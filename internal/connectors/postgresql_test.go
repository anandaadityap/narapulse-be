package connectors

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewPostgreSQLConnector(t *testing.T) {
	connector := NewPostgreSQLConnector()
	assert.NotNil(t, connector)
	assert.Nil(t, connector.db)
}

func TestPostgreSQLConnector_Connect_InvalidConfig(t *testing.T) {
	connector := NewPostgreSQLConnector()

	// Test with empty config
	err := connector.Connect(map[string]interface{}{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "host is required")

	// Test with missing required fields
	config := map[string]interface{}{
		"host": "localhost",
		// missing database, username, password (port has default value)
	}
	err = connector.Connect(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database is required")
}

func TestPostgreSQLConnector_Connect_ValidConfig(t *testing.T) {
	connector := NewPostgreSQLConnector()

	config := map[string]interface{}{
		"host":     "localhost",
		"port":     5432,
		"database": "test_db",
		"username": "test_user",
		"password": "test_pass",
		"ssl_mode": "disable",
	}

	// This will fail in CI/test environment without actual PostgreSQL
	// but we can test the connection string building logic
	err := connector.Connect(config)
	// We expect an error since there's no actual PostgreSQL server
	// but the error should be about connection, not config validation
	if err != nil {
		// Should not be a config validation error
		assert.NotContains(t, err.Error(), "is required")
	}
}

func TestPostgreSQLConnector_Disconnect(t *testing.T) {
	connector := NewPostgreSQLConnector()

	// Test disconnect without connection
	err := connector.Disconnect()
	assert.NoError(t, err)
}

func TestPostgreSQLConnector_TestConnection_NoConnection(t *testing.T) {
	connector := NewPostgreSQLConnector()

	err := connector.TestConnection()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no active connection")
}

func TestPostgreSQLConnector_GetSchema_NoConnection(t *testing.T) {
	connector := NewPostgreSQLConnector()

	schema, err := connector.GetSchema()
	assert.Error(t, err)
	assert.Nil(t, schema)
	assert.Contains(t, err.Error(), "no active connection")
}

func TestPostgreSQLConnector_GetData_NoConnection(t *testing.T) {
	connector := NewPostgreSQLConnector()

	data, err := connector.GetData("test_table", 10)
	assert.Error(t, err)
	assert.Nil(t, data)
	assert.Contains(t, err.Error(), "no active connection")
}