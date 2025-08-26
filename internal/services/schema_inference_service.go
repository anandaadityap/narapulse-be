package services

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	models "narapulse-be/internal/models/entity"
)

// SchemaInferenceService handles automatic schema detection and data type inference
type SchemaInferenceService struct{}

// NewSchemaInferenceService creates a new schema inference service
func NewSchemaInferenceService() *SchemaInferenceService {
	return &SchemaInferenceService{}
}

// InferSchemaFromSample infers schema from sample data
func (s *SchemaInferenceService) InferSchemaFromSample(sampleData []map[string]interface{}, sourceName string) (*models.Schema, error) {
	if len(sampleData) == 0 {
		return nil, fmt.Errorf("no sample data provided")
	}

	// Get all unique column names from sample data
	columnNames := s.extractColumnNames(sampleData)
	if len(columnNames) == 0 {
		return nil, fmt.Errorf("no columns found in sample data")
	}

	// Infer column types and properties
	columns := make([]models.Column, 0, len(columnNames))
	for _, columnName := range columnNames {
		column := s.inferColumnFromSample(columnName, sampleData)
		columns = append(columns, column)
	}

	// Convert columns to JSON
	columnsJSON, err := json.Marshal(columns)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal columns: %w", err)
	}

	// Convert sample data to JSON
	sampleDataJSON, err := json.Marshal(s.prepareSampleData(sampleData, 5))
	if err != nil {
		return nil, fmt.Errorf("failed to marshal sample data: %w", err)
	}

	// Create schema
	schema := &models.Schema{
		Name:        sourceName,
		DisplayName: s.generateDisplayName(sourceName),
		Description: fmt.Sprintf("Auto-detected schema for %s", sourceName),
		Columns:     models.JSON(columnsJSON),
		RowCount:    int64(len(sampleData)),
		SampleData:  models.JSON(sampleDataJSON),
	}

	return schema, nil
}

// InferColumnType infers the data type of a column from sample values
func (s *SchemaInferenceService) InferColumnType(values []interface{}) string {
	if len(values) == 0 {
		return "string"
	}

	// Count occurrences of each type
	typeCounts := map[string]int{
		"integer":   0,
		"float":     0,
		"boolean":   0,
		"date":      0,
		"datetime":  0,
		"time":      0,
		"email":     0,
		"url":       0,
		"phone":     0,
		"string":    0,
		"null":      0,
	}

	for _, value := range values {
		detectedType := s.detectValueType(value)
		typeCounts[detectedType]++
	}

	// Remove null count from total for percentage calculation
	totalNonNull := len(values) - typeCounts["null"]
	if totalNonNull == 0 {
		return "string" // All values are null, default to string
	}

	// Find the most common type (excluding null)
	mostCommonType := "string"
	maxCount := 0

	for dataType, count := range typeCounts {
		if dataType != "null" && count > maxCount {
			maxCount = count
			mostCommonType = dataType
		}
	}

	// If the most common type represents at least 70% of non-null values, use it
	threshold := float64(totalNonNull) * 0.7
	if float64(maxCount) >= threshold {
		return mostCommonType
	}

	// Otherwise, default to string
	return "string"
}

// detectValueType detects the type of a single value
func (s *SchemaInferenceService) detectValueType(value interface{}) string {
	if value == nil {
		return "null"
	}

	str := strings.TrimSpace(fmt.Sprintf("%v", value))
	if str == "" {
		return "null"
	}

	// Check for boolean
	if s.isBooleanValue(str) {
		return "boolean"
	}

	// Check for integer
	if s.isIntegerValue(str) {
		return "integer"
	}

	// Check for float
	if s.isFloatValue(str) {
		return "float"
	}

	// Check for date/time formats
	if s.isDateValue(str) {
		return "date"
	}
	if s.isDateTimeValue(str) {
		return "datetime"
	}
	if s.isTimeValue(str) {
		return "time"
	}

	// Check for special string formats
	if s.isEmailValue(str) {
		return "email"
	}
	if s.isURLValue(str) {
		return "url"
	}
	if s.isPhoneValue(str) {
		return "phone"
	}

	return "string"
}

// Type detection helper methods
func (s *SchemaInferenceService) isBooleanValue(str string) bool {
	lower := strings.ToLower(str)
	return lower == "true" || lower == "false" || lower == "1" || lower == "0" ||
		lower == "yes" || lower == "no" || lower == "y" || lower == "n"
}

func (s *SchemaInferenceService) isIntegerValue(str string) bool {
	_, err := strconv.Atoi(str)
	return err == nil
}

func (s *SchemaInferenceService) isFloatValue(str string) bool {
	_, err := strconv.ParseFloat(str, 64)
	return err == nil
}

