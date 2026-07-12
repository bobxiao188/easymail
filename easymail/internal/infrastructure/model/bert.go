/**
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

// Package model loads DistilBERT ONNX classifiers: a single .onnx with WordPiece + id2label embedded in metadata.
package model

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"runtime"
	"strings"
	"sync"
	"unicode"

	onnxruntime "github.com/yalue/onnxruntime_go"
)

var onnxInitOnce sync.Once
var onnxInitErr error

// InitONNXRuntime initializes the ONNX Runtime shared library once per process.
// libPath: absolute or relative path to onnxruntime.dll / libonnxruntime.so / .dylib.
// If empty, uses EASYMAIL_ONNXRUNTIME_PATH, then a default filename for the current OS (searched via OS loader path / cwd).
func InitONNXRuntime(libPath string) error {
	onnxInitOnce.Do(func() {
		p := strings.TrimSpace(libPath)
		if p == "" {
			p = strings.TrimSpace(os.Getenv("EASYMAIL_ONNXRUNTIME_PATH"))
		}
		if p == "" {
			switch runtime.GOOS {
			case "windows":
				p = "onnxruntime.dll"
			case "darwin":
				p = "libonnxruntime.dylib"
			default:
				p = "libonnxruntime.so"
			}
		}
		onnxruntime.SetSharedLibraryPath(p)
		onnxInitErr = onnxruntime.InitializeEnvironment()
	})
	return onnxInitErr
}

// TokenizerConfig mirrors HuggingFace tokenizer_config.json subset used for WordPiece.
type TokenizerConfig struct {
	DoLowerCase bool `json:"do_lower_case"`
	// TokenizeChineseChars matches HuggingFace BasicTokenizer.tokenize_chinese_chars.
	// When nil (JSON omitted), defaults to true like BertTokenizer.
	TokenizeChineseChars *bool  `json:"tokenize_chinese_chars,omitempty"`
	CLSToken             string `json:"cls_token"`
	SEPToken             string `json:"sep_token"`
	PadToken             string `json:"pad_token"`
	UNKToken             string `json:"unk_token"`
}

// WordPieceTokenizer loads vocab and tokenizer_config.json from explicit paths.
type WordPieceTokenizer struct {
	vocab      map[string]int
	config     TokenizerConfig
	clsID      int64
	sepID      int64
	unkID      int64
	padID      int64
	maxWordLen int
}

// NewWordPieceTokenizer loads tokenizer assets from tokenizerConfigPath and vocabPath.
func NewWordPieceTokenizer(vocabPath, tokenizerConfigPath string) (*WordPieceTokenizer, error) {
	cfgData, err := os.ReadFile(tokenizerConfigPath)
	if err != nil {
		return nil, fmt.Errorf("read tokenizer config %s: %w", tokenizerConfigPath, err)
	}
	vocabData, err := os.ReadFile(vocabPath)
	if err != nil {
		return nil, fmt.Errorf("read vocab %s: %w", vocabPath, err)
	}
	return NewWordPieceTokenizerFromBytes(vocabData, cfgData)
}

// NewWordPieceTokenizerFromBytes builds a WordPiece tokenizer from raw file contents (e.g. ONNX-embedded assets).
func NewWordPieceTokenizerFromBytes(vocabData, tokenizerConfigJSON []byte) (*WordPieceTokenizer, error) {
	var cfg TokenizerConfig
	if err := json.Unmarshal(tokenizerConfigJSON, &cfg); err != nil {
		return nil, fmt.Errorf("parse tokenizer config: %w", err)
	}

	vocab := make(map[string]int)
	lines := strings.Split(string(vocabData), "\n")
	for i, line := range lines {
		token := strings.TrimSpace(line)
		if token == "" {
			continue
		}
		vocab[token] = i
	}

	getID := func(t string) int64 {
		if id, ok := vocab[t]; ok {
			return int64(id)
		}
		return 100
	}

	t := &WordPieceTokenizer{
		vocab:      vocab,
		config:     cfg,
		clsID:      getID(cfg.CLSToken),
		sepID:      getID(cfg.SEPToken),
		unkID:      getID(cfg.UNKToken),
		padID:      getID(cfg.PadToken),
		maxWordLen: 200,
	}
	return t, nil
}

// isChineseChar mirrors HuggingFace BasicTokenizer._is_chinese_char (tokenization_bert.py).
func isChineseChar(r rune) bool {
	cp := uint32(r)
	return (cp >= 0x4E00 && cp <= 0x9FFF) ||
		(cp >= 0x3400 && cp <= 0x4DBF) ||
		(cp >= 0x20000 && cp <= 0x2A6DF) ||
		(cp >= 0x2A700 && cp <= 0x2B73F) ||
		(cp >= 0x2B740 && cp <= 0x2B81F) ||
		(cp >= 0x2B820 && cp <= 0x2CEAF) ||
		(cp >= 0xF900 && cp <= 0xFAFF) ||
		(cp >= 0x2F800 && cp <= 0x2FA1F)
}

// tokenizeChineseCharsPreprocess mirrors HuggingFace BasicTokenizer._tokenize_chinese_chars.
func tokenizeChineseCharsPreprocess(s string) string {
	var b strings.Builder
	b.Grow(len(s) + 32)
	for _, r := range s {
		if isChineseChar(r) {
			b.WriteByte(' ')
			b.WriteRune(r)
			b.WriteByte(' ')
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func (t *WordPieceTokenizer) tokenizeChineseCharsEnabled() bool {
	if t == nil {
		return true
	}
	if t.config.TokenizeChineseChars != nil {
		return *t.config.TokenizeChineseChars
	}
	return true
}

func (t *WordPieceTokenizer) basicTokenize(text string) []string {
	text = strings.Map(func(r rune) rune {
		if r == 0 || r == 0xFFFD || isControl(r) {
			return -1
		}
		if unicode.IsSpace(r) {
			return ' '
		}
		return r
	}, text)
	if t.tokenizeChineseCharsEnabled() {
		text = tokenizeChineseCharsPreprocess(text)
	}
	text = strings.TrimSpace(text)

	var tokens []string
	current := make([]rune, 0, len(text))

	flush := func() {
		if len(current) > 0 {
			s := string(current)
			if t.config.DoLowerCase {
				s = strings.ToLower(s)
			}
			tokens = append(tokens, s)
			current = current[:0]
		}
	}

	for _, r := range text {
		if unicode.IsSpace(r) {
			flush()
		} else if unicode.IsPunct(r) {
			flush()
			tokens = append(tokens, string(r))
		} else {
			current = append(current, r)
		}
	}
	flush()
	return tokens
}

func isControl(r rune) bool {
	if unicode.IsControl(r) {
		return true
	}
	return (r >= 0x0000 && r <= 0x001F && r != '\t' && r != '\n' && r != '\r') ||
		(r >= 0x007F && r <= 0x009F)
}

func (t *WordPieceTokenizer) wordPiece(token string) []int64 {
	if len(token) > t.maxWordLen {
		return []int64{t.unkID}
	}

	tokens := make([]int64, 0)
	start := 0
	for start < len(token) {
		end := len(token)
		found := false
		for ; start < end; end-- {
			substr := token[start:end]
			if start > 0 {
				substr = "##" + substr
			}
			if id, ok := t.vocab[substr]; ok {
				tokens = append(tokens, int64(id))
				found = true
				break
			}
		}
		if !found {
			return []int64{t.unkID}
		}
		start = end
	}
	return tokens
}

// Tokenize builds input_ids and attention_mask of length maxLen (padded with PAD).
func (t *WordPieceTokenizer) Tokenize(text string, maxLen int) ([]int64, []int64) {
	if maxLen < 8 {
		maxLen = 8
	}
	ids := make([]int64, maxLen)
	mask := make([]int64, maxLen)
	for i := range ids {
		ids[i] = t.padID
	}

	pos := 0
	ids[pos] = t.clsID
	mask[pos] = 1
	pos++

	words := t.basicTokenize(text)
	for _, word := range words {
		subTokens := t.wordPiece(word)
		for _, id := range subTokens {
			if pos >= maxLen-1 {
				goto done
			}
			ids[pos] = id
			mask[pos] = 1
			pos++
		}
	}

done:
	if pos < maxLen {
		ids[pos] = t.sepID
		mask[pos] = 1
	}
	return ids, mask
}

type onnxBertSession struct {
	inner    *onnxruntime.AdvancedSession
	inputIDs *onnxruntime.Tensor[int64]
	mask     *onnxruntime.Tensor[int64]
	logits   *onnxruntime.Tensor[float32]
	seqLen   int
	numCls   int
}

func inferLogitsNumClasses(modelPath string, fallback int) (int, error) {
	_, outputs, err := onnxruntime.GetInputOutputInfoWithOptions(modelPath, nil)
	if err != nil {
		if fallback > 0 {
			return fallback, nil
		}
		return 0, fmt.Errorf("onnx output info: %w", err)
	}
	for _, o := range outputs {
		if o.Name != "logits" {
			continue
		}
		d := o.Dimensions
		if len(d) < 2 {
			break
		}
		last := int(d[len(d)-1])
		if last > 0 {
			return last, nil
		}
	}
	if fallback > 0 {
		return fallback, nil
	}
	return 0, errors.New("onnx: could not infer logits class count")
}

func newONNXBertSession(modelPath string, seqLen int, numClasses int) (*onnxBertSession, error) {
	if seqLen < 8 || seqLen > 512 {
		return nil, fmt.Errorf("seqLen must be in [8,512], got %d", seqLen)
	}
	if numClasses < 2 {
		return nil, fmt.Errorf("numClasses must be >= 2, got %d", numClasses)
	}
	shape := onnxruntime.Shape{1, int64(seqLen)}
	outShape := onnxruntime.Shape{1, int64(numClasses)}

	inIDs, err := onnxruntime.NewTensor(shape, make([]int64, seqLen))
	if err != nil {
		return nil, err
	}
	inMask, err := onnxruntime.NewTensor(shape, make([]int64, seqLen))
	if err != nil {
		inIDs.Destroy()
		return nil, err
	}
	outLogits, err := onnxruntime.NewTensor(outShape, make([]float32, numClasses))
	if err != nil {
		inIDs.Destroy()
		inMask.Destroy()
		return nil, err
	}

	inNames := []string{"input_ids", "attention_mask"}
	outNames := []string{"logits"}

	session, err := onnxruntime.NewAdvancedSession(
		modelPath,
		inNames, outNames,
		[]onnxruntime.Value{inIDs, inMask},
		[]onnxruntime.Value{outLogits},
		nil,
	)
	if err != nil {
		inIDs.Destroy()
		inMask.Destroy()
		outLogits.Destroy()
		return nil, err
	}

	return &onnxBertSession{
		inner:    session,
		inputIDs: inIDs,
		mask:     inMask,
		logits:   outLogits,
		seqLen:   seqLen,
		numCls:   numClasses,
	}, nil
}

func (s *onnxBertSession) run(ids, mask []int64) ([]float32, error) {
	copy(s.inputIDs.GetData(), ids)
	copy(s.mask.GetData(), mask)
	if err := s.inner.Run(); err != nil {
		return nil, err
	}
	out := make([]float32, s.numCls)
	copy(out, s.logits.GetData())
	return out, nil
}

func (s *onnxBertSession) destroy() {
	if s.inner != nil {
		_ = s.inner.Destroy()
		s.inner = nil
	}
	if s.inputIDs != nil {
		_ = s.inputIDs.Destroy()
		s.inputIDs = nil
	}
	if s.mask != nil {
		_ = s.mask.Destroy()
		s.mask = nil
	}
	if s.logits != nil {
		_ = s.logits.Destroy()
		s.logits = nil
	}
}

func softmax32(logits []float32) []float64 {
	if len(logits) == 0 {
		return nil
	}
	out := make([]float64, len(logits))
	var max float64
	for i, v := range logits {
		fv := float64(v)
		if i == 0 || fv > max {
			max = fv
		}
	}
	var sum float64
	for i, v := range logits {
		out[i] = math.Exp(float64(v) - max)
		sum += out[i]
	}
	if sum == 0 {
		return out
	}
	for i := range out {
		out[i] /= sum
	}
	return out
}

func argmax(probs []float64) int {
	best := 0
	for i := 1; i < len(probs); i++ {
		if probs[i] > probs[best] {
			best = i
		}
	}
	return best
}

func labelForClassIndex(idx int, numCls int) string {
	if numCls == 2 && idx == 0 {
		return "ham"
	}
	if numCls == 2 && idx == 1 {
		return "spam"
	}
	return fmt.Sprintf("class_%d", idx)
}

// DistilBERTONNXEngine runs a DistilBERT ONNX classifier with WordPiece tokenization.
type DistilBERTONNXEngine struct {
	tokenizer *WordPieceTokenizer
	session   *onnxBertSession
	maxLen    int
	// classLabels from ONNX metadata id2label (index order matches logits); nil uses labelForClassIndex heuristic.
	classLabels []string
	mu          sync.Mutex
}

func countClassesFromID2LabelJSON(s string) int {
	if strings.TrimSpace(s) == "" {
		return 0
	}
	var m map[string]string
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		return 0
	}
	max := 0
	for k := range m {
		var i int
		if _, err := fmt.Sscanf(k, "%d", &i); err != nil {
			continue
		}
		if i+1 > max {
			max = i + 1
		}
	}
	return max
}

func parseID2LabelJSON(s string, numClasses int) []string {
	if strings.TrimSpace(s) == "" || numClasses < 1 {
		return nil
	}
	var m map[string]string
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		return nil
	}
	out := make([]string, numClasses)
	for i := 0; i < numClasses; i++ {
		key := fmt.Sprintf("%d", i)
		if lab, ok := m[key]; ok && lab != "" {
			out[i] = lab
		} else {
			out[i] = labelForClassIndex(i, numClasses)
		}
	}
	return out
}

// NewDistilBERTONNXEngine loads vocab + tokenizer_config + id2label from ONNX custom metadata
// (scripts/train/distiBERT/train.py inject_onnx_metadata). Only the .onnx file is required on disk.
func NewDistilBERTONNXEngine(onnxPath string, maxSeqLen int) (*DistilBERTONNXEngine, error) {
	if !onnxruntime.IsInitialized() {
		return nil, errors.New("ONNX runtime not initialized; call model.InitONNXRuntime first")
	}
	modelPath := strings.TrimSpace(onnxPath)
	if modelPath == "" {
		return nil, errors.New("empty onnx path")
	}
	if st, err := os.Stat(modelPath); err != nil || st.IsDir() {
		return nil, fmt.Errorf("onnx model not found: %s", modelPath)
	}
	seqLen := clampBertSeqLen(maxSeqLen)

	meta, err := onnxruntime.GetModelMetadata(modelPath)
	if err != nil {
		return nil, fmt.Errorf("onnx metadata: %w", err)
	}
	defer func() { _ = meta.Destroy() }()

	vocab, ok, err := meta.LookupCustomMetadataMap("vocab_txt")
	if err != nil || !ok || vocab == "" {
		return nil, errors.New("onnx missing metadata vocab_txt (re-export with train.py inject_onnx_metadata)")
	}
	tokJSON, ok, err := meta.LookupCustomMetadataMap("tokenizer_config_json")
	if err != nil || !ok || tokJSON == "" {
		return nil, errors.New("onnx missing metadata tokenizer_config_json")
	}
	id2JSON, _, _ := meta.LookupCustomMetadataMap("id2label")
	if ms, ok2, _ := meta.LookupCustomMetadataMap("max_seq_len"); ok2 && ms != "" {
		var m int
		if _, err := fmt.Sscanf(ms, "%d", &m); err == nil && m > 0 {
			seqLen = clampBertSeqLen(m)
		}
	}

	tok, err := NewWordPieceTokenizerFromBytes([]byte(vocab), []byte(tokJSON))
	if err != nil {
		return nil, err
	}
	ncGuess := countClassesFromID2LabelJSON(id2JSON)
	nc, err := inferLogitsNumClasses(modelPath, ncGuess)
	if err != nil {
		return nil, err
	}
	if nc < 2 && ncGuess >= 2 {
		nc = ncGuess
	}
	if nc < 2 {
		return nil, fmt.Errorf("invalid class count %d from onnx/metadata", nc)
	}
	labels := parseID2LabelJSON(id2JSON, nc)
	sess, err := newONNXBertSession(modelPath, seqLen, nc)
	if err != nil {
		return nil, err
	}
	return &DistilBERTONNXEngine{
		tokenizer:   tok,
		session:     sess,
		maxLen:      seqLen,
		classLabels: labels,
	}, nil
}

func clampBertSeqLen(n int) int {
	if n <= 0 {
		return 128
	}
	if n < 8 {
		return 8
	}
	if n > 512 {
		return 512
	}
	return n
}

// Close releases ONNX session and tensors.
func (e *DistilBERTONNXEngine) Close() {
	if e == nil || e.session == nil {
		return
	}
	e.session.destroy()
	e.session = nil
}

// PredictProbs returns softmax probabilities for every output class (index order matches the ONNX head).
func (e *DistilBERTONNXEngine) PredictProbs(text string) (labels []string, probs []float64, err error) {
	if e == nil || e.tokenizer == nil || e.session == nil {
		return nil, nil, errors.New("distilbert onnx: engine not initialized")
	}
	e.mu.Lock()
	defer e.mu.Unlock()

	ids, mask := e.tokenizer.Tokenize(text, e.session.seqLen)
	logits, err := e.session.run(ids, mask)
	if err != nil {
		return nil, nil, err
	}
	p := softmax32(logits)
	if len(p) == 0 {
		return nil, nil, errors.New("empty logits")
	}
	n := len(p)
	labels = make([]string, n)
	for i := 0; i < n; i++ {
		if e.classLabels != nil && i < len(e.classLabels) && e.classLabels[i] != "" {
			labels[i] = e.classLabels[i]
		} else {
			labels[i] = labelForClassIndex(i, n)
		}
	}
	probs = make([]float64, n)
	copy(probs, p)
	return labels, probs, nil
}

// Predict returns the winning label and its probability (softmax).
func (e *DistilBERTONNXEngine) Predict(text string) (label string, prob float64, err error) {
	labels, probs, err := e.PredictProbs(text)
	if err != nil {
		return "", 0, err
	}
	idx := argmax(probs)
	return labels[idx], probs[idx], nil
}

// ExtractDistilBERTClassLabelsFromONNX reads class label strings in logit index order from ONNX custom metadata id2label
// and the logits output shape (see scripts/train/distiBERT/train.py inject_onnx_metadata).
// onnxRuntimeLib is passed to InitONNXRuntime (empty uses EASYMAIL_ONNXRUNTIME_PATH / OS default library name).
func ExtractDistilBERTClassLabelsFromONNX(onnxPath, onnxRuntimeLib string) ([]string, error) {
	if err := InitONNXRuntime(onnxRuntimeLib); err != nil {
		return nil, fmt.Errorf("onnx runtime init: %w", err)
	}
	modelPath := strings.TrimSpace(onnxPath)
	if modelPath == "" {
		return nil, errors.New("empty onnx path")
	}
	if st, err := os.Stat(modelPath); err != nil || st.IsDir() {
		return nil, fmt.Errorf("onnx model not found: %s", modelPath)
	}

	meta, err := onnxruntime.GetModelMetadata(modelPath)
	if err != nil {
		return nil, fmt.Errorf("onnx metadata: %w", err)
	}
	defer func() { _ = meta.Destroy() }()

	id2JSON, _, _ := meta.LookupCustomMetadataMap("id2label")
	ncGuess := countClassesFromID2LabelJSON(id2JSON)
	nc, err := inferLogitsNumClasses(modelPath, ncGuess)
	if err != nil {
		return nil, err
	}
	if nc < 2 && ncGuess >= 2 {
		nc = ncGuess
	}
	if nc < 2 {
		return nil, fmt.Errorf("invalid class count %d from onnx/metadata", nc)
	}
	labels := parseID2LabelJSON(id2JSON, nc)
	if len(labels) != nc {
		return nil, errors.New("internal: label slice length mismatch")
	}
	return labels, nil
}

