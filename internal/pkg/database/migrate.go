package database

import (
	models "narapulse-be/internal/models/entity"
	"gorm.io/gorm"
)

// AutoMigrate runs GORM auto-migration for all models
// Note: Auto-migration is disabled as we use SQL migrations via Goose
// However, we enable it for NL2SQL models for development purposes
func AutoMigrate(db *gorm.DB) error {
	// Auto-migrate NL2SQL models
	return db.AutoMigrate(
		&models.NL2SQLQuery{},
		&models.QueryResult{},
	)
}