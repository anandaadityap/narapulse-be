package connectors

import (
	"context"
	"fmt"

	entity "narapulse-be/internal/models/entity"

	"cloud.google.com/go/bigquery"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

// BigQueryConnector implements the Connector interface for Google BigQuery
type BigQueryConnector struct {
	client    *bigquery.Client
	projectID string
	datasetID string
	ctx       context.Context
}

// NewBigQueryConnector creates a new BigQuery connector
func NewBigQueryConnector() *BigQueryConnector {
	return &BigQueryConnector{
		ctx: context.Background(),
	}
}

// Connect establishes a connection to BigQuery
func (b *BigQueryConnector) Connect(config map[string]interface{}) error {
	projectID, ok := config["project_id"].(string)
	if !ok {
		return fmt.Errorf("project_id is required")
	}
	b.projectID = projectID

	datasetID, ok := config["dataset_id"].(string)
	if !ok {
		return fmt.Errorf("dataset_id is required")
	}
	b.datasetID = datasetID

	// Check if service account key is provided
	var client *bigquery.Client
	var err error

	if serviceAccountKey, ok := config["service_account_key"].(string); ok && serviceAccountKey != "" {
		// Use service account key
		client, err = bigquery.NewClient(b.ctx, projectID, option.WithCredentialsJSON([]byte(serviceAccountKey)))
	} else {
		// Use default credentials (ADC - Application Default Credentials)
		client, err = bigquery.NewClient(b.ctx, projectID)
	}

	if err != nil {
		return fmt.Errorf("failed to create BigQuery client: %w", err)
	}

	b.client = client
	return nil
}

// Disconnect closes the BigQuery client
func (b *BigQueryConnector) Disconnect() error {
	if b.client != nil {
		return b.client.Close()
	}
	return nil
}

// TestConnection tests if the connection is working
func (b *BigQueryConnector) TestConnection() error {
	if b.client == nil {
		return fmt.Errorf("no active connection")
	}

	// Try to access the dataset to test connection
	dataset := b.client.Dataset(b.datasetID)
	_, err := dataset.Metadata(b.ctx)
	if err != nil {
		return fmt.Errorf("failed to access dataset: %w", err)
	}

	return nil
}

// GetSchema retrieves the schema information from BigQuery
func (b *BigQueryConnector) GetSchema() ([]entity.Column, error) {
	if b.client == nil {
		return nil, fmt.Errorf("no active connection")
	}

	dataset := b.client.Dataset(b.datasetID)

	// List all tables in the dataset
	it := dataset.Tables(b.ctx)
	var allColumns []entity.Column

	for {
		table, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to iterate tables: %w", err)
		}

		// Get table metadata
		meta, err := table.Metadata(b.ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get table metadata for %s: %w", table.TableID, err)
		}

		// Convert BigQuery schema to our schema format
		for _, field := range meta.Schema {
			column := entity.Column{
				Name:     fmt.Sprintf("%s.%s", table.TableID, field.Name),
				Type:     b.convertFieldType(field.Type),
				Nullable: !field.Required,
			}

			if field.Description != "" {
				column.Description = field.Description
			}

			allColumns = append(allColumns, column)
		}
	}

	return allColumns, nil
}

// GetData retrieves data from a specific table
func (b *BigQueryConnector) GetData(tableName string, limit int) ([]map[string]interface{}, error) {
	if b.client == nil {
		return nil, fmt.Errorf("no active connection")
	}

	if limit <= 0 {
		limit = 100 // default limit
	}

	// Sanitize table name to prevent SQL injection
	if !b.isValidTableName(tableName) {
		return nil, fmt.Errorf("invalid table name")
	}

	// Build the query
	query := fmt.Sprintf("SELECT * FROM `%s.%s.%s` LIMIT %d", 
		b.projectID, b.datasetID, tableName, limit)

	q := b.client.Query(query)
	it, err := q.Read(b.ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}

	var result []map[string]interface{}

	for {
		var row []bigquery.Value
		err := it.Next(&row)
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read row: %w", err)
		}

		// Convert row to map
		rowMap := make(map[string]interface{})
		for i, field := range it.Schema {
			if i < len(row) {
				rowMap[field.Name] = row[i]
			}
		}
		result = append(result, rowMap)
	}

	return result, nil
}

// convertFieldType converts BigQuery field types to standard types
func (b *BigQueryConnector) convertFieldType(bqType bigquery.FieldType) string {
	switch bqType {
	case bigquery.StringFieldType:
		return "string"
	case bigquery.BytesFieldType:
		return "bytes"
	case bigquery.IntegerFieldType:
		return "integer"
	case bigquery.FloatFieldType:
		return "float"
	case bigquery.BooleanFieldType:
		return "boolean"
	case bigquery.TimestampFieldType:
		return "timestamp"
	case bigquery.DateFieldType:
		return "date"
	case bigquery.TimeFieldType:
		return "time"
	case bigquery.DateTimeFieldType:
		return "datetime"
	case bigquery.NumericFieldType:
		return "decimal"
	case bigquery.BigNumericFieldType:
		return "decimal"
	case bigquery.GeographyFieldType:
		return "geography"
	case bigquery.JSONFieldType:
		return "json"
	case bigquery.RecordFieldType:
		return "record"
	default:
		return "string" // fallback to string for unknown types
	}
}

// isValidTableName checks if the table name is valid (basic SQL injection prevention)
func (b *BigQueryConnector) isValidTableName(tableName string) bool {
	// Allow only alphanumeric characters, underscores, and hyphens
	for _, char := range tableName {
		if !((char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || 
			(char >= '0' && char <= '9') || char == '_' || char == '-') {
			return false
		}
	}
	return len(tableName) > 0 && len(tableName) <= 1024 // BigQuery table name length limit
}