package handlers

import (
	"strconv"

	models "narapulse-be/internal/models/entity"
	"narapulse-be/internal/services"

	"github.com/gofiber/fiber/v2"
)

// RAGHandler handles RAG-related HTTP requests
type RAGHandler struct {
	ragService       *services.RAGService
	embeddingService *services.EmbeddingService
}

// NewRAGHandler creates a new RAG handler
func NewRAGHandler(ragService *services.RAGService, embeddingService *services.EmbeddingService) *RAGHandler {
	return &RAGHandler{
		ragService:       ragService,
		embeddingService: embeddingService,
	}
}

// SearchSimilar handles similarity search requests
// @Summary Search similar schema elements
// @Description Search for similar schema elements using vector similarity
// @Tags RAG
// @Accept json
// @Produce json
// @Param request body models.RAGSearchRequest true "Search request"
// @Success 200 {object} models.RAGSearchResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/rag/search [post]
func (h *RAGHandler) SearchSimilar(c *fiber.Ctx) error {
	var req models.RAGSearchRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{
			Code:    "INVALID_REQUEST_BODY",
			Message: err.Error(),
		})
	}

	// Validate request
	if req.Query == "" {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{
			Code:    "QUERY_REQUIRED",
			Message: "Please provide a search query",
		})
	}

	// Set defaults
	if req.TopK <= 0 {
		req.TopK = 5
	}
	if req.TopK > 20 {
		req.TopK = 20
	}

	// Perform search
	result, err := h.ragService.SearchSimilar(c.Context(), req.Query, req.DataSourceID, req.TopK, req.ElementTypes)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Code:    "SEARCH_FAILED",
			Message: err.Error(),
		})
	}

	return c.JSON(result)
}

// BuildNL2SQLContext builds context for NL2SQL conversion
// @Summary Build NL2SQL context
// @Description Build context information for natural language to SQL conversion
// @Tags RAG
// @Accept json
// @Produce json
// @Param data_source_id query int true "Data source ID"
// @Param query query string true "Natural language query"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/rag/nl2sql-context [get]
func (h *RAGHandler) BuildNL2SQLContext(c *fiber.Ctx) error {
	query := c.Query("query")
	if query == "" {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{
			Code:    "QUERY_PARAMETER_REQUIRED",
			Message: "Please provide a query parameter",
		})
	}

	dataSourceIDStr := c.Query("data_source_id")
	if dataSourceIDStr == "" {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{
			Code:    "DATA_SOURCE_ID_REQUIRED",
			Message: "Please provide a data_source_id parameter",
		})
	}

	dataSourceID, err := strconv.ParseUint(dataSourceIDStr, 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{
			Code:    "INVALID_DATA_SOURCE_ID",
			Message: "Data source ID must be a valid number",
		})
	}

	context, err := h.ragService.BuildNL2SQLContext(c.Context(), query, uint(dataSourceID))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Code:    "CONTEXT_BUILD_FAILED",
			Message: err.Error(),
		})
	}

	return c.JSON(context)
}

// GetAvailableSchemas returns available schemas for a data source
// @Summary Get available schemas
// @Description Get list of available schemas for a data source
// @Tags RAG
// @Produce json
// @Param data_source_id path int true "Data source ID"
// @Success 200 {array} map[string]interface{}
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/rag/schemas/{data_source_id} [get]
func (h *RAGHandler) GetAvailableSchemas(c *fiber.Ctx) error {
	dataSourceIDStr := c.Params("data_source_id")
	dataSourceID, err := strconv.ParseUint(dataSourceIDStr, 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{
			Code:    "INVALID_DATA_SOURCE_ID",
			Message: "Data source ID must be a valid number",
		})
	}

	schemas, err := h.ragService.GetAvailableSchemas(uint(dataSourceID))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Code:    "GET_SCHEMAS_FAILED",
			Message: err.Error(),
		})
	}

	return c.JSON(schemas)
}

// SyncSchemaEmbeddings synchronizes embeddings for a data source
// @Summary Sync schema embeddings
// @Description Synchronize embeddings for all schemas in a data source
// @Tags RAG
// @Accept json
// @Produce json
// @Param data_source_id path int true "Data source ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/rag/sync/{data_source_id} [post]
func (h *RAGHandler) SyncSchemaEmbeddings(c *fiber.Ctx) error {
	dataSourceIDStr := c.Params("data_source_id")
	dataSourceID, err := strconv.ParseUint(dataSourceIDStr, 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{
			Code:    "INVALID_DATA_SOURCE_ID",
			Message: "Data source ID must be a valid number",
		})
	}

	err = h.ragService.SyncSchemaEmbeddings(c.Context(), uint(dataSourceID))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Code:    "SYNC_EMBEDDINGS_FAILED",
			Message: err.Error(),
		})
	}

	return c.JSON(map[string]string{
		"message": "Schema embeddings synchronized successfully",
		"status":  "success",
	})
}

