package routes

import (
	"narapulse-be/internal/handlers"
	"narapulse-be/internal/middleware"

	"github.com/gofiber/fiber/v2"
)

// SetupRAGRoutes sets up RAG-related routes
func SetupRAGRoutes(app *fiber.App, ragHandler *handlers.RAGHandler) {
	// Create RAG route group
	rag := app.Group("/api/v1/rag")

	// Apply authentication middleware to all RAG routes
	rag.Use(middleware.AuthMiddleware())

	// Search and retrieval endpoints
	rag.Post("/search", ragHandler.SearchSimilar)
	rag.Get("/nl2sql-context", ragHandler.BuildNL2SQLContext)
	rag.Get("/nl2sql-prompt", ragHandler.GetEnhancedNL2SQLPrompt)

	// Schema management endpoints
	rag.Get("/schemas/:data_source_id", ragHandler.GetAvailableSchemas)
	rag.Post("/sync/:data_source_id", ragHandler.SyncSchemaEmbeddings)

	// KPI and Glossary management endpoints
	rag.Post("/kpi", ragHandler.EmbedKPIDefinition)
	rag.Post("/glossary", ragHandler.EmbedGlossaryTerm)

	// Embedding management endpoints
	rag.Delete("/embeddings/:data_source_id", ragHandler.DeleteEmbeddings)
}