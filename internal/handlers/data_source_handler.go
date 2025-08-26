package handlers

import (
	"strconv"

	entity "narapulse-be/internal/models/entity"
	services "narapulse-be/internal/services"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

type DataSourceHandler struct {
	dataSourceService services.DataSourceService
	validator         *validator.Validate
}

func NewDataSourceHandler(dataSourceService services.DataSourceService) *DataSourceHandler {
	return &DataSourceHandler{
		dataSourceService: dataSourceService,
		validator:         validator.New(),
	}
}

// CreateDataSource godoc
// @Summary Create a new data source
// @Description Create a new data source connection
// @Tags data-sources
// @Accept json
// @Produce json
// @Param data_source body models.DataSourceCreateRequest true "Data source configuration"
// @Success 201 {object} models.StandardResponse{data=models.DataSourceResponse}
// @Failure 400 {object} models.StandardResponse
// @Failure 401 {object} models.StandardResponse
// @Failure 500 {object} models.StandardResponse
// @Security ApiKeyAuth
// @Router /data-sources [post]
func (h *DataSourceHandler) CreateDataSource(c *fiber.Ctx) error {
	// Get user ID from context (set by auth middleware)
	userID := c.Locals("user_id").(uint)

	var req entity.DataSourceCreateRequest
	if err := c.BodyParser(&req); err != nil {
		return entity.BadRequestResponse(c, "Invalid request body", err.Error())
	}

	// Validate request
	if err := h.validator.Struct(&req); err != nil {
		return entity.BadRequestResponse(c, "Validation failed", err.Error())
	}

	// Create data source
	dataSource, err := h.dataSourceService.CreateDataSource(userID, &req)
	if err != nil {
		return entity.BadRequestResponse(c, "Failed to create data source", err.Error())
	}

	return c.Status(fiber.StatusCreated).JSON(entity.StandardResponse{
		Success: true,
		Message: "Data source created successfully",
		Data:    dataSource,
	})
}

// GetDataSources godoc
// @Summary Get user's data sources
// @Description Get all data sources for the authenticated user
// @Tags data-sources
// @Accept json
// @Produce json
// @Success 200 {object} models.StandardResponse{data=[]models.DataSourceResponse}
// @Failure 401 {object} models.StandardResponse
// @Failure 500 {object} models.StandardResponse
// @Security ApiKeyAuth
// @Router /data-sources [get]
func (h *DataSourceHandler) GetDataSources(c *fiber.Ctx) error {
	// Get user ID from context
	userID := c.Locals("user_id").(uint)

	dataSources, err := h.dataSourceService.GetUserDataSources(userID)
	if err != nil {
		return entity.InternalServerErrorResponse(c, "Failed to get data sources", err.Error())
	}

	return entity.SuccessResponse(c, "Data sources retrieved successfully", dataSources)
}

// GetDataSource godoc
// @Summary Get a specific data source
// @Description Get a data source by ID with its schemas
// @Tags data-sources
// @Accept json
// @Produce json
// @Param id path int true "Data Source ID"
// @Success 200 {object} models.StandardResponse{data=models.DataSourceResponse}
// @Failure 400 {object} models.StandardResponse
// @Failure 401 {object} models.StandardResponse
// @Failure 404 {object} models.StandardResponse
// @Failure 500 {object} models.StandardResponse
// @Security ApiKeyAuth
// @Router /data-sources/{id} [get]
func (h *DataSourceHandler) GetDataSource(c *fiber.Ctx) error {
	// Get user ID from context
	userID := c.Locals("user_id").(uint)

	// Parse data source ID
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return entity.BadRequestResponse(c, "Invalid data source ID", err.Error())
	}

	dataSource, err := h.dataSourceService.GetDataSource(uint(id), userID)
	if err != nil {
		return entity.NotFoundResponse(c, "Data source not found")
	}

	return entity.SuccessResponse(c, "Data source retrieved successfully", dataSource)
}

// UpdateDataSource godoc
// @Summary Update a data source
// @Description Update a data source configuration
// @Tags data-sources
// @Accept json
// @Produce json
// @Param id path int true "Data Source ID"
// @Param data_source body models.DataSourceUpdateRequest true "Data source update data"
// @Success 200 {object} models.StandardResponse{data=models.DataSourceResponse}
// @Failure 400 {object} models.StandardResponse
// @Failure 401 {object} models.StandardResponse
// @Failure 404 {object} models.StandardResponse
// @Failure 500 {object} models.StandardResponse
// @Security ApiKeyAuth
// @Router /data-sources/{id} [put]
func (h *DataSourceHandler) UpdateDataSource(c *fiber.Ctx) error {
	// Get user ID from context
	userID := c.Locals("user_id").(uint)

	// Parse data source ID
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return entity.BadRequestResponse(c, "Invalid data source ID", err.Error())
	}

	var req entity.DataSourceUpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return entity.BadRequestResponse(c, "Invalid request body", err.Error())
	}

	// Validate request
	if err := h.validator.Struct(&req); err != nil {
		return entity.BadRequestResponse(c, "Validation failed", err.Error())
	}

	// Update data source
	dataSource, err := h.dataSourceService.UpdateDataSource(uint(id), userID, &req)
	if err != nil {
		return entity.BadRequestResponse(c, "Failed to update data source", err.Error())
	}

	return entity.SuccessResponse(c, "Data source updated successfully", dataSource)
}

