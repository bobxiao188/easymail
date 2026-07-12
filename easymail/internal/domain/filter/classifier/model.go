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

package classifier

import (
	"strings"
	"time"
)

// Algorithm identifies the training/inference engine.
type Algorithm string

const (
	AlgorithmFastText   Algorithm = "FastText"
	AlgorithmDistilBERT Algorithm = "DistilBERT"
	AlgorithmXGBoost    Algorithm = "XGBoost"
)

// Language is an ISO 639-1 language code.
type Language string
type Languages []string

const (
	LanguageEnglish Language = "en"
	LanguageChinese Language = "zh"
)

// ClassLabels is the ordered distinct gold labels for a Model.
type ClassLabels []string

// Tokenizer identifies the tokenization strategy for preprocessing.
type Tokenizer string

const (
	TokenizerWordPiece               Tokenizer = "WordPiece"
	TokenizerGSE                     Tokenizer = "GSE"
	TokenizerDistilBERT              Tokenizer = "distilbert-base-cased"
	TokenizerDistilBERTMultiilingual Tokenizer = "distilbert-base-multilingual-cased"
)

// EmailField identifies an email part used for model input.
type EmailField string
type EmailFields []string

const (
	EmailFromName        EmailField = "from_name"
	EmailSubject         EmailField = "subject"
	EmailHtmlBody        EmailField = "html_body"
	EmailPlainTextBody   EmailField = "plain_text_body"
	EmailAttachmentNames EmailField = "attachment_names"
	EmailAttachBody      EmailField = "attach_body"
)

// ModelParams holds algorithm-specific hyperparameters (JSON-serializable).
type ModelParams struct {
	Algorithm           string   `json:"algorithm"`
	ConfigJSON          string   `json:"configJSON,omitempty"`
	ModelFile           string   `json:"modelFile,omitempty"`
	SpecialTokensMap    string   `json:"specialTokensMap,omitempty"`
	TokenizerJSON       string   `json:"tokenizerJSON,omitempty"`
	TokenizerConfigJSON string   `json:"tokenizerConfigJSON,omitempty"`
	VocabTXT            string   `json:"vocabTXT,omitempty"`
	LearningRate        *float64 `json:"learningRate,omitempty"`
	Epoch               *int     `json:"epoch,omitempty"`
	WordNgrams          *int     `json:"wordNgrams,omitempty"`
	Dim                 *int     `json:"dim,omitempty"`
	Loss                string   `json:"loss,omitempty"`
}

// TrainStatus tracks the lifecycle of a model training job.
type TrainStatus string

const (
	TrainStatusPending   TrainStatus = "pending"
	TrainStatusRunning   TrainStatus = "running"
	TrainStatusCompleted TrainStatus = "completed"
	TrainStatusFailed    TrainStatus = "failed"
)

// Model is a classifier model definition.
type Model struct {
	ID              uint          `json:"id"`
	Name            string        `json:"name"`
	Algorithm       Algorithm     `json:"algorithm"`
	Tokenizer       Tokenizer     `json:"tokenizer"`
	Languages       Languages     `json:"languages"`
	SavePath        string        `json:"savePath"`
	Params          ModelParams   `json:"params"`
	MaxTextLength   int           `json:"maxTextLength"`
	EmailFields     EmailFields   `json:"emailFields"`
	ClassLabels     ClassLabels   `json:"classLabels"`
	CreatedAt       time.Time     `json:"createdAt"`
	UpdatedAt       time.Time     `json:"updatedAt"`
	Enabled         bool          `json:"enabled"`
	ActivationReady bool          `json:"activationReady"`
	TrainStatus     TrainStatus   `json:"trainStatus"`
	TrainResult     string        `json:"trainResult"`
	TrainTime       time.Time     `json:"trainTime,omitempty"`
	IsDeleted       bool          `json:"isDeleted"`
	DeleteTime      time.Time     `json:"deleteTime"`
	CreatorId       int64         `json:"creatorId"`
}

// SanitizeFeatureKey maps Model.Name to the runtime feature key (lowercase; non [a-z0-9] -> '_').
func SanitizeFeatureKey(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	var b strings.Builder
	for _, r := range strings.ToLower(name) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
		if b.Len() >= 64 {
			break
		}
	}
	return strings.Trim(b.String(), "_")
}

// ModelScore is one classifier head output after merging multi-label distributions if needed.
type ModelScore struct {
	ModelID string
	Label   string
	Score   float64
}

// ModelMetadata describes a deployable classifier instance (DB row projection).
type ModelMetadata struct {
	ID          string
	Name        string
	Algorithm   Algorithm
	SavePath    string
	Enabled     bool
	ClassLabels []string
}
