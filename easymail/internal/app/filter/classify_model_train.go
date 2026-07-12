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

package filter

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"easymail/internal/domain/filter/classifier"
	"easymail/internal/infrastructure/cache"
	"easymail/internal/infrastructure/filter/classifier/fasttext"
	"easymail/internal/infrastructure/persistence/mysql"

	"gorm.io/gorm"
)

const (
	fastTextTrainLogMaxRunes = 256 * 1024
	fastTextTrainTimeout     = 4 * time.Hour
	// fastTextTrainOutputPrefix is the fasttext -output basename under model_root/<model_name>/ (model.bin, model.vec).
	fastTextTrainOutputPrefix = "model"
)

type syncTrainLog struct {
	mu sync.Mutex
	b  strings.Builder
}

func (w *syncTrainLog) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	n, err := w.b.Write(p)
	w.trimLocked()
	return n, err
}

func (w *syncTrainLog) trimLocked() {
	s := w.b.String()
	if len([]rune(s)) <= fastTextTrainLogMaxRunes {
		return
	}
	r := []rune(s)
	s = string(r[len(r)-fastTextTrainLogMaxRunes:])
	w.b.Reset()
	w.b.WriteString(s)
}

func (w *syncTrainLog) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.b.String()
}

func (w *syncTrainLog) Append(msg string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.b.WriteString(msg)
	w.trimLocked()
}

func (s *classifyModelService) listAllModelSamples(ctx context.Context, modelID uint) ([]classifier.Sample, error) {
	var pos []mysql.ModelSamplePO
	if err := s.db.WithContext(ctx).Model(&mysql.ModelSamplePO{}).Where("classify_model_id = ?", modelID).Order("id ASC").Find(&pos).Error; err != nil {
		return nil, err
	}
	rows := make([]classifier.Sample, len(pos))
	for i := range pos {
		rows[i] = *mysql.PoToModelSample(&pos[i])
	}
	return rows, nil
}

func fastTextTrainingLine(label, text string) string {
	lab := strings.ReplaceAll(strings.TrimSpace(label), " ", "_")
	lab = strings.ReplaceAll(lab, "\t", "_")
	body := fasttext.FastTextSupervisedBody(text)
	return "__label__" + lab + " " + body
}

// SanitizeClassifyModelDirStem maps a display name to a directory name under model_root (shared with FastText / DistilBERT ONNX layout).
func SanitizeClassifyModelDirStem(name string) string {
	var b strings.Builder
	for _, r := range strings.TrimSpace(name) {
		if r < 32 || strings.ContainsRune(`<>:"/\|?*`, r) {
			b.WriteRune('_')
		} else {
			b.WriteRune(r)
		}
	}
	out := strings.Trim(strings.TrimSpace(b.String()), " .")
	if out == "" {
		return "model"
	}
	return out
}

// fastTextTrainWorkDir returns the directory model_root/<sanitized_model_name>/ and the final .bin path (model.bin).
func (s *classifyModelService) fastTextTrainWorkDir(m *classifier.Model) (workDir string, binPath string, err error) {
	root := strings.TrimSpace(s.modelRoot)
	if root == "" {
		return "", "", ErrClassifyModelModelRootNotConfigured
	}
	absRoot, err := filepath.Abs(filepath.Clean(root))
	if err != nil {
		return "", "", err
	}
	sub := SanitizeClassifyModelDirStem(m.Name)
	workDir = filepath.Join(absRoot, sub)
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		return "", "", err
	}
	binPath = filepath.Join(workDir, fastTextTrainOutputPrefix+".bin")
	return workDir, binPath, nil
}

