package connectors

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewGoogleSheetsConnector(t *testing.T) {
	connector := NewGoogleSheetsConnector()
	assert.NotNil(t, connector)
	assert.Nil(t, connector.service)
	assert.NotNil(t, connector.ctx)
}

func TestGoogleSheetsConnector_Connect_InvalidConfig(t *testing.T) {
	connector := NewGoogleSheetsConnector()

	// Test with empty config
	err := connector.Connect(map[string]interface{}{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "spreadsheet_id is required")
}

func TestGoogleSheetsConnector_Connect_ValidConfig(t *testing.T) {
	connector := NewGoogleSheetsConnector()

	config := map[string]interface{}{
		"spreadsheet_id": "test-spreadsheet-id",
		"sheet_name":     "Sheet1",
	}

	// This will fail in CI/test environment without actual Google Sheets credentials
	// but we can test the config validation logic
	err := connector.Connect(config)
	// We expect an error since there's no actual Google Sheets credentials
	// but the error should be about authentication, not config validation
	if err != nil {
		// Should not be a config validation error
		assert.NotContains(t, err.Error(), "is required")
	}
}

func TestGoogleSheetsConnector_Connect_DefaultSheetName(t *testing.T) {
	connector := NewGoogleSheetsConnector()

	config := map[string]interface{}{
		"spreadsheet_id": "test-spreadsheet-id",
		// no sheet_name provided
	}

	err := connector.Connect(config)
	// Should set default sheet name
	assert.Equal(t, "Sheet1", connector.sheetName)

	// Error should not be about missing sheet_name
	if err != nil {
		assert.NotContains(t, err.Error(), "sheet_name")
	}
}

func TestGoogleSheetsConnector_Disconnect(t *testing.T) {
	connector := NewGoogleSheetsConnector()

	// Test disconnect without connection
	err := connector.Disconnect()
	assert.NoError(t, err)
	assert.Nil(t, connector.service)
}

func TestGoogleSheetsConnector_TestConnection_NoConnection(t *testing.T) {
	connector := NewGoogleSheetsConnector()

	err := connector.TestConnection()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no active connection")
}

func TestGoogleSheetsConnector_GetSchema_NoConnection(t *testing.T) {
	connector := NewGoogleSheetsConnector()

	schema, err := connector.GetSchema()
	assert.Error(t, err)
	assert.Nil(t, schema)
	assert.Contains(t, err.Error(), "no active connection")
}

func TestGoogleSheetsConnector_GetData_NoConnection(t *testing.T) {
	connector := NewGoogleSheetsConnector()

	data, err := connector.GetData("Sheet1", 10)
	assert.Error(t, err)
	assert.Nil(t, data)
	assert.Contains(t, err.Error(), "no active connection")
}

func TestGoogleSheetsConnector_InferColumnType(t *testing.T) {
	connector := NewGoogleSheetsConnector()

	tests := []struct {
		name     string
		values   [][]interface{}
		colIndex int
		expected string
	}{
		{
			name: "integer column",
			values: [][]interface{}{
				{"Name", "Age"},     // header
				{"John", "25"},     // data
				{"Jane", "30"},     // data
				{"Bob", "35"},      // data
			},
			colIndex: 1,
			expected: "integer",
		},
		{
			name: "string column",
			values: [][]interface{}{
				{"Name", "City"},
				{"John", "New York"},
				{"Jane", "London"},
				{"Bob", "Tokyo"},
			},
			colIndex: 1,
			expected: "string",
		},
		{
			name: "float column",
			values: [][]interface{}{
				{"Name", "Salary"},
				{"John", "50000.50"},
				{"Jane", "60000.75"},
				{"Bob", "55000.25"},
			},
			colIndex: 1,
			expected: "float",
		},
		{
			name: "boolean column",
			values: [][]interface{}{
				{"Name", "Active"},
				{"John", "true"},
				{"Jane", "false"},
				{"Bob", "true"},
			},
			colIndex: 1,
			expected: "boolean",
		},
		{
			name: "empty data",
			values: [][]interface{}{
				{"Name", "Empty"},
			},
			colIndex: 1,
			expected: "string",
		},
		{
			name: "mixed types default to string",
			values: [][]interface{}{
				{"Name", "Mixed"},
				{"John", "25"},
				{"Jane", "text"},
				{"Bob", "true"},
			},
			colIndex: 1,
			expected: "string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := connector.inferColumnType(tt.values, tt.colIndex)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetEnvOrDefault(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue string
		expected     string
	}{
		{
			name:         "returns default when env not set",
			key:          "NON_EXISTENT_ENV_VAR",
			defaultValue: "default_value",
			expected:     "default_value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getEnvOrDefault(tt.key, tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}