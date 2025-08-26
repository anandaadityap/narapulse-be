package connectors

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewBigQueryConnector(t *testing.T) {
	connector := NewBigQueryConnector()
	assert.NotNil(t, connector)
	assert.Nil(t, connector.client)
}

func TestBigQueryConnector_Connect_InvalidConfig(t *testing.T) {
	connector := NewBigQueryConnector()

	// Test with empty config
	err := connector.Connect(map[string]interface{}{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "project_id is required")

	// Test with missing dataset_id
	config := map[string]interface{}{
		"project_id": "test-project",
		// missing dataset_id
	}
	err = connector.Connect(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "dataset_id is required")
}

func TestBigQueryConnector_Connect_ValidConfig(t *testing.T) {
	connector := NewBigQueryConnector()

	config := map[string]interface{}{
		"project_id": "test-project",
		"dataset_id": "test_dataset",
	}

	// This will fail in CI/test environment without actual BigQuery credentials
	// but we can test the config validation logic
	err := connector.Connect(config)
	// We expect an error since there's no actual BigQuery credentials
	// but the error should be about authentication, not config validation
	if err != nil {
		// Should not be a config validation error
		assert.NotContains(t, err.Error(), "is required")
	}
}

func TestBigQueryConnector_Disconnect(t *testing.T) {
	connector := NewBigQueryConnector()

	// Test disconnect without connection
	err := connector.Disconnect()
	assert.NoError(t, err)
}

func TestBigQueryConnector_TestConnection_NoConnection(t *testing.T) {
	connector := NewBigQueryConnector()

	err := connector.TestConnection()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no active connection")
}

func TestBigQueryConnector_GetSchema_NoConnection(t *testing.T) {
	connector := NewBigQueryConnector()

	schema, err := connector.GetSchema()
	assert.Error(t, err)
	assert.Nil(t, schema)
	assert.Contains(t, err.Error(), "no active connection")
}

func TestBigQueryConnector_GetData_NoConnection(t *testing.T) {
	connector := NewBigQueryConnector()

	data, err := connector.GetData("test_table", 10)
	assert.Error(t, err)
	assert.Nil(t, data)
	assert.Contains(t, err.Error(), "no active connection")
}