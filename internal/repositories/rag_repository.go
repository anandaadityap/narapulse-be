package repositories

import (
	"fmt"
	"narapulse-be/internal/models/entity"

	"gorm.io/gorm"
)

type RAGRepository interface {
	// Schema Embeddings
	CreateSchemaEmbedding(embedding *models.SchemaEmbedding) error
	GetSchemaEmbeddingsByDataSource(dataSourceID uint) ([]models.SchemaEmbedding, error)
	SearchSimilarEmbeddings(embedding []float32, dataSourceID uint, limit int) ([]models.SchemaEmbedding, error)
	DeleteSchemaEmbeddingsByDataSource(dataSourceID uint) error

	// KPI Definitions
	CreateKPIDefinition(kpi *models.KPIDefinition) error
	GetKPIDefinitionsByUser(userID uint) ([]models.KPIDefinition, error)
	GetKPIDefinitionByID(id uint) (*models.KPIDefinition, error)
	UpdateKPIDefinition(kpi *models.KPIDefinition) error
	DeleteKPIDefinition(id uint) error
	SearchKPIDefinitions(userID uint, query string) ([]models.KPIDefinition, error)

	// Business Glossary
	CreateBusinessGlossary(glossary *models.BusinessGlossary) error
	GetBusinessGlossariesByUser(userID uint) ([]models.BusinessGlossary, error)
	GetBusinessGlossaryByID(id uint) (*models.BusinessGlossary, error)
	UpdateBusinessGlossary(glossary *models.BusinessGlossary) error
	DeleteBusinessGlossary(id uint) error
	SearchBusinessGlossaries(userID uint, query string) ([]models.BusinessGlossary, error)

	// RAG Query Context
	CreateRAGQueryContext(context *models.RAGQueryContext) error
	GetRAGQueryContextsByUser(userID uint, limit int) ([]models.RAGQueryContext, error)
}

type ragRepository struct {
	db *gorm.DB
}

func NewRAGRepository(db *gorm.DB) RAGRepository {
	return &ragRepository{db: db}
}

// Schema Embeddings Implementation
func (r *ragRepository) CreateSchemaEmbedding(embedding *models.SchemaEmbedding) error {
	return r.db.Create(embedding).Error
}

func (r *ragRepository) GetSchemaEmbeddingsByDataSource(dataSourceID uint) ([]models.SchemaEmbedding, error) {
	var embeddings []models.SchemaEmbedding
	err := r.db.Where("data_source_id = ?", dataSourceID).Find(&embeddings).Error
	return embeddings, err
}

func (r *ragRepository) SearchSimilarEmbeddings(embedding []float32, dataSourceID uint, limit int) ([]models.SchemaEmbedding, error) {
	var embeddings []models.SchemaEmbedding
	
	// Convert embedding to PostgreSQL vector format
	embeddingStr := "["
	for i, val := range embedding {
		if i > 0 {
			embeddingStr += ","
		}
		embeddingStr += fmt.Sprintf("%f", val)
	}
	embeddingStr += "]"
	
	err := r.db.Raw(`
		SELECT *, (embedding <=> ?::vector) as distance 
		FROM schema_embeddings 
		WHERE data_source_id = ? 
		ORDER BY embedding <=> ?::vector 
		LIMIT ?
	`, embeddingStr, dataSourceID, embeddingStr, limit).Scan(&embeddings).Error
	
	return embeddings, err
}

func (r *ragRepository) DeleteSchemaEmbeddingsByDataSource(dataSourceID uint) error {
	return r.db.Where("data_source_id = ?", dataSourceID).Delete(&models.SchemaEmbedding{}).Error
}

// KPI Definitions Implementation
func (r *ragRepository) CreateKPIDefinition(kpi *models.KPIDefinition) error {
	return r.db.Create(kpi).Error
}

func (r *ragRepository) GetKPIDefinitionsByUser(userID uint) ([]models.KPIDefinition, error) {
	var kpis []models.KPIDefinition
	err := r.db.Where("user_id = ? AND is_active = ?", userID, true).Find(&kpis).Error
	return kpis, err
}

func (r *ragRepository) GetKPIDefinitionByID(id uint) (*models.KPIDefinition, error) {
	var kpi models.KPIDefinition
	err := r.db.First(&kpi, id).Error
	if err != nil {
		return nil, err
	}
	return &kpi, nil
}

func (r *ragRepository) UpdateKPIDefinition(kpi *models.KPIDefinition) error {
	return r.db.Save(kpi).Error
}

func (r *ragRepository) DeleteKPIDefinition(id uint) error {
	return r.db.Delete(&models.KPIDefinition{}, id).Error
}

func (r *ragRepository) SearchKPIDefinitions(userID uint, query string) ([]models.KPIDefinition, error) {
	var kpis []models.KPIDefinition
	searchPattern := "%" + query + "%"
	err := r.db.Where("user_id = ? AND is_active = ? AND (name ILIKE ? OR description ILIKE ? OR category ILIKE ?)", 
		userID, true, searchPattern, searchPattern, searchPattern).Find(&kpis).Error
	return kpis, err
}

// Business Glossary Implementation
func (r *ragRepository) CreateBusinessGlossary(glossary *models.BusinessGlossary) error {
	return r.db.Create(glossary).Error
}

func (r *ragRepository) GetBusinessGlossariesByUser(userID uint) ([]models.BusinessGlossary, error) {
	var glossaries []models.BusinessGlossary
	err := r.db.Where("user_id = ? AND is_active = ?", userID, true).Find(&glossaries).Error
	return glossaries, err
}

func (r *ragRepository) GetBusinessGlossaryByID(id uint) (*models.BusinessGlossary, error) {
	var glossary models.BusinessGlossary
	err := r.db.First(&glossary, id).Error
	if err != nil {
		return nil, err
	}
	return &glossary, nil
}

func (r *ragRepository) UpdateBusinessGlossary(glossary *models.BusinessGlossary) error {
	return r.db.Save(glossary).Error
}

func (r *ragRepository) DeleteBusinessGlossary(id uint) error {
	return r.db.Delete(&models.BusinessGlossary{}, id).Error
}

func (r *ragRepository) SearchBusinessGlossaries(userID uint, query string) ([]models.BusinessGlossary, error) {
	var glossaries []models.BusinessGlossary
	searchPattern := "%" + query + "%"
	err := r.db.Where("user_id = ? AND is_active = ? AND (term ILIKE ? OR definition ILIKE ? OR category ILIKE ?)", 
		userID, true, searchPattern, searchPattern, searchPattern).Find(&glossaries).Error
	return glossaries, err
}

// RAG Query Context Implementation
func (r *ragRepository) CreateRAGQueryContext(context *models.RAGQueryContext) error {
	return r.db.Create(context).Error
}

func (r *ragRepository) GetRAGQueryContextsByUser(userID uint, limit int) ([]models.RAGQueryContext, error) {
	var contexts []models.RAGQueryContext
	err := r.db.Where("user_id = ?", userID).Order("created_at DESC").Limit(limit).Find(&contexts).Error
	return contexts, err
}