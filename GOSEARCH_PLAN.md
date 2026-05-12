# GoSearch — Distributed AI-Powered Search Engine
## End-to-End Implementation Plan

> **How to use this file in a new Claude session:**
> Say: "I have a project plan at ~/Downloads/resume/GOSEARCH_PLAN.md. Please read it and help me implement Phase 1, starting from Step 1."

---

## Context

Build a production-grade distributed search engine in Go to showcase senior backend skills for a Perplexity AI Search Golang Engineer application. The project indexes ~5M Wikipedia documents across a 3-node cluster with BM25 ranking, AI query expansion, circuit breakers, Prometheus/Grafana observability, and Kubernetes deployment.

**End state:** A live demo URL (Fly.io) + GitHub repo with real benchmark numbers to add to the resume.

---

## Repository Root

Create at: `~/gosearch/` (or wherever you prefer)

```
gosearch/
├── cmd/
│   ├── gateway/        # HTTP REST entry point
│   ├── router/         # gRPC fan-out with consistent hashing
│   ├── indexnode/      # Inverted index + BM25 + gRPC server
│   ├── merger/         # Result merge + deduplication
│   ├── reranker/       # AI query expansion + semantic reranking
│   └── ingester/       # Wikipedia XML dump parser
├── internal/
│   ├── index/          # index.go, bm25.go, posting.go, stopwords.go
│   ├── consistent/     # ring.go (consistent hash), etcd_sync.go
│   ├── circuit/        # breaker.go (wraps sony/gobreaker)
│   ├── metrics/        # prometheus.go (shared metric definitions)
│   └── middleware/     # auth.go, ratelimit.go
├── proto/search/       # search.proto — single contract for all services
├── deployments/
│   ├── docker-compose.yml
│   └── k8s/            # Deployment, Service, HPA, Ingress per service
├── scripts/
│   ├── download_wiki.sh
│   ├── ingest.sh
│   └── load_test.js    # k6 load test
├── monitoring/
│   ├── prometheus.yml
│   └── grafana-dashboard.json
└── .github/workflows/
    ├── ci.yml
    └── load-test.yml
```

---

## go.mod Dependencies

```
module github.com/<your-username>/gosearch

go 1.22

require (
    google.golang.org/grpc                  v1.63+
    google.golang.org/protobuf              v1.34+
    github.com/prometheus/client_golang     v1.19+
    github.com/sony/gobreaker              v0.5+
    go.etcd.io/etcd/client/v3              v3.5+
    github.com/dgraph-io/badger/v4         v4.x     // LSM index persistence
    github.com/go-chi/chi/v5               v5.0+
    golang.org/x/time                      latest   // token bucket rate limiter
    github.com/sashabaranov/go-openai      v1.x     // OpenAI + Claude-compatible
    go.uber.org/zap                        v1.27+
    github.com/stretchr/testify            v1.9+
    golang.org/x/sync                      latest   // errgroup
)
```

---

## Makefile Targets

```makefile
proto        # regenerate .pb.go from .proto
build        # go build ./cmd/...
test         # go test -race ./...
lint         # golangci-lint run
up           # docker compose up --build
down         # docker compose down -v
ingest       # scripts/ingest.sh (full 5M docs)
ingest-sample # ingest 10k docs for testing
load-test    # k6 run scripts/load_test.js
```

---

## Phase 1 — Core Index + Single-Node Search (Week 1)

**Goal:** `curl localhost:8080/search?q=golang` returns ranked Wikipedia results from a single in-process node.

### Step 1: Proto Contracts (Day 1-2)
**File:** `proto/search/search.proto`

Lock all message shapes before building any service.

