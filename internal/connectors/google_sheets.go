package connectors

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	entity "narapulse-be/internal/models/entity"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

// GoogleSheetsConnector implements the Connector interface for Google Sheets
type GoogleSheetsConnector struct {
	service       *sheets.Service
	spreadsheetID string
	sheetName     string
	ctx           context.Context
}

// NewGoogleSheetsConnector creates a new Google Sheets connector
func NewGoogleSheetsConnector() *GoogleSheetsConnector {
	return &GoogleSheetsConnector{
		ctx: context.Background(),
	}
}

// Connect establishes a connection to Google Sheets
func (g *GoogleSheetsConnector) Connect(config map[string]interface{}) error {
	spreadsheetID, ok := config["spreadsheet_id"].(string)
	if !ok {
		return fmt.Errorf("spreadsheet_id is required")
	}
	g.spreadsheetID = spreadsheetID

	sheetName, ok := config["sheet_name"].(string)
	if !ok {
		g.sheetName = "Sheet1" // default sheet name
	} else {
		g.sheetName = sheetName
	}

	// Check if we have OAuth2 tokens
	if accessToken, ok := config["access_token"].(string); ok && accessToken != "" {
		// Use OAuth2 token
		token := &oauth2.Token{
			AccessToken: accessToken,
		}

		if refreshToken, ok := config["refresh_token"].(string); ok && refreshToken != "" {
			token.RefreshToken = refreshToken
		}

		// Create OAuth2 config (you'll need to set these from environment or config)
		oauth2Config := &oauth2.Config{
			ClientID:     getEnvOrDefault("GOOGLE_CLIENT_ID", ""),
			ClientSecret: getEnvOrDefault("GOOGLE_CLIENT_SECRET", ""),
			Scopes:       []string{sheets.SpreadsheetsReadonlyScope},
			Endpoint:     google.Endpoint,
		}

		client := oauth2Config.Client(g.ctx, token)
		service, err := sheets.NewService(g.ctx, option.WithHTTPClient(client))
		if err != nil {
			return fmt.Errorf("failed to create Sheets service: %w", err)
		}
		g.service = service
	} else if credentialsJSON, ok := config["credentials_json"].(string); ok && credentialsJSON != "" {
		// Use service account credentials
		service, err := sheets.NewService(g.ctx, option.WithCredentialsJSON([]byte(credentialsJSON)))
		if err != nil {
			return fmt.Errorf("failed to create Sheets service with credentials: %w", err)
		}
		g.service = service
	} else {
		// Use default credentials (ADC)
		service, err := sheets.NewService(g.ctx)
		if err != nil {
			return fmt.Errorf("failed to create Sheets service with default credentials: %w", err)
		}
		g.service = service
	}

	return nil
}

// Disconnect closes the Google Sheets connection
func (g *GoogleSheetsConnector) Disconnect() error {
	// Google Sheets API doesn't require explicit disconnection
	g.service = nil
	return nil
}

// TestConnection tests if the connection is working
func (g *GoogleSheetsConnector) TestConnection() error {
	if g.service == nil {
		return fmt.Errorf("no active connection")
	}

	// Try to get spreadsheet metadata to test connection
	_, err := g.service.Spreadsheets.Get(g.spreadsheetID).Do()
	if err != nil {
		return fmt.Errorf("failed to access spreadsheet: %w", err)
	}

	return nil
}

