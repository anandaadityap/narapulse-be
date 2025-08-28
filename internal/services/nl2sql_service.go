package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	models "narapulse-be/internal/models/entity"
	"gorm.io/gorm"
)

// ConnectorService placeholder - will be implemented later
type ConnectorService struct {
	// TODO: Implement connector service
}

// AIService placeholder - will be implemented later  
type AIService struct {
	// TODO: Implement AI service
}

// NL2SQLService handles natural language to SQL conversion
type NL2SQLService struct {
	db               *gorm.DB
	sqlValidator     *SQLValidatorService
	connectorService *ConnectorService
	aiService        *AIService // Will be implemented later
	ragService       *RAGService
}

// NewNL2SQLService creates a new NL2SQL service
func NewNL2SQLService(db *gorm.DB, ragService *RAGService) *NL2SQLService {
	return &NL2SQLService{
		db:               db,
		sqlValidator:     NewSQLValidatorService(),
		connectorService: &ConnectorService{}, // Placeholder
		ragService:       ragService,
		// aiService will be initialized when AI integration is ready
	}
}

// ConvertNL2SQL converts natural language query to SQL
func (s *NL2SQLService) ConvertNL2SQL(userID uint, request *models.NL2SQLRequest) (*models.NL2SQLResponse, error) {
	// Validate data source access
	dataSource, err := s.validateDataSourceAccess(userID, request.DataSourceID)
	if err != nil {
		return nil, fmt.Errorf("data source validation failed: %v", err)
	}

	// Create query record
	query := &models.NL2SQLQuery{
		UserID:       userID,
		DataSourceID: request.DataSourceID,
		NLQuery:      request.NLQuery,
		Status:       models.QueryStatusPending,
		Type:         request.Type,
	}

	// Set default type if not provided
	if query.Type == "" {
		query.Type = models.QueryTypeAnalytics
	}

	// Store context
	if request.Context != nil {
		contextJSON, _ := json.Marshal(request.Context)
		query.Context = models.JSON(contextJSON)
	}

	// Save query to database
	if err := s.db.Create(query).Error; err != nil {
		return nil, fmt.Errorf("failed to create query record: %v", err)
	}

	// Build enhanced context using RAG system
	enhancedContext, err := s.buildEnhancedContext(dataSource, request.NLQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to build enhanced context: %v", err)
	}

	// Generate SQL using enhanced context
	generatedSQL, err := s.generateSQLWithRAG(request.NLQuery, enhancedContext)
	if err != nil {
		query.MarkFailed(err.Error())
		s.db.Save(query)
		return nil, fmt.Errorf("SQL generation failed: %v", err)
	}

	// Validate generated SQL
	validationResult, err := s.sqlValidator.ValidateSQL(generatedSQL)
	if err != nil {
		query.MarkFailed(fmt.Sprintf("SQL validation failed: %v", err))
		s.db.Save(query)
		return nil, fmt.Errorf("SQL validation failed: %v", err)
	}

	// Enforce LIMIT if not present
	if !validationResult.HasLimit {
		generatedSQL, err = s.sqlValidator.EnforceLimit(generatedSQL, 1000)
		if err != nil {
			query.MarkFailed(fmt.Sprintf("Failed to enforce LIMIT: %v", err))
			s.db.Save(query)
			return nil, fmt.Errorf("failed to enforce LIMIT: %v", err)
		}
		// Re-validate after adding LIMIT
		validationResult, _ = s.sqlValidator.ValidateSQL(generatedSQL)
	}

	// Set the generated SQL to the query object
	query.GeneratedSQL = generatedSQL

	// Check if query is safe to execute
	canExecute := s.sqlValidator.IsQuerySafe(validationResult)
	if canExecute {
		query.MarkCompleted(0, 0) // Will be updated when query is actually executed
	} else {
		query.MarkFailed("Query failed safety validation")
	}

	// Store metadata
	metadata := map[string]interface{}{
		"validation_result": validationResult,
		"enhanced_context":  enhancedContext,
		"generated_at":      time.Now(),
	}
	metadataJSON, _ := json.Marshal(metadata)
	query.Metadata = models.JSON(metadataJSON)

	// Save updated query
	if err := s.db.Save(query).Error; err != nil {
		return nil, fmt.Errorf("failed to update query record: %v", err)
	}

	// Prepare response
	response := &models.NL2SQLResponse{
		QueryID:       query.ID,
		GeneratedSQL:  generatedSQL,
		Validation:    *validationResult,
		EstimatedCost: validationResult.EstimatedCost,
		SafetyScore:   validationResult.SafetyScore,
		CanExecute:    canExecute,
		Messages:      []string{},
	}

	// Add messages based on validation
	if len(validationResult.Violations) > 0 {
		response.Messages = append(response.Messages, "Query has validation violations")
	}
	if len(validationResult.Warnings) > 0 {
		response.Messages = append(response.Messages, "Query has warnings")
	}
	if canExecute {
		response.Messages = append(response.Messages, "Query is ready for execution")
	}

	return response, nil
}

