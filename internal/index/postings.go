package index

import "sync"

// To identity which field the token comes from in the doc.
// We keep it in uint8 1 byte to save space and cache friendly when in slice.
type Field uint8

const (
	FieldTitle    Field = 1
	FieldAbstract Field = 2
)

// Posting represents one occurrence of token in doc in one field.
// we can have 2 positing for same doc if token appears in both title and abstract.

// Memory layout of Posting: 8 bytes (DocID) + 4 bytes (TF) + 1 byte (Field) = 13 bytes.
//
//	This is padded to 16 bytes for alignment, which is efficient for memory access.
type Posting struct {
	DocID uint64
	TF    float32
	Field Field
}

type PostingsList struct {
	mu       sync.RWMutex
	postings []Posting
}

func NewPostingsList() *PostingsList {
	return &PostingsList{}
}

func (pl *PostingsList) AddPosting(p Posting) {
	pl.mu.Lock()
	defer pl.mu.Unlock()
	pl.postings = append(pl.postings, p)
}

func (pl *PostingsList) Len() int {
	pl.mu.RLock()
	defer pl.mu.RUnlock()
	return len(pl.postings)
}

// Snapshot returns copy of postings list for safe concurrent access without locking.
// Cost: one allocation + memcpy of postings slice.
// If profiling shows this as bottleneck during query we can optimize.
func (pl *PostingsList) Snapshot() []Posting {
	pl.mu.RLock()
	defer pl.mu.RUnlock()
	snapshot := make([]Posting, len(pl.postings))
	copy(snapshot, pl.postings)
	return snapshot
}
