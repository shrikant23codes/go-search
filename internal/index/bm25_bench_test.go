package index

import (
	"fmt"
	"math/rand/v2"
	"strings"
	"testing"
)

var benchmarkVocab = []string{
	"machine", "learning", "neural", "network", "deep", "transformer",
	"golang", "python", "rust", "java", "kotlin", "swift",
	"search", "index", "query", "ranking", "scoring", "relevance",
	"kubernetes", "docker", "container", "cluster", "node", "pod",
	"database", "storage", "cache", "memory", "disk", "ssd",
	"distributed", "concurrent", "parallel", "asynchronous", "synchronous",
	"tcp", "http", "grpc", "websocket", "rest",
	"encryption", "hashing", "authentication", "authorization",
}

func generateDocs(numDocs int) []Document {
	r := rand.New(rand.NewPCG(42, 0))
	docs := make([]Document, numDocs)

	for i := 0; i < numDocs; i++ {
		titleLen := 5 + r.IntN(11)     // 5 to 15 words
		abstractLen := 30 + r.IntN(71) // 30 to 100 words
		docs[i] = Document{
			ID:       fmt.Sprintf("doc-%d", i),
			Title:    randomText(r, titleLen),
			Abstract: randomText(r, abstractLen),
		}
	}
	return docs
}

func randomText(r *rand.Rand, wordCount int) string {
	var sb strings.Builder
	for i := 0; i < wordCount; i++ {
		if i > 0 {
			sb.WriteByte(' ')
		}
		sb.WriteString(benchmarkVocab[r.IntN(len(benchmarkVocab))])
	}
	return sb.String()
}

func benchMarkSearch(b *testing.B, numDocs int) {
	idx := New()

	for _, doc := range generateDocs(numDocs) {
		if err := idx.Add(doc); err != nil {
			b.Fatalf("failed to add document: %v", err)
		}
	}

	queries := []string{
		"machine learning",
		"golang grpc",
		"distributed search index",
	}

	// Allocations are usually the bottleneck in Go (GC pressure), so always report them on perf-sensitive benchmarks
	b.ReportAllocs()

	for i := 0; b.Loop(); i++ {
		_ = idx.Search(queries[i%len(queries)], 10)
	}
}

func BenchmarkSearch_1kDocs(b *testing.B)   { benchMarkSearch(b, 1_000) }
func BenchmarkSearch_100kDocs(b *testing.B) { benchMarkSearch(b, 100_000) }

func BenchmarkSearch_1MDocs(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping 1M doc benchmark in short mode")
	}
	benchMarkSearch(b, 1_000_000)
}

// benchmark bulk ingest with batch size 500
func BenchmarkIndex_Batch500(b *testing.B) {
	docs := generateDocs(500)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; b.Loop(); i++ {
		idx := New()
		for _, doc := range docs {
			if err := idx.Add(doc); err != nil {
				b.Fatalf("failed to add document: %v", err)
			}
		}
	}
}
