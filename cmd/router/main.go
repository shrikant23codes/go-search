package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"github.com/shrikant23codes/gosearch/internal/consistent"
	pb "github.com/shrikant23codes/gosearch/proto/search"
	"google.golang.org/grpc"
)

func main() {
	port := envStr("PORT", "9100")
	etcdEndPoints := splitAndTrim(envStr("ETCD_ENDPOINTS", "localhost:2379"))

	ring := consistent.NewRing()

	watcher, err := consistent.NewEtcdWatcher(etcdEndPoints, ring)
	if err != nil {
		log.Fatalf("router: connect etcd error: %v", err)
	}
	defer watcher.Close()

	loadCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	// Pass ctx so that if etcd is not available, as we call etcd to fetch nodes. we dont block.
	if err := watcher.LoadExistingNodes(loadCtx); err != nil {
		log.Fatalf("router: loading nodes from etcd: %v", err)
		cancel()
	}

	cancel()
	// Start watching for changes in etcd in a separate goroutine
	go watcher.Watch(context.Background())

	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		log.Fatalf("router: listen on port %s .. error: %v", port, err)
	}

	server := NewRouterServer(ring, 50*time.Millisecond)
	defer server.Close()

	grpcServer := grpc.NewServer()
	pb.RegisterSearchServiceServer(grpcServer, server)

	log.Printf("router listening on :%s", port)

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("router: grpc server error: %v", err)
	}

}

func envStr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func splitAndTrim(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}
