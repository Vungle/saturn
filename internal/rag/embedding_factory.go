package rag

import (
	"fmt"
)

// EmbeddingProviderConfig contains embedding provider-specific settings
type EmbeddingProviderConfig struct {
	APIKey string `json:"apiKey,omitempty"` // API key for the embedding provider
}

// CreateEmbeddingProvider creates an embedding provider based on the provider name and config
// Supports: voyage, openai (future), cohere (future), etc.
func CreateEmbeddingProvider(providerName string, config EmbeddingProviderConfig) (EmbeddingProvider, error) {
	switch providerName {
	case "voyage":
		if config.APIKey == "" {
			return nil, fmt.Errorf("API key is required for Voyage embedding provider")
		}
		return NewVoyageClient(config.APIKey), nil

	default:
		return nil, fmt.Errorf("unsupported embedding provider: %s (supported: voyage)", providerName)
	}
}
