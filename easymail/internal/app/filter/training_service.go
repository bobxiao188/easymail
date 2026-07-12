/*
 * Copyright (c) 2026 easymail.my. All rights reserved.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Affero General Public License for more details.
 *
 * For commercial licensing inquiries, please contact: 3680010825@qq.com
 *
 * Author: bob.xiao
 * License: AGPLv3
 */

// Training service: launches ad-hoc FastText supervised training jobs directly
// from the admin UI. It aggregates public samples (selected by source tag) into
// per-target-class training lines, then reuses the same FastText pipeline used by
// classify-model training. A ClassifyModelPO is created as the training output so
// the result appears in Model Management.

package filter

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"easymail/internal/domain/filter/classifier"
	"easymail/internal/infrastructure/cache"
	"easymail/internal/infrastructure/persistence/mysql"

	"gorm.io/gorm"
)

// Training-service specific errors.
var (
	ErrTrainingModelNameRequired = errors.New("training_model_name_required")
	ErrTrainingMinClasses        = errors.New("training_min_classes")
	ErrTrainingClassNameInvalid  = errors.New("training_class_name_invalid")
	ErrTrainingClassNameDuplicate = errors.New("training_class_name_duplicate")
	ErrTrainingNoTags            = errors.New("training_no_tags")
)

var trainingClassNameRE = regexp.MustCompile(`^[a-z]{1,10}$`)

// SourceGroup selects one category and one or more of its tags, with an optional
// per-group sample limit applied when aggregating training examples.
type SourceGroup struct {
	Category  string   `json:"category"`
	Tags      []string `json:"tags"`
	LimitType string   `json:"limitType"` // unlimited | random | first | last | middle
	LimitN    int      `json:"limitN"`
}

// TargetClassMapping maps one training target class to a set of source groups.
type TargetClassMapping struct {
	TargetClass string        `json:"targetClass"`
	Sources     []SourceGroup `json:"sources"`
}

// TrainingRequest is the input to StartTraining.
type TrainingRequest struct {
	ModelName      string
	Algorithm      string
	Params         classifier.ModelParams
	SampleMappings []TargetClassMapping
}

// TrainingService launches ad-hoc training jobs from the admin UI.
type TrainingService interface {
	StartTraining(ctx context.Context, req TrainingRequest) (*mysql.TrainingTaskPO, error)
	GetTraining(ctx context.Context, id uint) (*mysql.TrainingTaskPO, error)
}

type trainingService struct {
	db          *gorm.DB
	fastTextExe string
	modelRoot   string
	sampleRepo  *mysql.PublicSampleRepository
}

// NewTrainingService builds a TrainingService. fastTextExe / modelRoot mirror the
// classify-model training configuration.
func NewTrainingService(db *gorm.DB, fastTextExecutable, modelRoot string) TrainingService {
	return &trainingService{
		db:          db,
		fastTextExe: strings.TrimSpace(fastTextExecutable),
		modelRoot:   strings.TrimSpace(modelRoot),
		sampleRepo:  mysql.NewPublicSampleRepository(db),
	}
}

// validateTrainingMappings enforces: >=2 classes, each name is lowercase 1-10
// letters, unique, and has at least one non-empty source tag.
func validateTrainingMappings(mappings []TargetClassMapping) error {
	if len(mappings) < 2 {
		return ErrTrainingMinClasses
	}
	seen := make(map[string]struct{}, len(mappings))
	for _, m := range mappings {
		name := strings.TrimSpace(m.TargetClass)
		if name == "" || !trainingClassNameRE.MatchString(name) {
			return ErrTrainingClassNameInvalid
		}
		if _, ok := seen[name]; ok {
			return ErrTrainingClassNameDuplicate
		}
		seen[name] = struct{}{}
		if !classHasTags(m) {
			return ErrTrainingNoTags
		}
	}
	return nil
}

// classHasTags reports whether a mapping has at least one source group that
// contributes at least one non-empty tag.
func classHasTags(m TargetClassMapping) bool {
	for _, g := range m.Sources {
		if len(nonEmptyStrings(g.Tags)) > 0 {
			return true
		}
	}
	return false
}

