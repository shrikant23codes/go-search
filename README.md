# go-search

A full-text search engine written in Go. Indexes documents by title and abstract, scores results with BM25, and persists the forward index to BadgerDB. Work in progress.

## internal/index

The core of the project. All types are concurrency-safe; concurrent reads and indexing are supported.

| File | What it does |
|---|---|
| `index.go` | Index orchestrator. Holds the in-memory postings map, forward doc store, and per-doc token counts. `Add` writes to memory first (so the doc is immediately searchable), then persists to BadgerDB. `Search` tokenizes the query, looks up postings per term, accumulates BM25 scores, and returns the top-K results sorted by score descending. Doc IDs are hashed to `uint64` with FNV-64a: fast, deterministic, good distribution for short strings. |
| `bm25.go` | BM25 scoring math. `TermScore` computes TF saturation; `IDF` uses the standard `ln((N - df + 0.5) / (df + 0.5) + 1)` formula with `+1` inside the log to keep IDF non-negative for very common terms. `FieldWeight` applies a 3x multiplier to title hits vs. 1x for abstract. Constants: `k1 = 1.5`, `b = 0.75`. |
| `postings.go` | `Posting` and `PostingsList`. Each posting is 13 bytes (DocID uint64 + TF float32 + Field uint8), padded to 16 for alignment. A term can have two postings for the same doc if it appears in both title and abstract. `Snapshot()` copies the slice under a read lock so callers do not hold the lock during scoring. |
| `persistence.go` | BadgerDB persistence via `Open` / `Close`. Only the forward index (doc content) is persisted. Postings are rebuilt in memory on `LoadFromDB` by re-running `addInMemory` over every stored document. Keys are namespaced with a `doc:` prefix; the iterator does a prefix scan, which is the standard pattern for LSM-style KV stores. Badger's closure-based transactions handle commit/rollback automatically. |
| `tokenize.go` | `StandardTokenizer`. Lowercases input, splits on non-alphanumeric characters, strips stop words. Stateless, so the index only locks after tokenization is done. |
| `stopwords.go` | A fixed set of common English stop words stored in a package-level map. Accessible only within the `index` package. |

## Running tests

```sh
go test -race ./internal/index/...
```

For verbose output:

```sh
go test -race -v ./internal/index/...
```

## Running benchmarks

Benchmarks are not written yet.

Once added, run them with:

```sh
go test -bench=. -benchmem ./internal/index/...
```

## Benchmarks

Synthetic 45-word vocabulary corpus (worst-case posting list sizes).

| Corpus | Latency | Memory |
|---|---|---|
| 1k docs | 274 μs/op | 0.17 MB/op |
| 100k docs | 31 ms/op | 13.7 MB/op |
| 1M docs | 516 ms/op | 155 MB/op |
| Index 500 docs | 5.79 ms/op | 4.83 MB/op |

Real Wikipedia data has a much larger vocabulary so posting lists will be sparser. Full numbers after ingest in Phase 3.

## Quick start

The project is a work in progress. The index layer is functional; a gRPC server layer and query API are planned next.

```go
idx := index.New()
idx.Add(index.Document{ID: "1", Title: "Go concurrency", Abstract: "goroutines and channels"})
results := idx.Search("concurrency", 10)
```

To open a persistent index backed by BadgerDB:

```go
idx, err := index.Open("/path/to/data")
defer idx.Close()
```

## Ingestion Flow

### 1) Run indexnode:

```
go build ./cmd/indexnode && ./indexnode
```

### 2) Running Ingester

````
$ chmod +x data/download_wiki.sh
$ ./download_wiki.sh
$ go build ./cmd/ingester && ./ingester \
    --dump-path=data/enwiki_content-20260510-00010.json.bz2 \
    --indexnode-addrs=localhost:9001 \
    --batch-size=50 \
    --limit=1000
```

This should give output like:

```
2026/05/19 12:59:15 indexed 50 docs (total ~1000)
2026/05/19 12:59:15 done: 1000 docs ingested in 955ms
```

### 3) Health Check grpc curl to get index count
```
$ brew install grpcurl

$ grpcurl -plaintext \
    -proto proto/search/search.proto \
    localhost:9001 \
    search.v1.SearchService/Health

Response = {
  "status": "OK",
  "indexSize": "1000"
}
```
