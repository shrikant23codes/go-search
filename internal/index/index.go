package index

import (
	"errors"
	"fmt"
	"hash/fnv"
	"sort"
	"sync"

	"github.com/dgraph-io/badger/v4"
)

type Document struct {
	ID       string
	Title    string
	Abstract string
}

// Document is embedded - we can access r.ID, r.Title, r.Abstract directly.
type SearchResult struct {
	Document
	Score float64
}

type Index struct {
	mu        sync.RWMutex
	postings  map[string]*PostingsList // term -> postings
	docs      map[uint64]Document      // docID -> original doc
	docLens   map[uint64]int           // docID -> token count for BM25 length norm
	totalLen  int                      // sum of all doc lengths; avgDL = totalLen / len(docs)
	tokenizer Tokenizer
	db        *badger.DB // For persisting docID -> docContent mapping
}

func New() *Index {
	return &Index{
		postings:  make(map[string]*PostingsList),
		docs:      make(map[uint64]Document),
		docLens:   make(map[uint64]int),
		tokenizer: NewStandardTokenizer(),
	}
}

// First insert in inmemory index and then persist to badgerDB. This way we can ensure that the document is searchable immediately after Add returns.
// But obviously a gap that we don't persist to badgerDB.
// Can have a background goroutine do it.
func (idx *Index) Add(doc Document) error {
	if err := idx.addInMemory(doc); err != nil {
		return err
	}

	if idx.db != nil {
		err := idx.persistDoc(doc)
		if err != nil {
			return fmt.Errorf("failed to persist document %q: %w", doc.ID, err)
		}
	}
	return nil
}

// Add document to index, return error on empty ID, duplicate ID
// or hash collision with different doc
func (idx *Index) addInMemory(doc Document) error {
	if doc.ID == "" {
		return errors.New("document ID is empty")
	}

	titleTokens := idx.tokenizer.Tokenize(doc.Title)
	abstractTokens := idx.tokenizer.Tokenize(doc.Abstract)

	titleTF := tabulate(titleTokens)
	abstractTF := tabulate(abstractTokens)
	docLen := len(titleTF) + len(abstractTF)
	id := hashID(doc.ID)

	// Now lock index after tokenizer as tokenizer is stateless
	idx.mu.Lock()
	defer idx.mu.Unlock()

	if existing, exists := idx.docs[id]; exists {
		if existing.ID == doc.ID {
			return fmt.Errorf("document %q already indexed", doc.ID)
		}
		return fmt.Errorf("Hash collision between %q and %q", existing.ID, doc.ID)
	}

	idx.docs[id] = doc
	idx.docLens[id] = docLen
	idx.totalLen += docLen

	for term, tf := range titleTF {
		idx.appendPosting(term, Posting{DocID: id, TF: float32(tf), Field: FieldTitle})
	}

	for term, tf := range abstractTF {
		idx.appendPosting(term, Posting{DocID: id, TF: float32(tf), Field: FieldAbstract})
	}

	return nil
}

func (idx *Index) Search(query string, topK int) []SearchResult {

	if topK <= 0 {
		return nil
	}

	queryTokens := idx.tokenizer.Tokenize(query)

	// All stopwords
	if len(queryTokens) == 0 {
		return nil
	}

	// For each query term get postings and compute scores for each doc
	n := len(idx.docs)
	if n == 0 {
		return nil
	}

	avgDocLen := float64(idx.totalLen) / float64(n)
	scores := make(map[uint64]float64) // docID -> sum of BM25 scores for all query terms

	for _, term := range queryTokens {
		pl, exists := idx.postings[term]
		if !exists {
			continue
		}
		idf := IDF(n, pl.Len())
		for _, p := range pl.Snapshot() {
			scores[p.DocID] += idf * FieldWeight(p.Field) * TermScore(float64(p.TF), float64(idx.docLens[p.DocID]), avgDocLen)
		}
	}

	// Now rank the documents by score and return topK results
	results := make([]SearchResult, 0, len(scores))
	for docID, score := range scores {
		doc := idx.docs[docID]
		results = append(results, SearchResult{Document: doc, Score: score})
	}

	// Sort results by score descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	if len(results) > topK {
		return results[:topK]
	}

	return results
}

// Size returns the number of indexed documents (used by HealthResponse).
func (idx *Index) Size() int {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return len(idx.docs)
}

func (idx *Index) appendPosting(term string, p Posting) {
	pl, ok := idx.postings[term]
	if !ok {
		pl = NewPostingsList()
		idx.postings[term] = pl
	}

	pl.AddPosting(p)
}

func tabulate(tokens []string) map[string]int {
	countMap := make(map[string]int, len(tokens))
	for _, t := range tokens {
		countMap[t]++
	}
	return countMap
}

// hashID maps a string ID to a uint64 using FNV-64a.
// FNV is fast, deterministic, and has good distribution for short strings —
// fine for our docID needs. We don't need a cryptographic hash.
func hashID(s string) uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(s))
	return h.Sum64()
}