func nonEmptyStrings(in []string) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		if t := strings.TrimSpace(s); t != "" {
			out = append(out, t)
		}
	}
	return out
}

// applySourceLimit trims the collected sample texts down to N according to the
// requested selection method. Non-positive N or "unlimited" returns all texts.
func applySourceLimit(texts []string, limitType string, n int) []string {
	if limitType == "" || limitType == "unlimited" || n <= 0 {
		return texts
	}
	switch limitType {
	case "random":
		rand.Shuffle(len(texts), func(i, j int) { texts[i], texts[j] = texts[j], texts[i] })
		if n < len(texts) {
			texts = texts[:n]
		}
		return texts
	case "first":
		if n > len(texts) {
			n = len(texts)
		}
		return texts[:n]
	case "last":
		if n > len(texts) {
			n = len(texts)
		}
		return texts[len(texts)-n:]
	case "middle":
		if n > len(texts) {
			n = len(texts)
		}
		start := (len(texts) - n) / 2
		if start < 0 {
			start = 0
		}
		return texts[start : start+n]
	default:
		return texts
	}
}

// StartTraining validates the request, persists a pending task, then runs the job.
func (s *trainingService) StartTraining(ctx context.Context, req TrainingRequest) (*mysql.TrainingTaskPO, error) {
	if strings.TrimSpace(s.fastTextExe) == "" {
		return nil, ErrFastTextExecutableNotConfigured
	}
	if strings.TrimSpace(s.modelRoot) == "" {
		return nil, ErrClassifyModelModelRootNotConfigured
	}
	if !strings.EqualFold(strings.TrimSpace(req.Algorithm), string(classifier.AlgorithmFastText)) {
		return nil, ErrClassifyModelTrainNotFastText
	}
	name := strings.TrimSpace(req.ModelName)
	if name == "" {
		return nil, ErrTrainingModelNameRequired
	}
	// Reuse the same name/feature-key validation as the classify-model CRUD.
	if err := ValidateClassifyModelDisplayName(ctx, s.db, name, 0); err != nil {
		return nil, err
	}
	if err := validateTrainingMappings(req.SampleMappings); err != nil {
		return nil, err
	}

	paramsJSON, _ := json.Marshal(req.Params)
	mappingsJSON, _ := json.Marshal(req.SampleMappings)
	task := &mysql.TrainingTaskPO{
		ModelName:      name,
		Algorithm:      string(classifier.AlgorithmFastText),
		Params:         string(paramsJSON),
		SampleMappings: string(mappingsJSON),
		Status:         string(classifier.TrainStatusPending),
		TrainResult:    "Training queued...\n",
	}
	if err := s.db.WithContext(ctx).Create(task).Error; err != nil {
		return nil, err
	}
	go s.runTrainJob(task.ID, name, req.Params, req.SampleMappings)
	return task, nil
}

// GetTraining returns a training task by ID.
func (s *trainingService) GetTraining(ctx context.Context, id uint) (*mysql.TrainingTaskPO, error) {
	var po mysql.TrainingTaskPO
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&po).Error; err != nil {
		return nil, err
	}
	return &po, nil
}