```protobuf
syntax = "proto3";
package search.v1;
option go_package = "github.com/<user>/gosearch/proto/search";

message Document {
  string id       = 1;
  string title    = 2;
  string abstract = 3;
  float  score    = 4;
}

message SearchRequest {
  string query    = 1;
  int32  top_k    = 2;
  int32  shard_id = 3;
}
message SearchResponse {
  repeated Document results    = 1;
  int64            latency_ms  = 2;
}

message IndexRequest {
  repeated Document docs     = 1;
  int32             shard_id = 2;
}
message IndexResponse {
  int64  indexed_count = 1;
  string error         = 2;
}

message HealthRequest  {}
message HealthResponse {
  string status     = 1;
  int64  index_size = 2;
}

service SearchService {
  rpc Search (SearchRequest)  returns (SearchResponse);
  rpc Index  (IndexRequest)   returns (IndexResponse);
  rpc Health (HealthRequest)  returns (HealthResponse);
}
```

Run `make proto` to generate `.pb.go` and `_grpc.pb.go`.

### Step 2: Inverted Index + BM25 (Day 2-3)
**Files:** `internal/index/posting.go`, `internal/index/bm25.go`, `internal/index/index.go`

- `Posting` struct: `{DocID uint64, TF float32, Field uint8}`
- Store as flat sorted `[]Posting` per term (cache-friendly, binary search)
- One `sync.RWMutex` per `PostingList`
- BM25 formula: `score = Σ IDF(qi) * TF*(k1+1) / (TF + k1*(1-b+b*|D|/avgdl))`
  - k1=1.5, b=0.75
  - IDF = `ln((N - df + 0.5) / (df + 0.5) + 1)`
- Field weights: title=3.0×, abstract=1.0×
- Tokenizer: lowercase → split on non-alphanumeric → remove stop words
- Persistence: **BadgerDB** (LSM, much better than BoltDB for bulk 5M doc ingestion)

**Write tests first** (`internal/index/bm25_test.go`):
- Exact match outscores partial match
- Title match outscores abstract-only match
- Rare term outscores common term
- Edge cases: empty query, single-doc corpus
- Run with `go test -race -v ./internal/index/...`

Also add benchmarks (`internal/index/bm25_bench_test.go`):
```go
func BenchmarkSearch_1kDocs(b *testing.B)
func BenchmarkSearch_100kDocs(b *testing.B)
func BenchmarkSearch_1MDocs(b *testing.B)
func BenchmarkIndex_Batch500(b *testing.B)
```
Include output in README: `BenchmarkSearch_1MDocs-8    2847 ns/op    1024 B/op    8 allocs/op`

### Step 3: IndexNode gRPC Server (Day 3-4)
**Files:** `cmd/indexnode/main.go`, `cmd/indexnode/server.go`

- Implement `SearchService` gRPC interface
- Config from env: `SHARD_ID`, `PORT`, `ETCD_ENDPOINTS`, `BADGER_PATH`
- Three ports per service: `PORT` (gRPC), `PORT+1000` (Prometheus `/metrics`), `PORT+2000` (pprof)
- Register with etcd on startup: 10s TTL lease + 3s keepalive heartbeat
- Add `import _ "net/http/pprof"` for profiling endpoint

### Step 4: Ingester — Wikipedia XML Parser (Day 4-5)
**Files:** `cmd/ingester/main.go`, `scripts/download_wiki.sh`

Data source:
```bash
curl -L "https://dumps.wikimedia.org/enwiki/latest/enwiki-latest-abstract.xml.gz" | gunzip > data/enwiki-abstract.xml
```

XML structure:
```xml
<feed>
  <doc>
    <title>Wikipedia: Go (programming language)</title>
    <abstract>Go is a statically typed...</abstract>
    <url>https://en.wikipedia.org/wiki/Go_(programming_language)</url>
  </doc>
</feed>
```

**CRITICAL:** Use `xml.NewDecoder` streaming (token-by-token), NOT `xml.Unmarshal`. Full unmarshal of 800MB → ~4GB RAM usage.

- `docID = hash(url)`
- Phase 1 shard assignment: `shardID = docID % numShards`
- Batch 500 docs per `IndexRequest`
- Flags: `--dump-path`, `--indexnode-addrs` (comma-sep), `--batch-size`

### Step 5: API Gateway (Day 5-6)
**Files:** `cmd/gateway/main.go`, `internal/middleware/auth.go`, `internal/middleware/ratelimit.go`

