package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

type EmbeddingRequest struct {
	Input string `json:"input"`
	Model string `json:"model"`
}

type EmbeddingResponse struct {
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

func getEmbedding(text, apiKey string) ([]float64, error) {
	reqBody := EmbeddingRequest{
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
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var embeddingResp EmbeddingResponse
	if err := json.Unmarshal(body, &embeddingResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(embeddingResp.Data) == 0 {
		return nil, fmt.Errorf("no embedding data returned")
	}

	return embeddingResp.Data[0].Embedding, nil
}

func formatEmbeddingAsGoSlice(embedding []float64) string {
	result := "[]float64{"
	for i, val := range embedding {
		result += fmt.Sprintf("%.8f", val)
		if i < len(embedding)-1 {
			result += ", "
		}
	}
	result += "}"
	return result
}

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("Error: OPENAI_API_KEY environment variable not set")
		os.Exit(1)
	}

	// Define two contrasting example strings
	examples := []string{
		"I hate the state of california.",
		"Washington is a really great place.",
	}

	fmt.Println("// Generated static embeddings for demo")
	fmt.Println("package main")
	fmt.Println()
	fmt.Println("type StaticEmbedding struct {")
	fmt.Println("\tText      string")
	fmt.Println("\tEmbedding []float64")
	fmt.Println("}")
	fmt.Println()
	fmt.Println("var staticExamples = []StaticEmbedding{")

	for i, text := range examples {

		embedding, err := getEmbedding(text, apiKey)
		if err != nil {
			fmt.Printf("Error generating embedding for %q: %v\n", text, err)
			os.Exit(1)
		}

		fmt.Printf("\t{\n")
		fmt.Printf("\t\tText: %q,\n", text)
		fmt.Printf("\t\tEmbedding: %s,\n", formatEmbeddingAsGoSlice(embedding))
		fmt.Printf("\t},\n")

		if i < len(examples)-1 {
			fmt.Println()
		}
	}

	fmt.Println("}")
}
