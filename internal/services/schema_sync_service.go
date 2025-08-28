package services

import (
	"context"
	"fmt"
	"log"
	"time"

	models "narapulse-be/internal/models/entity"
	"gorm.io/gorm"
)

// SchemaSyncService handles automatic synchronization of schema embeddings
type SchemaSyncService struct {
	db               *gorm.DB
	ragService       *RAGService
	embeddingService *EmbeddingService
}

// NewSchemaSyncService creates a new schema sync service
func NewSchemaSyncService(db *gorm.DB, ragService *RAGService, embeddingService *EmbeddingService) *SchemaSyncService {
	return &SchemaSyncService{
		db:               db,
		ragService:       ragService,
		embeddingService: embeddingService,
	}
}

// SyncAllDataSources synchronizes embeddings for all active data sources
func (s *SchemaSyncService) SyncAllDataSources(ctx context.Context) error {
	var dataSources []models.DataSource
	if err := s.db.Where("is_active = ?", true).Find(&dataSources).Error; err != nil {
		return fmt.Errorf("failed to get active data sources: %w", err)
	}

	for _, dataSource := range dataSources {
		if err := s.SyncDataSource(ctx, dataSource.ID); err != nil {
			log.Printf("Failed to sync data source %d: %v", dataSource.ID, err)
			// Continue with other data sources even if one fails
			continue
		}
	}

	return nil
}

// SyncDataSource synchronizes embeddings for a specific data source
func (s *SchemaSyncService) SyncDataSource(ctx context.Context, dataSourceID uint) error {
	// Get the data source
	var dataSource models.DataSource
	if err := s.db.First(&dataSource, dataSourceID).Error; err != nil {
		return fmt.Errorf("data source not found: %w", err)
	}

	// Check if sync is needed
	needSync, err := s.checkSyncNeeded(dataSourceID)
	if err != nil {
		return fmt.Errorf("failed to check sync status: %w", err)
	}

	if !needSync {
		log.Printf("Data source %d is already up to date", dataSourceID)
		return nil
	}

	// Perform synchronization
	log.Printf("Starting sync for data source %d (%s)", dataSourceID, dataSource.Name)

	// Remove old embeddings
	if err := s.removeOldEmbeddings(dataSourceID); err != nil {
		return fmt.Errorf("failed to remove old embeddings: %w", err)
	}

	// Generate new embeddings
	if err := s.ragService.SyncSchemaEmbeddings(ctx, dataSourceID); err != nil {
		return fmt.Errorf("failed to generate new embeddings: %w", err)
	}

	// Update sync timestamp
	if err := s.updateSyncTimestamp(dataSourceID); err != nil {
		return fmt.Errorf("failed to update sync timestamp: %w", err)
	}

	log.Printf("Successfully synced data source %d", dataSourceID)
	return nil
}

// checkSyncNeeded determines if synchronization is needed for a data source
func (s *SchemaSyncService) checkSyncNeeded(dataSourceID uint) (bool, error) {
	// Get the latest schema update time
	var latestSchemaUpdate time.Time
	if err := s.db.Model(&models.Schema{}).
		Where("data_source_id = ?", dataSourceID).
		Select("MAX(updated_at)").
		Scan(&latestSchemaUpdate).Error; err != nil {
		return false, fmt.Errorf("failed to get latest schema update: %w", err)
	}

	// Get the latest embedding sync time
	var latestEmbeddingSync time.Time
	if err := s.db.Model(&models.SchemaEmbedding{}).
		Where("data_source_id = ?", dataSourceID).
		Select("MAX(updated_at)").
		Scan(&latestEmbeddingSync).Error; err != nil {
		// If no embeddings exist, sync is needed
		if err == gorm.ErrRecordNotFound {
			return true, nil
		}
		return false, fmt.Errorf("failed to get latest embedding sync: %w", err)
	}

	// Sync is needed if schema was updated after last embedding sync
	return latestSchemaUpdate.After(latestEmbeddingSync), nil
}

