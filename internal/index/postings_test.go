package index

import (
	"sync"
	"testing"
)

func TestPostingList_NewIsEmpty(t *testing.T) {
	pl := NewPostingsList()
	if pl.Len() != 0 {
		t.Errorf("Len = %d want 0", pl.Len())
	}
}

func TestPostingList_AppendAndLen(t *testing.T) {
	pl := NewPostingsList()
	pl.AddPosting(Posting{DocID: 1, TF: 2.0, Field: FieldTitle})
	pl.AddPosting(Posting{DocID: 2, TF: 1.0, Field: FieldAbstract})

	if got := pl.Len(); got != 2 {
		t.Errorf("Len = %d want 2", got)
	}
}

func TestPostingList_SnapshotIsCopy(t *testing.T) {
	pl := NewPostingsList()
	pl.AddPosting(Posting{DocID: 1, TF: 2.0, Field: FieldTitle})

	snapshot := pl.Snapshot()
	snapshot[0].TF = 3.0 // modify snapshot

	snapShotAgain := pl.Snapshot()
	if snapShotAgain[0].TF != 2.0 {
		t.Errorf("Snapshot is not a copy, original TF = %f want 2.0", snapShotAgain[0].TF)
	}
}

func TestPostingList_ConcurrentAppendAndRead(t *testing.T) {
	// Spawn n goroutines for write and n goroutines for read controlled through a wait group

	pl := NewPostingsList()

	const n = 1000

	var wg sync.WaitGroup
	wg.Add(2 * n)

	// n writer goroutines
	for i := range n {
		// Pass loop variable as argument to avoid closure capture issue |
		//  Fixed in Go 1.21 with new loop variable scoping rules, but still good practice to avoid capture.
		go func(id uint64) {
			defer wg.Done()
			pl.AddPosting(Posting{DocID: id, TF: 1.0, Field: FieldTitle})
		}(uint64(i))
	}

	// n reader goroutines
	for range n {
		go func() {
			defer wg.Done()
			_ = pl.Snapshot()
		}()
	}

	wg.Wait()

	if got := pl.Len(); got != n {
		t.Errorf("Len = %d want %d", got, n)
	}
}
