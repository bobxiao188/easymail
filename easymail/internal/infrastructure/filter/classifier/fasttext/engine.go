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

// Package fasttext provides pure Go implementation for loading and predicting
// with FastText models trained in C++.
//
// This package parses the binary model format directly without requiring
// the original C++ library or any external dependencies.
package fasttext

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
)

const (
	// Magic number for fastText model files
	magicNumber = 793712314
	// Version 12 (1b)
	modelVersion = 12

	// Loss types
	lossHS      = 1
	lossNS      = 2
	lossSoftmax = 3
	lossOVA     = 4

	// Model types
	modelCBOW = 1
	modelSG   = 2
	modelSup  = 3
)

// LossName represents the loss function type
type LossName int

const (
	LossHierarchicalSoftmax LossName = lossHS
	LossNegativeSampling    LossName = lossNS
	LossSoftmax             LossName = lossSoftmax
	LossOneVsAll            LossName = lossOVA
)

// ModelName represents the model type
type ModelName int

const (
	ModelCBOW ModelName = modelCBOW
	ModelSG   ModelName = modelSG
	ModelSup  ModelName = modelSup
)

// Args holds the model parameters
type Args struct {
	Dim          int32
	Ws           int32
	Epoch        int32
	MinCount     int32
	Neg          int32
	WordNgrams   int32
	Loss         LossName
	Model        ModelName
	Bucket       int32
	Minn         int32
	Maxn         int32
	LrUpdateRate int32
	T            float64
}

// Entry represents a word or label entry in the dictionary
type Entry struct {
	Word     string
	Count    int64
	Type     int8 // 0 = word, 1 = label
	Subwords []int32
}

// Dictionary holds the vocabulary
type Dictionary struct {
	args     *Args
	words    []Entry
	word2int []int32
	nwords   int32
	nlabels  int32
	ntokens  int64
	size     int32
}

// Matrix represents a dense matrix of float32 values
type Matrix struct {
	m    int64 // rows
	n    int64 // cols
	data []float32
}

// Model represents a loaded fastText model
type Model struct {
	args   *Args
	dict   *Dictionary
	input  *Matrix
	output *Matrix
	quant  bool
	dim    int32
}

// Close closes the model and releases any resources.
// Currently a no-op since all data is in memory, but provided for API completeness.
func (m *Model) Close() error {
	return nil
}

// Prediction represents a single prediction result
type Prediction struct {
	Label string
	Prob  float64
}

// NewMatrix creates a new matrix with given dimensions
func NewMatrix(m, n int64) *Matrix {
	return &Matrix{
		m:    m,
		n:    n,
		data: make([]float32, m*n),
	}
}

// At returns the value at matrix[i][j]
func (m *Matrix) At(i, j int64) float32 {
	return m.data[i*m.n+j]
}

// Set sets the value at matrix[i][j]
func (m *Matrix) Set(i, j int64, v float32) {
	m.data[i*m.n+j] = v
}

// Rows returns the number of rows
func (m *Matrix) Rows() int64 {
	return m.m
}

// Cols returns the number of columns
func (m *Matrix) Cols() int64 {
	return m.n
}

// NewDictionary creates a new Dictionary
func NewDictionary(args *Args) *Dictionary {
	return &Dictionary{
		args:     args,
		word2int: make([]int32, 30000000),
	}
}

// LoadModel loads a fastText model from a file
func LoadModel(filename string) (*Model, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	return LoadModelFromReader(file)
}

