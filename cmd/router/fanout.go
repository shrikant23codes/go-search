package main

import (
	"context"
	"errors"
	"log"
	"sort"
	"sync"
	"time"

	"github.com/shrikant23codes/gosearch/internal/consistent"
	pb "github.com/shrikant23codes/gosearch/proto/search"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type RouterServer struct {
	pb.UnimplementedSearchServiceServer

	ring         *consistent.Ring
	shardTimeout time.Duration

	mu      sync.RWMutex
	clients map[string]pb.SearchServiceClient
	conns   map[string]*grpc.ClientConn
}

func NewRouterServer(ring *consistent.Ring, shardTimeout time.Duration) *RouterServer {
	return &RouterServer{
		ring:         ring,
		shardTimeout: shardTimeout,
		clients:      make(map[string]pb.SearchServiceClient),
		conns:        make(map[string]*grpc.ClientConn),
	}
}

func (r *RouterServer) Health(ctx context.Context, req *pb.HealthRequest) (*pb.HealthResponse, error) {
	nodes := r.ring.GetAll()
	if len(nodes) == 0 {
		return &pb.HealthResponse{Status: "DEGRADED"}, nil
	}
	return &pb.HealthResponse{
		Status:    "OK",
		IndexSize: int64(len(nodes)),
	}, nil
}

func (r *RouterServer) Close() {
	r.mu.Lock()
	defer r.mu.Unlock()
	for nodeId, conn := range r.conns {
		if err := conn.Close(); err != nil {
			log.Printf("router: Close error for conn %s: %v", nodeId, err)
		}
	}
}

func (r *RouterServer) Search(ctx context.Context, req *pb.SearchRequest) (*pb.SearchResponse, error) {
	if req.Query == "" {
		return nil, status.Error(codes.InvalidArgument, "query is empty")
	}

	topK := req.TopK
	if topK <= 0 {
		topK = 10
	}

	start := time.Now()

	responses, err := r.fanout(ctx, req)

	if err != nil {
		return nil, err
	}

	docs := mergeTemporary(responses, int(topK))

	return &pb.SearchResponse{
		Docs:      docs,
		LatencyMs: int64(time.Since(start).Milliseconds()),
	}, nil

}

func mergeTemporary(responses []*pb.SearchResponse, topK int) []*pb.Document {
	seen := make(map[string]*pb.Document)
	for _, resp := range responses {
		if resp == nil {
			continue
		}

		for _, doc := range resp.Docs {
			existing, ok := seen[doc.Id]
			if !ok || doc.Score > existing.Score {
				seen[doc.Id] = doc
			}
		}
	}

	docs := make([]*pb.Document, 0, len(seen))
	for _, doc := range seen {
		docs = append(docs, doc)
	}

	sort.Slice(docs, func(i, j int) bool {
		return docs[i].Score > docs[j].Score
	})

	if topK > 0 && len(docs) > topK {
		docs = docs[:topK]
	}

	return docs
}

func (r *RouterServer) fanout(ctx context.Context, req *pb.SearchRequest) ([]*pb.SearchResponse, error) {
	nodes := r.ring.GetAll()

	if len(nodes) == 0 {
		return nil, status.Error(codes.Unavailable, "no index nodes registered")
	}

	// By using list where each shard only works on one index we tackle concurrency by sharing memory
	// but partition ownership clearly. So no need of mutex here
	responses := make([]*pb.SearchResponse, len(nodes))
	errs := make([]error, len(nodes))

	var wg sync.WaitGroup
	wg.Add(len(nodes))

	for i, node := range nodes {
		go func(i int, n consistent.Node) {
			defer wg.Done()

			shardCtx, cancel := context.WithTimeout(ctx, r.shardTimeout)
			defer cancel()

			client, err := r.ClientFor(n)
			if err != nil {
				errs[i] = err
				return
			}

			shardReq := &pb.SearchRequest{
				Query:   req.Query,
				TopK:    req.TopK,
				ShardId: req.ShardId,
			}
			res, err := client.Search(shardCtx, shardReq)
			if err != nil {
				errs[i] = err
				return
			}

			responses[i] = res
		}(i, node)
	}

	wg.Wait()

	successes := 0
	for i, resp := range responses {
		if resp != nil {
			successes++
			continue
		}
		if errs[i] != nil {
			log.Printf("router: shard %s failed for %v", nodes[i].ID, errs[i])
		}

	}

	if successes == 0 {
		return nil, status.Error(codes.Unavailable, "All shards failed")
	}

	return responses, nil

}

func (r *RouterServer) ClientFor(node consistent.Node) (pb.SearchServiceClient, error) {
	if node.Address == "" {
		return nil, errors.New("node has empty address")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if client, ok := r.clients[node.Address]; ok {
		return client, nil
	}

	conn, err := grpc.NewClient(
		node.Address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)

	if err != nil {
		return nil, err
	}

	client := pb.NewSearchServiceClient(conn)
	r.clients[node.Address] = client
	// We cache conn for cleanup as client doesn't have close
	r.conns[node.Address] = conn

	return client, nil
}
