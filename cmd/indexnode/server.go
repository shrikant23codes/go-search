package main

import (
	"context"
	"time"

	"github.com/shrikant23codes/gosearch/internal/index"
	pb "github.com/shrikant23codes/gosearch/proto/search"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type indexNodeServer struct {
	pb.UnimplementedSearchServiceServer
	index   *index.Index
	shardId int32
}

func (s *indexNodeServer) Index(ct context.Context, req *pb.IndexRequest) (*pb.IndexResponse, error) {
	var count int64
	for _, doc := range req.Docs {
		err := s.index.Add(index.Document{
			ID:       doc.Id,
			Title:    doc.Title,
			Abstract: doc.Abstract,
		})
		if err != nil {
			continue
		}
		count++
	}
	return &pb.IndexResponse{
		IndexedCount: count,
	}, nil
}

func (s *indexNodeServer) Search(ct context.Context, req *pb.SearchRequest) (*pb.SearchResponse, error) {
	if req.Query == "" {
		return nil, status.Error(codes.InvalidArgument, "query cannot be empty..")
	}

	topK := int(req.TopK)
	if topK <= 0 {
		topK = 10
	}

	start := time.Now()
	results := s.index.Search(req.Query, topK)
	latency := time.Since(start)

	docs := make([]*pb.Document, len(results))
	for _, res := range results {
		docs = append(docs, &pb.Document{
			Id:       res.ID,
			Title:    res.Title,
			Abstract: res.Abstract,
			Score:    float32(res.Score),
		})
	}

	return &pb.SearchResponse{
		Docs:      docs,
		LatencyMs: int64(latency),
	}, nil
}

func (s *indexNodeServer) Health(ctx context.Context, req *pb.HealthRequest) (*pb.HealthResponse, error) {
	return &pb.HealthResponse{
		Status:    "OK",
		IndexSize: int64(s.index.Size()),
	}, nil
}