// LoadModelFromReader loads a fastText model from an io.Reader
func LoadModelFromReader(r io.Reader) (*Model, error) {
	model := &Model{}

	// Read and verify magic number
	var magic int32
	if err := binary.Read(r, binary.LittleEndian, &magic); err != nil {
		return nil, fmt.Errorf("failed to read magic: %w", err)
	}
	if magic != magicNumber {
		return nil, fmt.Errorf("invalid magic number: got %d, want %d", magic, magicNumber)
	}

	// Read version
	var version int32
	if err := binary.Read(r, binary.LittleEndian, &version); err != nil {
		return nil, fmt.Errorf("failed to read version: %w", err)
	}
	if version > modelVersion {
		return nil, fmt.Errorf("unsupported model version: %d (max supported: %d)", version, modelVersion)
	}

	// Read Args
	model.args = &Args{}
	if err := readArgs(r, model.args); err != nil {
		return nil, fmt.Errorf("failed to read args: %w", err)
	}

	// Read Dictionary
	model.dict = &Dictionary{args: model.args}
	if err := model.dict.load(r); err != nil {
		return nil, fmt.Errorf("failed to read dictionary: %w", err)
	}

	// Read quant_ flag
	var quant bool
	if err := binary.Read(r, binary.LittleEndian, &quant); err != nil {
		return nil, fmt.Errorf("failed to read quant flag: %w", err)
	}
	model.quant = quant

	// Read input matrix
	model.input = &Matrix{}
	if err := model.input.load(r); err != nil {
		return nil, fmt.Errorf("failed to read input matrix: %w", err)
	}

	// Read qout_ flag
	var qout bool
	if err := binary.Read(r, binary.LittleEndian, &qout); err != nil {
		return nil, fmt.Errorf("failed to read qout flag: %w", err)
	}

	// Read output matrix
	model.output = &Matrix{}
	if err := model.output.load(r); err != nil {
		return nil, fmt.Errorf("failed to read output matrix: %w", err)
	}

	model.dim = model.args.Dim

	return model, nil
}

func readArgs(r io.Reader, args *Args) error {
	if err := binary.Read(r, binary.LittleEndian, &args.Dim); err != nil {
		return err
	}
	if err := binary.Read(r, binary.LittleEndian, &args.Ws); err != nil {
		return err
	}
	if err := binary.Read(r, binary.LittleEndian, &args.Epoch); err != nil {
		return err
	}
	if err := binary.Read(r, binary.LittleEndian, &args.MinCount); err != nil {
		return err
	}
	if err := binary.Read(r, binary.LittleEndian, &args.Neg); err != nil {
		return err
	}
	if err := binary.Read(r, binary.LittleEndian, &args.WordNgrams); err != nil {
		return err
	}
	// Loss and Model are enum types, read as int32 first then convert
	var loss int32
	if err := binary.Read(r, binary.LittleEndian, &loss); err != nil {
		return err
	}
	args.Loss = LossName(loss)
	var model int32
	if err := binary.Read(r, binary.LittleEndian, &model); err != nil {
		return err
	}
	args.Model = ModelName(model)
	if err := binary.Read(r, binary.LittleEndian, &args.Bucket); err != nil {
		return err
	}
	if err := binary.Read(r, binary.LittleEndian, &args.Minn); err != nil {
		return err
	}
	if err := binary.Read(r, binary.LittleEndian, &args.Maxn); err != nil {
		return err
	}
	if err := binary.Read(r, binary.LittleEndian, &args.LrUpdateRate); err != nil {
		return err
	}
	if err := binary.Read(r, binary.LittleEndian, &args.T); err != nil {
		return err
	}
	return nil
}

func (d *Dictionary) load(r io.Reader) error {
	if err := binary.Read(r, binary.LittleEndian, &d.size); err != nil {
		return err
	}
	if err := binary.Read(r, binary.LittleEndian, &d.nwords); err != nil {
		return err
	}
	if err := binary.Read(r, binary.LittleEndian, &d.nlabels); err != nil {
		return err
	}
	if err := binary.Read(r, binary.LittleEndian, &d.ntokens); err != nil {
		return err
	}

	var pruneidxSize int64
	if err := binary.Read(r, binary.LittleEndian, &pruneidxSize); err != nil {
		return err
	}

	d.words = make([]Entry, d.size)
	for i := int32(0); i < d.size; i++ {
		// Read word (null-terminated string)
		wordBytes, err := readNullTerminatedBytes(r)
		if err != nil {
			return err
		}
		d.words[i].Word = string(wordBytes)

		if err := binary.Read(r, binary.LittleEndian, &d.words[i].Count); err != nil {
			return err
		}
		if err := binary.Read(r, binary.LittleEndian, &d.words[i].Type); err != nil {
			return err
		}
	}

	// Skip pruneidx (not needed for prediction)
	for i := int64(0); i < pruneidxSize; i++ {
		var first, second int32
		if err := binary.Read(r, binary.LittleEndian, &first); err != nil {
			return err
		}
		if err := binary.Read(r, binary.LittleEndian, &second); err != nil {
			return err
		}
	}

	// Initialize word2int lookup table
	word2intSize := int32((float64(d.size) / 0.7))
	d.word2int = make([]int32, word2intSize)
	for i := range d.word2int {
		d.word2int[i] = -1
	}
	for i := int32(0); i < d.size; i++ {
		h := d.find(d.words[i].Word)
		d.word2int[h] = i
	}

	// Initialize subwords for each word
	d.initNgrams()

	return nil
}

