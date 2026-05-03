# GoSearch — Makefile
#
# Targets are added as we reach each phase. v1 lives by the plan in
# GOSEARCH_PLAN.md and the scope lock in GOSEARCH_V1_LOCKED.md.
#
# Phase 1 only needs: proto, build, test, lint, tidy.
# Phase 2 adds: up, down (docker compose).
# Phase 3 adds: ingest, ingest-sample, load-test.

.PHONY: proto build test lint tidy clean

# Generate Go + gRPC code from search.proto.
#   --go_out / --go-grpc_out   : where to write generated files
#   paths=source_relative       : drop generated files next to the .proto
#                                 (rather than under a synthetic dir derived
#                                 from option go_package)
proto:
	protoc \
		--go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		proto/search/search.proto

# Compile every binary under cmd/.
build:
	go build ./cmd/...

# -race: detect data races at runtime. Slower but mandatory for a concurrent
# service like this — race conditions are the #1 thing reviewers will probe.
test:
	go test -race ./...

# golangci-lint runs many linters at once (errcheck, staticcheck, gosec, …).
# Install separately: brew install golangci-lint
lint:
	golangci-lint run

# Sync go.mod / go.sum with what the code actually imports.
# Run after generating new .pb.go files or adding new imports.
tidy:
	go mod tidy

# Wipe generated proto code. Useful when bumping protoc-gen-go versions.
clean:
	rm -f proto/search/*.pb.go
