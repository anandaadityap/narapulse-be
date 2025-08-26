package services

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	models "narapulse-be/internal/models/entity"
	"github.com/xwb1989/sqlparser"
)

// SQLValidatorService handles SQL validation and safety checks
type SQLValidatorService struct {
	allowedFunctions []string
	blockedKeywords  []string
	maxJoinTables    int
	maxRowLimit      int
}

// NewSQLValidatorService creates a new SQL validator service
func NewSQLValidatorService() *SQLValidatorService {
	return &SQLValidatorService{
		allowedFunctions: []string{
			// Aggregate functions
			"COUNT", "SUM", "AVG", "MIN", "MAX",
			// String functions
			"UPPER", "LOWER", "TRIM", "LENGTH", "SUBSTRING", "CONCAT",
			// Date functions
			"DATE", "YEAR", "MONTH", "DAY", "DATE_TRUNC", "DATE_ADD", "DATE_SUB",
			"EXTRACT", "NOW", "CURRENT_DATE", "CURRENT_TIMESTAMP",
			// Math functions
			"ROUND", "CEIL", "FLOOR", "ABS", "COALESCE", "NULLIF",
			// Conditional functions
			"CASE", "IF", "IFNULL",
		},
		blockedKeywords: []string{
			// DML operations
			"INSERT", "UPDATE", "DELETE", "MERGE", "UPSERT",
			// DDL operations
			"CREATE", "ALTER", "DROP", "TRUNCATE", "RENAME",
			// DCL operations
			"GRANT", "REVOKE",
			// Transaction control
			"COMMIT", "ROLLBACK", "SAVEPOINT",
			// System functions
			"EXEC", "EXECUTE", "CALL", "LOAD", "COPY",
			// File operations
			"INTO OUTFILE", "LOAD DATA", "SELECT INTO",
			// Administrative
			"SHOW", "DESCRIBE", "EXPLAIN", "ANALYZE",
		},
		maxJoinTables: 5,
		maxRowLimit:   10000,
	}
}

// ValidateSQL validates a SQL query for safety and compliance
func (s *SQLValidatorService) ValidateSQL(sql string) (*models.SQLValidationResult, error) {
	result := &models.SQLValidationResult{
		IsValid:      false,
		IsReadOnly:   false,
		HasLimit:     false,
		SafetyScore:  0.0,
		Violations:   []string{},
		Warnings:     []string{},
	}

	// Basic SQL sanitization
	sql = strings.TrimSpace(sql)
	if sql == "" {
		result.Violations = append(result.Violations, "Empty SQL query")
		return result, errors.New("empty SQL query")
	}

	// Check for blocked keywords
	if violations := s.checkBlockedKeywords(sql); len(violations) > 0 {
		result.Violations = append(result.Violations, violations...)
		return result, errors.New("SQL contains blocked operations")
	}

	// Parse SQL using sqlparser
	stmt, err := sqlparser.Parse(sql)
	if err != nil {
		result.Violations = append(result.Violations, fmt.Sprintf("SQL parsing error: %v", err))
		return result, fmt.Errorf("failed to parse SQL: %v", err)
	}

	// Validate that it's a SELECT statement
	selectStmt, ok := stmt.(*sqlparser.Select)
	if !ok {
		result.Violations = append(result.Violations, "Only SELECT statements are allowed")
		return result, errors.New("only SELECT statements are allowed")
	}

	result.IsReadOnly = true

	// Check for LIMIT clause
	result.HasLimit = s.hasLimitClause(selectStmt)
	if !result.HasLimit {
		result.Warnings = append(result.Warnings, "Query should include LIMIT clause for performance")
	}

	// Validate JOIN complexity
	if warnings := s.validateJoinComplexity(selectStmt); len(warnings) > 0 {
		result.Warnings = append(result.Warnings, warnings...)
	}

	// Validate functions
	if violations := s.validateFunctions(sql); len(violations) > 0 {
		result.Violations = append(result.Violations, violations...)
		return result, errors.New("SQL contains unauthorized functions")
	}

	// Check for potential security issues
	if warnings := s.checkSecurityIssues(sql); len(warnings) > 0 {
		result.Warnings = append(result.Warnings, warnings...)
	}

	// Calculate safety score
	result.SafetyScore = s.calculateSafetyScore(result)

	// Estimate cost (simplified)
	result.EstimatedCost = s.estimateQueryCost(selectStmt)

	result.IsValid = len(result.Violations) == 0

	return result, nil
}

// EnforceLimit adds or modifies LIMIT clause in SQL
func (s *SQLValidatorService) EnforceLimit(sql string, limit int) (string, error) {
	if limit <= 0 || limit > s.maxRowLimit {
		limit = s.maxRowLimit
	}

	stmt, err := sqlparser.Parse(sql)
	if err != nil {
		return "", fmt.Errorf("failed to parse SQL: %v", err)
	}

	selectStmt, ok := stmt.(*sqlparser.Select)
	if !ok {
		return "", errors.New("only SELECT statements are supported")
	}

	// Add or modify LIMIT clause
	selectStmt.Limit = &sqlparser.Limit{
		Rowcount: sqlparser.NewIntVal([]byte(fmt.Sprintf("%d", limit))),
	}

	return sqlparser.String(selectStmt), nil
}