- HTTP REST with `go-chi/chi`
- Routes: `GET /search`, `POST /index`, `GET /health`, `GET /metrics`
- Auth middleware: `X-API-Key` header checked against `GOSEARCH_API_KEYS` env var → 401 if missing/invalid
- Rate limit middleware: per-key token bucket (`golang.org/x/time/rate`), burst=20, rate=100 req/sec → 429 with `Retry-After`
- Phase 1: gateway calls indexnode directly (no router yet — wire router in Phase 2)

### Step 6: Single-Node Smoke Test (Day 6-7)

Integration test `cmd/gateway/integration_test.go`:
- Spin up indexnode in-process
- Ingest 1000 docs
- Assert top result for known query returns correct document

Verify manually:
```bash
./gosearch-indexnode --shard-id=0 --port=9001 &
./gosearch-ingester --dump-path=data/enwiki-abstract.xml --indexnode-addrs=localhost:9001 --limit=100000
curl -H "X-API-Key: dev-key" "localhost:8080/search?q=machine+learning&top_k=5"
```

**Milestone commit:** `feat: single-node BM25 search over Wikipedia abstracts with gRPC indexnode and HTTP gateway`

---

## Phase 2 — Multi-Node, Routing, Merging, Observability (Week 2)

**Goal:** 3 index nodes, consistent hash routing, circuit breakers, Prometheus/Grafana dashboard.

### Step 7: Consistent Hash Ring (Day 8-9)
**Files:** `internal/consistent/ring.go`, `internal/consistent/etcd_sync.go`

```go
type Ring struct {
    mu      sync.RWMutex
    vnodes  int              // 150 per physical node
    ring    []uint32         // sorted hash values
    hashMap map[uint32]Node
}

type Node struct {
    ID      string
    Address string  // host:port
}

func (r *Ring) Add(node Node)
func (r *Ring) Remove(nodeID string)
func (r *Ring) Get(key string) Node
func (r *Ring) GetN(key string, n int) []Node
func (r *Ring) GetAll() []Node
```

- Hash: `crc32.ChecksumIEEE`
- Virtual node keys: `hash(nodeID + "#" + strconv.Itoa(i))` for i in 0..149
- `Get` uses `sort.Search` → O(log(150 * numNodes))

**Why 150 vnodes:** With 3 nodes × 150 = 450 ring positions, distribution std dev ≈ 10%. At 50 vnodes it's 18%, at 300 it's 7% — 150 is the sweet spot (same as Cassandra uses).

`EtcdWatcher`:
- Watches etcd prefix `/gosearch/nodes/`
- On DELETE event (node crash/TTL expiry): calls `ring.Remove(nodeID)`
- On PUT event (node join): calls `ring.Add(node)`

**Tests (critical — interviewers will ask about these):**
- Distribution: ingest 100k keys → each node gets 28-38% of traffic
- Monotonicity: add 4th node → only ~25% of keys reassign (not more)
- Remove: remove a node → its keys go to next node only
- Concurrency: Add/Remove/Get concurrently under `-race` flag

### Step 8: Query Router (Day 9-10)
**Files:** `cmd/router/main.go`, `cmd/router/fanout.go`

Fan-out pattern (idiomatic Go — mention contrast with Java `CompletableFuture.allOf`):

```go
func (r *RouterServer) fanout(ctx context.Context, req *search.SearchRequest) ([]*search.SearchResponse, error) {
    nodes := r.ring.GetAll()
    results := make([]*search.SearchResponse, len(nodes))
    var wg sync.WaitGroup
    for i, node := range nodes {
        wg.Add(1)
        go func(idx int, n consistent.Node) {
            defer wg.Done()
            // Per-shard 50ms deadline from parent context
            shardCtx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
            defer cancel()
            cb := r.getBreaker(n.ID)
            resp, err := cb.Execute(func() (interface{}, error) {
                conn := r.pool.Get(n.Address)
                return search.NewSearchServiceClient(conn).Search(shardCtx, req)
            })
            if err == nil {
                results[idx] = resp.(*search.SearchResponse)
            }
        }(i, node)
    }
    wg.Wait()
    return results, nil  // tolerate minority shard failures → return partial results
}
```

