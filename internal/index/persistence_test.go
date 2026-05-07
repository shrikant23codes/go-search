package index

import "testing"

// First, add some docs, clsoe the index, reopen and then search for docs
func TestPersistence_RoundTrip(t *testing.T) {

	dir := t.TempDir()

	// Part 1: Create index, add docs and close

	{
		idx, err := Open(dir)
		if err != nil {
			t.Fatalf("failed to open index: %v", err)
		}
		defer idx.Close()

		docs := []Document{
			{ID: "doc1", Title: "golang concurrency", Abstract: "goroutines and channels"},
			{ID: "doc2", Title: "python data science", Abstract: "pandas and numpy"},
		}

		for _, doc := range docs {
			if err := idx.Add(doc); err != nil {
				t.Fatalf("failed to add document %q: %v", doc.ID, err)
			}
		}

		if err := idx.Close(); err != nil {
			t.Fatalf("Failed to close index: %v", err)
		}
	}

	// Part 2: Reopen index and search

	idx, err := Open(dir)

	if err != nil {
		t.Fatalf("failed to reopen index: %v", err)
	}
	defer idx.Close()

	if got := idx.Size(); got != 2 {
		t.Fatalf("expected 2 documents, got %d", got)
	}

	results := idx.Search("golang", 2)
	if len(results) != 1 && results[0].ID != "doc1" {
		t.Fatalf("expected to find doc1, got %v", results)
	}

}

func TestPersistence_OpenEmpty(t *testing.T) {
	dir := t.TempDir()

	idx, err := Open(dir)

	if err != nil {
		t.Fatalf("failed to open index: %v", err)
	}

	defer idx.Close()

	if got := idx.Size(); got != 0 {
		t.Fatalf("expected 0 documents, got %d", got)
	}

	results := idx.Search("golang", 2)
	if len(results) != 0 {
		t.Fatalf("expected 0 results, got %v", results)
	}
}
