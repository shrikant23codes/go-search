package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/shrikant23codes/gosearch/internal/index"
	pb "github.com/shrikant23codes/gosearch/proto/search"
	"google.golang.org/grpc"
)

func main() {
	shardId := envInt("SHARD_ID", 0)
	port := envInt("PORT", 9001)
	badgerPath := envStr("BADGER_PATH", fmt.Sprintf("data/shard-%d", shardId))

	idx, err := index.Open(badgerPath)

	if err != nil {
		log.Fatalf("Failed to open badget: %v", err)
	}

	// 3 servers on different ports:
	// grpcServr

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("failed to listen on port %d: %v", port, err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterSearchServiceServer(grpcServer, &indexNodeServer{index: idx, shardId: int32(shardId)})

	// Prometheus metrics server
	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.Handler())
		if err := http.ListenAndServe(fmt.Sprintf(":%d", port+1000), mux); err != nil {
			log.Fatalf("Failed to start Prometheus metrics server: %v", err)
		}
	}()

	// pprof
	go func() {
		if err := http.ListenAndServe(fmt.Sprintf(":%d", port+2000), nil); err != nil {
			log.Fatalf("Failed to start pprof server: %v", err)
		}
	}()

	log.Printf("indexnode shard=%d grpc=:%d metrics=:%d pprof=:%d", shardId, port, port+1000, port+2000)

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve grpc server: %v", err)
	}
}

func envStr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		return def
	}
	return def
}
