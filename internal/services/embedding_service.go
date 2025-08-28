package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	models "narapulse-be/internal/models/entity"
	"gorm.io/gorm"
)

// EmbeddingService handles vector embeddings for RAG system
type EmbeddingService struct {
	db     *gorm.DB
	apiKey string
	client *http.Client
}

// NewEmbeddingService creates a new embedding service
func NewEmbeddingService(db *gorm.DB, apiKey string) *EmbeddingService {
	return &EmbeddingService{
		db:     db,
		apiKey: apiKey,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// OpenAI Embedding API structures
type EmbeddingRequest struct {
	Input []string `json:"input"`
	Model string   `json:"model"`
}

type EmbeddingResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
	Model string `json:"model"`
	Usage struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
}

// GenerateEmbedding generates vector embedding for given text
func (s *EmbeddingService) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	if strings.TrimSpace(text) == "" {
		return nil, fmt.Errorf("text cannot be empty")
	}

	reqBody := EmbeddingRequest{
		Input: []string{text},
		Model: "text-embedding-ada-002",
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/embeddings", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var embeddingResp EmbeddingResponse
	if err := json.Unmarshal(body, &embeddingResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(embeddingResp.Data) == 0 {
		return nil, fmt.Errorf("no embedding data received")
	}

	return embeddingResp.Data[0].Embedding, nil
}

// EmbedSchema generates and stores embeddings for schema elements
func (s *EmbeddingService) EmbedSchema(ctx context.Context, dataSourceID uint, schemaID uint) error {
	// Get schema with columns
	var schema models.Schema
	if err := s.db.First(&schema, schemaID).Error; err != nil {
		return fmt.Errorf("failed to get schema: %w", err)
	}

	// Parse columns
	var columns []models.Column
	if err := json.Unmarshal(schema.Columns, &columns); err != nil {
		return fmt.Errorf("failed to parse columns: %w", err)
	}

	// Generate embedding for table/schema
	tableContent := s.buildTableContent(schema, columns)
	tableEmbedding, err := s.GenerateEmbedding(ctx, tableContent)
	if err != nil {
		return fmt.Errorf("failed to generate table embedding: %w", err)
	}

	// Store table embedding
	tableEmbeddingRecord := &models.SchemaEmbedding{
		DataSourceID: dataSourceID,
		SchemaID:     schemaID,
		ElementType:  "table",
		ElementName:  schema.Name,
		Content:      tableContent,
		Embedding:    tableEmbedding,
		Metadata:     models.JSON(`{"display_name":"` + schema.DisplayName + `","description":"` + schema.Description + `","row_count":` + fmt.Sprintf("%d", schema.RowCount) + `}`),
	}

	if err := s.db.Create(tableEmbeddingRecord).Error; err != nil {
		return fmt.Errorf("failed to store table embedding: %w", err)
	}

	// Generate embeddings for each column
	for _, column := range columns {
		columnContent := s.buildColumnContent(schema.Name, column)
		columnEmbedding, err := s.GenerateEmbedding(ctx, columnContent)
		if err != nil {
			continue // Skip failed embeddings but don't fail the whole process
		}

		columnEmbeddingRecord := &models.SchemaEmbedding{
			DataSourceID: dataSourceID,
			SchemaID:     schemaID,
			ElementType:  "column",
			ElementName:  column.Name,
			Content:      columnContent,
			Embedding:    columnEmbedding,
			Metadata:     models.JSON(fmt.Sprintf(`{"table":"%s","type":"%s","nullable":%t,"primary_key":%t}`, schema.Name, column.Type, column.Nullable, column.PrimaryKey)),
		}

		s.db.Create(columnEmbeddingRecord)
	}

	return nil
}

// EmbedKPIDefinition generates and stores embedding for KPI definition
func (s *EmbeddingService) EmbedKPIDefinition(ctx context.Context, kpi *models.KPIDefinition) error {
	content := s.buildKPIContent(kpi)
	embedding, err := s.GenerateEmbedding(ctx, content)
	if err != nil {
		return fmt.Errorf("failed to generate KPI embedding: %w", err)
	}

	// Store KPI embedding (using schema_id = 0 for KPIs)
	kpiEmbeddingRecord := &models.SchemaEmbedding{
		DataSourceID: 0, // KPIs are not tied to specific data sources
		SchemaID:     0,
		ElementType:  "kpi",
		ElementName:  kpi.Name,
		Content:      content,
		Embedding:    embedding,
		Metadata:     models.JSON(fmt.Sprintf(`{"category":"%s","unit":"%s","grain":"%s","user_id":%d}`, kpi.Category, kpi.Unit, kpi.Grain, kpi.UserID)),
	}

	if err := s.db.Create(kpiEmbeddingRecord).Error; err != nil {
		return fmt.Errorf("failed to store KPI embedding: %w", err)
	}

	return nil
}

// EmbedGlossaryTerm generates and stores embedding for glossary term
func (s *EmbeddingService) EmbedGlossaryTerm(ctx context.Context, glossary *models.BusinessGlossary) error {
	content := s.buildGlossaryContent(glossary)
	embedding, err := s.GenerateEmbedding(ctx, content)
	if err != nil {
		return fmt.Errorf("failed to generate glossary embedding: %w", err)
	}

	// Store glossary embedding (using schema_id = 0 for glossary)
	glossaryEmbeddingRecord := &models.SchemaEmbedding{
		DataSourceID: 0, // Glossary terms are not tied to specific data sources
		SchemaID:     0,
		ElementType:  "glossary",
		ElementName:  glossary.Term,
		Content:      content,
		Embedding:    embedding,
		Metadata:     models.JSON(fmt.Sprintf(`{"category":"%s","domain":"%s","user_id":%d}`, glossary.Category, glossary.Domain, glossary.UserID)),
	}

	if err := s.db.Create(glossaryEmbeddingRecord).Error; err != nil {
		return fmt.Errorf("failed to store glossary embedding: %w", err)
	}

	return nil
}

// DeleteEmbeddings removes embeddings for a specific schema
func (s *EmbeddingService) DeleteEmbeddings(dataSourceID uint, schemaID uint) error {
	return s.db.Where("data_source_id = ? AND schema_id = ?", dataSourceID, schemaID).Delete(&models.SchemaEmbedding{}).Error
}

// Helper methods to build content for embeddings
func (s *EmbeddingService) buildTableContent(schema models.Schema, columns []models.Column) string {
	var content strings.Builder
	content.WriteString(fmt.Sprintf("Table: %s", schema.Name))
	if schema.DisplayName != "" {
		content.WriteString(fmt.Sprintf(" (%s)", schema.DisplayName))
	}
	if schema.Description != "" {
		content.WriteString(fmt.Sprintf("\nDescription: %s", schema.Description))
	}
	content.WriteString(fmt.Sprintf("\nColumns: %d", len(columns)))
	content.WriteString(fmt.Sprintf("\nRow count: %d", schema.RowCount))

	content.WriteString("\nColumn details:")
	for _, col := range columns {
		content.WriteString(fmt.Sprintf("\n- %s (%s)", col.Name, col.Type))
		if col.Description != "" {
			content.WriteString(fmt.Sprintf(": %s", col.Description))
		}
	}

	return content.String()
}

func (s *EmbeddingService) buildColumnContent(tableName string, column models.Column) string {
	var content strings.Builder
	content.WriteString(fmt.Sprintf("Column: %s.%s", tableName, column.Name))
	content.WriteString(fmt.Sprintf("\nType: %s", column.Type))
	if column.Description != "" {
		content.WriteString(fmt.Sprintf("\nDescription: %s", column.Description))
	}
	if column.PrimaryKey {
		content.WriteString("\nPrimary Key: true")
	}
	if !column.Nullable {
		content.WriteString("\nNullable: false")
	}
	if len(column.SampleValues) > 0 {
		content.WriteString("\nSample values: ")
		for i, val := range column.SampleValues {
			if i > 0 {
				content.WriteString(", ")
			}
			content.WriteString(fmt.Sprintf("%v", val))
			if i >= 4 { // Limit to 5 sample values
				break
			}
		}
	}

	return content.String()
}

func (s *EmbeddingService) buildKPIContent(kpi *models.KPIDefinition) string {
	var content strings.Builder
	content.WriteString(fmt.Sprintf("KPI: %s", kpi.Name))
	if kpi.DisplayName != "" {
		content.WriteString(fmt.Sprintf(" (%s)", kpi.DisplayName))
	}
	if kpi.Description != "" {
		content.WriteString(fmt.Sprintf("\nDescription: %s", kpi.Description))
	}
	content.WriteString(fmt.Sprintf("\nFormula: %s", kpi.Formula))
	if kpi.Category != "" {
		content.WriteString(fmt.Sprintf("\nCategory: %s", kpi.Category))
	}
	if kpi.Unit != "" {
		content.WriteString(fmt.Sprintf("\nUnit: %s", kpi.Unit))
	}
	if kpi.Grain != "" {
		content.WriteString(fmt.Sprintf("\nGrain: %s", kpi.Grain))
	}

	return content.String()
}

func (s *EmbeddingService) buildGlossaryContent(glossary *models.BusinessGlossary) string {
	var content strings.Builder
	content.WriteString(fmt.Sprintf("Term: %s", glossary.Term))
	content.WriteString(fmt.Sprintf("\nDefinition: %s", glossary.Definition))
	if glossary.Category != "" {
		content.WriteString(fmt.Sprintf("\nCategory: %s", glossary.Category))
	}
	if glossary.Domain != "" {
		content.WriteString(fmt.Sprintf("\nDomain: %s", glossary.Domain))
	}

	// Add synonyms
	var synonyms []string
	if glossary.Synonyms != nil {
		json.Unmarshal(glossary.Synonyms, &synonyms)
		if len(synonyms) > 0 {
			content.WriteString(fmt.Sprintf("\nSynonyms: %s", strings.Join(synonyms, ", ")))
		}
	}

	// Add examples
	var examples []string
	if glossary.Examples != nil {
		json.Unmarshal(glossary.Examples, &examples)
		if len(examples) > 0 {
			content.WriteString(fmt.Sprintf("\nExamples: %s", strings.Join(examples, "; ")))
		}
	}

	return content.String()
}