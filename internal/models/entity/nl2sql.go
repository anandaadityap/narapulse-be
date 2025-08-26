package models

import (
	"time"

	"gorm.io/gorm"
)

// QueryStatus represents the status of a NL2SQL query
type QueryStatus string

const (
	QueryStatusPending   QueryStatus = "pending"
	QueryStatusRunning   QueryStatus = "running"
	QueryStatusCompleted QueryStatus = "completed"
	QueryStatusFailed    QueryStatus = "failed"
)

// QueryType represents the type of query
type QueryType string

const (
	QueryTypeAnalytics QueryType = "analytics"
	QueryTypeReport    QueryType = "report"
	QueryTypeExplore   QueryType = "explore"
)

// NL2SQLQuery represents a natural language to SQL query
type NL2SQLQuery struct {
	ID             uint           `json:"id" gorm:"primaryKey"`
	UserID         uint           `json:"user_id" gorm:"not null;index"`
	DataSourceID   uint           `json:"data_source_id" gorm:"not null;index"`
	NLQuery        string         `json:"nl_query" gorm:"type:text;not null"`
	GeneratedSQL   string         `json:"generated_sql" gorm:"type:text"`
	Status         QueryStatus    `json:"status" gorm:"default:pending"`
	Type           QueryType      `json:"type" gorm:"default:analytics"`
	Context        JSON           `json:"context" gorm:"type:jsonb"`
	Metadata       JSON           `json:"metadata" gorm:"type:jsonb"`
	ErrorMsg       string         `json:"error_msg" gorm:"type:text"`
	ExecutionTime  int64          `json:"execution_time"` // in milliseconds
	RowsReturned   int64          `json:"rows_returned"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `json:"-" gorm:"index"`

	// Relations - removed User and DataSource to avoid foreign key constraint issues
	// User       User       `json:"user,omitempty" gorm:"foreignKey:UserID"`
	// DataSource DataSource `json:"data_source,omitempty" gorm:"foreignKey:DataSourceID"`
	Results    []QueryResult `json:"results,omitempty" gorm:"foreignKey:QueryID"`
}

// QueryResult represents the result of a query execution
type QueryResult struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	QueryID   uint           `json:"query_id" gorm:"not null;index"`
	Columns   JSON           `json:"columns" gorm:"type:jsonb"` // Column definitions
	Data      JSON           `json:"data" gorm:"type:jsonb"` // Query result data
	RowCount  int64          `json:"row_count"`
	CreatedAt time.Time      `json:"created_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	// Relations
	Query NL2SQLQuery `json:"query" gorm:"foreignKey:QueryID"`
}

// SQLValidationResult represents the result of SQL validation
type SQLValidationResult struct {
	IsValid      bool     `json:"is_valid"`
	IsReadOnly   bool     `json:"is_read_only"`
	HasLimit     bool     `json:"has_limit"`
	EstimatedCost float64 `json:"estimated_cost"`
	SafetyScore  float64  `json:"safety_score"`
	Violations   []string `json:"violations"`
	Warnings     []string `json:"warnings"`
}

// QueryContext represents the context for NL2SQL generation
type QueryContext struct {
	AllowedTables []string               `json:"allowed_tables"`
	KPIIDs        []string               `json:"kpi_ids"`
	Filters       map[string]interface{} `json:"filters"`
	TimeRange     *TimeRange             `json:"time_range,omitempty"`
	MaxRows       int                    `json:"max_rows"`
}

// TimeRange represents a time range filter
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// Request/Response DTOs

// NL2SQLRequest represents a request to convert natural language to SQL
type NL2SQLRequest struct {
	NLQuery      string                 `json:"nl_query" validate:"required,min=1,max=1000"`
	DataSourceID uint                   `json:"data_source_id" validate:"required"`
	Context      map[string]interface{} `json:"context,omitempty"`
	Type         QueryType              `json:"type,omitempty"`
}

// NL2SQLResponse represents the response from NL2SQL conversion
type NL2SQLResponse struct {
	QueryID       uint                 `json:"query_id"`
	GeneratedSQL  string               `json:"generated_sql"`
	Validation    SQLValidationResult  `json:"validation"`
	EstimatedCost float64              `json:"estimated_cost"`
	SafetyScore   float64              `json:"safety_score"`
	Messages      []string             `json:"messages"`
	CanExecute    bool                 `json:"can_execute"`
}

// QueryExecutionRequest represents a request to execute a query
type QueryExecutionRequest struct {
	QueryID uint `json:"query_id" validate:"required"`
	Limit   int  `json:"limit,omitempty" validate:"min=1,max=10000"`
}

// QueryExecutionResponse represents the response from query execution
type QueryExecutionResponse struct {
	QueryID       uint                     `json:"query_id"`
	Columns       []Column                 `json:"columns"`
	Data          []map[string]interface{} `json:"data"`
	RowCount      int64                    `json:"row_count"`
	ExecutionTime int64                    `json:"execution_time"`
	Status        QueryStatus              `json:"status"`
	Message       string                   `json:"message,omitempty"`
}

// QueryHistoryResponse represents a query in the history
type QueryHistoryResponse struct {
	ID            uint        `json:"id"`
	NLQuery       string      `json:"nl_query"`
	GeneratedSQL  string      `json:"generated_sql"`
	Status        QueryStatus `json:"status"`
	Type          QueryType   `json:"type"`
	DataSourceID  uint        `json:"data_source_id"`
	DataSourceName string     `json:"data_source_name"`
	ExecutionTime int64       `json:"execution_time"`
	RowsReturned  int64       `json:"rows_returned"`
	CreatedAt     time.Time   `json:"created_at"`
	ErrorMsg      string      `json:"error_message,omitempty"`
}

// Methods

// ToHistoryResponse converts NL2SQLQuery to QueryHistoryResponse
func (q *NL2SQLQuery) ToHistoryResponse() *QueryHistoryResponse {
	return &QueryHistoryResponse{
		ID:             q.ID,
		NLQuery:        q.NLQuery,
		GeneratedSQL:   q.GeneratedSQL,
		Status:         q.Status,
		Type:           q.Type,
		DataSourceID:   q.DataSourceID,
		DataSourceName: "", // Will be populated by service layer
		ExecutionTime:  q.ExecutionTime,
		RowsReturned:   q.RowsReturned,
		CreatedAt:      q.CreatedAt,
		ErrorMsg:       q.ErrorMsg,
	}
}

// IsExecutable checks if the query can be executed
func (q *NL2SQLQuery) IsExecutable() bool {
	// Query is executable if it has generated SQL and is not failed
	return q.GeneratedSQL != "" && q.Status != QueryStatusFailed
}

// MarkCompleted marks the query as completed
func (q *NL2SQLQuery) MarkCompleted(executionTime int64, rowsReturned int64) {
	q.Status = QueryStatusCompleted
	q.ExecutionTime = executionTime
	q.RowsReturned = rowsReturned
	q.UpdatedAt = time.Now()
}

// MarkFailed marks the query as failed
func (q *NL2SQLQuery) MarkFailed(errorMsg string) {
	q.Status = QueryStatusFailed
	q.ErrorMsg = errorMsg
	q.UpdatedAt = time.Now()
}