package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"gorm.io/gorm"
)

// JSON is a custom type for handling JSON data in GORM
type JSON json.RawMessage

// Scan implements the Scanner interface for database/sql
func (j *JSON) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	switch s := value.(type) {
	case string:
		*j = JSON(s)
	case []byte:
		*j = JSON(s)
	default:
		return errors.New("cannot scan into JSON")
	}
	return nil
}

// Value implements the driver Valuer interface
func (j JSON) Value() (driver.Value, error) {
	if len(j) == 0 {
		return nil, nil
	}
	return string(j), nil
}

// DataSourceType represents the type of data source
type DataSourceType string

const (
	DataSourceTypeCSV        DataSourceType = "csv"
	DataSourceTypeExcel      DataSourceType = "excel"
	DataSourceTypePostgreSQL DataSourceType = "postgresql"
	DataSourceTypeBigQuery   DataSourceType = "bigquery"
	DataSourceTypeGoogleSheets DataSourceType = "google_sheets"
)

// ConnectionStatus represents the status of a data source connection
type ConnectionStatus string

const (
	ConnectionStatusActive     ConnectionStatus = "active"
	ConnectionStatusInactive   ConnectionStatus = "inactive"
	ConnectionStatusError      ConnectionStatus = "error"
	ConnectionStatusConnecting ConnectionStatus = "connecting"
)

// DataSource represents a data source configuration
type DataSource struct {
	ID          uint                   `json:"id" gorm:"primaryKey"`
	UserID      uint                   `json:"user_id" gorm:"not null;index"`
	Name        string                 `json:"name" gorm:"not null"`
	Description string                 `json:"description"`
	Type        DataSourceType         `json:"type" gorm:"not null"`
	Status      ConnectionStatus       `json:"status" gorm:"default:inactive"`
	Config      JSON                   `json:"config" gorm:"type:jsonb"` // Store connection configuration
	Metadata    JSON                   `json:"metadata" gorm:"type:jsonb"` // Store additional metadata
	LastTested  *time.Time             `json:"last_tested"`
	ErrorMsg    string                 `json:"error_message" gorm:"column:error_message"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	DeletedAt   gorm.DeletedAt         `json:"-" gorm:"index"`

	// Relationships
	User    User     `json:"user" gorm:"foreignKey:UserID"`
	Schemas []Schema `json:"schemas" gorm:"foreignKey:DataSourceID"`
}

// Schema represents the schema of a data source
type Schema struct {
	ID           uint           `json:"id" gorm:"primaryKey"`
	DataSourceID uint           `json:"data_source_id" gorm:"not null;index"`
	Name         string         `json:"name" gorm:"not null"` // table name, sheet name, etc.
	DisplayName  string         `json:"display_name"`
	Description  string         `json:"description"`
	Columns      JSON           `json:"columns" gorm:"type:jsonb"` // Store column definitions
	RowCount     int64          `json:"row_count"`
	SampleData   JSON           `json:"sample_data" gorm:"type:jsonb"` // Store sample rows
	IsActive     bool           `json:"is_active" gorm:"default:true"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `json:"-" gorm:"index"`

	// Relationships
	DataSource DataSource `json:"data_source" gorm:"foreignKey:DataSourceID"`
}

// Column represents a column definition in a schema
type Column struct {
	Name        string `json:"name"`
	Type        string `json:"type"`        // data type (string, integer, float, boolean, date, etc.)
	Nullable    bool   `json:"nullable"`
	PrimaryKey  bool   `json:"primary_key"`
	Description string `json:"description"`
	SampleValues []interface{} `json:"sample_values,omitempty"`
}

// ConnectionConfig represents configuration for different data source types
type ConnectionConfig struct {
	// For file uploads (CSV/Excel)
	FileName     string `json:"file_name,omitempty"`
	FilePath     string `json:"file_path,omitempty"`
	FileSize     int64  `json:"file_size,omitempty"`
	HasHeader    bool   `json:"has_header,omitempty"`
	Delimiter    string `json:"delimiter,omitempty"`
	Encoding     string `json:"encoding,omitempty"`

	// For database connections
	Host     string `json:"host,omitempty"`
	Port     int    `json:"port,omitempty"`
	Database string `json:"database,omitempty"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"` // Should be encrypted
	SSLMode  string `json:"ssl_mode,omitempty"`

	// For BigQuery
	ProjectID      string `json:"project_id,omitempty"`
	DatasetID      string `json:"dataset_id,omitempty"`
	CredentialsJSON string `json:"credentials_json,omitempty"` // Should be encrypted

	// For Google Sheets
	SpreadsheetID string `json:"spreadsheet_id,omitempty"`
	SheetName     string `json:"sheet_name,omitempty"`
	Range         string `json:"range,omitempty"`
	AccessToken   string `json:"access_token,omitempty"`   // Should be encrypted
	RefreshToken  string `json:"refresh_token,omitempty"` // Should be encrypted
}

