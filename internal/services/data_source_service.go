package services

import (
	"encoding/json"
	"fmt"
	models "narapulse-be/internal/models/entity"
	"narapulse-be/internal/repositories"
	"time"
)

type DataSourceService interface {
	CreateDataSource(userID uint, req *models.DataSourceCreateRequest) (*models.DataSourceResponse, error)
	GetDataSource(id uint, userID uint) (*models.DataSourceResponse, error)
	GetUserDataSources(userID uint) ([]models.DataSourceResponse, error)
	UpdateDataSource(id uint, userID uint, req *models.DataSourceUpdateRequest) (*models.DataSourceResponse, error)
	DeleteDataSource(id uint, userID uint) error
	TestConnection(req *models.TestConnectionRequest) (*models.TestConnectionResponse, error)
	RefreshSchema(id uint, userID uint) (*models.DataSourceResponse, error)
}

type dataSourceService struct {
	dataSourceRepo repositories.DataSourceRepository
	schemaRepo     repositories.SchemaRepository
	connectorSvc   *connectorService
}

func NewDataSourceService(dataSourceRepo repositories.DataSourceRepository, schemaRepo repositories.SchemaRepository, connectorSvc *connectorService) DataSourceService {
	return &dataSourceService{
		dataSourceRepo: dataSourceRepo,
		schemaRepo:     schemaRepo,
		connectorSvc:   connectorSvc,
	}
}

func (s *dataSourceService) CreateDataSource(userID uint, req *models.DataSourceCreateRequest) (*models.DataSourceResponse, error) {
	// Validate configuration based on data source type
	if err := s.validateConfig(req.Type, req.Config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Convert config to JSON
	configJSON, err := json.Marshal(req.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	// Create data source
	dataSource := &models.DataSource{
		UserID:      userID,
		Name:        req.Name,
		Description: req.Description,
		Type:        req.Type,
		Status:      models.ConnectionStatusInactive,
		Config:      models.JSON(configJSON),
	}

	if err := s.dataSourceRepo.Create(dataSource); err != nil {
		return nil, fmt.Errorf("failed to create data source: %w", err)
	}

	// Test connection and discover schema
	go s.testAndDiscoverSchema(dataSource)

	return dataSource.ToResponse(), nil
}

func (s *dataSourceService) GetDataSource(id uint, userID uint) (*models.DataSourceResponse, error) {
	dataSource, err := s.dataSourceRepo.GetWithSchemas(id)
	if err != nil {
		return nil, fmt.Errorf("data source not found: %w", err)
	}

	// Check ownership
	if dataSource.UserID != userID {
		return nil, fmt.Errorf("access denied")
	}

	return dataSource.ToResponse(), nil
}

func (s *dataSourceService) GetUserDataSources(userID uint) ([]models.DataSourceResponse, error) {
	dataSources, err := s.dataSourceRepo.GetByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get data sources: %w", err)
	}

	var responses []models.DataSourceResponse
	for _, ds := range dataSources {
		responses = append(responses, *ds.ToResponse())
	}

	return responses, nil
}

func (s *dataSourceService) UpdateDataSource(id uint, userID uint, req *models.DataSourceUpdateRequest) (*models.DataSourceResponse, error) {
	dataSource, err := s.dataSourceRepo.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("data source not found: %w", err)
	}

	// Check ownership
	if dataSource.UserID != userID {
		return nil, fmt.Errorf("access denied")
	}

	// Update fields
	if req.Name != "" {
		dataSource.Name = req.Name
	}
	if req.Description != "" {
		dataSource.Description = req.Description
	}
	if req.Config != nil {
		// Validate new configuration
		if err := s.validateConfig(dataSource.Type, req.Config); err != nil {
			return nil, fmt.Errorf("invalid configuration: %w", err)
		}

		configJSON, err := json.Marshal(req.Config)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal config: %w", err)
		}
		dataSource.Config = models.JSON(configJSON)
		dataSource.Status = models.ConnectionStatusInactive
	}

	if err := s.dataSourceRepo.Update(dataSource); err != nil {
		return nil, fmt.Errorf("failed to update data source: %w", err)
	}

	// If config was updated, test connection and refresh schema
	if req.Config != nil {
		go s.testAndDiscoverSchema(dataSource)
	}

	return dataSource.ToResponse(), nil
}

func (s *dataSourceService) DeleteDataSource(id uint, userID uint) error {
	dataSource, err := s.dataSourceRepo.GetByID(id)
	if err != nil {
		return fmt.Errorf("data source not found: %w", err)
	}

	// Check ownership
	if dataSource.UserID != userID {
		return fmt.Errorf("access denied")
	}

	// Delete associated schemas first
	if err := s.schemaRepo.DeleteByDataSourceID(id); err != nil {
		return fmt.Errorf("failed to delete schemas: %w", err)
	}

	// Delete data source
	if err := s.dataSourceRepo.Delete(id); err != nil {
		return fmt.Errorf("failed to delete data source: %w", err)
	}

	return nil
}

func (s *dataSourceService) TestConnection(req *models.TestConnectionRequest) (*models.TestConnectionResponse, error) {
	// Validate configuration
	if err := s.validateConfig(req.Type, req.Config); err != nil {
		return &models.TestConnectionResponse{
			Success: false,
			Message: fmt.Sprintf("Invalid configuration: %v", err),
		}, nil
	}

	// Test connection using connector service
	err := s.connectorSvc.TestConnection(*req)
	if err != nil {
		return &models.TestConnectionResponse{
			Success: false,
			Message: fmt.Sprintf("Connection failed: %v", err),
		}, nil
	}

	return &models.TestConnectionResponse{
		Success: true,
		Message: "Connection successful",
	}, nil
}

