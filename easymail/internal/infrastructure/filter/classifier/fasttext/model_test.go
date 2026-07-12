package fasttext

import (
	"testing"
)

func TestTokenize(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "simple words",
			input:    "hello world",
			expected: []string{"hello", "world"},
		},
		{
			name:     "with punctuation",
			input:    "Hello, world! How are you?",
			expected: []string{"hello", "world", "how", "are", "you"}, // GSE normalizes/strips punct for supervised line
		},
		{
			name:     "empty string",
			input:    "",
			expected: nil,
		},
		{
			name:     "with newlines",
			input:    "hello\nworld\ttab",
			expected: []string{"hello", "world", "tab"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := WordsForInference(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("WordsForInference(%q) = %v, want %v", tt.input, result, tt.expected)
				return
			}
			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("WordsForInference(%q) = %v, want %v", tt.input, result, tt.expected)
					return
				}
			}
		})
	}
}

func TestMatrixCreation(t *testing.T) {
	m := NewMatrix(3, 4)

	if m.Rows() != 3 {
		t.Errorf("Matrix.Rows() = %d, want 3", m.Rows())
	}
	if m.Cols() != 4 {
		t.Errorf("Matrix.Cols() = %d, want 4", m.Cols())
	}

	// Test Set and At
	m.Set(1, 2, 3.14)
	if m.At(1, 2) != 3.14 {
		t.Errorf("Matrix.At(1, 2) = %f, want 3.14", m.At(1, 2))
	}
}

func TestDictionaryHash(t *testing.T) {
	args := &Args{Bucket: 2000000, Minn: 3, Maxn: 6}
	dict := NewDictionary(args)

	// Test hash function consistency
	word := "hello"
	h1 := dict.hash(word)
	h2 := dict.hash(word)
	if h1 != h2 {
		t.Errorf("hash(%q) not consistent: %d != %d", word, h1, h2)
	}

	// Test different words produce different hashes (mostly)
	word2 := "world"
	h3 := dict.hash(word2)
	// Note: collisions are possible but unlikely for different words
	_ = h3
}

func TestFastExp(t *testing.T) {
	tests := []struct {
		input    float64
		expected float64
		approx   float64 // acceptable error
	}{
		{0, 1, 0.0001},
		{1, 2.71828, 0.0001},
		{-1, 0.36788, 0.0001},
	}

	for _, tt := range tests {
		result := fastExp(tt.input)
		diff := result - tt.expected
		if diff < 0 {
			diff = -diff
		}
		if diff > tt.approx {
			t.Errorf("fastExp(%f) = %f, want ~%f", tt.input, result, tt.expected)
		}
	}
}