func readNullTerminatedBytes(r io.Reader) ([]byte, error) {
	var result []byte
	for {
		b := make([]byte, 1)
		if _, err := r.Read(b); err != nil {
			return nil, err
		}
		if b[0] == 0 {
			return result, nil
		}
		result = append(result, b[0])
	}
}

// FNV-1a hash function (compatible with fastText)
func (d *Dictionary) hash(s string) uint32 {
	h := uint32(2166136261)
	for i := 0; i < len(s); i++ {
		h = h ^ uint32(int8(s[i]))
		h = h * 16777619
	}
	return h
}

func (d *Dictionary) find(s string) int32 {
	h := d.hash(s)
	size := int32(len(d.word2int))
	id := int32(h % uint32(size))
	for d.word2int[id] != -1 && d.words[d.word2int[id]].Word != s {
		id = (id + 1) % size
	}
	return id
}

func (d *Dictionary) initNgrams() {
	maxn := int(d.args.Maxn)
	minn := int(d.args.Minn)
	bucket := int(d.args.Bucket)

	for i := range d.words {
		word := d.words[i].Word
		if word == "</s>" {
			d.words[i].Subwords = []int32{int32(i)}
			continue
		}

		// Wrap with BOW and EOW
		concat := "<" + word + ">"

		subwords := []int32{int32(i)}
		// Generate character n-grams with proper UTF-8 handling
		runeIndices := runeIndicesInString(concat)

		for idx := 0; idx < len(runeIndices); idx++ {
			var ngram []byte
			charCount := 0
			for j := idx; j < len(runeIndices) && charCount < maxn; j++ {
				start := runeIndices[j]
				end := len(concat)
				if j+1 < len(runeIndices) {
					end = runeIndices[j+1]
				}
				ngram = append(ngram, concat[start:end]...)
				charCount++

				// Check boundary: first char of word (idx == 0) or last char of word
				isFirstChar := (start == 1)        // concat starts with "<"
				isLastChar := (end == len(concat)) // concat ends with ">"
				if charCount >= minn && !(charCount == 1 && (isFirstChar || isLastChar)) {
					h := d.hash(string(ngram)) % uint32(bucket)
					subwords = append(subwords, d.nwords+int32(h))
				}
			}
		}
		d.words[i].Subwords = subwords
	}
}

func (d *Dictionary) getId(word string) int32 {
	h := d.find(word)
	return d.word2int[h]
}

func (d *Dictionary) getSubwords(word string) []int32 {
	id := d.getId(word)
	if id >= 0 {
		return d.words[id].Subwords
	}

	// Out-of-vocabulary: compute subwords
	if word == "</s>" {
		return []int32{}
	}

	maxn := int(d.args.Maxn)
	minn := int(d.args.Minn)
	bucket := int(d.args.Bucket)

	concat := "<" + word + ">"
	subwords := []int32{}

	// Convert to rune indices for proper UTF-8 handling
	runeIndices := runeIndicesInString(concat)

	for i := 0; i < len(runeIndices); i++ {
		var ngram []byte
		charCount := 0
		for j := i; j < len(runeIndices) && charCount < maxn; j++ {
			start := runeIndices[j]
			end := len(concat)
			if j+1 < len(runeIndices) {
				end = runeIndices[j+1]
			}
			ngram = append(ngram, concat[start:end]...)
			charCount++

			// Check condition: ngram length >= minn AND NOT (single char at boundary)
			// Boundary means: first char of word (j == 0) OR last char of word (j == len(runeIndices)-1)
			// But we need to check if we're at the word boundary, not concat boundary
			isFirstChar := (start == 1)        // concat starts with "<"
			isLastChar := (end == len(concat)) // concat ends with ">"
			if charCount >= minn && !(charCount == 1 && (isFirstChar || isLastChar)) {
				h := d.hash(string(ngram)) % uint32(bucket)
				subwords = append(subwords, d.nwords+int32(h))
			}
		}
	}

	return subwords
}