// DeleteDataSource godoc
// @Summary Delete a data source
// @Description Delete a data source and its associated schemas
// @Tags data-sources
// @Accept json
// @Produce json
// @Param id path int true "Data Source ID"
// @Success 200 {object} models.StandardResponse
// @Failure 400 {object} models.StandardResponse
// @Failure 401 {object} models.StandardResponse
// @Failure 404 {object} models.StandardResponse
// @Failure 500 {object} models.StandardResponse
// @Security ApiKeyAuth
// @Router /data-sources/{id} [delete]
func (h *DataSourceHandler) DeleteDataSource(c *fiber.Ctx) error {
	// Get user ID from context
	userID := c.Locals("user_id").(uint)

	// Parse data source ID
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return entity.BadRequestResponse(c, "Invalid data source ID", err.Error())
	}

	// Delete data source
	if err := h.dataSourceService.DeleteDataSource(uint(id), userID); err != nil {
		return entity.BadRequestResponse(c, "Failed to delete data source", err.Error())
	}

	return entity.SuccessResponse(c, "Data source deleted successfully", nil)
}

// TestConnection godoc
// @Summary Test data source connection
// @Description Test connection to a data source without creating it
// @Tags data-sources
// @Accept json
// @Produce json
// @Param connection body models.TestConnectionRequest true "Connection configuration"
// @Success 200 {object} models.StandardResponse{data=models.TestConnectionResponse}
// @Failure 400 {object} models.StandardResponse
// @Failure 401 {object} models.StandardResponse
// @Failure 500 {object} models.StandardResponse
// @Security ApiKeyAuth
// @Router /data-sources/test-connection [post]
func (h *DataSourceHandler) TestConnection(c *fiber.Ctx) error {
	var req entity.TestConnectionRequest
	if err := c.BodyParser(&req); err != nil {
		return entity.BadRequestResponse(c, "Invalid request body", err.Error())
	}

	// Validate request
	if err := h.validator.Struct(&req); err != nil {
		return entity.BadRequestResponse(c, "Validation failed", err.Error())
	}

	// Test connection
	result, err := h.dataSourceService.TestConnection(&req)
	if err != nil {
		return entity.InternalServerErrorResponse(c, "Failed to test connection", err.Error())
	}

	return entity.SuccessResponse(c, "Connection test completed", result)
}

// RefreshSchema godoc
// @Summary Refresh data source schema
// @Description Refresh the schema information for a data source
// @Tags data-sources
// @Accept json
// @Produce json
// @Param id path int true "Data Source ID"
// @Success 200 {object} models.StandardResponse{data=models.DataSourceResponse}
// @Failure 400 {object} models.StandardResponse
// @Failure 401 {object} models.StandardResponse
// @Failure 404 {object} models.StandardResponse
// @Failure 500 {object} models.StandardResponse
// @Security ApiKeyAuth
// @Router /data-sources/{id}/refresh-schema [post]
func (h *DataSourceHandler) RefreshSchema(c *fiber.Ctx) error {
	// Get user ID from context
	userID := c.Locals("user_id").(uint)

	// Parse data source ID
	id, err := strconv.ParseUint(c.Params("id"), 10, 32)
	if err != nil {
		return entity.BadRequestResponse(c, "Invalid data source ID", err.Error())
	}

	// Refresh schema
	dataSource, err := h.dataSourceService.RefreshSchema(uint(id), userID)
	if err != nil {
		return entity.BadRequestResponse(c, "Failed to refresh schema", err.Error())
	}

	return entity.SuccessResponse(c, "Schema refreshed successfully", dataSource)
}

// UploadFile godoc
// @Summary Upload a file for CSV/Excel data source
// @Description Upload a CSV or Excel file to create a file-based data source
// @Tags data-sources
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "CSV or Excel file"
// @Success 200 {object} models.StandardResponse{data=models.FileUploadResponse}
// @Failure 400 {object} models.StandardResponse
// @Failure 401 {object} models.StandardResponse
// @Failure 500 {object} models.StandardResponse
// @Security ApiKeyAuth
// @Router /data-sources/upload [post]
func (h *DataSourceHandler) UploadFile(c *fiber.Ctx) error {
	// Get uploaded file
	file, err := c.FormFile("file")
	if err != nil {
		return entity.BadRequestResponse(c, "No file uploaded", err.Error())
	}

	// Validate file type
	allowedTypes := map[string]bool{
		"text/csv":                                true,
		"application/vnd.ms-excel":                true,
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet": true,
	}

	if !allowedTypes[file.Header.Get("Content-Type")] {
		return entity.BadRequestResponse(c, "Invalid file type. Only CSV and Excel files are allowed", nil)
	}

	// Validate file size (max 50MB)
	maxSize := int64(50 * 1024 * 1024) // 50MB
	if file.Size > maxSize {
		return entity.BadRequestResponse(c, "File too large. Maximum size is 50MB", nil)
	}

	// TODO: Implement file storage logic
	// For now, return a mock response
	response := &entity.FileUploadResponse{
		FileName: file.Filename,
		FilePath: "/uploads/" + file.Filename, // This should be the actual stored path
		FileSize: file.Size,
		MimeType: file.Header.Get("Content-Type"),
	}

	return entity.SuccessResponse(c, "File uploaded successfully", response)
}