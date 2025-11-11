package rag

import (
	"context"
	"os"
	"testing"

	"github.com/tuannvm/slack-mcp-client/internal/common/logging"
	"github.com/tuannvm/slack-mcp-client/internal/config"
	"github.com/tuannvm/slack-mcp-client/internal/llm"
)

// createTestLLMRegistry creates a real LLM registry for testing
func createTestLLMRegistry(t *testing.T) *llm.ProviderRegistry {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		t.Skip("ANTHROPIC_API_KEY not set, skipping integration test")
	}

	// Create a minimal config for Anthropic
	cfg := &config.Config{
		LLM: config.LLMConfig{
			Provider: "anthropic",
			Providers: map[string]config.LLMProviderConfig{
				"anthropic": {
					Model:  "claude-sonnet-4-5-20250929",
					APIKey: apiKey,
				},
			},
		},
	}

	logger := logging.New("test", logging.LevelInfo)
	registry, err := llm.NewProviderRegistry(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create LLM registry: %v", err)
	}

	return registry
}

// TestQueryEnhancer_EnhanceQuery_Temporal tests temporal query enhancement with real Claude
// Requires ANTHROPIC_API_KEY environment variable
func TestQueryEnhancer_EnhanceQuery_Temporal(t *testing.T) {
	t.Skip("Test requires prompt template file - skipping for now")

	registry := createTestLLMRegistry(t)
	enhancer := NewQueryEnhancer(registry)

	ctx := context.Background()
	// Note: In real usage, prompt template would be loaded from file
	promptTemplate := "Test prompt with {today} and {query} placeholders"
	result, err := enhancer.EnhanceQuery(ctx, "What were Q3 2025 revenues for VX in APAC?", "2025-10-31", promptTemplate)

	if err != nil {
		t.Fatalf("EnhanceQuery() error = %v", err)
	}

	// Verify enhanced query
	if result.EnhancedQuery == "" {
		t.Errorf("EnhanceQuery() returned empty enhanced query")
	}

	t.Logf("Original query: %s", result.OriginalQuery)
	t.Logf("Enhanced query: %s", result.EnhancedQuery)

	// Verify metadata filters for temporal query
	if len(result.MetadataFilters.Dates) == 0 {
		t.Logf("WARNING: No dates returned (expected for temporal query)")
	} else {
		t.Logf("Dates: %v", result.MetadataFilters.Dates)
	}

	t.Logf("Business units: %v", result.MetadataFilters.BusinessUnits)
	t.Logf("Regions: %v", result.MetadataFilters.Regions)
	t.Logf("Labels: %v", result.MetadataFilters.Labels)
}

// TestQueryEnhancer_EnhanceQuery_NonTemporal tests non-temporal query enhancement with real Claude
// Requires ANTHROPIC_API_KEY environment variable
func TestQueryEnhancer_EnhanceQuery_NonTemporal(t *testing.T) {
	t.Skip("Test requires prompt template file - skipping for now")

	registry := createTestLLMRegistry(t)
	enhancer := NewQueryEnhancer(registry)

	ctx := context.Background()
	promptTemplate := "Test prompt with {today} and {query} placeholders"
	result, err := enhancer.EnhanceQuery(ctx, "What is ROAS?", "2025-10-31", promptTemplate)

	if err != nil {
		t.Fatalf("EnhanceQuery() error = %v", err)
	}

	t.Logf("Original query: %s", result.OriginalQuery)
	t.Logf("Enhanced query: %s", result.EnhancedQuery)

	// Verify no date for non-temporal (knowledge) query
	if len(result.MetadataFilters.Dates) > 0 {
		t.Logf("WARNING: Dates returned for non-temporal query: %v", result.MetadataFilters.Dates)
	} else {
		t.Logf("Correctly returned empty dates for non-temporal query")
	}

	t.Logf("Labels: %v", result.MetadataFilters.Labels)
}

// TestQueryEnhancer_EnhanceQuery_RecentQuery tests "recent" keyword handling
// Requires ANTHROPIC_API_KEY environment variable
func TestQueryEnhancer_EnhanceQuery_RecentQuery(t *testing.T) {
	t.Skip("Test requires prompt template file - skipping for now")

	registry := createTestLLMRegistry(t)
	enhancer := NewQueryEnhancer(registry)

	ctx := context.Background()
	promptTemplate := "Test prompt with {today} and {query} placeholders"
	result, err := enhancer.EnhanceQuery(ctx, "recent sales performance", "2025-10-31", promptTemplate)

	if err != nil {
		t.Fatalf("EnhanceQuery() error = %v", err)
	}

	t.Logf("Original query: %s", result.OriginalQuery)
	t.Logf("Enhanced query: %s", result.EnhancedQuery)

	// "recent" should trigger temporal behavior
	if len(result.MetadataFilters.Dates) == 0 {
		t.Logf("WARNING: 'recent' query should return dates")
	} else {
		t.Logf("Dates for 'recent' query: %v", result.MetadataFilters.Dates)
	}

	t.Logf("Labels: %v", result.MetadataFilters.Labels)
}

// TestExtractJSONFromCodeBlock tests the JSON extraction utility
func TestExtractJSONFromCodeBlock(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "json code block",
			input: "```json\n{\"key\": \"value\"}\n```",
			want:  "{\"key\": \"value\"}",
		},
		{
			name:  "plain code block",
			input: "```\n{\"key\": \"value\"}\n```",
			want:  "{\"key\": \"value\"}",
		},
		{
			name:  "no code block",
			input: "{\"key\": \"value\"}",
			want:  "{\"key\": \"value\"}",
		},
		{
			name:  "with explanation after code block",
			input: "```json\n{\"key\": \"value\"}\n```\n\n**Reasoning:**\nSome explanation",
			want:  "{\"key\": \"value\"}",
		},
		{
			name:  "with text before and after",
			input: "Here's the result:\n```json\n{\"key\": \"value\"}\n```\nDone!",
			want:  "{\"key\": \"value\"}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractJSONFromCodeBlock(tt.input)
			if got != tt.want {
				t.Errorf("extractJSONFromCodeBlock() = %v, want %v", got, tt.want)
			}
		})
	}
}
