package models

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// SchemaEmbedding stores vector embeddings for schema elements
type SchemaEmbedding struct {
	ID           uint           `json:"id" gorm:"primaryKey"`
	DataSourceID uint           `json:"data_source_id" gorm:"not null;index"`
	SchemaID     uint           `json:"schema_id" gorm:"not null;index"`
	ElementType  string         `json:"element_type" gorm:"not null"` // table, column, kpi, glossary
	ElementName  string         `json:"element_name" gorm:"not null"`
	Content      string         `json:"content" gorm:"type:text"` // The text content that was embedded
	Embedding    []float32 `json:"-" gorm:"type:vector(1536)"` // OpenAI ada-002 embedding size
	Metadata     JSON           `json:"metadata" gorm:"type:jsonb"` // Additional metadata
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `json:"-" gorm:"index"`

	// Relations
	DataSource DataSource `json:"data_source" gorm:"foreignKey:DataSourceID"`
	Schema     Schema     `json:"schema" gorm:"foreignKey:SchemaID"`
}

// KPIDefinition stores business KPI definitions
type KPIDefinition struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	UserID      uint           `json:"user_id" gorm:"not null;index"`
	Name        string         `json:"name" gorm:"not null;uniqueIndex:idx_user_kpi_name"`
	DisplayName string         `json:"display_name"`
	Description string         `json:"description" gorm:"type:text"`
	Formula     string         `json:"formula" gorm:"type:text"` // SQL formula or calculation
	Category    string         `json:"category"` // revenue, marketing, operations, etc.
	Unit        string         `json:"unit"` // currency, percentage, count, etc.
	Grain       string         `json:"grain"` // daily, weekly, monthly, etc.
	Filters     JSON           `json:"filters" gorm:"type:jsonb"` // Default filters
	Tags        JSON           `json:"tags" gorm:"type:jsonb"` // Tags for categorization
	IsActive    bool           `json:"is_active" gorm:"default:true"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`

	// Relations
	User User `json:"user" gorm:"foreignKey:UserID"`
}

// BusinessGlossary stores business term definitions
type BusinessGlossary struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	UserID      uint           `json:"user_id" gorm:"not null;index"`
	Term        string         `json:"term" gorm:"not null;uniqueIndex:idx_user_term"`
	Definition  string         `json:"definition" gorm:"type:text;not null"`
	Synonyms    JSON           `json:"synonyms" gorm:"type:jsonb"` // Alternative terms
	Category    string         `json:"category"` // business, technical, domain-specific
	Domain      string         `json:"domain"` // finance, marketing, operations, etc.
	Examples    JSON           `json:"examples" gorm:"type:jsonb"` // Usage examples
	RelatedTerms JSON          `json:"related_terms" gorm:"type:jsonb"` // Related glossary terms
	IsActive    bool           `json:"is_active" gorm:"default:true"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`

	// Relations
	User User `json:"user" gorm:"foreignKey:UserID"`
}

// RAGQueryContext stores context for NL2SQL queries
type RAGQueryContext struct {
	ID           uint           `json:"id" gorm:"primaryKey"`
	UserID       uint           `json:"user_id" gorm:"not null;index"`
	DataSourceID uint           `json:"data_source_id" gorm:"not null;index"`
	Query        string         `json:"query" gorm:"type:text;not null"`
	Context      JSON           `json:"context" gorm:"type:jsonb"` // Retrieved context from RAG
	Embedding    []float32 `json:"-" gorm:"type:vector(1536)"` // Query embedding
	CreatedAt    time.Time      `json:"created_at"`

	// Relations
	User       User       `json:"user" gorm:"foreignKey:UserID"`
	DataSource DataSource `json:"data_source" gorm:"foreignKey:DataSourceID"`
}

// Request/Response DTOs
type KPIDefinitionRequest struct {
	Name        string                 `json:"name" validate:"required,min=1,max=100"`
	DisplayName string                 `json:"display_name" validate:"max=200"`
	Description string                 `json:"description" validate:"max=1000"`
	Formula     string                 `json:"formula" validate:"required"`
	Category    string                 `json:"category" validate:"max=50"`
	Unit        string                 `json:"unit" validate:"max=20"`
	Grain       string                 `json:"grain" validate:"max=20"`
	Filters     map[string]interface{} `json:"filters"`
	Tags        []string               `json:"tags"`
}

