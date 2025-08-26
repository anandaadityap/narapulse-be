package models

import "github.com/gofiber/fiber/v2"

// StandardResponse represents the standard API response format
type StandardResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Error   interface{} `json:"error,omitempty"`
	Meta    *Meta       `json:"meta,omitempty"`
}

// Meta represents pagination and additional metadata
type Meta struct {
	Page       int `json:"page,omitempty"`
	Limit      int `json:"limit,omitempty"`
	Total      int `json:"total,omitempty"`
	TotalPages int `json:"total_pages,omitempty"`
}

// ErrorResponse represents error details
type ErrorResponse struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

// Response helper functions
func SuccessResponse(c *fiber.Ctx, message string, data interface{}) error {
	return c.JSON(StandardResponse{
		Success: true,
		Message: message,
		Data:    data,
	})
}

func SuccessResponseWithMeta(c *fiber.Ctx, message string, data interface{}, meta *Meta) error {
	return c.JSON(StandardResponse{
		Success: true,
		Message: message,
		Data:    data,
		Meta:    meta,
	})
}

func ErrorResponseWithStatus(c *fiber.Ctx, status int, message string, err interface{}) error {
	return c.Status(status).JSON(StandardResponse{
		Success: false,
		Message: message,
		Error:   err,
	})
}

func BadRequestResponse(c *fiber.Ctx, message string, err interface{}) error {
	return ErrorResponseWithStatus(c, fiber.StatusBadRequest, message, err)
}

func UnauthorizedResponse(c *fiber.Ctx, message string) error {
	return ErrorResponseWithStatus(c, fiber.StatusUnauthorized, message, nil)
}

func ForbiddenResponse(c *fiber.Ctx, message string) error {
	return ErrorResponseWithStatus(c, fiber.StatusForbidden, message, nil)
}

func NotFoundResponse(c *fiber.Ctx, message string) error {
	return ErrorResponseWithStatus(c, fiber.StatusNotFound, message, nil)
}

func InternalServerErrorResponse(c *fiber.Ctx, message string, err interface{}) error {
	return ErrorResponseWithStatus(c, fiber.StatusInternalServerError, message, err)
}