// runTrainJob performs the async training: aggregate samples -> write train.txt ->
// run fasttext -> create/persist the classify model. On failure the produced model
// is soft-deleted so it does not block future reruns and does not clutter listings.
func (s *trainingService) runTrainJob(taskID uint, modelName string, params classifier.ModelParams, mappings []TargetClassMapping) {
	ctx, cancel := context.WithTimeout(context.Background(), fastTextTrainTimeout)
	defer cancel()

	logBuf := &syncTrainLog{}
	updateTask := func(status classifier.TrainStatus, modelID uint) {
		upd := map[string]interface{}{
			"status":       string(status),
			"train_result": logBuf.String(),
		}
		if modelID > 0 {
			upd["model_id"] = modelID
		}
		_ = s.db.Session(&gorm.Session{}).Model(&mysql.TrainingTaskPO{}).Where("id = ?", taskID).Updates(upd).Error
	}

	// 1. Aggregate samples into FastText training lines grouped by target class.
	var lines []string
	for _, m := range mappings {
		var classLines []string
		for _, grp := range m.Sources {
			tags := nonEmptyStrings(grp.Tags)
			if len(tags) == 0 {
				continue
			}
			q := s.db.WithContext(ctx).Model(&mysql.PublicSamplePO{})
			if cat := strings.TrimSpace(grp.Category); cat != "" {
				q = q.Where("category_id = (SELECT id FROM public_sample_categories WHERE name = ?)", cat)
			}
			q = q.Where("tag IN ?", tags)
			var texts []string
			if err := q.Pluck("text", &texts).Error; err != nil {
				logBuf.Append(fmt.Sprintf("\n[error] load samples for class %q: %v\n", m.TargetClass, err))
				updateTask(classifier.TrainStatusFailed, 0)
				return
			}
			texts = applySourceLimit(texts, grp.LimitType, grp.LimitN)
			for _, t := range texts {
				body := strings.TrimSpace(t)
				if body == "" {
					continue
				}
				classLines = append(classLines, fastTextTrainingLine(m.TargetClass, body))
			}
		}
		lines = append(lines, classLines...)
	}
	if len(lines) == 0 {
		logBuf.Append("No samples found for the selected tags.\n")
		updateTask(classifier.TrainStatusFailed, 0)
		return
	}
	logBuf.Append(fmt.Sprintf("Collected %d samples across %d classes.\n", len(lines), len(mappings)))

	// 2. Create the classify model PO (training).
	labels := make([]string, 0, len(mappings))
	for _, m := range mappings {
		labels = append(labels, strings.TrimSpace(m.TargetClass))
	}
	paramsJSON, _ := json.Marshal(params)
	labelsJSON, _ := json.Marshal(labels)
	modelPO := &mysql.ClassifyModelPO{
		Name:          modelName,
		Algorithm:     string(classifier.AlgorithmFastText),
		Languages:     "[\"en\",\"zh\"]",
		Params:        string(paramsJSON),
		MaxTextLength: 256,
		EmailFields:   "[]",
		ClassLabels:   string(labelsJSON),
		Enabled:       false,
		TrainStatus:   string(classifier.TrainStatusRunning),
		TrainResult:   logBuf.String(),
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	if err := s.db.WithContext(ctx).Create(modelPO).Error; err != nil {
		logBuf.Append(fmt.Sprintf("\n[error] create model: %v\n", err))
		updateTask(classifier.TrainStatusFailed, 0)
		return
	}
	modelID := modelPO.ID
	updateTask(classifier.TrainStatusRunning, modelID)

	// 3. Prepare working directory and train.txt.
	absRoot, err := filepath.Abs(filepath.Clean(strings.TrimSpace(s.modelRoot)))
	if err != nil {
		logBuf.Append(fmt.Sprintf("\n[error] model root: %v\n", err))
		s.failModel(modelID, logBuf)
		updateTask(classifier.TrainStatusFailed, modelID)
		return
	}
	workDir := filepath.Join(absRoot, SanitizeClassifyModelDirStem(modelName))
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		logBuf.Append(fmt.Sprintf("\n[error] mkdir: %v\n", err))
		s.failModel(modelID, logBuf)
		updateTask(classifier.TrainStatusFailed, modelID)
		return
	}
	trainPath := filepath.Join(workDir, "train.txt")
	f, err := os.Create(trainPath)
	if err != nil {
		logBuf.Append(fmt.Sprintf("\n[error] create train file: %v\n", err))
		s.failModel(modelID, logBuf)
		updateTask(classifier.TrainStatusFailed, modelID)
		return
	}
	w := bufio.NewWriter(f)
	for _, line := range lines {
		if _, err := w.WriteString(line + "\n"); err != nil {
			f.Close()
			logBuf.Append(fmt.Sprintf("\n[error] write train file: %v\n", err))
			s.failModel(modelID, logBuf)
			updateTask(classifier.TrainStatusFailed, modelID)
			return
		}
	}
	if err := w.Flush(); err != nil {
		f.Close()
		logBuf.Append(fmt.Sprintf("\n[error] flush train file: %v\n", err))
		s.failModel(modelID, logBuf)
		updateTask(classifier.TrainStatusFailed, modelID)
		return
	}
	f.Close()

	// 4. Run fasttext supervised training.
	binPath := filepath.Join(workDir, fastTextTrainOutputPrefix+".bin")
	args := fasttextSupervisedArgList(params)
	logBuf.Append(fmt.Sprintf("$ %s %s\n", s.fastTextExe, strings.Join(args, " ")))

	cmd := exec.CommandContext(ctx, s.fastTextExe, args...)
	cmd.Dir = workDir
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		logBuf.Append(fmt.Sprintf("\n[error] stdout pipe: %v\n", err))
		s.failModel(modelID, logBuf)
		updateTask(classifier.TrainStatusFailed, modelID)
		return
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		logBuf.Append(fmt.Sprintf("\n[error] stderr pipe: %v\n", err))
		s.failModel(modelID, logBuf)
		updateTask(classifier.TrainStatusFailed, modelID)
		return
	}
	if err := cmd.Start(); err != nil {
		logBuf.Append(fmt.Sprintf("\n[error] start fasttext: %v\n", err))
		s.failModel(modelID, logBuf)
		updateTask(classifier.TrainStatusFailed, modelID)
		return
	}

	doneOut := make(chan struct{}, 1)
	doneErr := make(chan struct{}, 1)
	go streamPipe(logBuf, stdout, doneOut)
	go streamPipe(logBuf, stderr, doneErr)

	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	waitDone := make(chan error, 1)
	go func() { waitDone <- cmd.Wait() }()

