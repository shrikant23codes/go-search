// This package contains inverted index, bm25 scoring and tokenization logi.
// All concurrency-safe; concurrent reads and indexing supported.
package index

// stopWords and isStopWord accessible only within index package files.

var stopWords = map[string]struct{}{
	"a": {}, "an": {}, "and": {}, "are": {}, "as": {}, "at": {},
	"be": {}, "but": {}, "by": {},
	"for": {}, "from": {},
	"has": {}, "have": {}, "he": {}, "her": {}, "his": {},
	"i": {}, "in": {}, "is": {}, "it": {}, "its": {},
	"of": {}, "on": {}, "or": {},
	"she":  {},
	"that": {}, "the": {}, "this": {}, "to": {},
	"was": {}, "were": {}, "will": {}, "with": {},
	"you": {}, "your": {},
}

func isStopWord(word string) bool {
	_, ok := stopWords[word]
	return ok
}
