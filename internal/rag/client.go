// Package rag provides a RAG client wrapper for MCP integration
package rag

import (
	"context"
	"fmt"
	"strings"

	"github.com/tuannvm/slack-mcp-client/internal/llm"
)

// Client wraps vector providers to implement the MCP tool interface
// This allows the LLM-MCP bridge to treat RAG as a regular MCP tool
type Client struct {
	provider          VectorProvider
	embeddingProvider EmbeddingProvider // Interface for embedding providers (Voyage, OpenAI, etc.)
	queryEnhancer     *QueryEnhancer
	llmRegistry       *llm.ProviderRegistry
	config            map[string]interface{} // Raw config for accessing provider-specific settings
}

// NewClient creates a new RAG client with simple provider (legacy compatibility)
func NewClient(ragDatabase string) *Client {
	config := map[string]interface{}{
		"provider":      "simple",
		"database_path": ragDatabase,
	}

	provider, err := CreateProviderFromConfig(config)
	if err != nil {
		// Fallback to simple provider for backward compatibility
		simpleProvider := NewSimpleProvider(ragDatabase)
		_ = simpleProvider.Initialize(context.Background())
		return &Client{
			provider: simpleProvider,
			config:   config,
		}
	}

	return &Client{
		provider: provider,
		config:   config,
	}
}

// NewClientWithProvider creates a new RAG client with specified provider
func NewClientWithProvider(providerType string, config map[string]interface{}) (*Client, error) {
	// Ensure provider type is set in config
	if config == nil {
		config = make(map[string]interface{})
	}
	config["provider"] = providerType

	provider, err := CreateProviderFromConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create provider: %w", err)
	}

	return &Client{
		provider: provider,
		config:   config,
	}, nil
}

// SetEnhancedSearchDependencies sets optional dependencies for enhanced RAG search
// If not set, will fall back to basic search without query enhancement
func (c *Client) SetEnhancedSearchDependencies(llmRegistry *llm.ProviderRegistry, embeddingProvider EmbeddingProvider) {
	c.llmRegistry = llmRegistry
	c.embeddingProvider = embeddingProvider

	// Initialize query enhancer if LLM registry is available
	if llmRegistry != nil {
		c.queryEnhancer = NewQueryEnhancer(llmRegistry)
	}
}

// CallTool implements the MCP tool interface for RAG operations
func (c *Client) CallTool(ctx context.Context, toolName string, args map[string]interface{}) (string, error) {
	if args == nil {
		return "", fmt.Errorf("arguments cannot be nil")
	}

	switch toolName {
	case "rag_search":
		return c.handleRAGSearch(ctx, args)
	case "rag_ingest":
		return c.handleRAGIngest(ctx, args)
	case "rag_stats":
		return c.handleRAGStats(ctx, args)
	default:
		return "", fmt.Errorf("unknown RAG tool: %s. Available tools: rag_search, rag_ingest, rag_stats", toolName)
	}
}

// handleRAGSearch processes search requests with enhanced pipeline
func (c *Client) handleRAGSearch(ctx context.Context, args map[string]interface{}) (string, error) {
	// Extract and validate query parameter
	originalQuery, err := c.extractStringParam(args, "query", true)
	if err != nil {
		return "", err
	}

	// Build search options
	// Extract max_results from config, default to 20
	maxResults := 20
	if c.config != nil {
		if maxResultsFloat, ok := c.config["max_results"].(float64); ok {
			maxResults = int(maxResultsFloat)
		} else if maxResultsInt, ok := c.config["max_results"].(int); ok {
			maxResults = maxResultsInt
		}
	}

	searchOpts := SearchOptions{
		Limit:    maxResults,
		Metadata: make(map[string]string),
	}

	// Step 1: Get today's date
	today := GetTodayDate()

	// Step 2: Enhance query with LLM (if available)
	var enhancedQuery string
	var dateFilter []string

	if c.queryEnhancer != nil {
		enhanced, err := c.queryEnhancer.EnhanceQuery(ctx, originalQuery, today)
		if err != nil {
			// Log error but continue with original query
			fmt.Printf("Warning: query enhancement failed: %v\n", err)
			enhancedQuery = originalQuery
		} else {
			enhancedQuery = enhanced.EnhancedQuery

			// Step 3: Expand date range if temporal query
			if enhanced.MetadataFilters.GeneratedDate != nil {
				dateFilter, err = ExpandDateRange(*enhanced.MetadataFilters.GeneratedDate, 7)
				if err != nil {
					fmt.Printf("Warning: date range expansion failed: %v\n", err)
				} else {
					searchOpts.DateFilter = dateFilter
				}
			}
		}
	} else {
		enhancedQuery = originalQuery
	}

	// Step 4: Embed query with embedding provider (if available)
	if c.embeddingProvider != nil {
		queryVector, err := c.embeddingProvider.EmbedQuery(ctx, enhancedQuery)
		if err != nil {
			return "", fmt.Errorf("failed to embed query: %w", err)
		}
		searchOpts.QueryVector = queryVector
	}

	// Step 5: Perform search using the provider
	results, err := c.provider.Search(ctx, enhancedQuery, searchOpts)
	if err != nil {
		return "", fmt.Errorf("search failed: %w", err)
	}

	// Format results for display
	if len(results) == 0 {
		return "No relevant context found for query: '" + originalQuery + "'", nil
	}

	// Step 6: Sort results by report_generated_date (newest first)
	// TODO: Add reranking step here in the future
	sortResultsByDate(results)

	// Build response string
	var response strings.Builder
	response.WriteString(fmt.Sprintf("Found %d relevant context(s) for '%s':\n", len(results), originalQuery))

	for i, result := range results {
		response.WriteString(fmt.Sprintf("--- Context %d ---\n", i+1))

		// Add source information if available
		if result.FileName != "" {
			response.WriteString(fmt.Sprintf("Source: %s", result.FileName))
			if result.Score > 0 {
				response.WriteString(fmt.Sprintf(" (score: %.2f)", result.Score))
			}
			response.WriteString("\n")
		}

		// Add metadata if available
		if date, exists := result.Metadata["report_generated_date"]; exists {
			response.WriteString(fmt.Sprintf("Date: %s\n", date))
		}

		// Add content
		response.WriteString(fmt.Sprintf("Content: %s\n", result.Content))

		// Add highlights if available
		if len(result.Highlights) > 0 {
			response.WriteString(fmt.Sprintf("Highlights: %s\n", strings.Join(result.Highlights, " | ")))
		}
	}

	return response.String(), nil
}

