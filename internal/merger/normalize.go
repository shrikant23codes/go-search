package merger

import (
	pb "github.com/shrikant23codes/gosearch/proto/search"
)

func normalizeShard(docs []*pb.Document) []*pb.Document {
	var minScore float32
	var maxScore float32
	initialized := false

	for _, doc := range docs {
		if doc == nil {
			continue
		}

		if !initialized {
			minScore = doc.Score
			maxScore = doc.Score
			initialized = true
		}

		if doc.Score < minScore {
			minScore = doc.Score
		}

		if doc.Score > maxScore {
			maxScore = doc.Score
		}
	}

	if !initialized {
		return nil
	}

	normalized := make([]*pb.Document, 0, len(docs))
	scoreRange := maxScore - minScore

	for _, doc := range docs {
		if doc == nil {
			continue
		}

		score := float32(1)

		if scoreRange > 0 {
			score = (doc.Score - minScore) / scoreRange
		}

		normalized = append(normalized, &pb.Document{
			Id:       doc.Id,
			Title:    doc.Title,
			Abstract: doc.Abstract,
			Score:    score,
		})
	}
	return normalized
}