func (s *dataSourceService) RefreshSchema(id uint, userID uint) (*models.DataSourceResponse, error) {
	dataSource, err := s.dataSourceRepo.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("data source not found: %w", err)
	}

	// Check ownership
	if dataSource.UserID != userID {
		return nil, fmt.Errorf("access denied")
	}

	// Delete existing schemas
	if err := s.schemaRepo.DeleteByDataSourceID(id); err != nil {
		return nil, fmt.Errorf("failed to delete existing schemas: %w", err)
	}

	// Discover new schema
	if err := s.discoverSchema(dataSource); err != nil {
		return nil, fmt.Errorf("failed to discover schema: %w", err)
	}

	// Get updated data source with schemas
	updatedDataSource, err := s.dataSourceRepo.GetWithSchemas(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get updated data source: %w", err)
	}

	return updatedDataSource.ToResponse(), nil
}

// Private helper methods
func (s *dataSourceService) validateConfig(dsType models.DataSourceType, config map[string]interface{}) error {
	switch dsType {
	case models.DataSourceTypeCSV, models.DataSourceTypeExcel:
		return s.validateFileConfig(config)
	case models.DataSourceTypePostgreSQL:
		return s.validatePostgreSQLConfig(config)
	case models.DataSourceTypeBigQuery:
		return s.validateBigQueryConfig(config)
	case models.DataSourceTypeGoogleSheets:
		return s.validateGoogleSheetsConfig(config)
	default:
		return fmt.Errorf("unsupported data source type: %s", dsType)
	}
}

func (s *dataSourceService) validateFileConfig(config map[string]interface{}) error {
	if _, ok := config["file_path"]; !ok {
		return fmt.Errorf("file_path is required")
	}
	return nil
}

func (s *dataSourceService) validatePostgreSQLConfig(config map[string]interface{}) error {
	requiredFields := []string{"host", "port", "database", "username", "password"}
	for _, field := range requiredFields {
		if _, ok := config[field]; !ok {
			return fmt.Errorf("%s is required", field)
		}
	}
	return nil
}

func (s *dataSourceService) validateBigQueryConfig(config map[string]interface{}) error {
	requiredFields := []string{"project_id", "dataset_id", "credentials_json"}
	for _, field := range requiredFields {
		if _, ok := config[field]; !ok {
			return fmt.Errorf("%s is required", field)
		}
	}
	return nil
}

func (s *dataSourceService) validateGoogleSheetsConfig(config map[string]interface{}) error {
	requiredFields := []string{"spreadsheet_id", "access_token"}
	for _, field := range requiredFields {
		if _, ok := config[field]; !ok {
			return fmt.Errorf("%s is required", field)
		}
	}
	return nil
}

func (s *dataSourceService) testAndDiscoverSchema(dataSource *models.DataSource) {
	// Parse config
	var config map[string]interface{}
	if err := json.Unmarshal(dataSource.Config, &config); err != nil {
		dataSource.Status = models.ConnectionStatusError
		dataSource.ErrorMsg = fmt.Sprintf("Invalid config: %v", err)
		s.dataSourceRepo.Update(dataSource)
		return
	}

	// Test connection
	testReq := models.TestConnectionRequest{
		Type:   dataSource.Type,
		Config: config,
	}

	err := s.connectorSvc.TestConnection(testReq)
	if err != nil {
		dataSource.Status = models.ConnectionStatusError
		dataSource.ErrorMsg = fmt.Sprintf("Connection failed: %v", err)
		now := time.Now()
		dataSource.LastTested = &now
		s.dataSourceRepo.Update(dataSource)
		return
	}

	// Connection successful, discover schema
	dataSource.Status = models.ConnectionStatusActive
	dataSource.ErrorMsg = ""
	now := time.Now()
	dataSource.LastTested = &now
	s.dataSourceRepo.Update(dataSource)

	// Discover schema
	s.discoverSchema(dataSource)
}

func (s *dataSourceService) discoverSchema(dataSource *models.DataSource) error {
	var config map[string]interface{}
	if err := json.Unmarshal(dataSource.Config, &config); err != nil {
		return err
	}

	columns, err := s.connectorSvc.DiscoverSchema(dataSource.Type, config)
	if err != nil {
		return err
	}

	// Create a default schema with discovered columns
	columnsJSON, err := json.Marshal(columns)
	if err != nil {
		return fmt.Errorf("failed to marshal columns: %w", err)
	}

	schema := &models.Schema{
		DataSourceID: dataSource.ID,
		Name:         "default",
		DisplayName:  "Default Schema",
		Columns:      models.JSON(columnsJSON),
		RowCount:     0, // Will be updated later
		IsActive:     true,
	}

	err = s.schemaRepo.Create(schema)
	if err != nil {
		return fmt.Errorf("failed to save schema: %w", err)
	}

	return nil
}

// SchemaInfo represents discovered schema information
type SchemaInfo struct {
	Name        string                   `json:"name"`
	DisplayName string                   `json:"display_name"`
	Description string                   `json:"description"`
	Columns     []models.Column          `json:"columns"`
	RowCount    int64                    `json:"row_count"`
	SampleData  []map[string]interface{} `json:"sample_data"`
}

// ConnectorServiceInterface interface for different data source connectors
type ConnectorServiceInterface interface {
	GetConnector(dsType models.DataSourceType) (Connector, error)
}

// Connector interface for different data source types
type Connector interface {
	Connect(config map[string]interface{}) error
	Disconnect() error
	TestConnection() error
	GetSchema() ([]models.Column, error)
	GetData(tableName string, limit int) ([]map[string]interface{}, error)
}