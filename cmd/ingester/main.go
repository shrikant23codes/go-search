package main

import (
	"bufio"
	"compress/bzip2"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	pb "github.com/shrikant23codes/gosearch/proto/search"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type cirrusDoc struct {
	Title       string `json:"title"`
	OpeningText string `json:"opening_text"`
	Namespace   int    `json:"namespace"`
}

func main() {
	dumpPath := flag.String("dump-path", "data/enwiki-cirrussearch.json.gz", "path to CirrusSearch .json.bz2 dump")
	addrs := flag.String("indexnode-addrs", "localhost:9001", "comma-separated indexnode gRPC addresses")
	batchSize := flag.Int("batch-size", 100, "docs per IndexRequest")
	limit := flag.Int("limit", 0, "max docs to ingest (0 = all)")
	flag.Parse()

	clients := makeClients(strings.Split(*addrs, ","))

	f, err := os.Open(*dumpPath)
	if err != nil {
		log.Fatalf("Failed to open dump file: %v", err)
	}
	defer f.Close()

	bz := bzip2.NewReader(f)

	var (
		batch    []*pb.Document
		ingested int
		start    = time.Now()
		isIndex  = true // CirrusSearch alternates: index line, doc line, index line, doc line
	)

	scanner := bufio.NewScanner(bz)
	scanner.Buffer(make([]byte, 4*1024*1024), 4*1024*1024) // 4mb buffer

	for scanner.Scan() {
		line := scanner.Bytes()

		if isIndex {
			isIndex = false
			continue // skip the {"index":{...}} line
		}
		isIndex = true

		var doc cirrusDoc
		if err := json.Unmarshal(line, &doc); err != nil {
			log.Printf("Malformed line.. %v", err)
			continue
		}

		// Only main namespace articles
		if doc.Namespace != 0 || doc.OpeningText == "" || doc.Title == "" {
			continue
		}

		batch = append(batch, &pb.Document{
			Id:       "https://en.wikipedia.org/wiki/" + strings.ReplaceAll(doc.Title, " ", "_"),
			Title:    doc.Title,
			Abstract: doc.OpeningText,
		})

		if len(batch) >= *batchSize {
			fmt.Println("[DEBUG] Sending batch of size", len(batch))
			sendBatch(clients, batch, ingested)
			ingested += len(batch)
			batch = batch[:0] // avoid realloc
		}
		if *limit > 0 && ingested >= *limit {
			break
		}

	}
	if err := scanner.Err(); err != nil {
		log.Fatalf("scan error: %v", err)
	}

	if len(batch) > 0 {
		sendBatch(clients, batch, ingested)
		ingested += len(batch)
	}

	log.Printf("done: %d docs ingested in %s", ingested, time.Since(start).Round(time.Millisecond))
}

func sendBatch(clients []pb.SearchServiceClient, batch []*pb.Document, cursor int) {
	client := clients[(cursor/len(batch))%len(clients)]

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := client.Index(ctx, &pb.IndexRequest{Docs: batch})
	if err != nil {
		log.Fatalf("Failed to index batch error is: %v", err)
	}

	log.Printf("indexed %d docs (total ~%d)", resp.IndexedCount, cursor+int(resp.IndexedCount))
}

func makeClients(addrs []string) []pb.SearchServiceClient {
	// Add 0 otherwise I was getting "index out of range" panic when cursor is 0 as it points to nil
	clients := make([]pb.SearchServiceClient, 0, len(addrs))
	for _, addr := range addrs {
		addr = strings.TrimSpace(addr)
		conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			log.Fatalf("Failed to connect to indexnode at %s: %v", addr, err)
		}
		// verify the node is reachable before ingesting
		c := pb.NewSearchServiceClient(conn)
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		_, err = c.Health(ctx, &pb.HealthRequest{})
		cancel()
		if err != nil {
			log.Fatalf("indexnode %s not reachable: %v", addr, err)
		}
		clients = append(clients, c)
	}
	return clients
}
