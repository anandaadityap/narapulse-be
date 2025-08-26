package services

import (
	"encoding/json"
	"fmt"
	"testing"

	models "narapulse-be/internal/models/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSchemaInferenceService(t *testing.T) {
	service := NewSchemaInferenceService()
	assert.NotNil(t, service)
}

func TestSchemaInferenceService_InferSchemaFromSample(t *testing.T) {
	service := NewSchemaInferenceService()

	tests := []struct {
		name       string
		sampleData []map[string]interface{}
		wantErr    bool
		expected   int // expected number of columns
	}{
		{
			name: "valid sample data",
			sampleData: []map[string]interface{}{
				{"name": "John", "age": 25, "active": true, "salary": 50000.50},
				{"name": "Jane", "age": 30, "active": false, "salary": 60000.75},
				{"name": "Bob", "age": 35, "active": true, "salary": 55000.25},
			},
			wantErr:  false,
			expected: 4,
		},
		{
			name:       "empty sample data",
			sampleData: []map[string]interface{}{},
			wantErr:    true,
			expected:   0,
		},
		{
			name:       "nil sample data",
			sampleData: nil,
			wantErr:    true,
			expected:   0,
		},
		{
			name: "single row sample",
			sampleData: []map[string]interface{}{
				{"id": 1, "name": "Test"},
			},
			wantErr:  false,
			expected: 2,
		},
		{
			name: "inconsistent columns",
			sampleData: []map[string]interface{}{
				{"name": "John", "age": 25},
				{"name": "Jane", "city": "NYC"},
				{"email": "bob@test.com", "age": 35},
			},
			wantErr:  false,
			expected: 4, // name, age, city, email
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema, err := service.InferSchemaFromSample(tt.sampleData, "test_source")

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, schema)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, schema)
				
				// Parse columns from JSON
				var columns []models.Column
				err = json.Unmarshal(schema.Columns, &columns)
				require.NoError(t, err)
				assert.Len(t, columns, tt.expected)
				
				// Verify sample data is stored
				var sampleData []map[string]interface{}
				err = json.Unmarshal(schema.SampleData, &sampleData)
				require.NoError(t, err)
				assert.NotEmpty(t, sampleData)
			}
		})
	}
}