// ExecuteQuery executes a validated NL2SQL query
func (s *NL2SQLService) ExecuteQuery(userID uint, request *models.QueryExecutionRequest) (*models.QueryExecutionResponse, error) {
	// Get query record
	var query models.NL2SQLQuery
	if err := s.db.Where("id = ? AND user_id = ?", request.QueryID, userID).First(&query).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("query not found")
		}
		return nil, fmt.Errorf("failed to get query: %v", err)
	}

	// Check if query is executable
	if !query.IsExecutable() {
		return nil, errors.New("query is not executable")
	}

	// Get data source
	var dataSource models.DataSource
	if err := s.db.First(&dataSource, query.DataSourceID).Error; err != nil {
		return nil, fmt.Errorf("failed to get data source: %v", err)
	}

	// Set default limit if not provided
	limit := request.Limit
	if limit <= 0 {
		limit = 1000
	}

	// Execute query using connector service
	startTime := time.Now()
	result, err := s.executeQueryOnDataSource(&dataSource, query.GeneratedSQL, limit)
	executionTime := time.Since(startTime).Milliseconds()

	if err != nil {
		// Update query with error
		query.Status = models.QueryStatusFailed
		query.ErrorMsg = err.Error()
		query.ExecutionTime = executionTime
		s.db.Save(&query)

		return &models.QueryExecutionResponse{
			QueryID:       query.ID,
			Status:        models.QueryStatusFailed,
			Message:       err.Error(),
			ExecutionTime: executionTime,
		}, nil
	}

	// Update query with success
	query.ExecutionTime = executionTime
	query.RowsReturned = int64(len(result.Data))
	s.db.Save(&query)

	// Store query result
	queryResult := &models.QueryResult{
		QueryID:  query.ID,
		RowCount: int64(len(result.Data)),
	}

	// Store columns
	columnsJSON, _ := json.Marshal(result.Columns)
	queryResult.Columns = models.JSON(columnsJSON)

	// Store data
	dataJSON, _ := json.Marshal(result.Data)
	queryResult.Data = models.JSON(dataJSON)

	// Save result
	s.db.Create(queryResult)

	return &models.QueryExecutionResponse{
		QueryID:       query.ID,
		Columns:       result.Columns,
		Data:          result.Data,
		RowCount:      int64(len(result.Data)),
		ExecutionTime: executionTime,
		Status:        models.QueryStatusCompleted,
		Message:       "Query executed successfully",
	}, nil
}

// GetQueryDetails gets details of a specific query
func (s *NL2SQLService) GetQueryDetails(userID uint, queryID uint) (*models.NL2SQLQuery, error) {
	var query models.NL2SQLQuery
	if err := s.db.Where("id = ? AND user_id = ?", queryID, userID).First(&query).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("query not found")
		}
		return nil, fmt.Errorf("failed to get query: %v", err)
	}
	return &query, nil
}

// DeleteQuery deletes a query from history
func (s *NL2SQLService) DeleteQuery(userID uint, queryID uint) error {
	// First check if query exists and belongs to user
	var query models.NL2SQLQuery
	if err := s.db.Where("id = ? AND user_id = ?", queryID, userID).First(&query).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("query not found")
		}
		return fmt.Errorf("failed to get query: %v", err)
	}

	// Delete associated query results first
	if err := s.db.Where("query_id = ?", queryID).Delete(&models.QueryResult{}).Error; err != nil {
		return fmt.Errorf("failed to delete query results: %v", err)
	}

	// Delete the query
	if err := s.db.Delete(&query).Error; err != nil {
		return fmt.Errorf("failed to delete query: %v", err)
	}

	return nil
}

// GetQueryHistory gets query history for a user
func (s *NL2SQLService) GetQueryHistory(userID uint, limit int, offset int) ([]*models.QueryHistoryResponse, error) {
	var queries []models.NL2SQLQuery

	query := s.db.Where("user_id = ?", userID).
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	if err := query.Find(&queries).Error; err != nil {
		return nil, fmt.Errorf("failed to get query history: %v", err)
	}

	var history []*models.QueryHistoryResponse
	for _, q := range queries {
		// Get data source info if needed
		var dataSource models.DataSource
		if q.DataSourceID > 0 {
			s.db.First(&dataSource, q.DataSourceID)
		}
		
		response := q.ToHistoryResponse()
		if q.DataSourceID > 0 {
			response.DataSourceName = dataSource.Name
		}
		history = append(history, response)
	}

	return history, nil
}