// sortResultsByDate sorts results by report_generated_date in descending order (newest first)
func sortResultsByDate(results []SearchResult) {
	// Simple bubble sort - adequate for small result sets
	for i := 0; i < len(results); i++ {
		for j := i + 1; j < len(results); j++ {
			dateI := results[i].Metadata["report_generated_date"]
			dateJ := results[j].Metadata["report_generated_date"]
			if dateJ > dateI { // Descending order
				results[i], results[j] = results[j], results[i]
			}
		}
	}
}

// handleRAGIngest processes document ingestion requests
func (c *Client) handleRAGIngest(ctx context.Context, args map[string]interface{}) (string, error) {
	// Extract file path parameter
	filePath, err := c.extractStringParam(args, "file_path", true)
	if err != nil {
		return "", err
	}

	// Extract optional metadata
	metadata := make(map[string]string)
	if metaParam, exists := args["metadata"]; exists {
		if metaMap, ok := metaParam.(map[string]interface{}); ok {
			for k, v := range metaMap {
				if str, ok := v.(string); ok {
					metadata[k] = str
				} else {
					metadata[k] = fmt.Sprintf("%v", v)
				}
			}
		}
	}

	// Ingest the file
	fileID, err := c.provider.IngestFile(ctx, filePath, metadata)
	if err != nil {
		return "", fmt.Errorf("ingestion failed: %w", err)
	}

	return fmt.Sprintf("Successfully ingested file: %s (ID: %s)", filePath, fileID), nil
}

// handleRAGStats returns statistics about the vector store
func (c *Client) handleRAGStats(ctx context.Context, args map[string]interface{}) (string, error) {
	stats, err := c.provider.GetStats(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get stats: %w", err)
	}

	var response strings.Builder
	response.WriteString("RAG Vector Store Statistics:\n")
	response.WriteString(fmt.Sprintf("Total Files: %d\n", stats.TotalFiles))
	response.WriteString(fmt.Sprintf("Total Chunks: %d\n", stats.TotalChunks))
	response.WriteString(fmt.Sprintf("Processing Files: %d\n", stats.ProcessingFiles))
	response.WriteString(fmt.Sprintf("Failed Files: %d\n", stats.FailedFiles))

	if stats.StorageSizeBytes > 0 {
		response.WriteString(fmt.Sprintf("Storage Size: %.2f MB\n", float64(stats.StorageSizeBytes)/(1024*1024)))
	}

	response.WriteString(fmt.Sprintf("Last Updated: %s\n", stats.LastUpdated.Format("2006-01-02 15:04:05")))

	return response.String(), nil
}

// extractStringParam extracts and validates a string parameter from args
func (c *Client) extractStringParam(args map[string]interface{}, paramName string, required bool) (string, error) {
	value, exists := args[paramName]
	if !exists {
		if required {
			return "", fmt.Errorf("missing required parameter: %s", paramName)
		}
		return "", nil
	}

	strValue, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("parameter %s must be a string, got %T", paramName, value)
	}

	if required && strings.TrimSpace(strValue) == "" {
		return "", fmt.Errorf("parameter %s cannot be empty", paramName)
	}

	return strValue, nil
}

// GetProvider returns the underlying vector provider (for testing/debugging)
func (c *Client) GetProvider() VectorProvider {
	return c.provider
}

// Close cleans up the client and its provider
func (c *Client) Close() error {
	if c.provider != nil {
		return c.provider.Close()
	}
	return nil
}