Loop:
	for {
		select {
		case <-tick.C:
			_ = s.db.Session(&gorm.Session{}).Model(&mysql.TrainingTaskPO{}).
				Where("id = ?", taskID).Update("train_result", logBuf.String()).Error
		case err := <-waitDone:
			<-doneOut
			<-doneErr
			if err != nil {
				if errors.Is(ctx.Err(), context.DeadlineExceeded) {
					logBuf.Append("\n[error] training timed out\n")
				} else {
					logBuf.Append(fmt.Sprintf("\n[error] fasttext exit: %v\n", err))
				}
				s.failModel(modelID, logBuf)
				updateTask(classifier.TrainStatusFailed, modelID)
				return
			}
			break Loop
		}
	}

	if _, err := os.Stat(binPath); err != nil {
		logBuf.Append(fmt.Sprintf("\n[error] missing output %s: %v\n", binPath, err))
		s.failModel(modelID, logBuf)
		updateTask(classifier.TrainStatusFailed, modelID)
		return
	}

	absBin, _ := filepath.Abs(binPath)
	logBuf.Append(fmt.Sprintf("\n[ok] model written to %s\n", absBin))
	now := time.Now()
	_ = s.db.Session(&gorm.Session{}).Model(&mysql.ClassifyModelPO{}).
		Where("id = ? AND is_deleted = ?", modelID, false).
		Updates(map[string]interface{}{
			"save_path":    absBin,
			"train_status": string(classifier.TrainStatusCompleted),
			"train_result": logBuf.String(),
			"train_time":   &now,
		}).Error
	cache.InvalidateClassifyModelsCache()

	updateTask(classifier.TrainStatusCompleted, modelID)
}

// failModel marks a produced model as failed (and soft-deleted) so it does not
// block future reruns with the same name.
func (s *trainingService) failModel(modelID uint, logBuf *syncTrainLog) {
	if modelID == 0 {
		return
	}
	_ = s.db.Session(&gorm.Session{}).Model(&mysql.ClassifyModelPO{}).
		Where("id = ? AND is_deleted = ?", modelID, false).
		Updates(map[string]interface{}{
			"is_deleted":   true,
			"train_status": string(classifier.TrainStatusFailed),
			"train_result": logBuf.String(),
		}).Error
}