// runeIndicesInString returns the starting byte indices of each rune in the string
func runeIndicesInString(s string) []int {
	indices := []int{0}
	for i := 0; i < len(s); {
		if s[i] >= 0x80 {
			// Multi-byte character, skip continuation bytes
			i++
			for i < len(s) && (s[i]&0xC0) == 0x80 {
				i++
			}
		} else {
			i++
		}
		if i < len(s) {
			indices = append(indices, i)
		}
	}
	return indices
}

func (d *Dictionary) getLabel(id int32) string {
	if id < 0 || id >= d.nlabels {
		return ""
	}
	return d.words[d.nwords+id].Word
}

func (m *Matrix) load(r io.Reader) error {
	if err := binary.Read(r, binary.LittleEndian, &m.m); err != nil {
		return err
	}
	if err := binary.Read(r, binary.LittleEndian, &m.n); err != nil {
		return err
	}
	m.data = make([]float32, m.m*m.n)
	if err := binary.Read(r, binary.LittleEndian, m.data); err != nil {
		return err
	}
	return nil
}

// GetWordVector returns the vector representation of a word
func (model *Model) GetWordVector(word string) []float32 {
	vec := make([]float32, model.dim)
	subwords := model.dict.getSubwords(word)

	if len(subwords) == 0 {
		return vec
	}

	for _, id := range subwords {
		for j := int32(0); j < model.dim; j++ {
			vec[j] += model.input.At(int64(id), int64(j))
		}
	}

	// Average
	invLen := 1.0 / float32(len(subwords))
	for j := int32(0); j < model.dim; j++ {
		vec[j] *= invLen
	}

	return vec
}

// PredictLine predicts labels for a single line of space-separated tokens (C++ fasttext convention).
func (model *Model) PredictLine(line string, k int32, threshold float32) ([]Prediction, error) {
	if model.args.Model != ModelSup {
		return nil, errors.New("model must be a supervised model for prediction")
	}

	words := WordsForInference(line)
	if len(words) == 0 {
		return []Prediction{}, nil
	}

	// Convert words to IDs and collect subwords
	var wordIds []int32
	for _, word := range words {
		ids := model.dict.getSubwords(word)
		wordIds = append(wordIds, ids...)
	}

	if len(wordIds) == 0 {
		return []Prediction{}, nil
	}

	// Compute input vector (average of word vectors) - use float64 for precision
	vec := make([]float64, model.dim)
	for _, id := range wordIds {
		for j := int32(0); j < model.dim; j++ {
			vec[j] += float64(model.input.At(int64(id), int64(j)))
		}
	}
	invLen := 1.0 / float64(len(wordIds))
	for j := int32(0); j < model.dim; j++ {
		vec[j] *= invLen
	}

	// Compute output scores using softmax
	predictions := model.computeSoftmaxFloat64(vec, k, threshold)

	return predictions, nil
}

