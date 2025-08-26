package connectors

import (
	"database/sql"
	"fmt"
	"strings"

	entity "narapulse-be/internal/models/entity"

	_ "github.com/lib/pq"
)

// PostgreSQLConnector implements the Connector interface for PostgreSQL databases
type PostgreSQLConnector struct {
	db *sql.DB
}

// NewPostgreSQLConnector creates a new PostgreSQL connector
func NewPostgreSQLConnector() *PostgreSQLConnector {
	return &PostgreSQLConnector{}
}

// Connect establishes a connection to PostgreSQL database
func (p *PostgreSQLConnector) Connect(config map[string]interface{}) error {
	host, ok := config["host"].(string)
	if !ok {
		return fmt.Errorf("host is required")
	}

	port, ok := config["port"].(string)
	if !ok {
		port = "5432" // default PostgreSQL port
	}

	database, ok := config["database"].(string)
	if !ok {
		return fmt.Errorf("database is required")
	}

	username, ok := config["username"].(string)
	if !ok {
		return fmt.Errorf("username is required")
	}

	password, ok := config["password"].(string)
	if !ok {
		return fmt.Errorf("password is required")
	}

	sslMode, ok := config["ssl_mode"].(string)
	if !ok {
		sslMode = "disable" // default SSL mode
	}

	// Build connection string
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, username, password, database, sslMode)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("failed to open database connection: %w", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	p.db = db
	return nil
}

// Disconnect closes the database connection
func (p *PostgreSQLConnector) Disconnect() error {
	if p.db != nil {
		return p.db.Close()
	}
	return nil
}

// TestConnection tests if the connection is working
func (p *PostgreSQLConnector) TestConnection() error {
	if p.db == nil {
		return fmt.Errorf("no active connection")
	}
	return p.db.Ping()
}

// GetSchema retrieves the schema information from PostgreSQL
func (p *PostgreSQLConnector) GetSchema() ([]entity.Column, error) {
	if p.db == nil {
		return nil, fmt.Errorf("no active connection")
	}

	// Query to get all tables and their columns
	query := `
		SELECT 
			t.table_name,
			c.column_name,
			c.data_type,
			c.is_nullable,
			c.column_default
		FROM information_schema.tables t
		JOIN information_schema.columns c ON t.table_name = c.table_name
		WHERE t.table_schema = 'public'
		ORDER BY t.table_name, c.ordinal_position
	`

	rows, err := p.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query schema: %w", err)
	}
	defer rows.Close()

	var columns []entity.Column

	for rows.Next() {
		var tableName, columnName, dataType, isNullable string
		var columnDefault sql.NullString

		if err := rows.Scan(&tableName, &columnName, &dataType, &isNullable, &columnDefault); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// Convert PostgreSQL data type to standard type
		standardType := p.convertDataType(dataType)

		// Create column
		column := entity.Column{
			Name:     fmt.Sprintf("%s.%s", tableName, columnName),
			Type:     standardType,
			Nullable: isNullable == "YES",
		}

		columns = append(columns, column)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return columns, nil
}

// GetData retrieves data from a specific table
func (p *PostgreSQLConnector) GetData(tableName string, limit int) ([]map[string]interface{}, error) {
	if p.db == nil {
		return nil, fmt.Errorf("no active connection")
	}

	if limit <= 0 {
		limit = 100 // default limit
	}

	// Sanitize table name to prevent SQL injection
	if !p.isValidTableName(tableName) {
		return nil, fmt.Errorf("invalid table name")
	}

	query := fmt.Sprintf("SELECT * FROM %s LIMIT %d", tableName, limit)
	rows, err := p.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query data: %w", err)
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}

	var result []map[string]interface{}

	for rows.Next() {
		// Create a slice of interface{} to hold the values
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range columns {
			valuePtrs[i] = &values[i]
		}

		// Scan the row into the value pointers
		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// Create a map for this row
		row := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			if val != nil {
				// Convert byte arrays to strings for better JSON serialization
				if b, ok := val.([]byte); ok {
					val = string(b)
				}
			}
			row[col] = val
		}
		result = append(result, row)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return result, nil
}

// convertDataType converts PostgreSQL data types to standard types
func (p *PostgreSQLConnector) convertDataType(pgType string) string {
	switch strings.ToLower(pgType) {
	case "integer", "int", "int4", "serial", "serial4":
		return "integer"
	case "bigint", "int8", "bigserial", "serial8":
		return "bigint"
	case "smallint", "int2", "smallserial", "serial2":
		return "smallint"
	case "decimal", "numeric":
		return "decimal"
	case "real", "float4":
		return "float"
	case "double precision", "float8":
		return "double"
	case "boolean", "bool":
		return "boolean"
	case "character", "char", "character varying", "varchar", "text":
		return "string"
	case "date":
		return "date"
	case "time", "time without time zone":
		return "time"
	case "timestamp", "timestamp without time zone", "timestamp with time zone", "timestamptz":
		return "timestamp"
	case "json", "jsonb":
		return "json"
	case "uuid":
		return "uuid"
	default:
		return "string" // fallback to string for unknown types
	}
}

// isValidTableName checks if the table name is valid (basic SQL injection prevention)
func (p *PostgreSQLConnector) isValidTableName(tableName string) bool {
	// Allow only alphanumeric characters, underscores, and dots
	for _, char := range tableName {
		if !((char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || 
			(char >= '0' && char <= '9') || char == '_' || char == '.') {
			return false
		}
	}
	return len(tableName) > 0 && len(tableName) <= 63 // PostgreSQL identifier length limit
}