// Request/Response DTOs
type DataSourceCreateRequest struct {
	Name        string                 `json:"name" validate:"required,min=1,max=100"`
	Description string                 `json:"description" validate:"max=500"`
	Type        DataSourceType         `json:"type" validate:"required"`
	Config      map[string]interface{} `json:"config" validate:"required"`
}

type DataSourceUpdateRequest struct {
	Name        string                 `json:"name" validate:"min=1,max=100"`
	Description string                 `json:"description" validate:"max=500"`
	Config      map[string]interface{} `json:"config"`
}

type DataSourceResponse struct {
	ID          uint                   `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Type        DataSourceType         `json:"type"`
	Status      ConnectionStatus       `json:"status"`
	Config      map[string]interface{} `json:"config,omitempty"` // Sensitive data should be masked
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	LastTested  *time.Time             `json:"last_tested"`
	ErrorMsg    string                 `json:"error_message,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	Schemas     []SchemaResponse       `json:"schemas,omitempty"`
}

type SchemaResponse struct {
	ID          uint                   `json:"id"`
	Name        string                 `json:"name"`
	DisplayName string                 `json:"display_name"`
	Description string                 `json:"description"`
	Columns     []Column               `json:"columns"`
	RowCount    int64                  `json:"row_count"`
	SampleData  []map[string]interface{} `json:"sample_data,omitempty"`
	IsActive    bool                   `json:"is_active"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

type TestConnectionRequest struct {
	Type   DataSourceType         `json:"type" validate:"required"`
	Config map[string]interface{} `json:"config" validate:"required"`
}

type TestConnectionResponse struct {
	Success bool     `json:"success"`
	Message string   `json:"message"`
	Schemas []string `json:"schemas,omitempty"` // Available tables/sheets
}

type FileUploadResponse struct {
	FileName string `json:"file_name"`
	FilePath string `json:"file_path"`
	FileSize int64  `json:"file_size"`
	MimeType string `json:"mime_type"`
}

// Helper methods
func (ds *DataSource) MaskSensitiveConfig() map[string]interface{} {
	var config map[string]interface{}
	if err := json.Unmarshal(ds.Config, &config); err != nil {
		return nil
	}

	// Mask sensitive fields
	sensitiveFields := []string{"password", "credentials_json", "access_token", "refresh_token"}
	for _, field := range sensitiveFields {
		if _, exists := config[field]; exists {
			config[field] = "***masked***"
		}
	}

	return config
}

func (ds *DataSource) ToResponse() *DataSourceResponse {
	var schemas []SchemaResponse
	for _, schema := range ds.Schemas {
		schemas = append(schemas, *schema.ToResponse())
	}

	return &DataSourceResponse{
		ID:          ds.ID,
		Name:        ds.Name,
		Description: ds.Description,
		Type:        ds.Type,
		Status:      ds.Status,
		Config:      ds.MaskSensitiveConfig(),
		LastTested:  ds.LastTested,
		ErrorMsg:    ds.ErrorMsg,
		CreatedAt:   ds.CreatedAt,
		UpdatedAt:   ds.UpdatedAt,
		Schemas:     schemas,
	}
}

func (s *Schema) ToResponse() *SchemaResponse {
	var columns []Column
	if err := json.Unmarshal(s.Columns, &columns); err == nil {
		// Successfully unmarshaled
	}

	var sampleData []map[string]interface{}
	if err := json.Unmarshal(s.SampleData, &sampleData); err == nil {
		// Successfully unmarshaled
	}

	return &SchemaResponse{
		ID:          s.ID,
		Name:        s.Name,
		DisplayName: s.DisplayName,
		Description: s.Description,
		Columns:     columns,
		RowCount:    s.RowCount,
		SampleData:  sampleData,
		IsActive:    s.IsActive,
		CreatedAt:   s.CreatedAt,
		UpdatedAt:   s.UpdatedAt,
	}
}