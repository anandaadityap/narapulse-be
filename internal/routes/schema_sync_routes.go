package routes

import (
	"github.com/gofiber/fiber/v2"
	"narapulse-be/internal/handlers"
	"narapulse-be/internal/middleware"
)

// SetupSchemaSyncRoutes sets up the schema synchronization routes
func SetupSchemaSyncRoutes(app *fiber.App, schemaSyncHandler *handlers.SchemaSyncHandler) {
	// Schema sync routes group
	schemaSync := app.Group("/api/v1/schema-sync")

	// Apply authentication middleware to all schema sync routes
	schemaSync.Use(middleware.AuthMiddleware())

	// Get sync status for all data sources
	schemaSync.Get("/status", schemaSyncHandler.GetSyncStatus)

	// Get sync status for specific data source
	schemaSync.Get("/status/:data_source_id", schemaSyncHandler.GetDataSourceSyncStatus)

	// Trigger sync for all data sources
	schemaSync.Post("/trigger", schemaSyncHandler.TriggerSyncAll)

	// Trigger sync for specific data source
	schemaSync.Post("/trigger/:data_source_id", schemaSyncHandler.TriggerSync)

	// Scheduled sync endpoint (for cron jobs)
	schemaSync.Post("/scheduled", schemaSyncHandler.ScheduledSync)
}