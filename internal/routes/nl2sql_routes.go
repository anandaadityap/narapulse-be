package routes

import (
	"narapulse-be/internal/handlers"
	"narapulse-be/internal/middleware"

	"github.com/gofiber/fiber/v2"
)

// SetupNL2SQLRoutes sets up NL2SQL related routes
func SetupNL2SQLRoutes(router fiber.Router, nl2sqlHandler *handlers.NL2SQLHandler) {
	// NL2SQL routes group
	nl2sql := router.Group("/nl2sql")
	
	// Apply authentication middleware to all NL2SQL routes
	nl2sql.Use(middleware.AuthMiddleware())

	// Convert natural language to SQL
	nl2sql.Post("/convert", nl2sqlHandler.ConvertNL2SQL)

	// Execute SQL query
	nl2sql.Post("/execute", nl2sqlHandler.ExecuteQuery)

	// Get query history
	nl2sql.Get("/history", nl2sqlHandler.GetQueryHistory)

	// Validate SQL without execution
	nl2sql.Post("/validate", nl2sqlHandler.ValidateSQL)

	// Query management routes
	queries := nl2sql.Group("/queries")
	
	// Get specific query details
	queries.Get("/:id", nl2sqlHandler.GetQueryDetails)

	// Delete query from history
	queries.Delete("/:id", nl2sqlHandler.DeleteQuery)
}