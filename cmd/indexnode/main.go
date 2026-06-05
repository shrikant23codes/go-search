package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/shrikant23codes/gosearch/internal/consistent"
	"github.com/shrikant23codes/gosearch/internal/index"
	pb "github.com/shrikant23codes/gosearch/proto/search"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
)

func main() {
	shardId := envInt("SHARD_ID", 0)
	port := envInt("PORT", 9001)
	badgerPath := envStr("BADGER_PATH", fmt.Sprintf("data/shard-%d", shardId))
	etcdEndpoints := strings.Split(envStr("ETCD_ENDPOINTS", "localhost:2379"), ",")

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

	go registerWithEtcd(etcdEndpoints, shardId, port)

	log.Printf("indexnode shard=%d grpc=:%d metrics=:%d pprof=:%d", shardId, port, port+1000, port+2000)

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("Failed to serve grpc server: %v", err)
	}
}

func registerWithEtcd(endpoints []string, shardId, port int) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
	})

	if err != nil {
		log.Printf("etcd connect failed: %v (cotinuing without registration)", err)
		return
	}
	defer cli.Close()

	key := fmt.Sprintf("/gosearch/nodes/shard-%d", shardId)
	nodeJSON, _ := json.Marshal(consistent.Node{
		ID:      fmt.Sprintf("shard-%d", shardId),
		Address: fmt.Sprintf("localhost:%d", port),
	})

	val := string(nodeJSON)

	// Register lease with etcd
	for {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		lease, err := cli.Grant(ctx, 10) // 10 sec TTL
		cancel()

		if err != nil {
			log.Printf("etcd grant: %v", err)
			time.Sleep(3 * time.Second)
			continue
		}

		_, err = cli.Put(context.Background(), key, val, clientv3.WithLease(lease.ID))
		if err != nil {
			log.Printf("etcd put: %v", err)
			time.Sleep(3 * time.Second)
			continue
		}

		ch, keepErr := cli.KeepAlive(context.Background(), lease.ID) // Spawns internal goroutine
		if keepErr != nil {
			log.Printf("etcd keepAlive: %v", err)
			time.Sleep(3 * time.Second)
			continue
		}

		for range ch {
			// drain channel to keep lease alive
		}
		log.Printf("etcd lease expired.. re-register")
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
