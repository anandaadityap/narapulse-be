package services

import (
	"encoding/csv"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"strconv"
	"strings"

	models "narapulse-be/internal/models/entity"
	"narapulse-be/internal/connectors"
	"github.com/xuri/excelize/v2"
)

// connectorService implements connector functionality
type connectorService struct{}

// NewConnectorService creates a new connector service
func NewConnectorService() *connectorService {
	return &connectorService{}
}

// TestConnection tests the connection to a data source
func (s *connectorService) TestConnection(request models.TestConnectionRequest) error {
	switch request.Type {
	case models.DataSourceTypePostgreSQL:
		return s.testPostgreSQLConnection(request.Config)
	case models.DataSourceTypeBigQuery:
		return s.testBigQueryConnection(request.Config)
	case models.DataSourceTypeGoogleSheets:
		return s.testGoogleSheetsConnection(request.Config)
	case models.DataSourceTypeCSV, models.DataSourceTypeExcel:
		// File-based sources don't need connection testing
		return nil
	default:
		return fmt.Errorf("unsupported data source type: %s", request.Type)
	}
}

// DiscoverSchema discovers the schema of a data source
func (s *connectorService) DiscoverSchema(dsType models.DataSourceType, config map[string]interface{}) ([]models.Column, error) {
	switch dsType {
	case models.DataSourceTypePostgreSQL:
		return s.discoverPostgreSQLSchema(config)
	case models.DataSourceTypeBigQuery:
		return s.discoverBigQuerySchema(config)
	case models.DataSourceTypeGoogleSheets:
		return s.discoverGoogleSheetsSchema(config)
	default:
		return nil, fmt.Errorf("unsupported data source type: %s", dsType)
	}
}

// ProcessFileUpload processes uploaded CSV/Excel files
func (s *connectorService) ProcessFileUpload(file *multipart.FileHeader) (*models.DataSource, []models.Column, error) {
	ext := strings.ToLower(filepath.Ext(file.Filename))
	
	switch ext {
	case ".csv":
		return s.processCSVFile(file)
	case ".xlsx", ".xls":
		return s.processExcelFile(file)
	default:
		return nil, nil, fmt.Errorf("unsupported file type: %s", ext)
	}
}

// PostgreSQL connection methods
func (s *connectorService) testPostgreSQLConnection(config map[string]interface{}) error {
	connector := connectors.NewPostgreSQLConnector()
	defer connector.Disconnect()

	if err := connector.Connect(config); err != nil {
		return fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}

	return connector.TestConnection()
}

func (s *connectorService) discoverPostgreSQLSchema(config map[string]interface{}) ([]models.Column, error) {
	connector := connectors.NewPostgreSQLConnector()
	defer connector.Disconnect()

	if err := connector.Connect(config); err != nil {
		return nil, fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}

	return connector.GetSchema()
}

// BigQuery connection methods (placeholder implementations)
func (s *connectorService) testBigQueryConnection(config map[string]interface{}) error {
	connector := connectors.NewBigQueryConnector()
	defer connector.Disconnect()

	if err := connector.Connect(config); err != nil {
		return fmt.Errorf("failed to connect to BigQuery: %w", err)
	}

	return connector.TestConnection()
}

func (s *connectorService) discoverBigQuerySchema(config map[string]interface{}) ([]models.Column, error) {
	connector := connectors.NewBigQueryConnector()
	defer connector.Disconnect()

	if err := connector.Connect(config); err != nil {
		return nil, fmt.Errorf("failed to connect to BigQuery: %w", err)
	}

	return connector.GetSchema()
}

// Google Sheets connection methods (placeholder implementations)
func (s *connectorService) testGoogleSheetsConnection(config map[string]interface{}) error {
	connector := connectors.NewGoogleSheetsConnector()
	defer connector.Disconnect()

	if err := connector.Connect(config); err != nil {
		return fmt.Errorf("failed to connect to Google Sheets: %w", err)
	}

	return connector.TestConnection()
}

func (s *connectorService) discoverGoogleSheetsSchema(config map[string]interface{}) ([]models.Column, error) {
	connector := connectors.NewGoogleSheetsConnector()
	defer connector.Disconnect()

	if err := connector.Connect(config); err != nil {
		return nil, fmt.Errorf("failed to connect to Google Sheets: %w", err)
	}

	return connector.GetSchema()
}