func (s *SchemaInferenceService) isDateValue(str string) bool {
	dateFormats := []string{
		"2006-01-02",
		"01/02/2006",
		"02/01/2006",
		"2006/01/02",
		"Jan 2, 2006",
		"January 2, 2006",
		"2-Jan-2006",
		"02-Jan-2006",
	}

	for _, format := range dateFormats {
		if _, err := time.Parse(format, str); err == nil {
			return true
		}
	}
	return false
}

func (s *SchemaInferenceService) isDateTimeValue(str string) bool {
	datetimeFormats := []string{
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05.000Z",
		"2006-01-02T15:04:05-07:00",
		"01/02/2006 15:04:05",
		"02/01/2006 15:04:05",
		time.RFC3339,
		time.RFC822,
	}

	for _, format := range datetimeFormats {
		if _, err := time.Parse(format, str); err == nil {
			return true
		}
	}
	return false
}

func (s *SchemaInferenceService) isTimeValue(str string) bool {
	timeFormats := []string{
		"15:04:05",
		"15:04",
		"3:04 PM",
		"3:04:05 PM",
	}

	for _, format := range timeFormats {
		if _, err := time.Parse(format, str); err == nil {
			return true
		}
	}
	return false
}

func (s *SchemaInferenceService) isEmailValue(str string) bool {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(str)
}

func (s *SchemaInferenceService) isURLValue(str string) bool {
	urlRegex := regexp.MustCompile(`^https?://[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}(/.*|\?.*)?$`)
	return urlRegex.MatchString(str)
}

func (s *SchemaInferenceService) isPhoneValue(str string) bool {
	// Remove common phone number separators
	cleanStr := regexp.MustCompile(`[\s\-\(\)\+\.]`).ReplaceAllString(str, "")
	// Check if it's all digits and has reasonable length
	phoneRegex := regexp.MustCompile(`^\d{7,15}$`)
	return phoneRegex.MatchString(cleanStr)
}

// Helper methods
func (s *SchemaInferenceService) extractColumnNames(sampleData []map[string]interface{}) []string {
	columnSet := make(map[string]bool)
	var columnNames []string

	for _, row := range sampleData {
		for columnName := range row {
			if !columnSet[columnName] {
				columnSet[columnName] = true
				columnNames = append(columnNames, columnName)
			}
		}
	}

	return columnNames
}

func (s *SchemaInferenceService) inferColumnFromSample(columnName string, sampleData []map[string]interface{}) models.Column {
	// Extract values for this column
	var values []interface{}
	nullCount := 0

	for _, row := range sampleData {
		if value, exists := row[columnName]; exists {
			values = append(values, value)
			if value == nil {
				nullCount++
			}
		} else {
			values = append(values, nil)
			nullCount++
		}
	}

	// Infer type
	dataType := s.InferColumnType(values)

	// Determine if column is nullable
	nullable := nullCount > 0

	return models.Column{
		Name:     columnName,
		Type:     dataType,
		Nullable: nullable,
	}
}

func (s *SchemaInferenceService) generateDisplayName(sourceName string) string {
	// Convert snake_case or kebab-case to Title Case
	words := regexp.MustCompile(`[_\-\s]+`).Split(sourceName, -1)
	var titleWords []string

	for _, word := range words {
		if len(word) > 0 {
			titleWords = append(titleWords, strings.Title(strings.ToLower(word)))
		}
	}

	return strings.Join(titleWords, " ")
}

func (s *SchemaInferenceService) prepareSampleData(sampleData []map[string]interface{}, limit int) []map[string]interface{} {
	if len(sampleData) <= limit {
		return sampleData
	}
	return sampleData[:limit]
}

// AnalyzeDataQuality analyzes the quality of data in a column
func (s *SchemaInferenceService) AnalyzeDataQuality(values []interface{}) map[string]interface{} {
	total := len(values)
	nullCount := 0
	emptyCount := 0
	uniqueValues := make(map[string]bool)

	for _, value := range values {
		if value == nil {
			nullCount++
			continue
		}

		str := strings.TrimSpace(fmt.Sprintf("%v", value))
		if str == "" {
			emptyCount++
			continue
		}

		uniqueValues[str] = true
	}

	completeness := float64(total-nullCount-emptyCount) / float64(total) * 100
	uniqueness := float64(len(uniqueValues)) / float64(total) * 100

	return map[string]interface{}{
		"total_count":      total,
		"null_count":       nullCount,
		"empty_count":      emptyCount,
		"unique_count":     len(uniqueValues),
		"completeness_pct": completeness,
		"uniqueness_pct":   uniqueness,
		"null_percentage":  float64(nullCount) / float64(total) * 100,
	}
}