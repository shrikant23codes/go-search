package index

import (
	"strings"
	"unicode"
)

type Tokenizer interface {
	Tokenize(s string) []string
}

type StandardTokenizer struct{}

func NewStandardTokenizer() *StandardTokenizer {
	return &StandardTokenizer{}
}

func (t *StandardTokenizer) Tokenize(s string) []string {
	s = strings.ToLower(s)

	fields := strings.FieldsFunc(s, func(r rune) bool {
		return !unicode.IsDigit(r) && !unicode.IsLetter(r)
	})

	out := make([]string, 0, len(fields))
	for _, field := range fields {
		if !isStopWord(field) {
			out = append(out, field)
		}
	}
	return out
}
