/*
 * Copyright (c) 2026 easymail.my. All rights reserved.
 *
 * License: AGPLv3
 */

package fasttext

import "easymail/internal/infrastructure/filter/tokenizer"

// Tokenize splits text the same way as WordsForInference (GSE + filtering).
func Tokenize(line string) []string {
	return tokenizer.GSEWords(line)
}

// WordsForInference returns GSE tokens for supervised FastText predict (same segmentation as training file lines).
func WordsForInference(line string) []string {
	return tokenizer.GSEWords(line)
}

// FastTextSupervisedBody returns the text field for one training line: GSE tokens joined by a single space
// (after the __label__ prefix is added by the caller).
func FastTextSupervisedBody(text string) string {
	return tokenizer.GSESupervisedBody(text)
}

// SpaceTokenize splits text by whitespace (raw C++ fasttext convention when input is already tokenized).
func SpaceTokenize(line string) []string {
	return tokenizer.SplitWhitespaceTokens(line)
}
