package rag

import (
	"context"
	"fmt"
	"sync"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3vectors"
	"github.com/aws/aws-sdk-go-v2/service/s3vectors/document"
	"github.com/aws/aws-sdk-go-v2/service/s3vectors/types"
	"github.com/tuannvm/slack-mcp-client/internal/common/logging"
)

// S3Provider implements VectorProvider using AWS S3 as storage backend
type S3Provider struct {
	bucketName      string
	indexName       string
	region          string
	config          map[string]interface{}
	s3vectorsClient *s3vectors.Client
	logger          *logging.Logger
	initOnce        sync.Once // Ensures initialization happens exactly once
	initErr         error     // Stores initialization error
}

// NewS3Provider creates a new S3-based vector provider
func NewS3Provider(config map[string]interface{}) (VectorProvider, error) {
	bucketName, ok := config["bucket_name"].(string)
	if !ok || bucketName == "" {
		return nil, fmt.Errorf("bucket_name is required in S3 provider config")
	}

	indexName, ok := config["index_name"].(string)
	if !ok || indexName == "" {
		indexName = "default" // default index name
	}

	region, ok := config["region"].(string)
	if !ok || region == "" {
		region = "us-east-1" // default region
	}

	// Create logger for S3 provider
	logger := logging.New("s3-provider", logging.LevelInfo)

	return &S3Provider{
		bucketName: bucketName,
		indexName:  indexName,
		region:     region,
		config:     config,
		logger:     logger,
	}, nil
}

// Initialize sets up the S3 vector provider
// Thread-safe: Uses sync.Once to ensure initialization happens exactly once
func (s *S3Provider) Initialize(ctx context.Context) error {
	s.initOnce.Do(func() {
		cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(s.region))
		if err != nil {
			s.initErr = fmt.Errorf("failed to load AWS config: %w", err)
			return
		}
		s.s3vectorsClient = s3vectors.NewFromConfig(cfg)
	})
	return s.initErr
}

// IngestFile ingests a single file into the vector store
func (s *S3Provider) IngestFile(ctx context.Context, filePath string, metadata map[string]string) (string, error) {
	// TODO: Upload file to S3, process and vectorize content
	return "", fmt.Errorf("not implemented")
}

// IngestFiles ingests multiple files into the vector store
func (s *S3Provider) IngestFiles(ctx context.Context, filePaths []string, metadata map[string]string) ([]string, error) {
	// TODO: Batch upload files to S3, process and vectorize content
	return nil, fmt.Errorf("not implemented")
}

// DeleteFile removes a file from the vector store
func (s *S3Provider) DeleteFile(ctx context.Context, fileID string) error {
	// TODO: Delete file from S3 and remove vectors
	return fmt.Errorf("not implemented")
}

// ListFiles lists files in the vector store
func (s *S3Provider) ListFiles(ctx context.Context, limit int) ([]FileInfo, error) {
	// TODO: List files from S3 bucket
	return nil, fmt.Errorf("not implemented")
}

