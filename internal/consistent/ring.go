package consistent

import (
	"fmt"
	"hash/crc32"
	"slices"
	"sort"
	"strconv"
	"sync"
)

// virtual nodes to distribute keys
// evenly
const vnodes = 150

type Node struct {
	ID      string
	Address string
}

type Ring struct {
	mu      sync.RWMutex
	ring    []uint32
	hashMap map[uint32]Node
}

func NewRing() *Ring {
	return &Ring{
		hashMap: make(map[uint32]Node),
	}
}

func (r *Ring) Add(node Node) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i := 0; i < vnodes; i++ {
		h := hash(node.ID + "#" + strconv.Itoa(i))
		r.ring = append(r.ring, h)
		r.hashMap[h] = node
	}
	// Always sort ring after adding nodes to maintain order.
	slices.Sort(r.ring)
}

func (r *Ring) Remove(nodeId string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := 0; i < vnodes; i++ {
		h := hash(nodeId + "#" + strconv.Itoa(i))
		delete(r.hashMap, h)
	}
	r.ring = r.ring[:0]
	for h := range r.hashMap {
		r.ring = append(r.ring, h)
	}
	// sort ring after rebuilding to maintain order.
	slices.Sort(r.ring)
}

func (r *Ring) Get(key string) (Node, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(r.ring) == 0 {
		return Node{}, false
	}

	h := hash(key)
	idx := sort.Search(len(r.ring), func(i int) bool {
		return r.ring[i] >= h
	})
	// wrap around the ring
	if idx == len(r.ring) {
		idx = 0
	}

	return r.hashMap[r.ring[idx]], true

}

func (r *Ring) GetAll() []Node {
	r.mu.RLock()
	defer r.mu.RUnlock()
	seen := make(map[string]struct{})

	nodes := make([]Node, 0)
	for _, node := range r.hashMap {
		if _, exists := seen[node.ID]; !exists {
			seen[node.ID] = struct{}{}
			nodes = append(nodes, node)
		}
	}

	return nodes
}

func (r *Ring) Len() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.ring)
}

func (r *Ring) String() string {
	nodes := r.GetAll()
	return fmt.Sprintf("Ring{nodes: %d, vnodes: %d}", len(nodes), len(nodes)*vnodes)
}

func hash(key string) uint32 {
	return crc32.ChecksumIEEE([]byte(key))
}
