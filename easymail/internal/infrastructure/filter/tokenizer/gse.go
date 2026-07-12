/*
 * Copyright (c) 2026 easymail.my. All rights reserved.
 *
 * License: AGPLv3
 */

package tokenizer

import (
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/go-ego/gse"
)

var gseSeg gse.Segmenter

// InitGSE loads GSE dictionaries from the given dictDir.
// The expected directory structure under dictDir is:
//
//	dictDir/zh/t_1.txt          — main Chinese dictionary
//	dictDir/zh/s_1.txt          — supplementary dictionary
//	dictDir/zh/stop_tokens.txt  — stop tokens
//	dictDir/zh/stop_word.txt    — stop words
//
// If dictDir does not exist, falls back to gse.LoadDict() (embedded defaults).
// Safe to call multiple times; subsequent calls reload the segmenter.
func InitGSE(dictDir string) error {
	zhDir := filepath.Join(dictDir, "zh")

	if fi, err := os.Stat(zhDir); err != nil || !fi.IsDir() {
		// Fall back to embedded default dictionaries
		return gseSeg.LoadDict()
	}

	var dictPaths []string

	// Main dictionary files
	for _, name := range []string{"t_1.txt", "s_1.txt"} {
		p := filepath.Join(zhDir, name)
		if _, err := os.Stat(p); err == nil {
			dictPaths = append(dictPaths, p)
		}
	}

	if len(dictPaths) == 0 {
		// No custom dictionary files found, fall back to embedded
		return gseSeg.LoadDict()
	}

	if err := gseSeg.LoadDict(dictPaths...); err != nil {
		return err
	}

	// Load stop words
	for _, name := range []string{"stop_tokens.txt", "stop_word.txt"} {
		p := filepath.Join(zhDir, name)
		if _, err := os.Stat(p); err == nil {
			_ = gseSeg.LoadStop(p)
		}
	}

	return nil
}

// GSEWords runs go-ego/gse on the full string, then drops whitespace-only and all-punctuation tokens.
// Used by FastText supervised training lines and inference so segmentation stays aligned.
func GSEWords(line string) []string {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil
	}
	parts := gseSeg.Segment([]byte(line))
	var words []string
	for _, p := range parts {
		word := strings.TrimSpace(p.Token().Text())
		if word == "" || word == " " || isPunctOnly(word) {
			continue
		}
		words = append(words, word)
	}
	return words
}

// GSESupervisedBody is GSE tokens joined by a single space (text column after __label__ line prefix).
func GSESupervisedBody(text string) string {
	return strings.Join(GSEWords(text), " ")
}

// SplitWhitespaceTokens splits on whitespace for inputs already tokenized (raw fastText convention).
func SplitWhitespaceTokens(line string) []string {
	fields := strings.Fields(line)
	var words []string
	for _, f := range fields {
		if f != "" && f != "</s>" {
			words = append(words, f)
		}
	}
	return words
}

func isPunctOnly(s string) bool {
	for _, r := range s {
		if !unicode.IsPunct(r) && r != ' ' {
			return false
		}
	}
	return true
}
