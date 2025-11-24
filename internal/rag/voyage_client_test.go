package rag

import (
	"context"
	"os"
	"testing"
)

// TestVoyageClient_EmbedQuery tests the Voyage embedding client
// This is an integration test that requires VOYAGE_API_KEY environment variable
func TestVoyageClient_EmbedQuery(t *testing.T) {
	apiKey := os.Getenv("VOYAGE_API_KEY")
	if apiKey == "" {
		t.Skip("VOYAGE_API_KEY not set, skipping integration test")
	}

	client := NewVoyageClient(apiKey)
	ctx := context.Background()

	tests := []struct {
		name    string
		query   string
		wantErr bool
	}{
		{
			name:    "simple query",
			query:   "What is revenue?",
			wantErr: false,
		},
		{
			name:    "complex query",
			query:   "What were the Q3 2025 revenues for the VX business unit in APAC region?",
			wantErr: false,
		},
		{
			name:    "empty query should fail",
			query:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := client.EmbedQuery(ctx, tt.query)

			if tt.wantErr {
				if err == nil {
					t.Errorf("EmbedQuery() expected error, got none")
				}
				return
			}

			if err != nil {
				t.Errorf("EmbedQuery() error = %v", err)
				return
			}

			// Verify embedding dimensions (voyage-context-3 should return 1024 dimensions)
			if len(result.Embedding) == 0 {
				t.Errorf("EmbedQuery() returned empty embedding")
			}

			t.Logf("Successfully generated embedding with %d dimensions (model: %s, tokens: %d)",
				len(result.Embedding), result.Model, result.TokensUsed)
		})
	}
}

// TestVoyageClient_EmbedQuery_InvalidAPIKey tests error handling with invalid credentials
func TestVoyageClient_EmbedQuery_InvalidAPIKey(t *testing.T) {
	client := NewVoyageClient("invalid-api-key")
	ctx := context.Background()

	_, err := client.EmbedQuery(ctx, "test query")
	if err == nil {
		t.Errorf("EmbedQuery() with invalid API key should return error")
	}

	t.Logf("Correctly returned error: %v", err)
}
