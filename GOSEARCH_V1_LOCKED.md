# GoSearch v1 — LOCKED SCOPE

**Locked on:** 2026-05-03
**Hard ship date:** 2026-05-24 (3 weeks)
**Why locked:** GoSearch v1 is mission-critical for the Railway-tier application sprint (Stance B). It's THE Go portfolio piece. Scope creep here = applications slip = LinkedIn plateau extends.

**Reference plan:** `~/Downloads/resume/GOSEARCH_PLAN.md` — v1 = exactly that plan, no more.

---

## What v1 IS (locked, do not reduce)

The full 3-phase plan from GOSEARCH_PLAN.md:

### Phase 1 — Core Index + Single-Node Search
- Proto contracts (search.proto)
- Inverted index + BM25 with tests + benchmarks
- IndexNode gRPC server with etcd registration
- Wikipedia XML streaming ingester
- API Gateway (chi) with auth + rate-limit middleware
- Single-node smoke test (1k docs, ranked results via curl)

### Phase 2 — Multi-Node + Observability
- Consistent hash ring (150 vnodes, etcd sync)
- Query Router (fan-out + circuit breakers + 50ms shard deadline)
- Circuit breaker (sony/gobreaker)
- Result Merger (min-max score normalization + dedup)
- Docker Compose multi-node (gateway, router, 3×indexnode, merger, reranker, etcd, Redis, Prometheus, Grafana)
- Prometheus metrics (8 metrics per table in plan) + Grafana dashboard JSON committed

### Phase 3 — AI + K8s + CI
- AI Reranker: query expansion (gpt-4o-mini, Redis-cached) + semantic rerank (text-embedding-3-small) + 0.4/0.6 score blend
- Kubernetes manifests + HPA (min=3, max=10, target CPU=60%) + PVC for BadgerDB
- GitHub Actions CI: lint, race tests, build, k6 load test with p99 < 200ms gate
- README with mermaid arch diagram + measured benchmark numbers + Grafana screenshot
- ARCHITECTURE.md with 6 key decisions documented
- Live demo URL on Fly.io

---

## What v1 IS NOT (anti-creep — reject these temptations)

**Reject these even if they feel "small" — that's the trap:**

- ❌ Adding more shards beyond 3 (the demo doesn't need it)
- ❌ Multi-region replication
- ❌ A web UI / search frontend (CLI + curl is fine)
- ❌ Authentication beyond X-API-Key (no OAuth, no JWT, no sessions)
- ❌ Adding Postgres / MySQL / "real" backing store (BadgerDB is the answer)
- ❌ Switching the embedding provider away from OpenAI text-embedding-3-small
- ❌ Adding tracing beyond the Prometheus metrics specified (OTel can come in v2)
- ❌ "Improving" the BM25 implementation (k1=1.5, b=0.75, ship it)
- ❌ Writing a custom consistent-hash library (use the design in the plan, don't optimize)
- ❌ Adding a Redis cluster (single Redis is fine for v1)
- ❌ Cross-shard query optimization (per-shard min-max norm is the v1 answer)
- ❌ Adding a "GoSearch Cloud" landing page
- ❌ Substituting Rust components "since you're learning Rust anyway"
- ❌ A second language SDK (no Python client, no JS client, no anything)
- ❌ More than the 6 binaries specified (gateway, router, indexnode, merger, reranker, ingester)

If a "small improvement" tempts you, write it down in `GOSEARCH_V2_IDEAS.md` and move on. v2 happens after Stance B applications go out, or never.

---

## Allowed v1 additions (because they help Stance B applications)

These are the ONLY additions allowed past the original plan, because they directly help Railway-tier interviews:

- ✅ **gRPC service health check exposed correctly** (HealthRequest/Response already in proto — just make sure it's wired)
- ✅ **OpenTelemetry trace export** from the gateway (small — adds ~1 day, demonstrates observability fluency for Railway/Datadog applications)
- ✅ **Prometheus client library exposed at `/metrics` per service** (already in plan, just confirm done)

That's it. Three additions, all small, all directly relevant to the observability-role pitch.

---

## Current status (FILL IN — do this today)

**Phase 1 progress:**
- [ ] Proto contracts (search.proto)
- [ ] Inverted index + BM25 + tests
- [ ] IndexNode gRPC server
- [ ] Wikipedia ingester
- [ ] API Gateway
- [ ] Single-node smoke test

**Phase 2 progress:**
- [ ] Consistent hash ring
- [ ] Query Router + circuit breakers
- [ ] Result Merger
- [ ] Docker Compose multi-node
- [ ] Prometheus + Grafana

**Phase 3 progress:**
- [ ] AI Reranker
- [ ] K8s manifests + HPA
- [ ] GitHub Actions CI + k6 load test
- [ ] README + ARCHITECTURE.md + benchmarks
- [ ] Fly.io deploy

**Stance B additions:**
- [ ] gRPC health check wired
- [ ] OTel trace export from gateway
- [ ] Prometheus /metrics endpoint per service

---

## Ship checklist (must all be true to call v1 done)

- [ ] `make build` — all 6 binaries compile
- [ ] `make up` — full stack starts from clean clone
- [ ] `make ingest-sample` — 10k docs distributed across 3 shards
- [ ] `curl` returns ranked results
- [ ] Grafana shows live metrics for all services
- [ ] `make load-test` — k6 reports p99 < 200ms at 1000 concurrent users
- [ ] `go test -race ./...` — all green
- [ ] Kill one indexnode → circuit breaker trips, partial results returned
- [ ] GitHub Actions green on main
- [ ] Public Fly.io URL live (gosearch.fly.dev or similar)
- [ ] README has architecture diagram + real benchmark numbers + Grafana screenshot
- [ ] ARCHITECTURE.md has 6 decisions documented

---

## Launch checklist (Day of ship)

- [ ] Public GitHub repo (clean commit history)
- [ ] Fly.io URL responds to curl with API key
- [ ] Launch blog post on shrikantj.dev (casual, observation-first per writing style memory)
- [ ] HN post (Show HN: GoSearch — distributed search engine in Go with AI reranking)
- [ ] /r/golang post
- [ ] /r/programming post
- [ ] lobste.rs post
- [ ] LinkedIn post (with link to blog + repo + Fly.io demo)
- [ ] Update CV with measured benchmark numbers in the GoSearch resume entry
- [ ] Update LinkedIn headline / About to mention GoSearch

---

## After ship — stop

After GoSearch v1 ships:

1. **Close the Go investment.** No GoSearch v2. No new Go side projects. Go is now a tool you use, not a portfolio thing to build.
2. **Pivot to Rust crash course** (Week 4 of Stance B sprint).
3. **Start applying** by Week 6 of sprint.

If you find yourself opening this file to add scope after ship date — close it and open `GOSEARCH_V2_IDEAS.md` instead. v2 is a future thing, not a now thing.
