package rag

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tuannvm/slack-mcp-client/internal/llm"
)

// MetadataFilters represents the metadata filters extracted from a query
type MetadataFilters struct {
	BusinessUnits []string `json:"business_units,omitempty"`
	Regions       []string `json:"regions,omitempty"`
	Dates         []string `json:"dates,omitempty"` // List of dates for temporal queries (YYYY-MM-DD format)
	Labels        []string `json:"labels,omitempty"`
}

// EnhancedQuery represents the result of query enhancement
type EnhancedQuery struct {
	EnhancedQuery   string          `json:"enhanced_query"`
	MetadataFilters MetadataFilters `json:"metadata_filters"`
	OriginalQuery   string          `json:"-"` // Not from LLM response
}

// QueryEnhancer enhances queries using LLM
type QueryEnhancer struct {
	llmRegistry *llm.ProviderRegistry
}

// NewQueryEnhancer creates a new query enhancer
func NewQueryEnhancer(llmRegistry *llm.ProviderRegistry) *QueryEnhancer {
	return &QueryEnhancer{
		llmRegistry: llmRegistry,
	}
}

// EnhanceQuery enhances a query by extracting metadata filters and improving the query text
func (qe *QueryEnhancer) EnhanceQuery(ctx context.Context, query string, today string) (*EnhancedQuery, error) {
	// Build the prompt by replacing placeholders
	prompt := strings.ReplaceAll(QueryEnhancementPromptTemplate, "{today}", today)
	prompt = strings.ReplaceAll(prompt, "{query}", query)

	// Get the primary LLM provider from registry
	provider, err := qe.llmRegistry.GetPrimaryProvider()
	if err != nil {
		return nil, fmt.Errorf("failed to get LLM provider: %w", err)
	}

	// Prepare message
	messages := []llm.RequestMessage{
		{
			Role:    "user",
			Content: prompt,
		},
	}

	// Call LLM with the prompt
	response, err := provider.GenerateChatCompletion(ctx, messages, llm.ProviderOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to call LLM: %w", err)
	}

	responseText := response.Content

	// Parse the JSON response
	var result EnhancedQuery
	if err := json.Unmarshal([]byte(responseText), &result); err != nil {
		// Try to extract JSON from code blocks if direct parsing fails
		responseText = extractJSONFromCodeBlock(responseText)
		if err := json.Unmarshal([]byte(responseText), &result); err != nil {
			return nil, fmt.Errorf("failed to parse LLM response as JSON: %w, response: %s", err, responseText)
		}
	}

	// Set the original query
	result.OriginalQuery = query

	return &result, nil
}

// extractJSONFromCodeBlock extracts JSON from markdown code blocks
func extractJSONFromCodeBlock(text string) string {
	// Try to find JSON in ```json or ``` code blocks
	text = strings.TrimSpace(text)

	// Look for opening fence (```json or ```)
	startIdx := strings.Index(text, "```json")
	fenceType := "```json"
	if startIdx == -1 {
		startIdx = strings.Index(text, "```")
		fenceType = "```"
	}

	// If we found an opening fence, extract content between fences
	if startIdx >= 0 {
		// Move past the opening fence
		text = text[startIdx+len(fenceType):]
		text = strings.TrimSpace(text)

		// Find the closing ``` (even if there's text after it)
		if endIdx := strings.Index(text, "```"); endIdx >= 0 {
			text = text[:endIdx]
		}
	}

	return strings.TrimSpace(text)
}