// Search performs a vector similarity search
func (s *S3Provider) Search(ctx context.Context, query string, options SearchOptions) ([]SearchResult, error) {
	if s.s3vectorsClient == nil {
		return nil, fmt.Errorf("s3 vectors client not initialized")
	}

	// S3 provider requires pre-computed query vector
	if len(options.QueryVector) == 0 {
		return nil, fmt.Errorf("query vector is required in SearchOptions for S3 provider")
	}

	// Set default limit if not specified
	limit := int32(options.Limit)
	if limit <= 0 {
		limit = 7
	}

	// Build the query input
	input := &s3vectors.QueryVectorsInput{
		VectorBucketName: &s.bucketName,
		IndexName:        &s.indexName,
		QueryVector:      &types.VectorDataMemberFloat32{Value: options.QueryVector},
		TopK:             &limit,
		ReturnDistance:   true,
		ReturnMetadata:   true,
	}

	// Build filter from options (generic - caller provides business logic)
	if len(options.DateFilter) > 0 || len(options.Metadata) > 0 {
		filter := make(map[string]interface{})

		// Add date filter ONLY if date_filter_field is configured
		if len(options.DateFilter) > 0 && s.config != nil {
			if dateFilterField, ok := s.config["date_filter_field"].(string); ok && dateFilterField != "" {
				filter[dateFilterField] = map[string]interface{}{
					"$in": options.DateFilter,
				}
				s.logger.DebugKV("Applying date filter", "field", dateFilterField, "dates", options.DateFilter)
			} else {
				// If date_filter_field is not configured, skip date filtering entirely
				s.logger.InfoKV("Date filter not applied: date_filter_field not configured", "provided_dates", options.DateFilter)
			}
		}

		// Add other metadata filters
		for key, value := range options.Metadata {
			filter[key] = value
		}

		// Wrap in document.NewLazyDocument for AWS SDK
		input.Filter = document.NewLazyDocument(filter)
	}

	// Execute the vector query
	output, err := s.s3vectorsClient.QueryVectors(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to query vectors: %w", err)
	}

	// Convert results to SearchResult format
	results := make([]SearchResult, 0, len(output.Vectors))
	for _, vector := range output.Vectors {
		// Calculate score from distance (assuming lower distance = higher score)
		score := float32(1.0)
		if vector.Distance != nil {
			// Convert distance to similarity score (inverse relationship)
			// You may want to adjust this formula based on your distance metric
			score = 1.0 / (1.0 + *vector.Distance)
		}

		searchResult := SearchResult{
			Score:    score,
			FileID:   *vector.Key,
			FileName: *vector.Key, // Using Key as filename for now
			Metadata: make(map[string]string),
		}

		// Extract content and metadata from S3 response
		if vector.Metadata != nil {
			// Use Smithy document Unmarshaler to convert to map
			var metadataMap map[string]interface{}
			err := vector.Metadata.UnmarshalSmithyDocument(&metadataMap)
			if err == nil {
				// Extract source_text as content
				if sourceText, exists := metadataMap["source_text"]; exists {
					if text, ok := sourceText.(string); ok {
						searchResult.Content = text
					}
				}

				// Convert metadata to string map
				for key, value := range metadataMap {
					if key == "source_text" {
						continue // Skip source_text as it's already in Content
					}
					// Convert value to string
					searchResult.Metadata[key] = fmt.Sprintf("%v", value)
				}

				// Log doc_id and date field (if configured)
				vectorKey := *vector.Key
				if dateField, ok := s.config["date_filter_field"].(string); ok && dateField != "" {
					if reportDate, exists := searchResult.Metadata[dateField]; exists {
						s.logger.DebugKV("S3 vector result", "vector_key", vectorKey, "date_field", dateField, "date_value", reportDate, "score", score)
					} else {
						s.logger.DebugKV("S3 vector result", "vector_key", vectorKey, "score", score)
					}
				} else {
					s.logger.DebugKV("S3 vector result", "vector_key", vectorKey, "score", score)
				}
			} else {
				// Failed to unmarshal metadata - skip this result
				s.logger.ErrorKV("Failed to unmarshal vector metadata, skipping result",
					"vector_key", *vector.Key,
					"score", score,
					"error", err)
				continue
			}
		}

		// Apply minimum score filter
		if options.MinScore > 0 && searchResult.Score < options.MinScore {
			continue
		}

		results = append(results, searchResult)
	}

	return results, nil
}

// Close cleans up resources
func (s *S3Provider) Close() error {
	// TODO: Clean up S3 client and connections
	return nil
}

// GetStats returns statistics about the vector store
func (s *S3Provider) GetStats(ctx context.Context) (*VectorStoreStats, error) {
	// TODO: Gather stats from S3 bucket
	return &VectorStoreStats{}, fmt.Errorf("not implemented")
}

func init() {
	// Register the S3 provider factory
	RegisterVectorProvider("s3", NewS3Provider)
}