func fasttextSupervisedArgList(p classifier.ModelParams) []string {
	args := []string{
		"supervised",
		"-input", "train.txt",
		"-output", fastTextTrainOutputPrefix,
	}
	if p.LearningRate != nil && *p.LearningRate > 0 {
		args = append(args, "-lr", trimFloatStr(*p.LearningRate))
	}

	if p.Epoch != nil && *p.Epoch > 0 {
		args = append(args, "-epoch", strconv.Itoa(*p.Epoch))
	}

	if p.WordNgrams != nil && *p.WordNgrams > 0 {
		args = append(args, "-wordNgrams", strconv.Itoa(*p.WordNgrams))
	}

	if p.Dim != nil && *p.Dim > 0 {
		args = append(args, "-dim", strconv.Itoa(*p.Dim))
	}
	if ls := strings.TrimSpace(p.Loss); ls != "" {
		args = append(args, "-loss", ls)
	}
	return args
}

func trimFloatStr(f float64) string {
	s := strconv.FormatFloat(f, 'g', -1, 64)
	return s
}

func streamPipe(w io.Writer, r io.Reader, done chan<- struct{}) {
	defer func() { done <- struct{}{} }()
	io.Copy(w, r)
}

// StartFastTextTraining validates configuration and runs fasttext supervised asynchronously.
func (s *classifyModelService) StartFastTextTraining(ctx context.Context, id int64) error {
	if strings.TrimSpace(s.fastTextExe) == "" {
		return ErrFastTextExecutableNotConfigured
	}
	if strings.TrimSpace(s.modelRoot) == "" {
		return ErrClassifyModelModelRootNotConfigured
	}
	mid := uint(id)
	if _, loaded := s.trainRunning.LoadOrStore(mid, struct{}{}); loaded {
		return ErrClassifyModelTrainAlreadyRunning
	}
	cleanupMap := true
	defer func() {
		if cleanupMap {
			s.trainRunning.Delete(mid)
		}
	}()

	m, err := s.loadClassifyModelByID(ctx, id)
	if err != nil {
		return err
	}
	if m.Algorithm != classifier.AlgorithmFastText {
		return ErrClassifyModelTrainNotFastText
	}
	samples, err := s.listAllModelSamples(ctx, mid)
	if err != nil {
		return err
	}
	if len(samples) == 0 {
		return ErrClassifyModelTrainNoSamples
	}

	if err := s.db.WithContext(ctx).Model(&mysql.ClassifyModelPO{}).
		Where("id = ? AND is_deleted = ?", mid, false).
		Updates(map[string]interface{}{
			"train_status": string(classifier.TrainStatusRunning),
			"train_result": fmt.Sprintf("Starting training (%d samples)...\n", len(samples)),
		}).Error; err != nil {
		return err
	}
	cache.InvalidateClassifyModelsCache()
	cleanupMap = false
	go func() {
		defer s.trainRunning.Delete(mid)
		s.runFastTextTrainJob(mid, samples)
	}()
	return nil
}