**Key decisions:**
- 50ms per-shard deadline (if gateway sets 200ms budget, shard gets 50ms, rest for merge/rerank)
- Tolerate 1-of-3 shard failure: partial results > 500 error for a search engine (availability > completeness)

### Step 9: Circuit Breaker (Day 9-10)
**File:** `internal/circuit/breaker.go`

```go
func NewBreaker(name string) *gobreaker.CircuitBreaker {
    return gobreaker.NewCircuitBreaker(gobreaker.Settings{
        Name:        name,
        MaxRequests: 3,              // half-open: allow 3 probe requests
        Interval:    10 * time.Second,
        Timeout:     30 * time.Second,
        ReadyToTrip: func(counts gobreaker.Counts) bool {
            return counts.ConsecutiveFailures > 5 ||
                (counts.Requests > 10 && counts.TotalFailures/counts.Requests > 0.5)
        },
    })
}
```

### Step 10: Result Merger (Day 10-11)
**Files:** `cmd/merger/merge.go`, `cmd/merger/dedup.go`

1. Collect all docs from shard responses → flat slice
2. Deduplicate by DocID
3. **Score normalization:** min-max normalize each shard's scores to [0,1] before merging
   - Why: BM25 IDF is computed per-shard (different doc counts), so raw scores aren't comparable across shards
4. Sort by normalized score descending → return top-K

Note in ARCHITECTURE.md: Production approach would use global IDF stored in Redis, updated by ingester.

### Step 11: Docker Compose Multi-Node (Day 11-12)
**File:** `deployments/docker-compose.yml`

Single multi-stage Dockerfile for all 6 binaries:
```dockerfile
FROM golang:1.22-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/gosearch-gateway   ./cmd/gateway
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/gosearch-router    ./cmd/router
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/gosearch-indexnode ./cmd/indexnode
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/gosearch-merger    ./cmd/merger
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/gosearch-reranker  ./cmd/reranker
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/gosearch-ingester  ./cmd/ingester

FROM gcr.io/distroless/static:nonroot
COPY --from=builder /out/ /
```

Compose services: etcd, 3×indexnode, router, gateway, merger, reranker, Redis, Prometheus, Grafana.
`make up` must work from a clean clone — this is the first thing a reviewer will do.

### Step 12: Prometheus + Grafana (Day 12-13)
**Files:** `internal/metrics/prometheus.go`, `monitoring/prometheus.yml`, `monitoring/grafana-dashboard.json`

| Metric | Type | Labels | Service |
|--------|------|--------|---------|
| `gosearch_requests_total` | Counter | service, method, status | all |
| `gosearch_request_duration_seconds` | Histogram | service, method | all |
| `gosearch_index_size_docs` | Gauge | shard_id | indexnode |
| `gosearch_index_size_bytes` | Gauge | shard_id | indexnode |
| `gosearch_fanout_duration_seconds` | Histogram | — | router |
| `gosearch_circuit_state` | Gauge | node_id | router |
| `gosearch_rerank_duration_seconds` | Histogram | model | reranker |
| `gosearch_ai_tokens_used` | Counter | model, direction | reranker |

Dashboard rows: Traffic (RPS, p50/p99, error rate) | Index Health (docs/shard, ingest rate) | Router (fanout latency, circuit state) | AI Layer (reranker latency, token usage)

Commit exported Grafana JSON to repo. Include dashboard screenshot in README.

**Milestone commit:** `feat: 3-node distributed search with consistent hashing, circuit breakers, and Prometheus/Grafana observability`

---

## Phase 3 — AI Layer, Kubernetes, CI/CD, Load Testing (Week 3)

**Goal:** AI reranking live, K8s manifests with HPA, GitHub Actions CI with SLO gates, Fly.io public URL.

### Step 13: AI Reranker (Day 15-16)
**Files:** `cmd/reranker/expand.go`, `cmd/reranker/rerank.go`, `cmd/reranker/pipeline.go`