// GetSchema retrieves the schema information from Google Sheets
func (g *GoogleSheetsConnector) GetSchema() ([]entity.Column, error) {
	if g.service == nil {
		return nil, fmt.Errorf("no active connection")
	}

	// Get spreadsheet metadata
	spreadsheet, err := g.service.Spreadsheets.Get(g.spreadsheetID).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get spreadsheet: %w", err)
	}

	var allColumns []entity.Column

	// Process each sheet
	for _, sheet := range spreadsheet.Sheets {
		sheetTitle := sheet.Properties.Title
		
		// Get the first few rows to infer schema
		readRange := fmt.Sprintf("%s!1:3", sheetTitle) // Read first 3 rows
		resp, err := g.service.Spreadsheets.Values.Get(g.spreadsheetID, readRange).Do()
		if err != nil {
			continue // Skip sheets that can't be read
		}

		if len(resp.Values) == 0 {
			continue // Skip empty sheets
		}

		// Assume first row contains headers
		headerRow := resp.Values[0]
		for i, header := range headerRow {
			headerStr, ok := header.(string)
			if !ok {
				headerStr = fmt.Sprintf("Column_%d", i+1)
			}

			// Infer data type from sample data
			dataType := g.inferColumnType(resp.Values, i)

			column := entity.Column{
				Name:     fmt.Sprintf("%s.%s", sheetTitle, headerStr),
				Type:     dataType,
				Nullable: true, // Google Sheets cells can be empty
			}

			allColumns = append(allColumns, column)
		}
	}

	return allColumns, nil
}

// GetData retrieves data from a specific sheet
func (g *GoogleSheetsConnector) GetData(sheetName string, limit int) ([]map[string]interface{}, error) {
	if g.service == nil {
		return nil, fmt.Errorf("no active connection")
	}

	if limit <= 0 {
		limit = 100 // default limit
	}

	// Build range - get headers first
	headerRange := fmt.Sprintf("%s!1:1", sheetName)
	headerResp, err := g.service.Spreadsheets.Values.Get(g.spreadsheetID, headerRange).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get headers: %w", err)
	}

	if len(headerResp.Values) == 0 || len(headerResp.Values[0]) == 0 {
		return nil, fmt.Errorf("no headers found")
	}

	headers := headerResp.Values[0]

	// Get data rows
	dataRange := fmt.Sprintf("%s!2:%d", sheetName, limit+1) // +1 because we start from row 2
	dataResp, err := g.service.Spreadsheets.Values.Get(g.spreadsheetID, dataRange).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get data: %w", err)
	}

	var result []map[string]interface{}

	for _, row := range dataResp.Values {
		rowMap := make(map[string]interface{})
		for i, header := range headers {
			headerStr, ok := header.(string)
			if !ok {
				headerStr = fmt.Sprintf("Column_%d", i+1)
			}

			var value interface{}
			if i < len(row) {
				value = row[i]
			} else {
				value = nil
			}

			rowMap[headerStr] = value
		}
		result = append(result, rowMap)
	}

	return result, nil
}

// inferColumnType infers the data type of a column based on sample values
func (g *GoogleSheetsConnector) inferColumnType(values [][]interface{}, columnIndex int) string {
	if len(values) <= 1 {
		return "string" // default to string if no data
	}

	// Collect sample values (skip header row)
	var samples []interface{}
	for i := 1; i < len(values) && i < 10; i++ { // Check up to 10 rows
		if columnIndex < len(values[i]) {
			samples = append(samples, values[i][columnIndex])
		}
	}

	if len(samples) == 0 {
		return "string"
	}

	// Count type occurrences
	intCount := 0
	floatCount := 0
	boolCount := 0
	stringCount := 0

	for _, sample := range samples {
		if sample == nil {
			continue
		}

		str := fmt.Sprintf("%v", sample)
		str = strings.TrimSpace(str)

		if str == "" {
			continue
		}

		// Check if it's a boolean
		if strings.ToLower(str) == "true" || strings.ToLower(str) == "false" {
			boolCount++
			continue
		}

		// Check if it's an integer
		if _, err := strconv.Atoi(str); err == nil {
			intCount++
			continue
		}

		// Check if it's a float
		if _, err := strconv.ParseFloat(str, 64); err == nil {
			floatCount++
			continue
		}

		stringCount++
	}

	// Determine the most common type
	total := intCount + floatCount + boolCount + stringCount
	if total == 0 {
		return "string"
	}

	// If more than 70% of samples are of one type, use that type
	threshold := float64(total) * 0.7

	if float64(intCount) >= threshold {
		return "integer"
	}
	if float64(floatCount) >= threshold {
		return "float"
	}
	if float64(boolCount) >= threshold {
		return "boolean"
	}

	return "string" // default to string
}

// getEnvOrDefault gets environment variable or returns default value
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}