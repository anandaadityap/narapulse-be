package database

import (
	"gorm.io/gorm"
)

// AutoMigrate runs GORM auto-migration for all models
// Note: Auto-migration is disabled as we use SQL migrations via Goose
func AutoMigrate(db *gorm.DB) error {
	// Skip auto-migration since we use SQL migrations
	// return db.AutoMigrate(
	//	&models.User{},
	// )
	return nil
}