func (s *classifyModelService) runFastTextTrainJob(mid uint, samples []classifier.Sample) {
	ctx, cancel := context.WithTimeout(context.Background(), fastTextTrainTimeout)
	defer cancel()

	logBuf := &syncTrainLog{}
	finish := func(status classifier.TrainStatus, tailMsg string) {
		now := time.Now()
		logBuf.Append(tailMsg)
		_ = s.db.Session(&gorm.Session{}).Model(&mysql.ClassifyModelPO{}).
			Where("id = ? AND is_deleted = ?", mid, false).
			Updates(map[string]interface{}{
				"train_status": string(status),
				"train_result": logBuf.String(),
				"train_time":   &now,
			}).Error
		cache.InvalidateClassifyModelsCache()
		cache.InvalidateClassifyModelsCache()
	}

	m, err := s.loadClassifyModelByID(context.Background(), int64(mid))
	if err != nil {
		finish(classifier.TrainStatusFailed, fmt.Sprintf("\n[error] load model: %v\n", err))
		return
	}
	workDir, targetBin, err := s.fastTextTrainWorkDir(m)
	if err != nil {
		finish(classifier.TrainStatusFailed, fmt.Sprintf("\n[error] train directory: %v\n", err))
		return
	}
	logBuf.Append(fmt.Sprintf("work_dir=%s\n", workDir))

	trainPath := filepath.Join(workDir, "train.txt")
	f, err := os.Create(trainPath)
	if err != nil {
		finish(classifier.TrainStatusFailed, fmt.Sprintf("\n[error] train file: %v\n", err))
		return
	}
	w := bufio.NewWriter(f)
	for _, row := range samples {
		line := fastTextTrainingLine(row.Label, row.Text)
		if _, err := w.WriteString(line + "\n"); err != nil {
			f.Close()
			finish(classifier.TrainStatusFailed, fmt.Sprintf("\n[error] write train line: %v\n", err))
			return
		}
	}
	if err := w.Flush(); err != nil {
		f.Close()
		finish(classifier.TrainStatusFailed, fmt.Sprintf("\n[error] flush train file: %v\n", err))
		return
	}
	if err := f.Close(); err != nil {
		finish(classifier.TrainStatusFailed, fmt.Sprintf("\n[error] close train file: %v\n", err))
		return
	}

	args := fasttextSupervisedArgList(m.Params)
	logBuf.Append(fmt.Sprintf("$ %s %s\n", s.fastTextExe, strings.Join(args, " ")))

	cmd := exec.CommandContext(ctx, s.fastTextExe, args...)
	cmd.Dir = workDir

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		finish(classifier.TrainStatusFailed, fmt.Sprintf("\n[error] stdout pipe: %v\n", err))
		return
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		finish(classifier.TrainStatusFailed, fmt.Sprintf("\n[error] stderr pipe: %v\n", err))
		return
	}

	if err := cmd.Start(); err != nil {
		finish(classifier.TrainStatusFailed, fmt.Sprintf("\n[error] start fasttext: %v\n", err))
		return
	}

	doneOut := make(chan struct{}, 1)
	doneErr := make(chan struct{}, 1)
	go streamPipe(logBuf, stdout, doneOut)
	go streamPipe(logBuf, stderr, doneErr)

	tick := time.NewTicker(time.Second)
	defer tick.Stop()

	waitDone := make(chan error, 1)
	go func() {
		waitDone <- cmd.Wait()
	}()

Loop:
	for {
		select {
		case <-tick.C:
			_ = s.db.Session(&gorm.Session{}).Model(&mysql.ClassifyModelPO{}).
				Where("id = ? AND is_deleted = ?", mid, false).
				Update("train_result", logBuf.String()).Error
		case err := <-waitDone:
			<-doneOut
			<-doneErr
			if err != nil {
				if errors.Is(ctx.Err(), context.DeadlineExceeded) {
					logBuf.Append("\n[error] training timed out\n")
				} else {
					logBuf.Append(fmt.Sprintf("\n[error] fasttext exit: %v\n", err))
				}
				finish(classifier.TrainStatusFailed, "")
				return
			}
			break Loop
		}
	}

	if _, err := os.Stat(targetBin); err != nil {
		finish(classifier.TrainStatusFailed, fmt.Sprintf("\n[error] missing output %s: %v\n", targetBin, err))
		return
	}

	now := time.Now()
	absBin, _ := filepath.Abs(targetBin)
	logBuf.Append(fmt.Sprintf("\n[ok] model written to %s\n", absBin))
	_ = s.db.Session(&gorm.Session{}).Model(&mysql.ClassifyModelPO{}).
		Where("id = ? AND is_deleted = ?", mid, false).
		Updates(map[string]interface{}{
			"save_path":    absBin,
			"train_status": string(classifier.TrainStatusCompleted),
			"train_result": logBuf.String(),
			"train_time":   &now,
		}).Error
	slog.Info("fasttext train completed", "classify_model_id", mid, "bin", absBin)
	if err := s.recomputeAndPersistClassLabels(context.Background(), s.db.Session(&gorm.Session{}), mid, true); err != nil {
		slog.Warn("classify_model sync class_labels after train", "classify_model_id", mid, "err", err)
	}
	cache.InvalidateClassifyModelsCache()
	cache.InvalidateClassifyModelsCache()
}