**Query expansion** (`expand.go`):
- Model: `gpt-4o-mini` (10× cheaper than gpt-4o, good enough for synonym generation)
- Prompt: `"Given the search query '{q}', generate 4 semantically related queries as a JSON array."`
- Cache in Redis: key `expand:sha256(normalized_query)`, TTL 1 hour
- Cost math: at 1000 RPS with 1% unique query rate, caching reduces LLM calls by 99×

**Semantic reranking** (`rerank.go`):
- Embeddings: `text-embedding-3-small` (1536 dims, ~$0.02/M tokens)
- Score blend: `final = 0.4 * bm25_normalized + 0.6 * cosine_similarity`
- Batch embedding calls (up to 2048 inputs per API call)

**Pipeline** (`pipeline.go`):
```go
variants, _ := expand(ctx, query)           // ["query", "variant1", ...]
// Fan-out all variants in parallel with 100ms timeout
var allDocs []*search.Document
resultCh := make(chan []*search.Document, len(variants))
for _, v := range variants {
    go func(q string) { resultCh <- searchViaRouter(ctx, q) }(v)
}
timeout := time.After(100 * time.Millisecond)
for range variants {
    select {
    case docs := <-resultCh: allDocs = append(allDocs, docs...)
    case <-timeout:           break
    }
}
merged := merger.Merge(dedup(allDocs), topK*2)
return rerank(ctx, query, merged)
```

The 100ms timeout on variant collection is deliberate resilience — don't block on slow variants.

### Step 14: Kubernetes Manifests (Day 16-17)
**Files:** `deployments/k8s/{service}/deployment.yaml`, `deployments/k8s/indexnode/hpa.yaml`

- `Deployment` + `Service` + gRPC liveness/readiness probes for each service
- `HPA` for indexnode: min=3, max=10, target CPU=60%
- `PersistentVolumeClaim` for BadgerDB storage per indexnode replica
- `NGINX Ingress` for gateway

### Step 15: GitHub Actions CI/CD (Day 17-18)
**File:** `.github/workflows/ci.yml`

```yaml
jobs:
  lint:   golangci-lint (errcheck, staticcheck, gosec)
  test:   go test -race -count=1 -timeout=5m ./...  (with etcd service container)
  build:  docker multi-arch build → push to GHCR on main
  load-test: (main only) docker compose up → ingest 10k docs → k6 → assert p99 < 200ms
```

**File:** `scripts/load_test.js` (k6):
```javascript
export const options = {
  stages: [
    { duration: '30s', target: 100 },
    { duration: '2m',  target: 1000 },
    { duration: '30s', target: 0 },
  ],
  thresholds: {
    http_req_duration: ['p(99)<200'],
    http_req_failed:   ['rate<0.01'],
  },
};
```

### Step 16: Documentation + Polish (Day 18-21)

**README.md must have:**
- Architecture diagram (Mermaid in GitHub markdown)
- `make up` quick-start that works from a clean clone
- Real benchmark numbers from k6 run (not estimated)
- Grafana dashboard screenshot

**ARCHITECTURE.md key decisions (prepare verbal answers for interviews):**
1. BadgerDB vs BoltDB — LSM vs B-tree write throughput for bulk ingestion
2. 150 vnodes — distribution std dev tradeoff (10% at 150 vs 18% at 50)
3. gRPC over REST — typed contracts, binary perf, future streaming support
4. Partial result tolerance — availability > completeness for search (CAP theorem applied)
5. Per-shard IDF problem — why min-max normalization is needed
6. Redis caching for query expansions — 99× LLM call reduction math

**Go patterns to call out** (key differentiators as a Java engineer):
- `sync.WaitGroup` fan-out vs Java `CompletableFuture.allOf`
- `context` propagation for deadline inheritance across service boundaries
- `errgroup` for structured concurrency in AI pipeline
- Fine-grained `sync.RWMutex` per posting list (vs Java `ConcurrentHashMap` segment locks)
- Table-driven tests with `t.Run`

**Milestone commit:** `feat: AI query expansion, K8s deployment, GitHub Actions CI with load test SLO gates`

---

## Multi-Node Testing Options

