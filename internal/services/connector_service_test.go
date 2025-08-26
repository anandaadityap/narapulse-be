package services

import (
	"bytes"
	"mime/multipart"
	"net/textproto"
	"path/filepath"
	"strings"
	"testing"

	models "narapulse-be/internal/models/entity"
	"github.com/stretchr/testify/assert"
)

func TestNewConnectorService(t *testing.T) {
	service := NewConnectorService()
	assert.NotNil(t, service)
}

func TestConnectorService_TestConnection(t *testing.T) {
	service := NewConnectorService()

	tests := []struct {
		name    string
		request models.TestConnectionRequest
		wantErr bool
	}{
		{
			name: "unsupported data source type",
			request: models.TestConnectionRequest{
				Type:   "unsupported",
				Config: map[string]interface{}{},
			},
			wantErr: true,
		},
		{
			name: "CSV type should not error",
			request: models.TestConnectionRequest{
				Type:   models.DataSourceTypeCSV,
				Config: map[string]interface{}{},
			},
			wantErr: false,
		},
		{
			name: "Excel type should not error",
			request: models.TestConnectionRequest{
				Type:   models.DataSourceTypeExcel,
				Config: map[string]interface{}{},
			},
			wantErr: false,
		},
		{
			name: "PostgreSQL with invalid config",
			request: models.TestConnectionRequest{
				Type:   models.DataSourceTypePostgreSQL,
				Config: map[string]interface{}{}, // empty config
			},
			wantErr: true,
		},
		{
			name: "BigQuery with invalid config",
			request: models.TestConnectionRequest{
				Type:   models.DataSourceTypeBigQuery,
				Config: map[string]interface{}{}, // empty config
			},
			wantErr: true,
		},
		{
			name: "Google Sheets with invalid config",
			request: models.TestConnectionRequest{
				Type:   models.DataSourceTypeGoogleSheets,
				Config: map[string]interface{}{}, // empty config
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.TestConnection(tt.request)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConnectorService_DiscoverSchema(t *testing.T) {
	service := NewConnectorService()

	tests := []struct {
		name    string
		dsType  models.DataSourceType
		config  map[string]interface{}
		wantErr bool
	}{
		{
			name:    "unsupported data source type",
			dsType:  "unsupported",
			config:  map[string]interface{}{},
			wantErr: true,
		},
		{
			name:    "PostgreSQL with invalid config",
			dsType:  models.DataSourceTypePostgreSQL,
			config:  map[string]interface{}{}, // empty config
			wantErr: true,
		},
		{
			name:    "BigQuery with invalid config",
			dsType:  models.DataSourceTypeBigQuery,
			config:  map[string]interface{}{}, // empty config
			wantErr: true,
		},
		{
			name:    "Google Sheets with invalid config",
			dsType:  models.DataSourceTypeGoogleSheets,
			config:  map[string]interface{}{}, // empty config
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema, err := service.DiscoverSchema(tt.dsType, tt.config)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, schema)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, schema)
			}
		})
	}
}

func TestConnectorService_ProcessFileUpload(t *testing.T) {
	service := NewConnectorService()

	tests := []struct {
		name     string
		filename string
		content  string
		wantErr  bool
	}{
		{
			name:     "unsupported file type",
			filename: "test.txt",
			content:  "some content",
			wantErr:  true,
		},
		{
			name:     "valid CSV file",
			filename: "test.csv",
			content:  "name,age,city\nJohn,25,NYC\nJane,30,LA",
			wantErr:  false,
		},
		{
			name:     "empty CSV file",
			filename: "empty.csv",
			content:  "",
			wantErr:  true,
		},
		{
			name:     "CSV with only headers",
			filename: "headers.csv",
			content:  "name,age,city",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a multipart file header
			fileHeader := createTestFileHeader(tt.filename, tt.content)

			dataSource, columns, err := service.ProcessFileUpload(fileHeader)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, dataSource)
				assert.Nil(t, columns)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, dataSource)
				assert.NotNil(t, columns)
				// DataSource name should be filename without extension
				expectedName := strings.TrimSuffix(tt.filename, filepath.Ext(tt.filename))
				assert.Equal(t, expectedName, dataSource.Name)
			}
		})
	}
}

func TestConnectorService_InferDataType(t *testing.T) {
	service := NewConnectorService()

	tests := []struct {
		name       string
		sampleRows [][]string
		colIndex   int
		expected   string
	}{
		{
			name: "integer column",
			sampleRows: [][]string{
				{"John", "25"},
				{"Jane", "30"},
				{"Bob", "35"},
				{"Alice", "40"},
				{"Charlie", "45"},
			},
			colIndex: 1,
			expected: "integer",
		},
		{
			name: "float column",
			sampleRows: [][]string{
				{"John", "50000.50"},
				{"Jane", "60000.75"},
				{"Bob", "55000.25"},
				{"Alice", "45000.80"},
				{"Charlie", "70000.90"},
			},
			colIndex: 1,
			expected: "decimal",
		},
		{
			name: "boolean column",
			sampleRows: [][]string{
				{"name", "active"},
				{"John", "true"},
				{"Jane", "false"},
				{"Bob", "true"},
			},
			colIndex: 1,
			expected: "text",
		},
		{
			name: "string column",
			sampleRows: [][]string{
				{"name", "city"},
				{"John", "New York"},
				{"Jane", "Los Angeles"},
				{"Bob", "Chicago"},
			},
			colIndex: 1,
			expected: "text",
		},
		{
			name: "mixed types default to string",
			sampleRows: [][]string{
				{"name", "mixed"},
				{"John", "25"},
				{"Jane", "text"},
				{"Bob", "true"},
			},
			colIndex: 1,
			expected: "text",
		},
		{
			name: "empty column",
			sampleRows: [][]string{
				{"name", "empty"},
			},
			colIndex: 1,
			expected: "text",
		},
		{
			name: "out of bounds column",
			sampleRows: [][]string{
				{"name"},
				{"John"},
			},
			colIndex: 1,
			expected: "text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.inferDataType(tt.sampleRows, tt.colIndex)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Helper function to create a test multipart file header
func createTestFileHeader(filename, content string) *multipart.FileHeader {
	// Create a buffer to write our multipart form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Create a form file field
	part, _ := writer.CreateFormFile("file", filename)
	part.Write([]byte(content))
	writer.Close()

	// Parse the multipart form
	reader := multipart.NewReader(body, writer.Boundary())
	form, _ := reader.ReadForm(int64(len(content)) + 1024)

	// Get the file header
	if files, ok := form.File["file"]; ok && len(files) > 0 {
		return files[0]
	}

	// Fallback: create a simple file header manually
	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", `form-data; name="file"; filename="`+filename+`"`)
	header.Set("Content-Type", "application/octet-stream")

	return &multipart.FileHeader{
		Filename: filename,
		Header:   header,
		Size:     int64(len(content)),
	}
}