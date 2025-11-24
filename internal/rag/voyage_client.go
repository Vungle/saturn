package rag

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	voyageAPIURL = "https://api.voyageai.com/v1/contextualizedembeddings"
	voyageModel  = "voyage-context-3"
)

// VoyageClient is a client for the Voyage AI contextualized embeddings API
type VoyageClient struct {
	apiKey     string
	httpClient *http.Client
}

// NewVoyageClient creates a new Voyage AI client
func NewVoyageClient(apiKey string) *VoyageClient {
	return &VoyageClient{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// voyageContextualEmbedRequest represents the request payload for contextualized embeddings
type voyageContextualEmbedRequest struct {
	Inputs    [][]string `json:"inputs"`     // List of lists: outer=batch, inner=context
	InputType string     `json:"input_type"` // "query" or "document"
	Model     string     `json:"model"`
}

// voyageEmbeddingItem represents a single embedding result
type voyageEmbeddingItem struct {
	Object    string    `json:"object"`
	Embedding []float32 `json:"embedding"`
	Index     int       `json:"index"`
}

// voyageDataItem represents a data item in the response
type voyageDataItem struct {
	Object string                `json:"object"`
	Data   []voyageEmbeddingItem `json:"data"`
	Index  int                   `json:"index"`
}

// voyageContextualEmbedResponse represents the response from Voyage API
type voyageContextualEmbedResponse struct {
	Object string           `json:"object"`
	Data   []voyageDataItem `json:"data"`
	Model  string           `json:"model"`
	Usage  struct {
		TotalTokens int `json:"total_tokens"`
	} `json:"usage"`
}

// EmbeddingResult contains the embedding vector and token usage
type EmbeddingResult struct {
	Embedding  []float32
	TokensUsed int
	Model      string
}

// EmbedQuery embeds a query string using Voyage's contextualized embeddings API
// Returns the embedding vector and token usage
func (c *VoyageClient) EmbedQuery(ctx context.Context, query string) (*EmbeddingResult, error) {
	// Prepare request payload
	// inputs is a list of lists: [["query"]] for single query
	reqBody := voyageContextualEmbedRequest{
		Inputs:    [][]string{{query}},
		InputType: "query",
		Model:     voyageModel,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", voyageAPIURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check for HTTP errors
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("voyage API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var voyageResp voyageContextualEmbedResponse
	if err := json.Unmarshal(body, &voyageResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Extract embedding from response structure
	// Response structure: data[0].data[0].embedding
	if len(voyageResp.Data) == 0 {
		return nil, fmt.Errorf("empty data array in response")
	}
	if len(voyageResp.Data[0].Data) == 0 {
		return nil, fmt.Errorf("empty embedding data in response")
	}

	embedding := voyageResp.Data[0].Data[0].Embedding
	if len(embedding) == 0 {
		return nil, fmt.Errorf("empty embedding vector")
	}

	return &EmbeddingResult{
		Embedding:  embedding,
		TokensUsed: voyageResp.Usage.TotalTokens,
		Model:      voyageResp.Model,
	}, nil
}
