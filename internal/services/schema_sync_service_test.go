package services

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSchemaSyncService_Validation(t *testing.T) {
	// Test service creation
	service := NewSchemaSyncService(nil, nil, nil)
	assert.NotNil(t, service)
}

func TestSchemaSyncService_GetSyncStatus(t *testing.T) {
	service := &SchemaSyncService{}

	// Test getting sync status with nil db should panic
	assert.Panics(t, func() {
		service.GetSyncStatus()
	})
}

func TestSchemaSyncService_CheckSyncNeeded(t *testing.T) {
	service := &SchemaSyncService{}

	// Test sync needed check with nil db should panic
	assert.Panics(t, func() {
		service.checkSyncNeeded(999)
	})
}

func TestSchemaSyncService_UpdateSyncTimestamp(t *testing.T) {
	service := &SchemaSyncService{}

	// Test updating sync timestamp with nil db should panic
	assert.Panics(t, func() {
		service.updateSyncTimestamp(999)
	})
}

func TestSchemaSyncService_ScheduledSync(t *testing.T) {
	service := &SchemaSyncService{}

	// Test scheduled sync with nil db should panic
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	assert.Panics(t, func() {
		service.ScheduledSync(ctx)
	})
}

func TestSchemaSyncService_TriggerSync(t *testing.T) {
	service := &SchemaSyncService{}

	// Test trigger sync with nil db should panic
	ctx := context.Background()
	assert.Panics(t, func() {
		service.TriggerSync(ctx, 999)
	})
}