| Option | Cost | Best For | Notes |
|--------|------|----------|-------|
| **Docker Compose** | Free | Development, README quick-start | `make up`, fastest loop |
| **Kind** (K8s in Docker) | Free | HPA demo, node failure simulation | Real K8s API, ~3 min startup |
| **Fly.io** | ~$15/mo | Resume link, public demo URL | `gosearch.fly.dev`, multi-region easy |
| Minikube | Free | — | No advantage over Kind, skip |
| AWS EKS | ~$120/mo | Only with credits | Overkill, cost risk |

**Recommended strategy:**
- **Dev:** Docker Compose (fastest iteration)
- **Demo video:** Kind (kill a worker node, watch circuit breakers trip in Grafana)
- **Resume link:** Fly.io (`gosearch.fly.dev` — include in resume alongside GitHub link)

**Kind quick-start:**
```bash
# deployments/kind-config.yaml
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
- role: worker   # indexnode 1
- role: worker   # indexnode 2
- role: worker   # indexnode 3

kind create cluster --config deployments/kind-config.yaml
kubectl apply -f deployments/k8s/
# Simulate node failure:
docker stop kind-worker2
# Watch circuit breaker trip in Grafana
```

---

## End-to-End Verification Checklist

```
[ ] make build                  — all 6 binaries compile
[ ] make up                     — all services start (gateway, router, 3×indexnode, merger, reranker, etcd, Redis, Prometheus, Grafana)
[ ] make ingest-sample          — 10k Wikipedia docs distributed across 3 shards
[ ] curl search                 — returns ranked results
[ ] Grafana localhost:3000      — live metrics for all services
[ ] make load-test              — k6 reports p99 < 200ms at 1000 concurrent users
[ ] go test -race ./...         — all tests pass
[ ] Kill one indexnode          — circuit breaker trips, partial results still returned
[ ] GitHub Actions              — CI green on main
```

---

## Resume Entry (fill in measured numbers after load test)

```
GOSEARCH | Distributed AI-Powered Search Engine
github.com/<user>/gosearch | gosearch.fly.dev

Production-grade distributed search engine in Go over 5M Wikipedia documents
across a 3-node cluster with BM25 ranking, AI query expansion, and full observability.
• Inverted index with BM25, BadgerDB persistence, fine-grained RWMutex concurrency
• Consistent hash ring (150 vnodes, etcd-backed topology) routing queries across index shards
• AI query expansion via OpenAI gpt-4o-mini + semantic reranking with text-embedding-3-small
• Circuit breakers (sony/gobreaker), per-key rate limiting, Prometheus/Grafana dashboards
• GitHub Actions CI: golangci-lint, -race tests, k6 load test gating p99 < 200ms SLO
• Load tested to [N]k RPS on 3 nodes; measured p99 latency [X]ms
Tech: Go 1.22, gRPC/protobuf, BadgerDB, etcd, Redis, Prometheus, Docker Compose, Kubernetes
```

---

## Implementation Order Summary

| Week | Days | Deliverable |
|------|------|-------------|
| 1 | 1-2 | Proto contracts, go.mod, Makefile |
| 1 | 2-3 | Inverted index + BM25 (with tests) |
| 1 | 3-4 | IndexNode gRPC server |
| 1 | 4-5 | Ingester (Wikipedia XML streaming parser) |
| 1 | 5-6 | API Gateway (auth + rate limit middleware) |
| 1 | 6-7 | Single-node smoke test + integration test |
| 2 | 8-9 | Consistent hash ring + etcd sync |
| 2 | 9-10 | Query Router (fan-out + circuit breakers) |
| 2 | 10-11 | Result Merger (score normalization + dedup) |
| 2 | 11-12 | Docker Compose multi-node setup |
| 2 | 12-14 | Prometheus + Grafana + pprof |
| 3 | 15-16 | AI Reranker (expansion + semantic reranking) |
| 3 | 16-17 | Kubernetes manifests + HPA |
| 3 | 17-18 | GitHub Actions CI/CD + k6 load test |
| 3 | 18-21 | README, ARCHITECTURE.md, benchmarks, Fly.io deploy |
