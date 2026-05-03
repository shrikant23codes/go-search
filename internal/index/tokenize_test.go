package index

import (
	"slices"
	"testing"
)

func TestStandardTokenizer(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  []string
	}{
		{"empty input", "", nil},
		{"single word lowercased", "Hello", []string{"hello"}},
		{"strips stop words", "the quick brown fox", []string{"quick", "brown", "fox"}},
		{"all stop words returns empty", "the and of", nil},
		{"punctuation splits tokens", "Go-lang's syntax!", []string{"go", "lang", "s", "syntax"}},
		{"numbers preserved", "Python 3.11", []string{"python", "3", "11"}},
		{"mixed case normalized", "GoSearch BM25", []string{"gosearch", "bm25"}},
		{"unicode letters preserved", "café résumé", []string{"café", "résumé"}},
	}

	tokenizer := NewStandardTokenizer()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tokenizer.Tokenize(tc.input)
			if !slices.Equal(tc.want, got) {
				t.Errorf("Tokenizer(%q) = %v, want %v ", tc.input, got, tc.want)
			}
		})
	}

}