// EmbedKPIDefinition embeds a KPI definition
// @Summary Embed KPI definition
// @Description Create vector embedding for a KPI definition
// @Tags RAG
// @Accept json
// @Produce json
// @Param request body models.KPIDefinitionRequest true "KPI definition request"
// @Success 201 {object} models.KPIDefinitionResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/rag/kpi [post]
func (h *RAGHandler) EmbedKPIDefinition(c *fiber.Ctx) error {
	var req models.KPIDefinitionRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{
			Code:    "INVALID_REQUEST_BODY",
			Message: err.Error(),
		})
	}

	// Validate request
	if req.Name == "" || req.Description == "" {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{
			Code:    "NAME_AND_DESCRIPTION_REQUIRED",
			Message: "Please provide both name and description for the KPI",
		})
	}

	// Create KPI definition from request
	kpi := &models.KPIDefinition{
		Name:        req.Name,
		DisplayName: req.DisplayName,
		Description: req.Description,
		Formula:     req.Formula,
		Category:    req.Category,
		Unit:        req.Unit,
		Grain:       req.Grain,
		// Convert filters and tags to JSON
	}

	err := h.embeddingService.EmbedKPIDefinition(c.Context(), kpi)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Code:    "EMBED_KPI_FAILED",
			Message: err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(map[string]interface{}{
		"message": "KPI definition embedded successfully",
		"id":      kpi.ID,
		"name":    kpi.Name,
	})
}

// EmbedGlossaryTerm embeds a business glossary term
// @Summary Embed glossary term
// @Description Create vector embedding for a business glossary term
// @Tags RAG
// @Accept json
// @Produce json
// @Param request body models.BusinessGlossaryRequest true "Glossary term request"
// @Success 201 {object} models.BusinessGlossaryResponse
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/rag/glossary [post]
func (h *RAGHandler) EmbedGlossaryTerm(c *fiber.Ctx) error {
	var req models.BusinessGlossaryRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{
			Code:    "INVALID_REQUEST_BODY",
			Message: err.Error(),
		})
	}

	// Validate request
	if req.Term == "" || req.Definition == "" {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{
			Code:    "TERM_AND_DEFINITION_REQUIRED",
			Message: "Please provide both term and definition",
		})
	}

	// Create glossary term from request
	glossary := &models.BusinessGlossary{
		Term:       req.Term,
		Definition: req.Definition,
		Category:   req.Category,
		Domain:     req.Domain,
		// Convert arrays to JSON
	}

	err := h.embeddingService.EmbedGlossaryTerm(c.Context(), glossary)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Code:    "EMBED_GLOSSARY_FAILED",
			Message: err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(map[string]interface{}{
		"message": "Glossary term embedded successfully",
		"id":      glossary.ID,
		"term":    glossary.Term,
	})
}

// GetEnhancedNL2SQLPrompt builds enhanced prompt for NL2SQL
// @Summary Get enhanced NL2SQL prompt
// @Description Build an enhanced prompt with context for NL2SQL conversion
// @Tags RAG
// @Produce json
// @Param data_source_id query int true "Data source ID"
// @Param query query string true "Natural language query"
// @Success 200 {object} map[string]string
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/rag/nl2sql-prompt [get]
func (h *RAGHandler) GetEnhancedNL2SQLPrompt(c *fiber.Ctx) error {
	query := c.Query("query")
	if query == "" {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{
			Code:    "QUERY_PARAMETER_REQUIRED",
			Message: "Please provide a query parameter",
		})
	}

	dataSourceIDStr := c.Query("data_source_id")
	if dataSourceIDStr == "" {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{
			Code:    "DATA_SOURCE_ID_REQUIRED",
			Message: "Please provide a data_source_id parameter",
		})
	}

	dataSourceID, err := strconv.ParseUint(dataSourceIDStr, 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{
			Code:    "INVALID_DATA_SOURCE_ID",
			Message: "Data source ID must be a valid number",
		})
	}

	prompt, err := h.ragService.BuildEnhancedNL2SQLPrompt(c.Context(), query, uint(dataSourceID))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Code:    "BUILD_PROMPT_FAILED",
			Message: err.Error(),
		})
	}

	return c.JSON(map[string]string{
		"prompt": prompt,
		"query":  query,
	})
}

// DeleteEmbeddings deletes embeddings for a data source
// @Summary Delete embeddings
// @Description Delete all embeddings for a specific data source
// @Tags RAG
// @Produce json
// @Param data_source_id path int true "Data source ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} models.ErrorResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /api/v1/rag/embeddings/{data_source_id} [delete]
func (h *RAGHandler) DeleteEmbeddings(c *fiber.Ctx) error {
	dataSourceIDStr := c.Params("data_source_id")
	dataSourceID, err := strconv.ParseUint(dataSourceIDStr, 10, 32)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{
			Code:    "INVALID_DATA_SOURCE_ID",
			Message: "Data source ID must be a valid number",
		})
	}

	// Get optional schema_id parameter
	schemaIDStr := c.Query("schema_id")
	var schemaID uint = 0
	if schemaIDStr != "" {
		parsedSchemaID, err := strconv.ParseUint(schemaIDStr, 10, 32)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(models.ErrorResponse{
				Code:    "INVALID_SCHEMA_ID",
				Message: "Schema ID must be a valid number",
			})
		}
		schemaID = uint(parsedSchemaID)
	}

	err = h.embeddingService.DeleteEmbeddings(uint(dataSourceID), schemaID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(models.ErrorResponse{
			Code:    "DELETE_EMBEDDINGS_FAILED",
			Message: err.Error(),
		})
	}

	message := "All embeddings deleted successfully"
	if schemaID > 0 {
		message = "Schema embeddings deleted successfully"
	}

	return c.JSON(map[string]string{
		"message": message,
		"status":  "success",
	})
}