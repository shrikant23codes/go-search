package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	gsmiddleware "github.com/shrikant23codes/gosearch/internal/middleware"
	pb "github.com/shrikant23codes/gosearch/proto/search"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	port := envStr("PORT", "8080")
	indexNodeAddr := envStr("INDEXNODE_ADDR", "localhost:9001")
	apiKeys := strings.Split(envStr("GOSEARCH_API_KEYS", "dev-key"), ",")

	// grpc connection to indexnode
	conn, err := grpc.NewClient(indexNodeAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to indexnode at %s: %v", &indexNodeAddr, err)
	}

	client := pb.NewSearchServiceClient(conn)

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(gsmiddleware.Auth(apiKeys))
	r.Use(gsmiddleware.RateLimit())

	r.Get("/search", searchHandler(client))
	r.Get("/health", healthHandler(client))
	// For 3rd party handler use Handle as it returns handler instead of handler func
	r.Handle("/metrics", promhttp.Handler())

	log.Printf("Gateway listening on :%s", port)
	// Gateway is our http server
	if err := http.ListenAndServe(fmt.Sprintf(":%s", port), r); err != nil {
		log.Fatalf("Gateway HTTP serve: %v", err)
	}

}

func envStr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func searchHandler(client pb.SearchServiceClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("q")
		if query == "" {
			http.Error(w, "missing q param", http.StatusBadRequest)
			return
		}

		topK := int32(10)
		ctx := r.Context()
		// We pass the context from http request to grpc call.
		// If client cancels http req, grpc call also get canclled.
		resp, err := client.Search(ctx, &pb.SearchRequest{Query: query, TopK: topK})
		if err != nil {
			http.Error(w, fmt.Sprintf("search failed with error: %s", err), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"latency_ms":%d,"results":[`, resp.LatencyMs)
		for i, doc := range resp.Docs {
			if i > 0 {
				fmt.Fprintf(w, ",")
			}
			fmt.Fprintf(w, `{"id":%q,"title":%q,"abstract":%q,"score":%f}`, doc.Id, doc.Title, doc.Abstract, doc.Score)
		}
		fmt.Fprintf(w, "]}")
	}
}

func healthHandler(client pb.SearchServiceClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		resp, err := client.Health(ctx, &pb.HealthRequest{})
		if err != nil {
			http.Error(w, "indexnode unreachable", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":%q,"index_size":%d}`, resp.Status, resp.IndexSize)
	}
}
