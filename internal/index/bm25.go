package index

import "math"

// BM25 hyperparameters;
const (
	BM25K1 = 1.5  // Controls how fast TF saturates
	BM25B  = 0.75 // How long documents are penalized
)

func FieldWeight(field Field) float64 {
	switch field {
	case FieldTitle:
		return 3.0
	case FieldAbstract:
		return 1.0
	default:
		return 0.0
	}
}

// IDF computes the inverse document frequency of a term
// IDF(t) = ln( (N - df + 0.5) / (df + 0.5) + 1 )
// Note: +1 inside log to prevent negative IDF for very common terms.
// n = total number of documents in the corpus
// df = number of documents containing the term (postings list lenght)
func IDF(n, df int) float64 {
	return math.Log((float64(n-df)+0.5)/(float64(df)+0.5) + 1.0)
}

// TermScore computes BM25 score contribution of a term in a document.
// score = tf * (k1 + 1) / ( tf + k1 * (1 - b + b * |D|/avgdl) )
// tf = term frequency in the document (from posting)
// |D| = document length (number of tokens in the document)
// avgdl = average document length in the corpus
func TermScore(tf, docLen, avgDocLen float64) float64 {
	if tf == 0 {
		return 0.0
	}

	lengthNorm := 1.0 - BM25B + BM25B*(docLen/avgDocLen)
	return (tf * (BM25K1 + 1)) / (tf + BM25K1*lengthNorm)
}
