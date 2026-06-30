package consistent

import "testing"

func TestAddSameNodeIsIdempotentAndUpdatesAddress(t *testing.T) {
	ring := NewRing()

	ring.Add(Node{
		ID:      "shard-1",
		Address: "localhost:9001",
	})
	ring.Add(Node{
		ID:      "shard-1",
		Address: "localhost:9101",
	})

	if got := ring.Len(); got != vnodes {
		t.Fatalf("ring contains %d positions, want %d", got, vnodes)
	}

	nodes := ring.GetAll()
	if len(nodes) != 1 {
		t.Fatalf("GetAll returned %d nodes, want 1", len(nodes))
	}

	if nodes[0].Address != "localhost:9101" {
		t.Fatalf(
			"node address = %q, want localhost:9101",
			nodes[0].Address,
		)
	}
}
