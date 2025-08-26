package repositories

import (
	"narapulse-be/internal/models/entity"

	"gorm.io/gorm"
)

type DataSourceRepository interface {
	Create(dataSource *models.DataSource) error
	GetByID(id uint) (*models.DataSource, error)
	GetByUserID(userID uint) ([]models.DataSource, error)
	Update(dataSource *models.DataSource) error
	Delete(id uint) error
	GetWithSchemas(id uint) (*models.DataSource, error)
	TestConnection(dataSource *models.DataSource) error
}

type dataSourceRepository struct {
	db *gorm.DB
}

func NewDataSourceRepository(db *gorm.DB) DataSourceRepository {
	return &dataSourceRepository{
		db: db,
	}
}

func (r *dataSourceRepository) Create(dataSource *models.DataSource) error {
	return r.db.Create(dataSource).Error
}

func (r *dataSourceRepository) GetByID(id uint) (*models.DataSource, error) {
	var dataSource models.DataSource
	err := r.db.First(&dataSource, id).Error
	if err != nil {
		return nil, err
	}
	return &dataSource, nil
}

func (r *dataSourceRepository) GetByUserID(userID uint) ([]models.DataSource, error) {
	var dataSources []models.DataSource
	err := r.db.Where("user_id = ?", userID).Find(&dataSources).Error
	return dataSources, err
}

func (r *dataSourceRepository) Update(dataSource *models.DataSource) error {
	return r.db.Save(dataSource).Error
}

func (r *dataSourceRepository) Delete(id uint) error {
	return r.db.Delete(&models.DataSource{}, id).Error
}

func (r *dataSourceRepository) GetWithSchemas(id uint) (*models.DataSource, error) {
	var dataSource models.DataSource
	err := r.db.Preload("Schemas").First(&dataSource, id).Error
	if err != nil {
		return nil, err
	}
	return &dataSource, nil
}

func (r *dataSourceRepository) TestConnection(dataSource *models.DataSource) error {
	// This method will be implemented by specific connector services
	// For now, just update the last_tested timestamp
	return r.db.Model(dataSource).Update("last_tested", "NOW()").Error
}

// Schema Repository
type SchemaRepository interface {
	Create(schema *models.Schema) error
	GetByID(id uint) (*models.Schema, error)
	GetByDataSourceID(dataSourceID uint) ([]models.Schema, error)
	Update(schema *models.Schema) error
	Delete(id uint) error
	DeleteByDataSourceID(dataSourceID uint) error
}

type schemaRepository struct {
	db *gorm.DB
}

func NewSchemaRepository(db *gorm.DB) SchemaRepository {
	return &schemaRepository{
		db: db,
	}
}

func (r *schemaRepository) Create(schema *models.Schema) error {
	return r.db.Create(schema).Error
}

func (r *schemaRepository) GetByID(id uint) (*models.Schema, error) {
	var schema models.Schema
	err := r.db.First(&schema, id).Error
	if err != nil {
		return nil, err
	}
	return &schema, nil
}

func (r *schemaRepository) GetByDataSourceID(dataSourceID uint) ([]models.Schema, error) {
	var schemas []models.Schema
	err := r.db.Where("data_source_id = ?", dataSourceID).Find(&schemas).Error
	return schemas, err
}

func (r *schemaRepository) Update(schema *models.Schema) error {
	return r.db.Save(schema).Error
}

func (r *schemaRepository) Delete(id uint) error {
	return r.db.Delete(&models.Schema{}, id).Error
}

func (r *schemaRepository) DeleteByDataSourceID(dataSourceID uint) error {
	return r.db.Where("data_source_id = ?", dataSourceID).Delete(&models.Schema{}).Error
}