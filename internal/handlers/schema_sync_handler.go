package handlers

import (
	"strconv"

	models "narapulse-be/internal/models/entity"
	"narapulse-be/internal/services"
	"github.com/gofiber/fiber/v2"
)

// SchemaSyncHandler handles schema synchronization API endpoints
type SchemaSyncHandler struct {
	schemaSyncService *services.SchemaSyncService
}

// NewSchemaSyncHandler creates a new schema sync handler
func NewSchemaSyncHandler(schemaSyncService *services.SchemaSyncService) *SchemaSyncHandler {
	return &SchemaSyncHandler{
		schemaSyncService: schemaSyncService,
	}
}

// GetSyncStatus returns synchronization status for all data sources
// @Summary Get synchronization status
// @Description Get the synchronization status for all active data sources
// @Tags Schema Sync
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "Sync status retrieved successfully"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/schema-sync/status [get]
func (h *SchemaSyncHandler) GetSyncStatus(c *fiber.Ctx) error {
	status, err := h.schemaSyncService.GetSyncStatus()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Code:    "SYNC_STATUS_ERROR",
			Message: "Failed to get sync status",
			Details: err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(map[string]interface{}{
		"message": "Sync status retrieved successfully",
		"data":    status,
	})
}

// TriggerSyncAll triggers synchronization for all data sources
// @Summary Trigger sync for all data sources
// @Description Manually trigger synchronization for all active data sources
// @Tags Schema Sync
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "Sync triggered successfully"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/schema-sync/trigger-all [post]
func (h *SchemaSyncHandler) TriggerSyncAll(c *fiber.Ctx) error {
	if err := h.schemaSyncService.SyncAllDataSources(c.Context()); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Code:    "SYNC_ALL_ERROR",
			Message: "Failed to sync all data sources",
			Details: err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(map[string]interface{}{
		"message": "Sync triggered successfully for all data sources",
	})
}

// TriggerSync triggers synchronization for a specific data source
// @Summary Trigger sync for specific data source
// @Description Manually trigger synchronization for a specific data source
// @Tags Schema Sync
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param data_source_id path int true "Data Source ID"
// @Success 200 {object} map[string]interface{} "Sync triggered successfully"
// @Failure 400 {object} models.ErrorResponse "Invalid data source ID"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/schema-sync/trigger/{data_source_id} [post]
func (h *SchemaSyncHandler) TriggerSync(c *fiber.Ctx) error {
	dataSourceIDStr := c.Params("data_source_id")
	dataSourceID, err := strconv.ParseUint(dataSourceIDStr, 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{
			Code:    "INVALID_DATA_SOURCE_ID",
			Message: "Invalid data source ID",
			Details: err.Error(),
		})
	}

	if err := h.schemaSyncService.TriggerSync(c.Context(), uint(dataSourceID)); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Code:    "SYNC_ERROR",
			Message: "Failed to sync data source",
			Details: err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(map[string]interface{}{
		"message": "Sync triggered successfully",
		"data_source_id": dataSourceID,
	})
}

// GetDataSourceSyncStatus returns synchronization status for a specific data source
// @Summary Get sync status for specific data source
// @Description Get the synchronization status for a specific data source
// @Tags Schema Sync
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param data_source_id path int true "Data Source ID"
// @Success 200 {object} map[string]interface{} "Sync status retrieved successfully"
// @Failure 400 {object} models.ErrorResponse "Invalid data source ID"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/schema-sync/status/{data_source_id} [get]
func (h *SchemaSyncHandler) GetDataSourceSyncStatus(c *fiber.Ctx) error {
	dataSourceIDStr := c.Params("data_source_id")
	dataSourceID, err := strconv.ParseUint(dataSourceIDStr, 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{
			Code:    "INVALID_DATA_SOURCE_ID",
			Message: "Invalid data source ID",
			Details: err.Error(),
		})
	}

	// Get all sync status and filter for the specific data source
	allStatus, err := h.schemaSyncService.GetSyncStatus()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Code:    "SYNC_STATUS_ERROR",
			Message: "Failed to get sync status",
			Details: err.Error(),
		})
	}

	// Find the specific data source status
	for _, status := range allStatus {
		if status.DataSourceID == uint(dataSourceID) {
			return c.Status(fiber.StatusOK).JSON(map[string]interface{}{
				"message": "Sync status retrieved successfully",
				"data":    status,
			})
		}
	}

	return c.Status(fiber.StatusNotFound).JSON(models.ErrorResponse{
		Code:    "DATA_SOURCE_NOT_FOUND",
		Message: "Data source not found or not active",
	})
}

// ScheduledSync endpoint for triggering scheduled synchronization
// @Summary Trigger scheduled sync
// @Description Trigger scheduled synchronization (typically called by cron jobs)
// @Tags Schema Sync
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "Scheduled sync completed successfully"
// @Failure 500 {object} models.ErrorResponse "Internal server error"
// @Router /api/v1/schema-sync/scheduled [post]
func (h *SchemaSyncHandler) ScheduledSync(c *fiber.Ctx) error {
	if err := h.schemaSyncService.ScheduledSync(c.Context()); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Code:    "SCHEDULED_SYNC_ERROR",
			Message: "Scheduled sync failed",
			Details: err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(map[string]interface{}{
		"message": "Scheduled sync completed successfully",
	})
}