// validateDataSourceAccess validates user access to data source
func (s *NL2SQLService) validateDataSourceAccess(userID uint, dataSourceID uint) (*models.DataSource, error) {
	var dataSource models.DataSource
	if err := s.db.Where("id = ? AND user_id = ?", dataSourceID, userID).First(&dataSource).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("data source not found or access denied")
		}
		return nil, fmt.Errorf("failed to validate data source access: %v", err)
	}

	if dataSource.Status != models.ConnectionStatusActive {
		return nil, errors.New("data source is not active")
	}

	return &dataSource, nil
}

// buildSchemaContext builds schema context for AI prompt
func (s *NL2SQLService) buildSchemaContext(dataSource *models.DataSource) (map[string]interface{}, error) {
	// Get schemas for the data source
	var schemas []models.Schema
	if err := s.db.Where("data_source_id = ? AND is_active = ?", dataSource.ID, true).Find(&schemas).Error; err != nil {
		return nil, fmt.Errorf("failed to get schemas: %v", err)
	}

	context := map[string]interface{}{
		"data_source_type": dataSource.Type,
		"data_source_name": dataSource.Name,
		"schemas":          []map[string]interface{}{},
	}

	for _, schema := range schemas {
		// Parse columns
		var columns []models.Column
		if err := json.Unmarshal(schema.Columns, &columns); err == nil {
			schemaInfo := map[string]interface{}{
				"name":         schema.Name,
				"display_name": schema.DisplayName,
				"description":  schema.Description,
				"columns":      columns,
				"row_count":    schema.RowCount,
			}
			context["schemas"] = append(context["schemas"].([]map[string]interface{}), schemaInfo)
		}
	}

	return context, nil
}

// buildEnhancedContext builds context using RAG system for better NL2SQL conversion
func (s *NL2SQLService) buildEnhancedContext(dataSource *models.DataSource, nlQuery string) (map[string]interface{}, error) {
	// Get basic schema context
	schemaContext, err := s.buildSchemaContext(dataSource)
	if err != nil {
		return nil, fmt.Errorf("failed to build schema context: %v", err)
	}

	// Use RAG service to build enhanced context
	ragContext, err := s.ragService.BuildNL2SQLContext(context.Background(), nlQuery, dataSource.ID)
	if err != nil {
		// If RAG fails, fallback to basic schema context
		return schemaContext, nil
	}

	// Merge schema context with RAG context
	enhancedContext := map[string]interface{}{
		"data_source_type":   dataSource.Type,
		"data_source_name":   dataSource.Name,
		"schemas":            schemaContext["schemas"],
		"similar_schemas":    ragContext["similar_schemas"],
		"relevant_kpis":      ragContext["relevant_kpis"],
		"business_glossary":  ragContext["business_glossary"],
		"query_examples":     ragContext["query_examples"],
		"enhanced_prompt":    ragContext["enhanced_prompt"],
	}

	return enhancedContext, nil
}

// generateSQL generates SQL from natural language (mock implementation)
func (s *NL2SQLService) generateSQL(nlQuery string, schemaContext map[string]interface{}) (string, error) {
	// This is a mock implementation
	// In the real implementation, this will call the AI service
	
	// Simple pattern matching for demo purposes
	if contains(nlQuery, []string{"sales", "revenue", "total"}) {
		return "SELECT SUM(amount) as total_sales FROM sales WHERE date >= '2024-01-01' LIMIT 1000", nil
	}
	
	if contains(nlQuery, []string{"count", "number", "how many"}) {
		return "SELECT COUNT(*) as total_count FROM sales LIMIT 1000", nil
	}
	
	if contains(nlQuery, []string{"average", "avg", "mean"}) {
		return "SELECT AVG(amount) as average_amount FROM sales LIMIT 1000", nil
	}

	// Default fallback
	return "SELECT * FROM sales LIMIT 100", nil
}

// generateSQLWithRAG generates SQL using enhanced context from RAG system
func (s *NL2SQLService) generateSQLWithRAG(nlQuery string, enhancedContext map[string]interface{}) (string, error) {
	// Extract enhanced prompt if available
	enhancedPrompt, hasPrompt := enhancedContext["enhanced_prompt"].(string)
	
	// If we have an enhanced prompt from RAG, use it for better SQL generation
	if hasPrompt && enhancedPrompt != "" {
		// TODO: When AI service is implemented, use enhanced prompt
		// For now, use enhanced context for better pattern matching
		return s.generateSQLWithEnhancedPatterns(nlQuery, enhancedContext)
	}
	
	// Fallback to basic generation with schema context
	schemaContext := map[string]interface{}{
		"data_source_type": enhancedContext["data_source_type"],
		"data_source_name": enhancedContext["data_source_name"],
		"schemas":          enhancedContext["schemas"],
	}
	return s.generateSQL(nlQuery, schemaContext)
}

