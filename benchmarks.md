## Benchmarks

Package: `github.com/shrikant23codes/gosearch/internal/index`

| Benchmark | ns/op | MB/op | allocs/op |
|---|---|---|---|
| Search_1kDocs | 273,786 | 0.17 | 28 |
| Search_100kDocs | 30,974,149 | 13.70 | 538 |
| Search_1MDocs | 516,498,625 | 155.03 | 7,732 |
| Index_Batch500 | 5,790,058 | 4.83 | 5,680 |


Run: `go test -bench=. -benchmem ./internal/index/...`
Note: synthetic corpus, 45-word vocabulary. Posting lists are worst-case size vs real Wikipedia.


13.7MB per search op at 100k docs is high. That's almost entirely the Snapshot() copies — for each query term we copy the full posting list. 538 allocs across a 3-term query means ~180 allocations per term, which maps to the scoring intermediate structs and the scores map.

### On the 516ms number

516ms at 1M docs on a 45-word vocab. The synthetic corpus is the worst case. Every term's posting list has ~22k entries (1M docs / 45 words). A 3-term query copies ~1M postings before scoring even starts.

Two things driving this:

1. `Snapshot()` allocates a full copy of each posting list before iterating. At 1M docs the copies dominate. Fix: pass a callback to the PostingList so scoring happens inside the read lock with no allocation.

2. Accumulating scores into a `map[uint64]float64` then sorting the whole map is O(N log N). Fix: use a max-heap of size K — only keep the top-K candidates, never sort the full scored set.