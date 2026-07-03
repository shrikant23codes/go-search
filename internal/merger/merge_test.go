package merger

import (
	"testing"

	pb "github.com/shrikant23codes/gosearch/proto/search"
)

func TestMergeNormalizesEachShardBeforeSorting(t *testing.T) {
	responses := []*pb.SearchResponse{
		{
			Docs: []*pb.Document{
				{Id: "a-low", Score: 10},
				{Id: "a-high", Score: 20},
			},
		},
		{
			Docs: []*pb.Document{
				{Id: "b-low", Score: 100},
				{Id: "b-high", Score: 200},
			},
		},
	}

	got := Merge(responses, 4)

	wantIDs := []string{
		"a-high",
		"b-high",
		"a-low",
		"b-low",
	}

	if len(got) != len(wantIDs) {
		t.Fatalf("got %d documents, want %d", len(got), len(wantIDs))
	}

	for i, wantID := range wantIDs {
		if got[i].Id != wantID {
			t.Errorf(
				"result %d ID = %q, want %q",
				i,
				got[i].Id,
				wantID,
			)
		}
	}
}

func TestMergeDeduplicatesUsingHighestNormalizedScore(t *testing.T) {
	responses := []*pb.SearchResponse{
		{
			Docs: []*pb.Document{
				{Id: "duplicate", Score: 10},
				{Id: "a", Score: 0},
			},
		},
		{
			Docs: []*pb.Document{
				{Id: "duplicate", Score: 100},
				{Id: "b", Score: 200},
			},
		},
	}

	got := Merge(responses, 10)

	if len(got) != 3 {
		t.Fatalf("got %d documents, want 3", len(got))
	}

	var duplicate *pb.Document
	for _, doc := range got {
		if doc.Id == "duplicate" {
			duplicate = doc
			break
		}
	}

	if duplicate == nil {
		t.Fatal("duplicate document missing")
	}

	if duplicate.Score != 1 {
		t.Fatalf(
			"duplicate score = %v, want highest normalized score 1",
			duplicate.Score,
		)
	}
}

func TestMergeAppliesDeterministicTopK(t *testing.T) {
	responses := []*pb.SearchResponse{
		{
			Docs: []*pb.Document{
				{Id: "c", Score: 5},
				{Id: "a", Score: 5},
				{Id: "b", Score: 5},
			},
		},
	}

	got := Merge(responses, 2)

	if len(got) != 2 {
		t.Fatalf("got %d documents, want 2", len(got))
	}

	if got[0].Id != "a" || got[1].Id != "b" {
		t.Fatalf(
			"got IDs %q, %q; want a, b",
			got[0].Id,
			got[1].Id,
		)
	}
}