// Predict predicts labels for a slice of words
func (model *Model) Predict(words []string, k int32, threshold float32) ([]Prediction, error) {
	if model.args.Model != ModelSup {
		return nil, errors.New("model must be a supervised model for prediction")
	}

	if len(words) == 0 {
		return []Prediction{}, nil
	}

	// Convert words to IDs and collect subwords
	var wordIds []int32
	for _, word := range words {
		ids := model.dict.getSubwords(word)
		wordIds = append(wordIds, ids...)
	}

	if len(wordIds) == 0 {
		return []Prediction{}, nil
	}

	// Compute input vector (average of word vectors) - use float64 for precision
	vec := make([]float64, model.dim)
	for _, id := range wordIds {
		for j := int32(0); j < model.dim; j++ {
			vec[j] += float64(model.input.At(int64(id), int64(j)))
		}
	}
	invLen := 1.0 / float64(len(wordIds))
	for j := int32(0); j < model.dim; j++ {
		vec[j] *= invLen
	}

	// Compute output scores using softmax
	predictions := model.computeSoftmaxFloat64(vec, k, threshold)

	return predictions, nil
}

// computeSoftmaxFloat64 computes softmax for float64 input vectors
func (model *Model) computeSoftmaxFloat64(input []float64, k int32, threshold float32) []Prediction {
	nlabels := int64(model.dict.nlabels)

	// Compute scores for all labels
	scores := make([]float64, nlabels)
	maxScore := float64(-1e20)

	for i := int64(0); i < nlabels; i++ {
		var sum float64
		for j := int32(0); j < model.dim; j++ {
			sum += input[j] * float64(model.output.At(i, int64(j)))
		}
		scores[i] = sum
		if sum > maxScore {
			maxScore = sum
		}
	}

	// Compute softmax
	expScores := make([]float64, nlabels)
	var expSum float64
	for i := int64(0); i < nlabels; i++ {
		expScores[i] = fastExp(scores[i] - maxScore)
		expSum += expScores[i]
	}

	// Normalize and convert to probabilities
	probs := make([]float64, nlabels)
	for i := int64(0); i < nlabels; i++ {
		probs[i] = expScores[i] / expSum
	}

	// Find top k predictions
	type probLabel struct {
		prob  float64
		label int64
	}
	var predictions []probLabel
	for i := int64(0); i < nlabels; i++ {
		if probs[i] >= float64(threshold) {
			predictions = append(predictions, probLabel{probs[i], i})
		}
	}

	// Sort by probability descending
	sort.Slice(predictions, func(i, j int) bool {
		return predictions[i].prob > predictions[j].prob
	})

	// Take top k
	if int32(len(predictions)) > k {
		predictions = predictions[:k]
	}

	// Convert to Prediction structs
	result := make([]Prediction, len(predictions))
	for i, p := range predictions {
		result[i] = Prediction{
			Label: model.dict.getLabel(int32(p.label)),
			Prob:  p.prob,
		}
	}

	return result
}

// GetDimension returns the embedding dimension
func (model *Model) GetDimension() int32 {
	return model.dim
}

// IsQuantized returns whether the model is quantized
func (model *Model) IsQuantized() bool {
	return model.quant
}

// GetArgs returns the model arguments
func (model *Model) GetArgs() *Args {
	return model.args
}

// GetNWords returns the number of words in the vocabulary
func (model *Model) GetNWords() int32 {
	return model.dict.nwords
}

// GetNLabels returns the number of labels
func (model *Model) GetNLabels() int32 {
	return model.dict.nlabels
}

// GetAllLabels returns all labels in the model
func (model *Model) GetAllLabels() []string {
	labels := make([]string, model.dict.nlabels)
	for i := int32(0); i < model.dict.nlabels; i++ {
		labels[i] = model.dict.getLabel(i)
	}
	return labels
}

// DictGetLabel returns the label at the given index
func (model *Model) DictGetLabel(id int32) string {
	return model.dict.getLabel(id)
}

// GetAllWords returns all words in the vocabulary
func (model *Model) GetAllWords() []string {
	words := make([]string, model.dict.nwords)
	for i := int32(0); i < model.dict.nwords; i++ {
		words[i] = model.dict.words[i].Word
	}
	return words
}

// DictGetSubwords returns the subword IDs for a list of words
func (model *Model) DictGetSubwords(words []string) []int32 {
	var wordIds []int32
	for _, word := range words {
		ids := model.dict.getSubwords(word)
		wordIds = append(wordIds, ids...)
	}
	return wordIds
}