// File processing methods
func (s *connectorService) processCSVFile(file *multipart.FileHeader) (*models.DataSource, []models.Column, error) {
	src, err := file.Open()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open CSV file: %w", err)
	}
	defer src.Close()
	
	reader := csv.NewReader(src)
	
	// Read header row
	headers, err := reader.Read()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read CSV headers: %w", err)
	}
	
	// Read a few sample rows to infer data types
	sampleRows := make([][]string, 0, 10)
	for i := 0; i < 10; i++ {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, nil, fmt.Errorf("failed to read CSV row: %w", err)
		}
		sampleRows = append(sampleRows, row)
	}
	
	// Create data source
	dataSource := &models.DataSource{
		Name:        strings.TrimSuffix(file.Filename, filepath.Ext(file.Filename)),
		Type:        models.DataSourceTypeCSV,
		Description: fmt.Sprintf("CSV file: %s", file.Filename),
		Status:      models.ConnectionStatusActive,
	}
	
	// Infer column types
	columns := make([]models.Column, len(headers))
	for i, header := range headers {
		columns[i] = models.Column{
			Name:     header,
			Type:     s.inferDataType(sampleRows, i),
			Nullable: true, // CSV columns are generally nullable
		}
	}
	
	return dataSource, columns, nil
}

func (s *connectorService) processExcelFile(file *multipart.FileHeader) (*models.DataSource, []models.Column, error) {
	src, err := file.Open()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open Excel file: %w", err)
	}
	defer src.Close()

	// Create temporary file to read Excel
	tempData := make([]byte, file.Size)
	_, err = io.ReadFull(src, tempData)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read Excel file: %w", err)
	}

	f, err := excelize.OpenReader(strings.NewReader(string(tempData)))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse Excel file: %w", err)
	}
	defer f.Close()

	// Get the first sheet
	sheetName := f.GetSheetName(0)
	if sheetName == "" {
		return nil, nil, fmt.Errorf("no sheets found in Excel file")
	}

	rows, err := f.GetRows(sheetName)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read Excel rows: %w", err)
	}

	if len(rows) == 0 {
		return nil, nil, fmt.Errorf("Excel file is empty")
	}

	// Create data source
	dataSource := &models.DataSource{
		Name:        strings.TrimSuffix(file.Filename, filepath.Ext(file.Filename)),
		Type:        models.DataSourceTypeExcel,
		Description: fmt.Sprintf("Excel file: %s", file.Filename),
		Status:      models.ConnectionStatusActive,
	}

	// First row as headers
	headers := rows[0]
	columns := make([]models.Column, len(headers))

	// Analyze data types from sample rows
	for i, header := range headers {
		dataType := "text" // default
		if len(rows) > 1 {
			// Sample first few rows to determine data type
			dataType = s.inferDataTypeFromRows(rows[1:], i)
		}

		columns[i] = models.Column{
			Name:     strings.TrimSpace(header),
			Type:     dataType,
			Nullable: true,
		}
	}

	return dataSource, columns, nil
}

// inferDataType infers the data type from sample data
func (s *connectorService) inferDataType(sampleRows [][]string, columnIndex int) string {
	if len(sampleRows) == 0 {
		return "text"
	}
	
	hasNumbers := 0
	hasDecimals := 0
	totalRows := 0
	
	for _, row := range sampleRows {
		if columnIndex >= len(row) {
			continue
		}
		
		value := strings.TrimSpace(row[columnIndex])
		if value == "" {
			continue
		}
		
		totalRows++
		
		// Try to parse as integer
		if _, err := strconv.Atoi(value); err == nil {
			hasNumbers++
			continue
		}
		
		// Try to parse as float
		if _, err := strconv.ParseFloat(value, 64); err == nil {
			hasDecimals++
			continue
		}
	}
	
	if totalRows == 0 {
		return "text"
	}
	
	// If more than 80% are decimals, consider it decimal
	if float64(hasDecimals)/float64(totalRows) > 0.8 {
		return "decimal"
	}
	
	// If more than 80% are integers, consider it integer
	if float64(hasNumbers)/float64(totalRows) > 0.8 {
		return "integer"
	}
	
	// Default to text
	return "text"
}

// inferDataTypeFromRows infers the data type from Excel rows
func (s *connectorService) inferDataTypeFromRows(rows [][]string, columnIndex int) string {
	if len(rows) == 0 {
		return "text"
	}
	
	hasNumbers := 0
	hasDecimals := 0
	totalRows := 0
	
	for _, row := range rows {
		if columnIndex >= len(row) {
			continue
		}
		
		value := strings.TrimSpace(row[columnIndex])
		if value == "" {
			continue
		}
		
		totalRows++
		
		// Try to parse as integer
		if _, err := strconv.Atoi(value); err == nil {
			hasNumbers++
			continue
		}
		
		// Try to parse as float
		if _, err := strconv.ParseFloat(value, 64); err == nil {
			hasDecimals++
			continue
		}
	}
	
	if totalRows == 0 {
		return "text"
	}
	
	// If more than 80% are decimals, consider it decimal
	if float64(hasDecimals)/float64(totalRows) > 0.8 {
		return "decimal"
	}
	
	// If more than 80% are integers, consider it integer
	if float64(hasNumbers)/float64(totalRows) > 0.8 {
		return "integer"
	}
	
	// Default to text
	return "text"
}