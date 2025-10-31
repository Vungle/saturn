package rag

import (
	"context"
	"os"
	"testing"
)

// TestS3Provider_Search tests the S3 vector search functionality
// This is an integration test that requires:
// - AWS_REGION, S3_VECTOR_BUCKET, S3_VECTOR_INDEX environment variables
// - AWS credentials configured (via AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY, or IAM role)
// - An existing S3 vector store with indexed data
func TestS3Provider_Search(t *testing.T) {
	bucketName := os.Getenv("S3_VECTOR_BUCKET")
	indexName := os.Getenv("S3_VECTOR_INDEX")
	region := os.Getenv("AWS_REGION")

	if bucketName == "" || indexName == "" {
		t.Skip("S3_VECTOR_BUCKET or S3_VECTOR_INDEX not set, skipping integration test")
	}

	if region == "" {
		region = "us-east-1"
	}

	// Create S3 provider
	config := map[string]interface{}{
		"bucket_name": bucketName,
		"index_name":  indexName,
		"region":      region,
	}

	provider, err := NewS3Provider(config)
	if err != nil {
		t.Fatalf("NewS3Provider() error = %v", err)
	}

	// Initialize the provider
	ctx := context.Background()
	err = provider.Initialize(ctx)
	if err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}

	// Test vector search
	// Note: This requires a pre-computed query vector
	// In real usage, this would come from Voyage embeddings
	tests := []struct {
		name        string
		query       string
		queryVector []float32
		options     SearchOptions
		wantErr     bool
	}{
		{
			name:  "search with no vector should fail",
			query: "test query",
			options: SearchOptions{
				Limit: 5,
			},
			wantErr: true,
		},
		// TODO: Add test with actual query vector once we have sample data
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.options.QueryVector = tt.queryVector

			results, err := provider.Search(ctx, tt.query, tt.options)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Search() expected error, got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Search() error = %v", err)
				return
			}

			t.Logf("Search returned %d results", len(results))

			// Verify results structure
			for i, result := range results {
				if result.FileID == "" {
					t.Errorf("Result %d has empty FileID", i)
				}
				if result.Score < 0 {
					t.Errorf("Result %d has negative score: %f", i, result.Score)
				}
				t.Logf("Result %d: FileID=%s, Score=%.4f, Content length=%d",
					i, result.FileID, result.Score, len(result.Content))
			}
		})
	}
}

// TestS3Provider_Search_WithFilters tests metadata filtering
func TestS3Provider_Search_WithFilters(t *testing.T) {
	bucketName := os.Getenv("S3_VECTOR_BUCKET")
	indexName := os.Getenv("S3_VECTOR_INDEX")
	voyageAPIKey := os.Getenv("VOYAGE_API_KEY")

	if bucketName == "" || indexName == "" {
		t.Skip("S3_VECTOR_BUCKET or S3_VECTOR_INDEX not set, skipping integration test")
	}

	if voyageAPIKey == "" {
		t.Skip("VOYAGE_API_KEY not set, skipping integration test")
	}

	// Create Voyage client to generate real embeddings
	voyageClient := NewVoyageClient(voyageAPIKey)
	ctx := context.Background()

	// Generate query embedding using Voyage
	queryVector, err := voyageClient.EmbedQuery(ctx, "revenue performance metrics")
	if err != nil {
		t.Fatalf("Failed to generate query embedding: %v", err)
	}
	t.Logf("Generated query embedding with %d dimensions", len(queryVector))

	config := map[string]interface{}{
		"bucket_name": bucketName,
		"index_name":  indexName,
		"region":      "us-east-1",
	}

	provider, err := NewS3Provider(config)
	if err != nil {
		t.Fatalf("NewS3Provider() error = %v", err)
	}

	err = provider.Initialize(ctx)
	if err != nil {
		t.Fatalf("Initialize() error = %v", err)
	}

	// Test with date filter
	t.Run("search with date filter", func(t *testing.T) {
		dateFilter := []string{"2025-10-31", "2025-10-30", "2025-10-29"}

		results, err := provider.Search(ctx, "revenue performance metrics", SearchOptions{
			QueryVector: queryVector,
			DateFilter:  dateFilter,
			Limit:       5,
		})

		if err != nil {
			t.Logf("Search with date filter: %v (may fail if no matching data)", err)
		} else {
			t.Logf("Found %d results with date filter", len(results))
			for _, result := range results {
				if date, exists := result.Metadata["report_generated_date"]; exists {
					t.Logf("  - Date: %s", date)
				}
			}
		}
	})

	// Test with metadata filter
	t.Run("search with metadata filter", func(t *testing.T) {
		results, err := provider.Search(ctx, "revenue performance metrics", SearchOptions{
			QueryVector: queryVector,
			Metadata: map[string]string{
				"business_units": "VX",
			},
			Limit: 5,
		})

		if err != nil {
			t.Logf("Search with metadata filter: %v (may fail if no matching data)", err)
		} else {
			t.Logf("Found %d results with business unit filter", len(results))
		}
	})

	// Test with no filter
	t.Run("search with metadata filter", func(t *testing.T) {
		results, err := provider.Search(ctx, "revenue performance metrics", SearchOptions{
			QueryVector: queryVector,
			Limit:       5,
		})

		if err != nil {
			t.Logf("Search with no metadata filter: %v (may fail if no matching data)", err)
		} else {
			t.Logf("Found %d results with no filter", len(results))
		}
	})
}

// TestS3Provider_Config tests configuration validation
func TestS3Provider_Config(t *testing.T) {
	tests := []struct {
		name    string
		config  map[string]interface{}
		wantErr bool
	}{
		{
			name: "valid config",
			config: map[string]interface{}{
				"bucket_name": "test-bucket",
				"index_name":  "test-index",
				"region":      "us-east-1",
			},
			wantErr: false,
		},
		{
			name: "missing bucket_name",
			config: map[string]interface{}{
				"region": "us-east-1",
			},
			wantErr: true,
		},
		{
			name: "default region",
			config: map[string]interface{}{
				"bucket_name": "test-bucket",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewS3Provider(tt.config)

			if tt.wantErr {
				if err == nil {
					t.Errorf("NewS3Provider() expected error, got none")
				}
				return
			}

			if err != nil {
				t.Errorf("NewS3Provider() error = %v", err)
				return
			}

			if provider == nil {
				t.Errorf("NewS3Provider() returned nil provider")
			}
		})
	}
}