func TestSchemaInferenceService_InferColumnType(t *testing.T) {
	service := NewSchemaInferenceService()

	tests := []struct {
		name     string
		values   []interface{}
		expected string
	}{
		{
			name:     "integer values",
			values:   []interface{}{1, 2, 3, 4, 5},
			expected: "integer",
		},
		{
			name:     "float values",
			values:   []interface{}{1.5, 2.7, 3.14, 4.0},
			expected: "float",
		},
		{
			name:     "boolean values",
			values:   []interface{}{true, false, true, false},
			expected: "boolean",
		},
		{
			name:     "string values",
			values:   []interface{}{"hello", "world", "test"},
			expected: "string",
		},
		{
			name:     "email values",
			values:   []interface{}{"john@test.com", "jane@example.org", "bob@company.net"},
			expected: "email",
		},
		{
			name:     "url values",
			values:   []interface{}{"https://example.com", "http://test.org", "https://company.net"},
			expected: "url",
		},
		{
			name:     "date values",
			values:   []interface{}{"2023-01-01", "2023-12-31", "2024-06-15"},
			expected: "date",
		},
		{
			name:     "datetime values",
			values:   []interface{}{"2023-01-01T10:00:00Z", "2023-12-31T23:59:59Z"},
			expected: "datetime",
		},
		{
			name:     "mixed types default to string",
			values:   []interface{}{1, "hello", true, 3.14},
			expected: "string",
		},
		{
			name:     "empty values",
			values:   []interface{}{},
			expected: "string",
		},
		{
			name:     "nil values",
			values:   []interface{}{nil, nil, nil},
			expected: "string",
		},
		{
			name:     "mixed with nulls",
			values:   []interface{}{2, nil, 3, nil, 5},
			expected: "integer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.InferColumnType(tt.values)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSchemaInferenceService_IsBooleanValue(t *testing.T) {
	service := NewSchemaInferenceService()

	tests := []struct {
		name     string
		value    interface{}
		expected bool
	}{
		{"true boolean", true, true},
		{"false boolean", false, true},
		{"string true", "true", true},
		{"string false", "false", true},
		{"string True", "True", true},
		{"string FALSE", "FALSE", true},
		{"string yes", "yes", true},
		{"string no", "no", true},
		{"string 1", "1", true},
		{"string 0", "0", true},
		{"integer 1", 1, true},
		{"integer 0", 0, true},
		{"string hello", "hello", false},
		{"integer 2", 2, false},
		{"float 1.0", 1.0, true},
		{"nil value", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valueStr, ok := tt.value.(string)
			if !ok {
				valueStr = fmt.Sprintf("%v", tt.value)
			}
			result := service.isBooleanValue(valueStr)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSchemaInferenceService_IsEmailValue(t *testing.T) {
	service := NewSchemaInferenceService()

	tests := []struct {
		name     string
		value    interface{}
		expected bool
	}{
		{"valid email", "test@example.com", true},
		{"valid email with subdomain", "user@mail.example.org", true},
		{"valid email with numbers", "user123@test123.net", true},
		{"invalid email no @", "testexample.com", false},
		{"invalid email no domain", "test@", false},
		{"invalid email no user", "@example.com", false},
		{"not string", 123, false},
		{"nil value", nil, false},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valueStr, ok := tt.value.(string)
			if !ok {
				valueStr = fmt.Sprintf("%v", tt.value)
			}
			result := service.isEmailValue(valueStr)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSchemaInferenceService_IsURLValue(t *testing.T) {
	service := NewSchemaInferenceService()

	tests := []struct {
		name     string
		value    interface{}
		expected bool
	}{
		{"valid https URL", "https://example.com", true},
		{"valid http URL", "http://test.org", true},
		{"valid URL with path", "https://example.com/path/to/page", true},
		{"valid URL with query", "https://example.com?param=value", true},
		{"invalid URL no protocol", "example.com", false},
		{"invalid URL malformed", "ht://example", false},
		{"not string", 123, false},
		{"nil value", nil, false},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valueStr, ok := tt.value.(string)
			if !ok {
				valueStr = fmt.Sprintf("%v", tt.value)
			}
			result := service.isURLValue(valueStr)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSchemaInferenceService_IsDateValue(t *testing.T) {
	service := NewSchemaInferenceService()

	tests := []struct {
		name     string
		value    interface{}
		expected bool
	}{
		{"valid date YYYY-MM-DD", "2023-01-01", true},
		{"valid date MM/DD/YYYY", "01/01/2023", true},
		{"invalid date DD-MM-YYYY", "01-01-2023", false},
		{"valid date DD/MM/YYYY", "01/01/2023", true},
		{"invalid date format", "2023/13/01", false},
		{"invalid date string", "not-a-date", false},
		{"not string", 123, false},
		{"nil value", nil, false},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valueStr, ok := tt.value.(string)
			if !ok {
				valueStr = fmt.Sprintf("%v", tt.value)
			}
			result := service.isDateValue(valueStr)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSchemaInferenceService_IsDateTimeValue(t *testing.T) {
	service := NewSchemaInferenceService()

	tests := []struct {
		name     string
		value    interface{}
		expected bool
	}{
		{"valid datetime ISO", "2023-01-01T10:00:00Z", true},
		{"valid datetime with timezone", "2023-01-01T10:00:00+07:00", true},
		{"valid datetime RFC3339", "2023-01-01T10:00:00.000Z", true},
		{"valid datetime format", "2023-01-01 10:00:00", true},
		{"invalid datetime string", "not-a-datetime", false},
		{"not string", 123, false},
		{"nil value", nil, false},
		{"empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valueStr, ok := tt.value.(string)
			if !ok {
				valueStr = fmt.Sprintf("%v", tt.value)
			}
			result := service.isDateTimeValue(valueStr)
			assert.Equal(t, tt.expected, result)
		})
	}
}