package merger

import pb "github.com/shrikant23codes/gosearch/proto/search"

func deduplicate(shards [][]*pb.Document) []*pb.Document {
	bestById := make(map[string]*pb.Document)

	for _, shard := range shards {
		for _, doc := range shard {
			if doc == nil || doc.Id == "" {
				continue
			}
			existing, exists := bestById[doc.Id]
			if !exists || doc.Score > existing.Score {
				bestById[doc.Id] = doc
			}
		}
	}

	docs := make([]*pb.Document, 0, len(bestById))
	for _, doc := range bestById {
		docs = append(docs, doc)
	}
	return docs
}
