package main

import (
	"math"
)

func cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) {
		return 0.0
	}

	var dotProduct, normA, normB float64

	for i := 0; i < len(a); i++ {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	normA = math.Sqrt(normA)
	normB = math.Sqrt(normB)

	if normA == 0 || normB == 0 {
		return 0.0
	}

	return dotProduct / (normA * normB)
}

type SimilarityResult struct {
	Text       string
	Similarity float64
}

func compareWithStaticEmbeddings(inputEmbedding []float64) []SimilarityResult {
	results := make([]SimilarityResult, len(staticExamples))

	for i, example := range staticExamples {
		similarity := cosineSimilarity(inputEmbedding, example.Embedding)
		results[i] = SimilarityResult{
			Text:       example.Text,
			Similarity: similarity,
		}
	}

	return results
}
