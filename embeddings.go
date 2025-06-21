package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

type EmbeddingsService struct {
	apiKey string
	client *http.Client
}

type OpenAIEmbeddingRequest struct {
	Input string `json:"input"`
	Model string `json:"model"`
}

type OpenAIEmbeddingResponse struct {
	Object string `json:"object"`
	Data   []struct {
		Object    string    `json:"object"`
		Embedding []float64 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
	Model string `json:"model"`
	Usage struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
}

func NewEmbeddingsService() *EmbeddingsService {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("Warning: OPENAI_API_KEY environment variable not set")
	}

	return &EmbeddingsService{
		apiKey: apiKey,
		client: &http.Client{},
	}
}

func (e *EmbeddingsService) GenerateEmbedding(text string) ([]float64, error) {
	if e.apiKey == "" {
		fmt.Printf("Cannot generate embedding: API key not configured\n")
		return nil, fmt.Errorf("API key not configured")
	}

	reqBody := OpenAIEmbeddingRequest{
		Input: text,
		Model: "text-embedding-3-small",
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.openai.com/v1/embeddings", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+e.apiKey)

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("API error (status %d): %s\n", resp.StatusCode, string(body))
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var embeddingResp OpenAIEmbeddingResponse
	if err := json.Unmarshal(body, &embeddingResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(embeddingResp.Data) > 0 {
		embedding := embeddingResp.Data[0].Embedding
		return embedding, nil
	}

	return nil, fmt.Errorf("no embedding data returned")
}
