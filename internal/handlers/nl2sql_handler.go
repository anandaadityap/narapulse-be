package handlers

import (
	"strconv"

	models "narapulse-be/internal/models/entity"
	"narapulse-be/internal/services"

	"github.com/gofiber/fiber/v2"
)

// NL2SQLHandler handles NL2SQL related HTTP requests
type NL2SQLHandler struct {
	nl2sqlService *services.NL2SQLService
}

// NewNL2SQLHandler creates a new NL2SQL handler
func NewNL2SQLHandler(nl2sqlService *services.NL2SQLService) *NL2SQLHandler {
	return &NL2SQLHandler{
		nl2sqlService: nl2sqlService,
	}
}

// ConvertNL2SQL handles natural language to SQL conversion
func (h *NL2SQLHandler) ConvertNL2SQL(c *fiber.Ctx) error {
	// Get user ID from context
	userID := c.Locals("user_id")
	if userID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "User not authenticated",
		})
	}

	// Parse request body
	var request models.NL2SQLRequest
	if err := c.BodyParser(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid request format: " + err.Error(),
		})
	}

	// Validate required fields
	if request.NLQuery == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Natural language query is required",
		})
	}

	if request.DataSourceID == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Data source ID is required",
		})
	}

	// Convert NL to SQL
	response, err := h.nl2sqlService.ConvertNL2SQL(userID.(uint), &request)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to convert query: " + err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"data":    response,
	})
}

// ExecuteQuery handles SQL query execution
func (h *NL2SQLHandler) ExecuteQuery(c *fiber.Ctx) error {
	// Get user ID from context
	userID := c.Locals("user_id")
	if userID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "User not authenticated",
		})
	}

	// Parse request body
	var request models.QueryExecutionRequest
	if err := c.BodyParser(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid request format: " + err.Error(),
		})
	}

	// Validate required fields
	if request.QueryID == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Query ID is required",
		})
	}

	// Execute query
	response, err := h.nl2sqlService.ExecuteQuery(userID.(uint), &request)
	if err != nil {
		if err.Error() == "query not found" {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"success": false,
				"message": "Query not found",
			})
		}
		if err.Error() == "query is not executable" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"success": false,
				"message": "Query is not executable",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to execute query: " + err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"data":    response,
	})
}

// GetQueryHistory handles getting query history
func (h *NL2SQLHandler) GetQueryHistory(c *fiber.Ctx) error {
	// Get user ID from context
	userID := c.Locals("user_id")
	if userID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "User not authenticated",
		})
	}

	// Parse query parameters
	limitStr := c.Query("limit", "50")
	offsetStr := c.Query("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid limit parameter",
		})
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid offset parameter",
		})
	}

	// Limit maximum results to prevent abuse
	if limit > 1000 {
		limit = 1000
	}

	// Get query history
	history, err := h.nl2sqlService.GetQueryHistory(userID.(uint), limit, offset)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to get query history: " + err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"data":    history,
	})
}

// ValidateSQL handles SQL validation without execution
func (h *NL2SQLHandler) ValidateSQL(c *fiber.Ctx) error {
	// Get user ID from context
	userID := c.Locals("user_id")
	if userID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "User not authenticated",
		})
	}

	// Parse request body
	var request map[string]string
	if err := c.BodyParser(&request); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid request format: " + err.Error(),
		})
	}

	sql, exists := request["sql"]
	if !exists || sql == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "SQL query is required",
		})
	}

	// Create SQL validator service
	validator := services.NewSQLValidatorService()

	// Validate SQL
	result, err := validator.ValidateSQL(sql)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"message": "Failed to validate SQL: " + err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"data":    result,
	})
}

// GetQueryDetails handles getting detailed information about a specific query
func (h *NL2SQLHandler) GetQueryDetails(c *fiber.Ctx) error {
	// Get user ID from context
	userID := c.Locals("user_id")
	if userID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "User not authenticated",
		})
	}

	// Parse query ID from path
	queryIDStr := c.Params("id")
	queryIDUint, err := strconv.ParseUint(queryIDStr, 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid query ID",
		})
	}

	// Get query details
	query, err := h.nl2sqlService.GetQueryDetails(userID.(uint), uint(queryIDUint))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"message": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": "Query details retrieved successfully",
		"data":    query,
	})
}

// DeleteQuery handles deleting a query from history
func (h *NL2SQLHandler) DeleteQuery(c *fiber.Ctx) error {
	// Get user ID from context
	userID := c.Locals("user_id")
	if userID == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"message": "User not authenticated",
		})
	}

	// Parse query ID from path
	queryIDStr := c.Params("id")
	queryIDUint, err := strconv.ParseUint(queryIDStr, 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Invalid query ID",
		})
	}

	// Delete query
	err = h.nl2sqlService.DeleteQuery(userID.(uint), uint(queryIDUint))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"message": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success": true,
		"message": "Query deleted successfully",
	})
}