type KPIDefinitionResponse struct {
	ID          uint                   `json:"id"`
	Name        string                 `json:"name"`
	DisplayName string                 `json:"display_name"`
	Description string                 `json:"description"`
	Formula     string                 `json:"formula"`
	Category    string                 `json:"category"`
	Unit        string                 `json:"unit"`
	Grain       string                 `json:"grain"`
	Filters     map[string]interface{} `json:"filters"`
	Tags        []string               `json:"tags"`
	IsActive    bool                   `json:"is_active"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

type BusinessGlossaryRequest struct {
	Term         string   `json:"term" validate:"required,min=1,max=100"`
	Definition   string   `json:"definition" validate:"required,min=1,max=1000"`
	Synonyms     []string `json:"synonyms"`
	Category     string   `json:"category" validate:"max=50"`
	Domain       string   `json:"domain" validate:"max=50"`
	Examples     []string `json:"examples"`
	RelatedTerms []string `json:"related_terms"`
}

type BusinessGlossaryResponse struct {
	ID           uint      `json:"id"`
	Term         string    `json:"term"`
	Definition   string    `json:"definition"`
	Synonyms     []string  `json:"synonyms"`
	Category     string    `json:"category"`
	Domain       string    `json:"domain"`
	Examples     []string  `json:"examples"`
	RelatedTerms []string  `json:"related_terms"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type RAGSearchRequest struct {
	Query        string `json:"query" validate:"required"`
	DataSourceID uint   `json:"data_source_id" validate:"required"`
	TopK         int    `json:"top_k" validate:"min=1,max=20"`
	ElementTypes []string `json:"element_types"` // filter by element types
}

type RAGSearchResult struct {
	ElementType string                 `json:"element_type"`
	ElementName string                 `json:"element_name"`
	Content     string                 `json:"content"`
	Score       float64                `json:"score"`
	Metadata    map[string]interface{} `json:"metadata"`
}

type RAGSearchResponse struct {
	Results []RAGSearchResult `json:"results"`
	Query   string            `json:"query"`
	TopK    int               `json:"top_k"`
}

// Helper methods
func (k *KPIDefinition) ToResponse() *KPIDefinitionResponse {
	var filters map[string]interface{}
	if k.Filters != nil {
		_ = json.Unmarshal(k.Filters, &filters)
	}

	var tags []string
	if k.Tags != nil {
		_ = json.Unmarshal(k.Tags, &tags)
	}

	return &KPIDefinitionResponse{
		ID:          k.ID,
		Name:        k.Name,
		DisplayName: k.DisplayName,
		Description: k.Description,
		Formula:     k.Formula,
		Category:    k.Category,
		Unit:        k.Unit,
		Grain:       k.Grain,
		Filters:     filters,
		Tags:        tags,
		IsActive:    k.IsActive,
		CreatedAt:   k.CreatedAt,
		UpdatedAt:   k.UpdatedAt,
	}
}

func (g *BusinessGlossary) ToResponse() *BusinessGlossaryResponse {
	var synonyms []string
	if g.Synonyms != nil {
		_ = json.Unmarshal(g.Synonyms, &synonyms)
	}

	var examples []string
	if g.Examples != nil {
		_ = json.Unmarshal(g.Examples, &examples)
	}

	var relatedTerms []string
	if g.RelatedTerms != nil {
		_ = json.Unmarshal(g.RelatedTerms, &relatedTerms)
	}

	return &BusinessGlossaryResponse{
		ID:           g.ID,
		Term:         g.Term,
		Definition:   g.Definition,
		Synonyms:     synonyms,
		Category:     g.Category,
		Domain:       g.Domain,
		Examples:     examples,
		RelatedTerms: relatedTerms,
		IsActive:     g.IsActive,
		CreatedAt:    g.CreatedAt,
		UpdatedAt:    g.UpdatedAt,
	}
}