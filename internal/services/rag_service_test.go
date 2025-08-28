package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
	models "narapulse-be/internal/models/entity"
)

// TestRAGService_CosineSimilarity tests the cosineSimilarity functionality
func TestRAGService_CosineSimilarity(t *testing.T) {
	// Create a simple RAG service instance for testing utility functions
	ragService := &RAGService{}

	// Test vectors
	vec1 := []float32{1.0, 0.0, 0.0}
	vec2 := []float32{0.0, 1.0, 0.0}
	vec3 := []float32{1.0, 0.0, 0.0}
	vec4 := []float32{0.5, 0.5, 0.0}

	// Test orthogonal vectors (should be 0)
	similarity1 := ragService.cosineSimilarity(vec1, vec2)
	assert.InDelta(t, 0.0, similarity1, 0.001)

	// Test identical vectors (should be 1)
	similarity2 := ragService.cosineSimilarity(vec1, vec3)
	assert.InDelta(t, 1.0, similarity2, 0.001)

	// Test partial similarity
	similarity3 := ragService.cosineSimilarity(vec1, vec4)
	assert.True(t, similarity3 > 0 && similarity3 < 1)
}

// TestRAGService_BuildSchemaContext tests the buildSchemaContext functionality
func TestRAGService_BuildSchemaContext(t *testing.T) {
	ragService := &RAGService{}

	// Mock search results
	results := []models.RAGSearchResult{
		{
			ElementType: "table",
			ElementName: "sales",
			Content:     "Sales transaction table",
			Score:       0.95,
			Metadata:    map[string]interface{}{"schema": "public"},
		},
		{
			ElementType: "column",
			ElementName: "amount",
			Content:     "Sales amount column",
			Score:       0.88,
			Metadata:    map[string]interface{}{"table": "sales"},
		},
	}

	context := ragService.buildSchemaContext(results)

	assert.NotNil(t, context)
	assert.Contains(t, context, "tables")
	assert.Contains(t, context, "columns")

	// Check if tables and columns are properly categorized
	tables, ok := context["tables"].(map[string]interface{})
	assert.True(t, ok)
	assert.Len(t, tables, 1)
	assert.Contains(t, tables, "sales")

	// Check table structure
	salesTable, ok := tables["sales"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "sales", salesTable["name"])

	// Check columns structure
	columns, ok := context["columns"].(map[string][]interface{})
	assert.True(t, ok)
	assert.Contains(t, columns, "sales")
	assert.Len(t, columns["sales"], 1)

	// Check column structure
	amountColumn := columns["sales"][0].(map[string]interface{})
	assert.Equal(t, "amount", amountColumn["name"])
}

// TestRAGService_BuildKPIContext tests the buildKPIContext functionality
func TestRAGService_BuildKPIContext(t *testing.T) {
	ragService := &RAGService{}

	// Mock KPI search results
	results := []models.RAGSearchResult{
		{
			ElementType: "kpi",
			ElementName: "total_revenue",
			Content:     "SUM(sales.amount)",
			Score:       0.9,
		},
	}

	kpis := ragService.buildKPIContext(results)

	assert.NotNil(t, kpis)
	assert.Len(t, kpis, 1)
	assert.Equal(t, "total_revenue", kpis[0]["name"])
	assert.Equal(t, "SUM(sales.amount)", kpis[0]["description"])
	assert.Equal(t, 0.9, kpis[0]["score"])
}

// TestRAGService_BuildGlossaryContext tests the buildGlossaryContext functionality
func TestRAGService_BuildGlossaryContext(t *testing.T) {
	ragService := &RAGService{}

	// Mock glossary search results
	results := []models.RAGSearchResult{
		{
			ElementType: "glossary",
			ElementName: "revenue",
			Content:     "Total income from sales",
			Score:       0.8,
		},
	}

	glossary := ragService.buildGlossaryContext(results)

	assert.NotNil(t, glossary)
	assert.Len(t, glossary, 1)
	assert.Equal(t, "revenue", glossary[0]["term"])
	assert.Equal(t, "Total income from sales", glossary[0]["definition"])
	assert.Equal(t, 0.8, glossary[0]["score"])
}

// TestRAGService_Validation tests basic validation of RAG service
func TestRAGService_Validation(t *testing.T) {
	// Test that RAGService can be created
	ragService := &RAGService{}
	assert.NotNil(t, ragService)

	// Test cosine similarity edge cases
	emptyVec1 := []float32{}
	emptyVec2 := []float32{}
	similarity := ragService.cosineSimilarity(emptyVec1, emptyVec2)
	assert.Equal(t, 0.0, similarity) // Should handle empty vectors gracefully

	// Test zero vectors
	zeroVec1 := []float32{0.0, 0.0, 0.0}
	zeroVec2 := []float32{0.0, 0.0, 0.0}
	similarity2 := ragService.cosineSimilarity(zeroVec1, zeroVec2)
	assert.Equal(t, 0.0, similarity2) // Should handle zero vectors
}