// checkBlockedKeywords checks for blocked SQL keywords
func (s *SQLValidatorService) checkBlockedKeywords(sql string) []string {
	var violations []string
	sqlUpper := strings.ToUpper(sql)

	for _, keyword := range s.blockedKeywords {
		if strings.Contains(sqlUpper, keyword) {
			violations = append(violations, fmt.Sprintf("Blocked keyword detected: %s", keyword))
		}
	}

	return violations
}

// hasLimitClause checks if the SELECT statement has a LIMIT clause
func (s *SQLValidatorService) hasLimitClause(stmt *sqlparser.Select) bool {
	return stmt.Limit != nil
}

// validateJoinComplexity validates the complexity of JOIN operations
func (s *SQLValidatorService) validateJoinComplexity(stmt *sqlparser.Select) []string {
	var warnings []string

	// Count tables in FROM clause
	tableCount := s.countTablesInFrom(stmt.From)

	if tableCount > s.maxJoinTables {
		warnings = append(warnings, fmt.Sprintf("Query joins too many tables (%d > %d)", tableCount, s.maxJoinTables))
	}

	return warnings
}

// countTablesInFrom counts the number of tables in FROM clause
func (s *SQLValidatorService) countTablesInFrom(from []sqlparser.TableExpr) int {
	count := 0
	for _, tableExpr := range from {
		switch t := tableExpr.(type) {
		case *sqlparser.AliasedTableExpr:
			count++
		case *sqlparser.JoinTableExpr:
			count += s.countTablesInJoin(t)
		}
	}
	return count
}

// countTablesInJoin counts tables in JOIN expressions
func (s *SQLValidatorService) countTablesInJoin(join *sqlparser.JoinTableExpr) int {
	count := 0
	
	// Count left side
	if _, ok := join.LeftExpr.(*sqlparser.AliasedTableExpr); ok {
		count++
	}
	
	// Count right side
	if _, ok := join.RightExpr.(*sqlparser.AliasedTableExpr); ok {
		count++
	}
	
	return count
}

// validateFunctions validates that only allowed functions are used
func (s *SQLValidatorService) validateFunctions(sql string) []string {
	var violations []string

	// Simple regex to find function calls
	funcRegex := regexp.MustCompile(`(?i)\b([A-Z_]+)\s*\(`)
	matches := funcRegex.FindAllStringSubmatch(sql, -1)

	for _, match := range matches {
		if len(match) > 1 {
			funcName := strings.ToUpper(match[1])
			if !s.isFunctionAllowed(funcName) {
				violations = append(violations, fmt.Sprintf("Unauthorized function: %s", funcName))
			}
		}
	}

	return violations
}

// isFunctionAllowed checks if a function is in the allowed list
func (s *SQLValidatorService) isFunctionAllowed(funcName string) bool {
	for _, allowed := range s.allowedFunctions {
		if allowed == funcName {
			return true
		}
	}
	return false
}

// checkSecurityIssues checks for potential security issues
func (s *SQLValidatorService) checkSecurityIssues(sql string) []string {
	var warnings []string
	sqlUpper := strings.ToUpper(sql)

	// Check for potential SQL injection patterns
	suspiciousPatterns := []string{
		"--", "/*", "*/", ";",
		"UNION", "OR 1=1", "AND 1=1",
		"DROP", "DELETE", "UPDATE",
	}

	for _, pattern := range suspiciousPatterns {
		if strings.Contains(sqlUpper, pattern) {
			warnings = append(warnings, fmt.Sprintf("Potentially suspicious pattern detected: %s", pattern))
		}
	}

	return warnings
}

// calculateSafetyScore calculates a safety score based on validation results
func (s *SQLValidatorService) calculateSafetyScore(result *models.SQLValidationResult) float64 {
	score := 1.0

	// Deduct points for violations
	score -= float64(len(result.Violations)) * 0.3

	// Deduct points for warnings
	score -= float64(len(result.Warnings)) * 0.1

	// Bonus for having LIMIT
	if result.HasLimit {
		score += 0.1
	}

	// Bonus for being read-only
	if result.IsReadOnly {
		score += 0.2
	}

	// Ensure score is between 0 and 1
	if score < 0 {
		score = 0
	}
	if score > 1 {
		score = 1
	}

	return score
}

// estimateQueryCost provides a simple cost estimation
func (s *SQLValidatorService) estimateQueryCost(stmt *sqlparser.Select) float64 {
	cost := 0.01 // Base cost

	// Add cost for each table
	tableCount := s.countTablesInFrom(stmt.From)
	cost += float64(tableCount) * 0.005

	// Add cost for JOINs
	if tableCount > 1 {
		cost += float64(tableCount-1) * 0.01
	}

	// Add cost for complex WHERE clauses
	if stmt.Where != nil {
		cost += 0.005
	}

	// Add cost for GROUP BY
	if stmt.GroupBy != nil {
		cost += 0.01
	}

	// Add cost for ORDER BY
	if stmt.OrderBy != nil {
		cost += 0.005
	}

	return cost
}

// IsQuerySafe checks if a query meets safety requirements
func (s *SQLValidatorService) IsQuerySafe(result *models.SQLValidationResult) bool {
	return result.IsValid && result.IsReadOnly && result.SafetyScore >= 0.7
}