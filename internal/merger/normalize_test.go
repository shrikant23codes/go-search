package merger

import (
	"testing"

	pb "github.com/shrikant23codes/gosearch/proto/search"
)

func TestNormalizeShard(t *testing.T) {
	input := []*pb.Document{
		{Id: "low", Score: 2}, {Id: "middle", Score: 6},
		{Id: "high", Score: 10},
	}

	got := normalizeShard(input)
	wantScores := []float32{0, 0.5, 1}

	if len(wantScores) != len(input) {
		t.Fatalf("got %d documents, want %d", len(got), len(wantScores))
	}

	for i, want := range wantScores {
		if !almostEqual(got[i].Score, want, 1e-6) {
			t.Errorf(
				"document %q score = %v, want %v",
				got[i].Id,
				got[i].Score,
				want,
			)
		}
	}
}

func almostEqual(a, b, epsilon float32) bool {
	return abs(a-b) < epsilon
}

func abs(x float32) float32 {
	if x < 0 {
		return -x
	}
	return x
}