// removeOldEmbeddings removes existing embeddings for a data source
func (s *SchemaSyncService) removeOldEmbeddings(dataSourceID uint) error {
	result := s.db.Where("data_source_id = ?", dataSourceID).Delete(&models.SchemaEmbedding{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete old embeddings: %w", result.Error)
	}

	log.Printf("Removed %d old embeddings for data source %d", result.RowsAffected, dataSourceID)
	return nil
}

// updateSyncTimestamp updates the sync timestamp for tracking
func (s *SchemaSyncService) updateSyncTimestamp(dataSourceID uint) error {
	// Update the data source's updated_at timestamp to track sync
	return s.db.Model(&models.DataSource{}).
		Where("id = ?", dataSourceID).
		Update("updated_at", time.Now()).Error
}

// ScheduledSync performs scheduled synchronization (can be called by cron job)
func (s *SchemaSyncService) ScheduledSync(ctx context.Context) error {
	log.Println("Starting scheduled schema synchronization")
	start := time.Now()

	if err := s.SyncAllDataSources(ctx); err != nil {
		log.Printf("Scheduled sync failed: %v", err)
		return err
	}

	duration := time.Since(start)
	log.Printf("Scheduled sync completed in %v", duration)
	return nil
}

// GetSyncStatus returns the synchronization status for all data sources
func (s *SchemaSyncService) GetSyncStatus() ([]SyncStatusInfo, error) {
	var dataSources []models.DataSource
	if err := s.db.Where("is_active = ?", true).Find(&dataSources).Error; err != nil {
		return nil, fmt.Errorf("failed to get data sources: %w", err)
	}

	var statusList []SyncStatusInfo
	for _, ds := range dataSources {
		status, err := s.getDataSourceSyncStatus(ds.ID)
		if err != nil {
			log.Printf("Failed to get sync status for data source %d: %v", ds.ID, err)
			continue
		}
		status.DataSourceName = ds.Name
		statusList = append(statusList, status)
	}

	return statusList, nil
}

// getDataSourceSyncStatus gets sync status for a specific data source
func (s *SchemaSyncService) getDataSourceSyncStatus(dataSourceID uint) (SyncStatusInfo, error) {
	status := SyncStatusInfo{
		DataSourceID: dataSourceID,
	}

	// Get schema count
	if err := s.db.Model(&models.Schema{}).
		Where("data_source_id = ? AND is_active = ?", dataSourceID, true).
		Count(&status.SchemaCount).Error; err != nil {
		return status, fmt.Errorf("failed to count schemas: %w", err)
	}

	// Get embedding count
	if err := s.db.Model(&models.SchemaEmbedding{}).
		Where("data_source_id = ?", dataSourceID).
		Count(&status.EmbeddingCount).Error; err != nil {
		return status, fmt.Errorf("failed to count embeddings: %w", err)
	}

	// Get last sync time
	if err := s.db.Model(&models.SchemaEmbedding{}).
		Where("data_source_id = ?", dataSourceID).
		Select("MAX(updated_at)").
		Scan(&status.LastSyncTime).Error; err != nil && err != gorm.ErrRecordNotFound {
		return status, fmt.Errorf("failed to get last sync time: %w", err)
	}

	// Check if sync is needed
	needSync, err := s.checkSyncNeeded(dataSourceID)
	if err != nil {
		return status, fmt.Errorf("failed to check sync status: %w", err)
	}
	status.NeedSync = needSync

	return status, nil
}

// SyncStatusInfo represents synchronization status information
type SyncStatusInfo struct {
	DataSourceID   uint      `json:"data_source_id"`
	DataSourceName string    `json:"data_source_name"`
	SchemaCount    int64     `json:"schema_count"`
	EmbeddingCount int64     `json:"embedding_count"`
	LastSyncTime   time.Time `json:"last_sync_time"`
	NeedSync       bool      `json:"need_sync"`
}

// TriggerSync manually triggers synchronization for a data source
func (s *SchemaSyncService) TriggerSync(ctx context.Context, dataSourceID uint) error {
	log.Printf("Manual sync triggered for data source %d", dataSourceID)
	return s.SyncDataSource(ctx, dataSourceID)
}

// AutoSyncOnSchemaChange automatically syncs when schema changes are detected
func (s *SchemaSyncService) AutoSyncOnSchemaChange(ctx context.Context, dataSourceID uint) error {
	// This method can be called from schema inference service or data source handlers
	// when schema changes are detected
	log.Printf("Auto sync triggered for data source %d due to schema change", dataSourceID)
	return s.SyncDataSource(ctx, dataSourceID)
}