// generateSQLWithEnhancedPatterns uses enhanced context for better pattern matching
func (s *NL2SQLService) generateSQLWithEnhancedPatterns(nlQuery string, enhancedContext map[string]interface{}) (string, error) {
	// Get relevant KPIs and business terms
	relevantKPIs, _ := enhancedContext["relevant_kpis"].([]models.KPIDefinition)
	businessGlossary, _ := enhancedContext["business_glossary"].([]models.BusinessGlossary)
	
	// Enhanced pattern matching using KPIs and business terms
	for _, kpi := range relevantKPIs {
		if contains(strings.ToLower(nlQuery), []string{strings.ToLower(kpi.Name)}) {
			// Use KPI definition to generate more accurate SQL
			if kpi.Formula != "" {
				return kpi.Formula + " LIMIT 1000", nil
			}
		}
	}
	
	// Check business glossary for domain-specific terms
	for _, term := range businessGlossary {
		if contains(strings.ToLower(nlQuery), []string{strings.ToLower(term.Term)}) {
			// Use business context for better SQL generation
			// This is a simplified implementation
			if contains(term.Definition, []string{"sum", "total", "aggregate"}) {
				return fmt.Sprintf("SELECT SUM(%s) as total FROM %s LIMIT 1000", term.Term, "main_table"), nil
			}
		}
	}
	
	// Fallback to basic patterns
	if contains(nlQuery, []string{"sales", "revenue", "total"}) {
		return "SELECT SUM(amount) as total_sales FROM sales WHERE date >= '2024-01-01' LIMIT 1000", nil
	}
	
	if contains(nlQuery, []string{"count", "number", "how many"}) {
		return "SELECT COUNT(*) as total_count FROM sales LIMIT 1000", nil
	}
	
	if contains(nlQuery, []string{"average", "avg", "mean"}) {
		return "SELECT AVG(amount) as average_amount FROM sales LIMIT 1000", nil
	}

	// Default fallback
	return "SELECT * FROM sales LIMIT 100", nil
}

// executeQueryOnDataSource executes query on the specified data source
func (s *NL2SQLService) executeQueryOnDataSource(dataSource *models.DataSource, sql string, limit int) (*QueryResult, error) {
	// Use connector service to execute query
	switch dataSource.Type {
	case models.DataSourceTypePostgreSQL:
		return s.executePostgreSQLQuery(dataSource, sql, limit)
	case models.DataSourceTypeBigQuery:
		return s.executeBigQueryQuery(dataSource, sql, limit)
	case models.DataSourceTypeCSV, models.DataSourceTypeExcel:
		return s.executeFileQuery(dataSource, sql, limit)
	default:
		return nil, fmt.Errorf("unsupported data source type: %s", dataSource.Type)
	}
}

// QueryResult represents the result of a query execution
type QueryResult struct {
	Columns []models.Column            `json:"columns"`
	Data    []map[string]interface{}   `json:"data"`
}

// executePostgreSQLQuery executes query on PostgreSQL
func (s *NL2SQLService) executePostgreSQLQuery(dataSource *models.DataSource, sql string, limit int) (*QueryResult, error) {
	// Mock implementation - in real scenario, use the PostgreSQL connector
	return &QueryResult{
		Columns: []models.Column{
			{Name: "id", Type: "integer"},
			{Name: "amount", Type: "decimal"},
			{Name: "date", Type: "date"},
		},
		Data: []map[string]interface{}{
			{"id": 1, "amount": 100.50, "date": "2024-01-15"},
			{"id": 2, "amount": 250.75, "date": "2024-01-16"},
		},
	}, nil
}

// executeBigQueryQuery executes query on BigQuery
func (s *NL2SQLService) executeBigQueryQuery(dataSource *models.DataSource, sql string, limit int) (*QueryResult, error) {
	// Mock implementation - in real scenario, use the BigQuery connector
	return &QueryResult{
		Columns: []models.Column{
			{Name: "total_sales", Type: "decimal"},
		},
		Data: []map[string]interface{}{
			{"total_sales": 15750.25},
		},
	}, nil
}

// executeFileQuery executes query on CSV/Excel files
func (s *NL2SQLService) executeFileQuery(dataSource *models.DataSource, sql string, limit int) (*QueryResult, error) {
	// Mock implementation - in real scenario, use DuckDB or similar for SQL on files
	return &QueryResult{
		Columns: []models.Column{
			{Name: "name", Type: "string"},
			{Name: "value", Type: "decimal"},
		},
		Data: []map[string]interface{}{
			{"name": "Product A", "value": 100.0},
			{"name": "Product B", "value": 200.0},
		},
	}, nil
}

// Helper function to check if string contains any of the keywords
func contains(text string, keywords []string) bool {
	text = strings.ToLower(text)
	for _, keyword := range keywords {
		if strings.Contains(text, strings.ToLower(keyword)) {
			return true
		}
	}
	return false
}