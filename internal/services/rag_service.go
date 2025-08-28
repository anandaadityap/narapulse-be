package services

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"

	models "narapulse-be/internal/models/entity"
	"gorm.io/gorm"
)

// RAGService handles Retrieval Augmented Generation operations
type RAGService struct {
	db               *gorm.DB
	embeddingService *EmbeddingService
}

// NewRAGService creates a new RAG service
func NewRAGService(db *gorm.DB, embeddingService *EmbeddingService) *RAGService {
	return &RAGService{
		db:               db,
		embeddingService: embeddingService,
	}
}

// SearchResult represents a search result with similarity score
type SearchResult struct {
	Embedding *models.SchemaEmbedding
	Score     float64
}

// SearchSimilar performs similarity search using cosine similarity
func (s *RAGService) SearchSimilar(ctx context.Context, query string, dataSourceID uint, topK int, elementTypes []string) (*models.RAGSearchResponse, error) {
	if topK <= 0 {
		topK = 5
	}
	if topK > 20 {
		topK = 20
	}

	// Generate embedding for the query
	queryEmbedding, err := s.embeddingService.GenerateEmbedding(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	// Build query conditions
	queryBuilder := s.db.Model(&models.SchemaEmbedding{})

	// Filter by data source (0 means global like KPIs and glossary)
	if dataSourceID > 0 {
		queryBuilder = queryBuilder.Where("data_source_id = ? OR data_source_id = 0", dataSourceID)
	} else {
		queryBuilder = queryBuilder.Where("data_source_id = 0")
	}

	// Filter by element types if specified
	if len(elementTypes) > 0 {
		queryBuilder = queryBuilder.Where("element_type IN ?", elementTypes)
	}

	// Get all relevant embeddings
	var embeddings []models.SchemaEmbedding
	if err := queryBuilder.Find(&embeddings).Error; err != nil {
		return nil, fmt.Errorf("failed to retrieve embeddings: %w", err)
	}

	// Calculate similarity scores
	var results []SearchResult
	for _, embedding := range embeddings {
		score := s.cosineSimilarity(queryEmbedding, embedding.Embedding)
		results = append(results, SearchResult{
			Embedding: &embedding,
			Score:     score,
		})
	}

	// Sort by similarity score (descending)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	// Take top K results
	if len(results) > topK {
		results = results[:topK]
	}

	// Convert to response format
	var searchResults []models.RAGSearchResult
	for _, result := range results {
		var metadata map[string]interface{}
		if result.Embedding.Metadata != nil {
			json.Unmarshal(result.Embedding.Metadata, &metadata)
		}

		searchResults = append(searchResults, models.RAGSearchResult{
			ElementType: result.Embedding.ElementType,
			ElementName: result.Embedding.ElementName,
			Content:     result.Embedding.Content,
			Score:       result.Score,
			Metadata:    metadata,
		})
	}

	return &models.RAGSearchResponse{
		Results: searchResults,
		Query:   query,
		TopK:    topK,
	}, nil
}

// BuildNL2SQLContext builds context for NL2SQL conversion
func (s *RAGService) BuildNL2SQLContext(ctx context.Context, query string, dataSourceID uint) (map[string]interface{}, error) {
	// Search for relevant schema elements
	schemaResults, err := s.SearchSimilar(ctx, query, dataSourceID, 10, []string{"table", "column"})
	if err != nil {
		return nil, fmt.Errorf("failed to search schema: %w", err)
	}

	// Search for relevant KPIs
	kpiResults, err := s.SearchSimilar(ctx, query, 0, 5, []string{"kpi"})
	if err != nil {
		return nil, fmt.Errorf("failed to search KPIs: %w", err)
	}

	// Search for relevant glossary terms
	glossaryResults, err := s.SearchSimilar(ctx, query, 0, 5, []string{"glossary"})
	if err != nil {
		return nil, fmt.Errorf("failed to search glossary: %w", err)
	}

	// Build context object
	context := map[string]interface{}{
		"query":           query,
		"data_source_id":  dataSourceID,
		"schema_context":  s.buildSchemaContext(schemaResults.Results),
		"kpi_context":     s.buildKPIContext(kpiResults.Results),
		"glossary_context": s.buildGlossaryContext(glossaryResults.Results),
		"timestamp":       ctx.Value("timestamp"),
	}

	return context, nil
}

// GetAvailableSchemas returns available schemas for a data source
func (s *RAGService) GetAvailableSchemas(dataSourceID uint) ([]map[string]interface{}, error) {
	var embeddings []models.SchemaEmbedding
	if err := s.db.Where("data_source_id = ? AND element_type = ?", dataSourceID, "table").Find(&embeddings).Error; err != nil {
		return nil, fmt.Errorf("failed to get schemas: %w", err)
	}

	var schemas []map[string]interface{}
	for _, embedding := range embeddings {
		var metadata map[string]interface{}
		if embedding.Metadata != nil {
			json.Unmarshal(embedding.Metadata, &metadata)
		}

		schema := map[string]interface{}{
			"name":         embedding.ElementName,
			"display_name": metadata["display_name"],
			"description":  metadata["description"],
			"row_count":    metadata["row_count"],
		}
		schemas = append(schemas, schema)
	}

	return schemas, nil
}

// SyncSchemaEmbeddings synchronizes embeddings for a data source
func (s *RAGService) SyncSchemaEmbeddings(ctx context.Context, dataSourceID uint) error {
	// Get all schemas for the data source
	var schemas []models.Schema
	if err := s.db.Where("data_source_id = ? AND is_active = ?", dataSourceID, true).Find(&schemas).Error; err != nil {
		return fmt.Errorf("failed to get schemas: %w", err)
	}

	// Delete existing embeddings for this data source
	if err := s.embeddingService.DeleteEmbeddings(dataSourceID, 0); err != nil {
		return fmt.Errorf("failed to delete existing embeddings: %w", err)
	}

	// Generate new embeddings for each schema
	for _, schema := range schemas {
		if err := s.embeddingService.EmbedSchema(ctx, dataSourceID, schema.ID); err != nil {
			// Log error but continue with other schemas
			fmt.Printf("Failed to embed schema %s: %v\n", schema.Name, err)
			continue
		}
	}

	return nil
}

// Helper methods
func (s *RAGService) cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0.0
	}

	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += float64(a[i] * b[i])
		normA += float64(a[i] * a[i])
		normB += float64(b[i] * b[i])
	}

	if normA == 0.0 || normB == 0.0 {
		return 0.0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

func (s *RAGService) buildSchemaContext(results []models.RAGSearchResult) map[string]interface{} {
	tables := make(map[string]interface{})
	columns := make(map[string][]interface{})

	for _, result := range results {
		if result.ElementType == "table" {
			tables[result.ElementName] = map[string]interface{}{
				"name":        result.ElementName,
				"description": result.Content,
				"score":       result.Score,
				"metadata":    result.Metadata,
			}
		} else if result.ElementType == "column" {
			tableName := ""
			if result.Metadata != nil {
				if table, ok := result.Metadata["table"].(string); ok {
					tableName = table
				}
			}

			columnInfo := map[string]interface{}{
				"name":        result.ElementName,
				"description": result.Content,
				"score":       result.Score,
				"metadata":    result.Metadata,
			}

			if tableName != "" {
				columns[tableName] = append(columns[tableName], columnInfo)
			}
		}
	}

	return map[string]interface{}{
		"tables":  tables,
		"columns": columns,
	}
}

func (s *RAGService) buildKPIContext(results []models.RAGSearchResult) []map[string]interface{} {
	var kpis []map[string]interface{}
	for _, result := range results {
		kpi := map[string]interface{}{
			"name":        result.ElementName,
			"description": result.Content,
			"score":       result.Score,
			"metadata":    result.Metadata,
		}
		kpis = append(kpis, kpi)
	}
	return kpis
}

func (s *RAGService) buildGlossaryContext(results []models.RAGSearchResult) []map[string]interface{} {
	var glossary []map[string]interface{}
	for _, result := range results {
		term := map[string]interface{}{
			"term":        result.ElementName,
			"definition":  result.Content,
			"score":       result.Score,
			"metadata":    result.Metadata,
		}
		glossary = append(glossary, term)
	}
	return glossary
}

// Enhanced NL2SQL prompt building
func (s *RAGService) BuildEnhancedNL2SQLPrompt(ctx context.Context, query string, dataSourceID uint) (string, error) {
	context, err := s.BuildNL2SQLContext(ctx, query, dataSourceID)
	if err != nil {
		return "", fmt.Errorf("failed to build context: %w", err)
	}

	var promptBuilder strings.Builder

	// System prompt
	promptBuilder.WriteString("You are an expert SQL generator. Convert natural language queries to SQL using the provided schema context.\n\n")

	// Schema context
	if schemaCtx, ok := context["schema_context"].(map[string]interface{}); ok {
		promptBuilder.WriteString("AVAILABLE TABLES AND COLUMNS:\n")
		if tables, ok := schemaCtx["tables"].(map[string]interface{}); ok {
			for tableName, tableInfo := range tables {
				if info, ok := tableInfo.(map[string]interface{}); ok {
					promptBuilder.WriteString(fmt.Sprintf("Table: %s\n", tableName))
					if desc, ok := info["description"].(string); ok {
						promptBuilder.WriteString(fmt.Sprintf("Description: %s\n", desc))
					}
				}
			}
		}

		if columns, ok := schemaCtx["columns"].(map[string][]interface{}); ok {
			for tableName, tableCols := range columns {
				promptBuilder.WriteString(fmt.Sprintf("\nColumns for %s:\n", tableName))
				for _, col := range tableCols {
					if colInfo, ok := col.(map[string]interface{}); ok {
						if name, ok := colInfo["name"].(string); ok {
							promptBuilder.WriteString(fmt.Sprintf("- %s", name))
							if metadata, ok := colInfo["metadata"].(map[string]interface{}); ok {
								if colType, ok := metadata["type"].(string); ok {
									promptBuilder.WriteString(fmt.Sprintf(" (%s)", colType))
								}
							}
							promptBuilder.WriteString("\n")
						}
					}
				}
			}
		}
	}

	// KPI context
	if kpiCtx, ok := context["kpi_context"].([]map[string]interface{}); ok && len(kpiCtx) > 0 {
		promptBuilder.WriteString("\nRELEVANT KPIs:\n")
		for _, kpi := range kpiCtx {
			if name, ok := kpi["name"].(string); ok {
				promptBuilder.WriteString(fmt.Sprintf("- %s", name))
				if desc, ok := kpi["description"].(string); ok {
					promptBuilder.WriteString(fmt.Sprintf(": %s", desc))
				}
				promptBuilder.WriteString("\n")
			}
		}
	}

	// Glossary context
	if glossaryCtx, ok := context["glossary_context"].([]map[string]interface{}); ok && len(glossaryCtx) > 0 {
		promptBuilder.WriteString("\nBUSINESS TERMS:\n")
		for _, term := range glossaryCtx {
			if name, ok := term["term"].(string); ok {
				promptBuilder.WriteString(fmt.Sprintf("- %s", name))
				if def, ok := term["definition"].(string); ok {
					promptBuilder.WriteString(fmt.Sprintf(": %s", def))
				}
				promptBuilder.WriteString("\n")
			}
		}
	}

	// Query and instructions
	promptBuilder.WriteString(fmt.Sprintf("\nQUERY: %s\n\n", query))
	promptBuilder.WriteString("INSTRUCTIONS:\n")
	promptBuilder.WriteString("1. Generate a SELECT-only SQL query\n")
	promptBuilder.WriteString("2. Use only the tables and columns provided above\n")
	promptBuilder.WriteString("3. Include appropriate WHERE clauses, JOINs, and aggregations\n")
	promptBuilder.WriteString("4. Add LIMIT clause for large result sets\n")
	promptBuilder.WriteString("5. Return only the SQL query, no explanations\n")

	return promptBuilder.String(), nil
}