package merger

import (
	"sort"

	pb "github.com/shrikant23codes/gosearch/proto/search"
)

func Merge(responses []*pb.SearchResponse, topK int) []*pb.Document {
	shards := make([][]*pb.Document, 0, len(responses))

	for _, response := range responses {
		if response == nil {
			continue
		}

		normalizedDocs := normalizeShard(response.Docs)
		if len(normalizedDocs) > 0 {
			shards = append(shards, normalizedDocs)
		}
	}

	dedupDocs := deduplicate(shards)

	sort.Slice(dedupDocs, func(i, j int) bool {
		if dedupDocs[i].Score == dedupDocs[j].Score {
			return dedupDocs[i].Id < dedupDocs[j].Id
		}
		return dedupDocs[i].Score > dedupDocs[j].Score
	})

	if len(dedupDocs) > topK {
		dedupDocs = dedupDocs[:topK]
	}

	return dedupDocs
}
