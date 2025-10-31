package rag

import (
	"fmt"
	"os"
)

// CreateEmbeddingProvider creates an embedding provider based on the provider name
// Supports: voyage, openai (future), cohere (future), etc.
func CreateEmbeddingProvider(providerName string) (EmbeddingProvider, error) {
	switch providerName {
	case "voyage":
		apiKey := os.Getenv("VOYAGE_API_KEY")
		if apiKey == "" {
			return nil, fmt.Errorf("VOYAGE_API_KEY environment variable not set")
		}
		return NewVoyageClient(apiKey), nil

	default:
		return nil, fmt.Errorf("unsupported embedding provider: %s (supported: voyage)", providerName)
	}
}
