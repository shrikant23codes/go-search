package index

import (
	"fmt"
	"testing"
)

func newTestIndex(t *testing.T, docs ...Document) *Index {
	t.Helper()
	idx := New()
	for _, doc := range docs {
		if err := idx.Add(doc); err != nil {
			t.Fatalf("failed to add document %q: %v", doc.ID, err)
		}
	}
	return idx
}

func TestIndex_ExactMatchBeatsPartial(t *testing.T) {
	idx := newTestIndex(t,
		Document{ID: "both", Title: "machine learning", Abstract: ""},
		Document{ID: "partial", Title: "machine intelligence", Abstract: ""},
	)

	results := idx.Search("machine learning", 2)
	if len(results) < 2 {
		t.Fatalf("expected at least 2 results, got %d", len(results))
	}
	if results[0].ID != "both" {
		t.Errorf("expected 'both' to be ranked higher(exact match) than 'partial', got %q", results[0].ID)
	}
}

// Due to field weight
func TestIndex_TitleBeatsAbstract(t *testing.T) {
	idx := newTestIndex(t,
		Document{ID: "title_match", Title: "golang", Abstract: "abstract text"},
		Document{ID: "abs_match", Title: "no match", Abstract: "golang"},
	)

	results := idx.Search("golang", 2)
	if len(results) < 2 {
		t.Fatalf("expected at least 2 results, got %d", len(results))
	}

	if results[0].ID != "title_match" {
		t.Errorf("Expected 'title_match' to be higher ranked, got %q", results[0].ID)
	}
}

func TestIndex_RareTermBeatsCommon(t *testing.T) {
	docs := []Document{
		{ID: "rare-doc", Title: "bitcoin", Abstract: ""},
	}

	for i := range 5 {
		docs = append(docs, Document{
			ID:    fmt.Sprintf("common-%d", i),
			Title: "popular",
		})
	}

	idx := newTestIndex(t, docs...)

	results := idx.Search("bitcoin", 2)

	if len(results) < 1 {
		t.Fatalf("expected at least 1 result, got %d", len(results))
	}

	if results[0].ID != "rare-doc" {
		t.Errorf("Expected 'rare-doc' to be higher ranked, got %q", results[0].ID)
	}
}

func TestIndex_EmptyQuery(t *testing.T) {
	idx := newTestIndex(t, Document{ID: "doc", Title: "hello world"})
	if r := idx.Search("", 10); len(r) != 0 {
		t.Errorf("empty query should return no results, got %v", r)
	}
}

func TestIndex_SingleDocCorpus(t *testing.T) {
	idx := newTestIndex(t, Document{ID: "only", Title: "the only document"})
	results := idx.Search("only", 10)
	if len(results) != 1 || results[0].ID != "only" {
		t.Errorf("single-doc corpus search broken, got %v", results)
	}
}

func TestIndex_UnknownTerm(t *testing.T) {
	idx := newTestIndex(t, Document{ID: "doc", Title: "hello world"})
	if r := idx.Search("zzzzzz", 10); len(r) != 0 {
		t.Errorf("unknown term should return no results, got %v", r)
	}
}

func TestIndex_NonPositiveTopK(t *testing.T) {
	idx := newTestIndex(t, Document{ID: "doc", Title: "hello world"})
	if r := idx.Search("hello", 0); len(r) != 0 {
		t.Errorf("topK=0 should return empty, got %v", r)
	}
	if r := idx.Search("hello", -1); len(r) != 0 {
		t.Errorf("topK=-1 should return empty, got %v", r)
	}
}

func TestIndex_Size(t *testing.T) {
	idx := New()
	if got := idx.Size(); got != 0 {
		t.Errorf("empty Size = %d, want 0", got)
	}
	_ = idx.Add(Document{ID: "a", Title: "x"})
	_ = idx.Add(Document{ID: "b", Title: "y"})
	if got := idx.Size(); got != 2 {
		t.Errorf("Size after 2 adds = %d, want 2", got)
	}
}
