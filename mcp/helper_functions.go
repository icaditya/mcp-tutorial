package mcp

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

// Helper functions for recon-saas API interactions

// ValidationResult holds the result of validation mode processing
type ValidationResult struct {
	Mode             string            `json:"mode"`
	Approved         bool              `json:"approved"`
	RuleExpressions  map[string]string `json:"rule_expressions"`
	ValidationNotes  []string          `json:"validation_notes"`
	RequiresApproval bool              `json:"requires_approval"`
}

// applyValidationMode applies validation logic based on the selected mode
func applyValidationMode(mode string, approveExpressions bool, masterSourceID1, masterSourceID2 string) (*ValidationResult, error) {
	result := &ValidationResult{
		Mode:            mode,
		RuleExpressions: make(map[string]string),
		ValidationNotes: make([]string, 0),
	}

	// Generate rule expressions
	result.RuleExpressions["reconciled"] = fmt.Sprintf("%s.EntityID == %s.EntityID && %s.Amount.Equal(%s.Amount)",
		masterSourceID1, masterSourceID2, masterSourceID1, masterSourceID2)
	result.RuleExpressions["amount_mismatch"] = fmt.Sprintf("%s.EntityID == %s.EntityID && !%s.Amount.Equal(%s.Amount)",
		masterSourceID1, masterSourceID2, masterSourceID1, masterSourceID2)
	result.RuleExpressions["missing_record"] = "NoRecord.Value == true"

	switch mode {
	case "automatic":
		result.Approved = true
		result.RequiresApproval = false
		result.ValidationNotes = append(result.ValidationNotes, "Automatic validation: All rule expressions approved automatically")

	case "guided":
		result.Approved = approveExpressions
		result.RequiresApproval = true
		if approveExpressions {
			result.ValidationNotes = append(result.ValidationNotes, "Guided validation: User approved rule expressions")
		} else {
			result.ValidationNotes = append(result.ValidationNotes, "Guided validation: User rejected rule expressions")
			return nil, fmt.Errorf("rule expressions were not approved by user in guided mode")
		}

	case "manual":
		result.Approved = false
		result.RequiresApproval = true
		result.ValidationNotes = append(result.ValidationNotes, "Manual validation: Expressions require manual review and modification")
		if !approveExpressions {
			return nil, fmt.Errorf("manual validation mode requires user approval of rule expressions")
		}
		result.Approved = true
		result.ValidationNotes = append(result.ValidationNotes, "Manual validation: User manually approved expressions")

	default:
		return nil, fmt.Errorf("unsupported validation mode: %s", mode)
	}

	return result, nil
}

// generateMerchantSourceName generates merchant source names based on the selected naming strategy
func generateMerchantSourceName(baseName, strategy string, index int) string {
	switch strategy {
	case "descriptive":
		return baseName + " - Merchant Portal"
	case "timestamp":
		timestamp := time.Now().Format("20060102_150405")
		return fmt.Sprintf("%s_%s", baseName, timestamp)
	case "sequential":
		return fmt.Sprintf("%s_%03d", baseName, index)
	case "custom":
		// For custom strategy, we could allow user-defined patterns
		// For now, default to descriptive with custom suffix
		return baseName + " - Custom Source"
	default:
		// Fallback to descriptive
		return baseName + " - Merchant Portal"
	}
}

// analyzeFile analyzes a file and returns metadata about columns and data based on file type
func analyzeFile(filePath, fileID, fileType string) (map[string]interface{}, error) {
	switch fileType {
	case "csv":
		return analyzeCSVFile(filePath, fileID)
	case "excel":
		return analyzeExcelFile(filePath, fileID)
	default:
		return nil, fmt.Errorf("unsupported file type: %s", fileType)
	}
}

// analyzeCSVFile analyzes a CSV file and returns metadata about columns and data
func analyzeCSVFile(filePath, fileID string) (map[string]interface{}, error) {
	// Open and read the CSV file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %v", filePath, err)
	}
	defer file.Close()

	reader := csv.NewReader(file)

	// Read all records
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV file %s: %v", filePath, err)
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("CSV file %s is empty", filePath)
	}

	// Extract headers and data
	headers := records[0]
	dataRows := records[1:]
	totalRows := len(dataRows)
	totalColumns := len(headers)

	// Analyze each column for patterns and characteristics
	columnAnalysis := make(map[string]map[string]interface{})
	for i, columnName := range headers {
		analysis := analyzeColumn(columnName, i, dataRows)
		columnAnalysis[columnName] = analysis
	}

	// Identify EntityID candidates
	entityIDCandidates := identifyEntityIDCandidates(headers, columnAnalysis)

	// Identify Amount candidates
	amountCandidates := identifyAmountCandidates(headers, columnAnalysis)

	// Determine recommendations
	var recommendedEntityID, recommendedAmount string
	if len(entityIDCandidates) > 0 {
		if entityMap, ok := entityIDCandidates[0].(map[string]interface{}); ok {
			if name, ok := entityMap["column_name"].(string); ok {
				recommendedEntityID = name
			}
		}
	}
	if len(amountCandidates) > 0 {
		if amountMap, ok := amountCandidates[0].(map[string]interface{}); ok {
			if name, ok := amountMap["column_name"].(string); ok {
				recommendedAmount = name
			}
		}
	}

	return map[string]interface{}{
		"filename":             filePath,
		"file_type":            "csv",
		"total_rows":           totalRows,
		"total_columns":        totalColumns,
		"all_columns":          headers,
		"entityid_candidates":  entityIDCandidates,
		"recommended_entityid": recommendedEntityID,
		"amount_candidates":    amountCandidates,
		"recommended_amount":   recommendedAmount,
		"user_selections": map[string]interface{}{
			"selected_entityid": recommendedEntityID,
			"selected_amount":   recommendedAmount,
		},
	}, nil
}

// analyzeExcelFile analyzes an Excel file and returns metadata about columns and data
func analyzeExcelFile(filePath, fileID string) (map[string]interface{}, error) {
	// Open the Excel file
	file, err := excelize.OpenFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open Excel file %s: %v", filePath, err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			fmt.Printf("Warning: failed to close Excel file: %v\n", err)
		}
	}()

	// Get the first sheet name (or use the active sheet)
	sheets := file.GetSheetList()
	if len(sheets) == 0 {
		return nil, fmt.Errorf("Excel file %s has no sheets", filePath)
	}

	sheetName := sheets[0] // Use the first sheet

	// Get all rows from the sheet
	rows, err := file.GetRows(sheetName)
	if err != nil {
		return nil, fmt.Errorf("failed to read Excel sheet %s: %v", sheetName, err)
	}

	if len(rows) == 0 {
		return nil, fmt.Errorf("Excel sheet %s is empty", sheetName)
	}

	// Extract headers and data rows
	headers := rows[0]
	dataRows := rows[1:]
	totalRows := len(dataRows)
	totalColumns := len(headers)

	// Convert Excel rows to the same format as CSV for analysis
	// Ensure all rows have the same number of columns as headers
	normalizedDataRows := make([][]string, len(dataRows))
	for i, row := range dataRows {
		normalizedRow := make([]string, len(headers))
		for j := 0; j < len(headers); j++ {
			if j < len(row) {
				normalizedRow[j] = strings.TrimSpace(row[j])
			} else {
				normalizedRow[j] = "" // Fill missing columns with empty strings
			}
		}
		normalizedDataRows[i] = normalizedRow
	}

	// Analyze each column for patterns and characteristics
	columnAnalysis := make(map[string]map[string]interface{})
	for i, columnName := range headers {
		analysis := analyzeColumn(columnName, i, normalizedDataRows)
		columnAnalysis[columnName] = analysis
	}

	// Identify EntityID candidates
	entityIDCandidates := identifyEntityIDCandidates(headers, columnAnalysis)

	// Identify Amount candidates
	amountCandidates := identifyAmountCandidates(headers, columnAnalysis)

	// Determine recommendations
	var recommendedEntityID, recommendedAmount string
	if len(entityIDCandidates) > 0 {
		if entityMap, ok := entityIDCandidates[0].(map[string]interface{}); ok {
			if name, ok := entityMap["column_name"].(string); ok {
				recommendedEntityID = name
			}
		}
	}
	if len(amountCandidates) > 0 {
		if amountMap, ok := amountCandidates[0].(map[string]interface{}); ok {
			if name, ok := amountMap["column_name"].(string); ok {
				recommendedAmount = name
			}
		}
	}

	return map[string]interface{}{
		"filename":             filePath,
		"file_type":            "excel",
		"sheet_name":           sheetName,
		"total_sheets":         len(sheets),
		"total_rows":           totalRows,
		"total_columns":        totalColumns,
		"all_columns":          headers,
		"entityid_candidates":  entityIDCandidates,
		"recommended_entityid": recommendedEntityID,
		"amount_candidates":    amountCandidates,
		"recommended_amount":   recommendedAmount,
		"user_selections": map[string]interface{}{
			"selected_entityid": recommendedEntityID,
			"selected_amount":   recommendedAmount,
		},
	}, nil
}

// analyzeColumn analyzes individual column characteristics
func analyzeColumn(columnName string, columnIndex int, dataRows [][]string) map[string]interface{} {
	if len(dataRows) == 0 {
		return map[string]interface{}{
			"unique_count":      0,
			"unique_percentage": 0.0,
			"data_type":         "unknown",
			"sample_values":     []string{},
			"is_numeric":        false,
			"is_monetary":       false,
		}
	}

	// Extract column values
	values := make([]string, 0, len(dataRows))
	uniqueValues := make(map[string]bool)
	numericCount := 0
	monetaryCount := 0

	sampleSize := len(dataRows)
	if sampleSize > 100 {
		sampleSize = 100 // Analyze first 100 rows for performance
	}

	for i := 0; i < sampleSize && i < len(dataRows); i++ {
		if columnIndex < len(dataRows[i]) {
			value := strings.TrimSpace(dataRows[i][columnIndex])
			if value != "" {
				values = append(values, value)
				uniqueValues[value] = true

				// Check if numeric
				if isNumeric(value) {
					numericCount++
				}

				// Check if monetary
				if isMonetary(value) {
					monetaryCount++
				}
			}
		}
	}

	uniqueCount := len(uniqueValues)
	uniquePercentage := float64(uniqueCount) / float64(len(values)) * 100

	// Get sample values (first 5 unique values)
	sampleValues := make([]string, 0, 5)
	count := 0
	for value := range uniqueValues {
		if count >= 5 {
			break
		}
		sampleValues = append(sampleValues, value)
		count++
	}

	// Determine data type
	dataType := "string"
	isNumericCol := float64(numericCount)/float64(len(values)) > 0.8
	isMonetaryCol := float64(monetaryCount)/float64(len(values)) > 0.7

	if isMonetaryCol {
		dataType = "monetary"
	} else if isNumericCol {
		dataType = "numeric"
	}

	return map[string]interface{}{
		"unique_count":      uniqueCount,
		"unique_percentage": uniquePercentage,
		"data_type":         dataType,
		"sample_values":     sampleValues,
		"is_numeric":        isNumericCol,
		"is_monetary":       isMonetaryCol,
		"total_values":      len(values),
	}
}

// identifyEntityIDCandidates identifies potential EntityID columns
func identifyEntityIDCandidates(headers []string, columnAnalysis map[string]map[string]interface{}) []interface{} {
	candidates := make([]interface{}, 0)

	// Priority patterns for EntityID columns
	idPatterns := []string{
		"(?i)(transaction[_\\s]*id|txn[_\\s]*id)",
		"(?i)(entity[_\\s]*id|ent[_\\s]*id)",
		"(?i)(instance[_\\s]*id|inst[_\\s]*id)",
		"(?i)(reference[_\\s]*(number|no|num|id)|ref[_\\s]*(no|num|id))",
		"(?i)(receipt[_\\s]*(number|no|num|id))",
		"(?i)(order[_\\s]*(number|no|num|id))",
		"(?i)^id$",
		"(?i)(unique[_\\s]*id)",
	}

	type candidate struct {
		columnName       string
		confidence       float64
		reason           string
		uniquePercentage float64
	}

	var candidateList []candidate

	for _, columnName := range headers {
		analysis := columnAnalysis[columnName]
		uniquePercentage := analysis["unique_percentage"].(float64)

		confidence := 0.0
		reason := ""

		// Check naming patterns (highest priority)
		for i, pattern := range idPatterns {
			if matched, _ := regexp.MatchString(pattern, columnName); matched {
				confidence = 0.95 - float64(i)*0.05 // Decrease confidence for lower priority patterns
				reason = "ID-like naming pattern with high uniqueness"
				break
			}
		}

		// Check uniqueness-based scoring
		if confidence == 0.0 && uniquePercentage >= 95.0 {
			confidence = 0.80
			reason = "Very high uniqueness suggests unique identifier"
		} else if confidence == 0.0 && uniquePercentage >= 85.0 {
			confidence = 0.70
			reason = "High uniqueness suggests potential identifier"
		}

		// Boost confidence for high uniqueness even with pattern match
		if confidence > 0.0 && uniquePercentage >= 98.0 {
			confidence = minFloat(confidence+0.1, 0.98)
		}

		// Only include candidates with reasonable confidence and uniqueness
		if confidence >= 0.60 && uniquePercentage >= 75.0 {
			candidateList = append(candidateList, candidate{
				columnName:       columnName,
				confidence:       confidence,
				reason:           reason,
				uniquePercentage: uniquePercentage,
			})
		}
	}

	// Sort by confidence (highest first)
	for i := 0; i < len(candidateList)-1; i++ {
		for j := i + 1; j < len(candidateList); j++ {
			if candidateList[j].confidence > candidateList[i].confidence {
				candidateList[i], candidateList[j] = candidateList[j], candidateList[i]
			}
		}
	}

	// Convert to expected format
	for _, candidate := range candidateList {
		candidates = append(candidates, map[string]interface{}{
			"column_name":       candidate.columnName,
			"unique_percentage": candidate.uniquePercentage,
			"confidence":        candidate.confidence,
			"reason":            candidate.reason,
		})
	}

	return candidates
}

// identifyAmountCandidates identifies potential Amount columns
func identifyAmountCandidates(headers []string, columnAnalysis map[string]map[string]interface{}) []interface{} {
	candidates := make([]interface{}, 0)

	// Priority patterns for Amount columns
	amountPatterns := []string{
		"(?i)^amount$",
		"(?i)(net[_\\s]*amount|netamount)",
		"(?i)(gross[_\\s]*amount|grossamount)",
		"(?i)(total[_\\s]*amount|totalamount)",
		"(?i)(transaction[_\\s]*amount|txn[_\\s]*amount)",
		"(?i)(payment[_\\s]*amount)",
		"(?i)(balance)",
		"(?i)(value)",
		"(?i)(price)",
		"(?i)(sum)",
		"(?i).*amount.*",
		"(?i).*_amt$",
		"(?i).*amt_.*",
	}

	type candidate struct {
		columnName   string
		confidence   float64
		reason       string
		sampleValues []string
	}

	var candidateList []candidate

	for _, columnName := range headers {
		analysis := columnAnalysis[columnName]
		isMonetary := analysis["is_monetary"].(bool)
		isNumeric := analysis["is_numeric"].(bool)
		sampleValues := analysis["sample_values"].([]string)

		confidence := 0.0
		reason := ""

		// Check naming patterns with monetary data
		for i, pattern := range amountPatterns {
			if matched, _ := regexp.MatchString(pattern, columnName); matched {
				if isMonetary {
					confidence = 0.95 - float64(i)*0.03
					reason = "Amount-like naming with monetary values"
				} else if isNumeric {
					confidence = 0.80 - float64(i)*0.03
					reason = "Amount-like naming with numeric values"
				} else {
					confidence = 0.60 - float64(i)*0.03
					reason = "Amount-like naming pattern"
				}
				break
			}
		}

		// Check data characteristics without pattern match
		if confidence == 0.0 && isMonetary {
			confidence = 0.75
			reason = "Contains monetary values (currency format)"
		} else if confidence == 0.0 && isNumeric {
			confidence = 0.60
			reason = "Contains numeric values"
		}

		// Only include candidates with reasonable confidence
		if confidence >= 0.50 {
			candidateList = append(candidateList, candidate{
				columnName:   columnName,
				confidence:   confidence,
				reason:       reason,
				sampleValues: sampleValues,
			})
		}
	}

	// Sort by confidence (highest first)
	for i := 0; i < len(candidateList)-1; i++ {
		for j := i + 1; j < len(candidateList); j++ {
			if candidateList[j].confidence > candidateList[i].confidence {
				candidateList[i], candidateList[j] = candidateList[j], candidateList[i]
			}
		}
	}

	// Convert to expected format
	for _, candidate := range candidateList {
		candidates = append(candidates, map[string]interface{}{
			"column_name":   candidate.columnName,
			"sample_values": candidate.sampleValues,
			"confidence":    candidate.confidence,
			"reason":        candidate.reason,
		})
	}

	return candidates
}

// isNumeric checks if a string represents a numeric value
func isNumeric(s string) bool {
	// Remove common non-numeric characters that might be in monetary values
	s = strings.ReplaceAll(s, ",", "")
	s = strings.ReplaceAll(s, "$", "")
	s = strings.ReplaceAll(s, "₹", "")
	s = strings.ReplaceAll(s, "€", "")
	s = strings.ReplaceAll(s, "£", "")
	s = strings.TrimSpace(s)

	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}

// isMonetary checks if a string represents a monetary value
func isMonetary(s string) bool {
	// Check for currency symbols
	currencyPattern := `(?i)[$₹€£¥¢]|\b(usd|eur|gbp|inr|jpy)\b`
	if matched, _ := regexp.MatchString(currencyPattern, s); matched {
		return true
	}

	// Check for decimal pattern with 2 decimal places (common for currency)
	decimalPattern := `^\d+\.\d{2}$`
	if matched, _ := regexp.MatchString(decimalPattern, s); matched {
		return true
	}

	// Check for comma-separated thousands with decimal
	commaPattern := `^\d{1,3}(,\d{3})*(\.\d{2})?$`
	if matched, _ := regexp.MatchString(commaPattern, s); matched {
		return true
	}

	// For simple numeric values, check if they're reasonable monetary amounts
	cleanValue := strings.ReplaceAll(s, ",", "")
	if val, err := strconv.ParseFloat(cleanValue, 64); err == nil {
		// Consider values between 0.01 and 999999999.99 as potentially monetary
		return val >= 0.01 && val <= 999999999.99 &&
			(strings.Contains(s, ".") || len(cleanValue) <= 8)
	}

	return false
}

// minFloat returns the minimum of two float64 values
func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// makeReconSaaSAPICall makes authenticated API calls to recon-saas service
func makeReconSaaSAPICall(ctx context.Context, method, endpoint string, payload interface{}) (map[string]interface{}, error) {
	const baseURL = "https://recon-saas.dev.razorpay.in"
	const authHeader = "Basic cmVjb24tc2FhczpyZWNvbi1zYWFz"

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	var reqBody io.Reader
	if payload != nil {
		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal payload: %v", err)
		}
		reqBody = bytes.NewReader(payloadBytes)
	}

	req, err := http.NewRequestWithContext(ctx, method, baseURL+endpoint, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authHeader)

	fmt.Printf("🌐 DEBUG: Making %s request to: %s\n", method, baseURL+endpoint)
	if payload != nil {
		payloadBytes, _ := json.Marshal(payload)
		fmt.Printf("📤 DEBUG: Request payload: %s\n", string(payloadBytes))
	}

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("❌ DEBUG: HTTP request failed: %v\n", err)
		return nil, fmt.Errorf("API call failed: %v", err)
	}
	defer resp.Body.Close()

	fmt.Printf("📥 DEBUG: Response status: %s\n", resp.Status)
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("❌ DEBUG: Failed to read response body: %v\n", err)
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	fmt.Printf("📥 DEBUG: Response body: %s\n", string(responseBody))
	if resp.StatusCode >= 400 {
		fmt.Printf("❌ DEBUG: API error %d: %s\n", resp.StatusCode, string(responseBody))
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(responseBody))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(responseBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	return result, nil
}

// makeReconSaaSAPICallString makes authenticated API calls to recon-saas service and returns string response
func makeReconSaaSAPICallString(ctx context.Context, method, endpoint string, payload interface{}) (string, error) {
	const baseURL = "https://recon-saas.dev.razorpay.in"
	const authHeader = "Basic cmVjb24tc2FhczpyZWNvbi1zYWFz"

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	var reqBody io.Reader
	if payload != nil {
		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			return "", fmt.Errorf("failed to marshal payload: %v", err)
		}
		reqBody = bytes.NewReader(payloadBytes)
	}

	req, err := http.NewRequestWithContext(ctx, method, baseURL+endpoint, reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", authHeader)

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("API call failed: %v", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %v", err)
	}

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("API error %d: %s", resp.StatusCode, string(responseBody))
	}

	// Handle string response in quotes (e.g., "R2QTjA6dPvcygE")
	responseStr := strings.TrimSpace(string(responseBody))
	if strings.HasPrefix(responseStr, "\"") && strings.HasSuffix(responseStr, "\"") {
		// Remove surrounding quotes
		responseStr = responseStr[1 : len(responseStr)-1]
	}

	return responseStr, nil
}

// fetchMerchantSources fetches merchant sources for a given merchant
func fetchMerchantSources(ctx context.Context, merchantID string) ([]map[string]interface{}, error) {
	endpoint := fmt.Sprintf("/v1/admin-recon-saas/sources/merchant/%s", merchantID)
	result, err := makeReconSaaSAPICall(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	sources, ok := result["sources"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response format for merchant sources")
	}

	var merchantSources []map[string]interface{}
	for _, source := range sources {
		if sourceMap, ok := source.(map[string]interface{}); ok {
			merchantSources = append(merchantSources, sourceMap)
		}
	}

	return merchantSources, nil
}

// fetchLookups fetches lookups for a given merchant
func fetchLookups(ctx context.Context, merchantID string) ([]map[string]interface{}, error) {
	endpoint := fmt.Sprintf("/v1/admin-recon-saas/lookup/merchant/%s", merchantID)
	result, err := makeReconSaaSAPICall(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	lookups, ok := result["lookups"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response format for lookups")
	}

	var lookupList []map[string]interface{}
	for _, lookup := range lookups {
		if lookupMap, ok := lookup.(map[string]interface{}); ok {
			lookupList = append(lookupList, lookupMap)
		}
	}

	return lookupList, nil
}

// fetchMerchantReconProcesses fetches merchant reconciliation processes
func fetchMerchantReconProcesses(ctx context.Context, merchantID string) ([]map[string]interface{}, error) {
	endpoint := fmt.Sprintf("/v1/admin-recon-saas/recon_process/merchant/%s", merchantID)
	result, err := makeReconSaaSAPICall(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	processes, ok := result["processes"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response format for recon processes")
	}

	var processList []map[string]interface{}
	for _, process := range processes {
		if processMap, ok := process.(map[string]interface{}); ok {
			processList = append(processList, processMap)
		}
	}

	return processList, nil
}

// getAvailableIDs fetches all available IDs for a merchant
func getAvailableIDs(ctx context.Context, merchantID string) (map[string]interface{}, error) {
	merchantSources, err := fetchMerchantSources(ctx, merchantID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch merchant sources: %v", err)
	}

	lookups, err := fetchLookups(ctx, merchantID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch lookups: %v", err)
	}

	reconProcesses, err := fetchMerchantReconProcesses(ctx, merchantID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch recon processes: %v", err)
	}

	return map[string]interface{}{
		"merchant_sources": merchantSources,
		"lookups":          lookups,
		"recon_processes":  reconProcesses,
		"summary": map[string]interface{}{
			"merchant_id":            merchantID,
			"merchant_sources_count": len(merchantSources),
			"lookups_count":          len(lookups),
			"recon_processes_count":  len(reconProcesses),
		},
	}, nil
}

// createMasterSource creates a master source via recon-saas API
func createMasterSource(ctx context.Context, name, columnsJSON, entityIDColumn, amountColumn string) (string, error) {
	var columns []string
	if err := json.Unmarshal([]byte(columnsJSON), &columns); err != nil {
		return "", fmt.Errorf("invalid columns JSON: %v", err)
	}

	// Generate source schema (all columns as string type)
	sourceSchema := make([]map[string]string, len(columns))
	for i, col := range columns {
		sourceSchema[i] = map[string]string{
			"name": col,
			"type": "string",
		}
	}

	// Generate mapping config with snake_case destinations
	mappingConfig := make([]map[string]string, len(columns))
	for i, col := range columns {
		var destination string
		switch col {
		case entityIDColumn:
			destination = "EntityID"
		case amountColumn:
			destination = "Amount"
		default:
			// Convert to snake_case
			destination = strings.ToLower(strings.ReplaceAll(col, " ", "_"))
			destination = strings.ReplaceAll(destination, "-", "_")
		}

		mappingConfig[i] = map[string]string{
			"value":       "",
			"source":      col,
			"destination": destination,
		}
	}

	payload := map[string]interface{}{
		"name": name,
		"config": map[string]interface{}{
			"is_internal":       false,
			"unique_keys":       []string{"EntityID"},
			"allow_upload":      false,
			"ingest_to_db":      true,
			"skip_top_rows":     0,
			"split_file_basis":  "",
			"report_enrichment": true,
			"sub_source_config": map[string]interface{}{
				"amount_key":                 "",
				"aggregate_config":           nil,
				"grouping_columns":           nil,
				"sub_master_source_id":       "",
				"enable_sub_source_creation": false,
			},
			"row_hash_value_based_split_config": map[string]interface{}{
				"column_joiner":                    "",
				"header_hash_to_master_source_map": nil,
			},
		},
		"source_schema":  sourceSchema,
		"mapping_config": mappingConfig,
	}

	result, err := makeReconSaaSAPICall(ctx, "POST", "/v1/admin-recon-saas/sources/create", payload)
	if err != nil {
		return "", err
	}

	if id, ok := result["id"].(string); ok {
		return id, nil
	}

	return fmt.Sprintf("mock_master_source_%d", time.Now().Unix()), nil
}

// createMerchantSource creates a merchant source via recon-saas API
func createMerchantSource(ctx context.Context, merchantID, masterSourceID, name string) (string, error) {
	payload := map[string]interface{}{
		"name":             name,
		"merchant_id":      merchantID,
		"master_source_id": masterSourceID,
		"config": map[string]interface{}{
			"cc_emails":          nil,
			"bcc_emails":         nil,
			"allow_upload":       true,
			"reporting_emails":   nil,
			"split_file_basis":   "",
			"beam_sftp_push_job": "",
			"row_hash_value_based_split_config": map[string]interface{}{
				"column_joiner":                    "",
				"header_hash_to_master_source_map": nil,
			},
		},
		"source_schema":  nil,
		"mapping_config": nil,
	}

	// Use the string-specific API call function for create_merchant endpoint
	merchantSourceID, err := makeReconSaaSAPICallString(ctx, "POST", "/v1/admin-recon-saas/sources/create_merchant", payload)
	if err != nil {
		return "", err
	}

	// The response should be a string ID (e.g., "R2QTjA6dPvcygE")
	if merchantSourceID != "" {
		return merchantSourceID, nil
	}

	return fmt.Sprintf("mock_merchant_source_%d", time.Now().Unix()), nil
}

// createReconStates creates reconciliation states via recon-saas API
func createReconStates(ctx context.Context, merchantID, source1Name, source2Name string) (map[string]interface{}, error) {
	states := []map[string]interface{}{
		{
			"name":     "Reconciled",
			"priority": 2,
			"remarks":  "success",
		},
		{
			"name":     "Unreconciled",
			"priority": 3,
			"remarks":  "Amount mismatch",
		},
		{
			"name":     "Unreconciled",
			"priority": 3,
			"remarks":  fmt.Sprintf("Record not found in %s", source1Name),
		},
		{
			"name":     "Unreconciled",
			"priority": 3,
			"remarks":  fmt.Sprintf("Record not found in %s", source2Name),
		},
	}

	results := make(map[string]interface{})
	stateNames := []string{"reconciled_state", "amount_mismatch_state", "missing_file1_state", "missing_file2_state"}

	for i, state := range states {
		payload := map[string]interface{}{
			"merchant_id": merchantID,
			"name":        state["name"],
			"priority":    state["priority"],
			"remarks":     state["remarks"],
		}

		result, err := makeReconSaaSAPICall(ctx, "POST", "/v1/admin-recon-saas/recon_state", payload)
		if err != nil {
			return nil, fmt.Errorf("failed to create state %s: %v", stateNames[i], err)
		}

		var stateID string
		if id, ok := result["id"].(string); ok {
			stateID = id
		} else {
			stateID = fmt.Sprintf("mock_recon_state_%d_%d", time.Now().Unix(), i)
		}

		results[stateNames[i]] = map[string]interface{}{
			"recon_state_id": stateID,
			"name":           state["name"],
			"priority":       state["priority"],
			"remarks":        state["remarks"],
			"api_response":   result,
		}
	}

	return results, nil
}

// createReconRules creates reconciliation rules via recon-saas API
func createReconRules(ctx context.Context, merchantID, masterSourceID1, masterSourceID2 string, reconStates map[string]interface{}) (map[string]interface{}, error) {
	// Extract recon state IDs
	getStateID := func(stateName string) string {
		if state, ok := reconStates[stateName].(map[string]interface{}); ok {
			if stateID, ok := state["recon_state_id"].(string); ok {
				return stateID
			}
		}
		return fmt.Sprintf("mock_state_%s", stateName)
	}

	rules := []map[string]interface{}{
		{
			"name":           "Reconciled Transactions",
			"expression":     fmt.Sprintf("%s.EntityID == %s.EntityID && %s.Amount.Equal(%s.Amount)", masterSourceID1, masterSourceID2, masterSourceID1, masterSourceID2),
			"recon_state_id": getStateID("reconciled_state"),
			"type":           "reconciled",
		},
		{
			"name":           "Amount Mismatch Transactions",
			"expression":     fmt.Sprintf("%s.EntityID == %s.EntityID && !%s.Amount.Equal(%s.Amount)", masterSourceID1, masterSourceID2, masterSourceID1, masterSourceID2),
			"recon_state_id": getStateID("amount_mismatch_state"),
			"type":           "amount_mismatch",
		},
		{
			"name":           "Missing Record in File 1",
			"expression":     "NoRecord.Value == true",
			"recon_state_id": getStateID("missing_file1_state"),
			"type":           "missing_record",
		},
		{
			"name":           "Missing Record in File 2",
			"expression":     "NoRecord.Value == true",
			"recon_state_id": getStateID("missing_file2_state"),
			"type":           "missing_record",
		},
	}

	results := make(map[string]interface{})
	ruleNames := []string{"reconciled_rule", "amount_mismatch_rule", "missing_record_rule_file1", "missing_record_rule_file2"}

	for i, rule := range rules {
		payload := map[string]interface{}{
			"merchant_id":    merchantID,
			"name":           rule["name"],
			"expression":     rule["expression"],
			"sources":        []string{masterSourceID1, masterSourceID2},
			"recon_state_id": rule["recon_state_id"],
			"type":           "recon",
			"case_rule":      false,
		}

		result, err := makeReconSaaSAPICall(ctx, "POST", "/v1/admin-recon-saas/rule", payload)
		if err != nil {
			return nil, fmt.Errorf("failed to create rule %s: %v", ruleNames[i], err)
		}

		var ruleID string
		if id, ok := result["id"].(string); ok {
			ruleID = id
		} else {
			ruleID = fmt.Sprintf("mock_rule_%d_%d", time.Now().Unix(), i)
		}

		results[ruleNames[i]] = map[string]interface{}{
			"rule_id":        ruleID,
			"name":           rule["name"],
			"expression":     rule["expression"],
			"recon_state_id": rule["recon_state_id"],
			"sources":        []string{masterSourceID1, masterSourceID2},
			"api_response":   result,
		}
	}

	return results, nil
}

// createReconRulesWithValidation creates reconciliation rules via recon-saas API with validation
func createReconRulesWithValidation(ctx context.Context, merchantID, masterSourceID1, masterSourceID2 string, reconStates map[string]interface{}, validationResult *ValidationResult) (map[string]interface{}, error) {
	// Extract recon state IDs
	getStateID := func(stateName string) string {
		if state, ok := reconStates[stateName].(map[string]interface{}); ok {
			if stateID, ok := state["recon_state_id"].(string); ok {
				return stateID
			}
		}
		return fmt.Sprintf("mock_state_%s", stateName)
	}

	// Use expressions from validation result if available, otherwise use defaults
	reconciledExpr := validationResult.RuleExpressions["reconciled"]
	amountMismatchExpr := validationResult.RuleExpressions["amount_mismatch"]
	missingRecordExpr := validationResult.RuleExpressions["missing_record"]

	if reconciledExpr == "" {
		reconciledExpr = fmt.Sprintf("%s.EntityID == %s.EntityID && %s.Amount.Equal(%s.Amount)", masterSourceID1, masterSourceID2, masterSourceID1, masterSourceID2)
	}
	if amountMismatchExpr == "" {
		amountMismatchExpr = fmt.Sprintf("%s.EntityID == %s.EntityID && !%s.Amount.Equal(%s.Amount)", masterSourceID1, masterSourceID2, masterSourceID1, masterSourceID2)
	}
	if missingRecordExpr == "" {
		missingRecordExpr = "NoRecord.Value == true"
	}

	rules := []map[string]interface{}{
		{
			"name":           "Reconciled Transactions",
			"expression":     reconciledExpr,
			"recon_state_id": getStateID("reconciled_state"),
			"type":           "reconciled",
		},
		{
			"name":           "Amount Mismatch Transactions",
			"expression":     amountMismatchExpr,
			"recon_state_id": getStateID("amount_mismatch_state"),
			"type":           "amount_mismatch",
		},
		{
			"name":           "Missing Record in File 1",
			"expression":     missingRecordExpr,
			"recon_state_id": getStateID("missing_file1_state"),
			"type":           "missing_record",
		},
		{
			"name":           "Missing Record in File 2",
			"expression":     missingRecordExpr,
			"recon_state_id": getStateID("missing_file2_state"),
			"type":           "missing_record",
		},
	}

	results := make(map[string]interface{})
	ruleNames := []string{"reconciled_rule", "amount_mismatch_rule", "missing_record_rule_file1", "missing_record_rule_file2"}

	for i, rule := range rules {
		payload := map[string]interface{}{
			"merchant_id":    merchantID,
			"name":           rule["name"],
			"expression":     rule["expression"],
			"sources":        []string{masterSourceID1, masterSourceID2},
			"recon_state_id": rule["recon_state_id"],
			"type":           "recon",
			"case_rule":      false,
		}

		result, err := makeReconSaaSAPICall(ctx, "POST", "/v1/admin-recon-saas/rule", payload)
		if err != nil {
			return nil, fmt.Errorf("failed to create rule %s: %v", ruleNames[i], err)
		}

		var ruleID string
		if id, ok := result["id"].(string); ok {
			ruleID = id
		} else {
			ruleID = fmt.Sprintf("mock_rule_%d_%d", time.Now().Unix(), i)
		}

		results[ruleNames[i]] = map[string]interface{}{
			"rule_id":          ruleID,
			"name":             rule["name"],
			"expression":       rule["expression"],
			"recon_state_id":   rule["recon_state_id"],
			"sources":          []string{masterSourceID1, masterSourceID2},
			"validation_mode":  validationResult.Mode,
			"validation_notes": validationResult.ValidationNotes,
			"api_response":     result,
		}
	}

	// Add validation summary to results
	results["validation_summary"] = map[string]interface{}{
		"mode":              validationResult.Mode,
		"approved":          validationResult.Approved,
		"requires_approval": validationResult.RequiresApproval,
		"validation_notes":  validationResult.ValidationNotes,
		"rule_expressions":  validationResult.RuleExpressions,
	}

	return results, nil
}

// createLookup creates a lookup configuration via recon-saas API
func createLookup(ctx context.Context, merchantID, source1Name, source2Name string) (string, error) {
	payload := map[string]interface{}{
		"name":        fmt.Sprintf("Entity Lookup for %s and %s", source1Name, source2Name),
		"merchant_id": merchantID,
		"config": []map[string]interface{}{
			{
				"source":  "record_internal",
				"columns": []string{"EntityID"},
			},
		},
	}

	result, err := makeReconSaaSAPICall(ctx, "POST", "/v1/admin-recon-saas/lookup", payload)
	if err != nil {
		return "", err
	}

	if id, ok := result["id"].(string); ok {
		return id, nil
	}

	return fmt.Sprintf("mock_lookup_%d", time.Now().Unix()), nil
}

// createMasterReconProcess creates a master reconciliation process via recon-saas API
func createMasterReconProcess(ctx context.Context, source1Name, source2Name, lookupID, masterSourceID1, masterSourceID2 string, ruleIDs []string, source1Columns, source2Columns, source1EntityID, source2EntityID, source1Amount, source2Amount string) (string, error) {
	processName := fmt.Sprintf("%s to %s Reconciliation", source1Name, source2Name)
	productID := fmt.Sprintf("%s_%s",
		strings.ToUpper(strings.ReplaceAll(source1Name[:3], " ", "")),
		strings.ToUpper(strings.ReplaceAll(source2Name[:3], " ", "")))

	// Parse column arrays
	var columns1, columns2 []string
	if err := json.Unmarshal([]byte(source1Columns), &columns1); err != nil {
		return "", fmt.Errorf("invalid source1_columns JSON: %v", err)
	}
	if err := json.Unmarshal([]byte(source2Columns), &columns2); err != nil {
		return "", fmt.Errorf("invalid source2_columns JSON: %v", err)
	}

	// Generate frontend columns (union of all columns from both files)
	frontendCols := make([]string, 0)
	seenColumns := make(map[string]bool)

	// Add columns from source 1
	for _, col := range columns1 {
		if !seenColumns[col] {
			frontendCols = append(frontendCols, col)
			seenColumns[col] = true
		}
	}

	// Add columns from source 2
	for _, col := range columns2 {
		if !seenColumns[col] {
			frontendCols = append(frontendCols, col)
			seenColumns[col] = true
		}
	}

	// Generate column mappings for source 1
	source1ColumnMap := make([]map[string]string, 0)
	for _, col := range columns1 {
		var destination string
		switch col {
		case source1EntityID:
			destination = "EntityID"
		case source1Amount:
			destination = "Amount"
		default:
			// Convert to snake_case
			destination = strings.ToLower(strings.ReplaceAll(col, " ", "_"))
			destination = strings.ReplaceAll(destination, "-", "_")
		}

		source1ColumnMap = append(source1ColumnMap, map[string]string{
			"report_column": col,
			"source_column": destination,
		})
	}

	// Generate column mappings for source 2
	source2ColumnMap := make([]map[string]string, 0)
	for _, col := range columns2 {
		var destination string
		switch col {
		case source2EntityID:
			destination = "EntityID"
		case source2Amount:
			destination = "Amount"
		default:
			// Convert to snake_case
			destination = strings.ToLower(strings.ReplaceAll(col, " ", "_"))
			destination = strings.ReplaceAll(destination, "-", "_")
		}

		source2ColumnMap = append(source2ColumnMap, map[string]string{
			"report_column": col,
			"source_column": destination,
		})
	}

	payload := map[string]interface{}{
		"name":       processName,
		"product_id": productID,
		"type":       "Gateway",
		"lookup_config": []map[string]interface{}{
			{
				"config": map[string]string{
					masterSourceID1: lookupID,
				},
				"streaming_source_id": masterSourceID2,
			},
		},
		"rules": map[string]interface{}{
			"rule_ids": ruleIDs,
		},
		"sources":  []string{masterSourceID1, masterSourceID2},
		"sequence": []interface{}{},
		"report_config": map[string]interface{}{
			"frontend_cols": frontendCols,
			"source_report_config": []map[string]interface{}{
				{
					"column_map":       source1ColumnMap,
					"master_source_id": masterSourceID1,
				},
				{
					"column_map":       source2ColumnMap,
					"master_source_id": masterSourceID2,
				},
			},
		},
	}

	result, err := makeReconSaaSAPICall(ctx, "POST", "/v1/admin-recon-saas/recon_process/master", payload)
	if err != nil {
		return "", err
	}

	if id, ok := result["id"].(string); ok {
		return id, nil
	}

	return fmt.Sprintf("mock_master_recon_process_%d", time.Now().Unix()), nil
}

// createMerchantReconProcess creates a merchant reconciliation process via recon-saas API
func createMerchantReconProcess(ctx context.Context, merchantID, masterReconProcessID, merchantSourceID1, merchantSourceID2 string) (string, error) {
	payload := map[string]interface{}{
		"merchant_id":             merchantID,
		"master_recon_process_id": masterReconProcessID,
		"sources":                 []string{merchantSourceID1, merchantSourceID2},
	}

	result, err := makeReconSaaSAPICall(ctx, "POST", "/v1/admin-recon-saas/recon_process/merchant", payload)
	if err != nil {
		return "", err
	}

	if id, ok := result["id"].(string); ok {
		return id, nil
	}

	return fmt.Sprintf("mock_merchant_recon_process_%d", time.Now().Unix()), nil
}

// extractStateIDs extracts recon state IDs from recon states map
func extractStateIDs(reconStates map[string]interface{}) map[string]string {
	stateIDs := make(map[string]string)

	for stateName, stateData := range reconStates {
		if state, ok := stateData.(map[string]interface{}); ok {
			if stateID, ok := state["recon_state_id"].(string); ok {
				stateIDs[stateName] = stateID
			}
		}
	}

	return stateIDs
}

// extractRuleIDs extracts rule IDs from rules map
func extractRuleIDs(rules map[string]interface{}) map[string]string {
	ruleIDs := make(map[string]string)

	for ruleName, ruleData := range rules {
		if rule, ok := ruleData.(map[string]interface{}); ok {
			if ruleID, ok := rule["rule_id"].(string); ok {
				ruleIDs[ruleName] = ruleID
			}
		}
	}

	return ruleIDs
}

// Helper functions for aggregation tool

// contains checks if a string slice contains a specific string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// performAggregationAnalysis analyzes the file and performs aggregation logic
func performAggregationAnalysis(filePath, fileType, groupingColumn1, groupingColumn2, aggregationColumn, aggregationFunction string) (map[string]interface{}, error) {
	var records [][]string
	var err error

	// Read file based on type
	if fileType == "csv" {
		records, err = readCSVFile(filePath)
	} else if fileType == "excel" {
		records, err = readExcelFile(filePath)
	} else {
		return nil, fmt.Errorf("unsupported file type: %s", fileType)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to read file: %v", err)
	}

	if len(records) < 2 {
		return nil, fmt.Errorf("file must have at least a header and one data row")
	}

	// Get column indices
	header := records[0]
	groupingCol1Index := -1
	groupingCol2Index := -1
	aggregationColIndex := -1

	for i, col := range header {
		if col == groupingColumn1 {
			groupingCol1Index = i
		}
		if col == groupingColumn2 {
			groupingCol2Index = i
		}
		if col == aggregationColumn {
			aggregationColIndex = i
		}
	}

	if groupingCol1Index == -1 || groupingCol2Index == -1 || aggregationColIndex == -1 {
		return nil, fmt.Errorf("one or more required columns not found in file")
	}

	// Group data by the two grouping columns
	groupedData := make(map[string][]float64)
	groupKeys := make([]string, 0)

	for i := 1; i < len(records); i++ {
		row := records[i]
		maxIndex := groupingCol1Index
		if groupingCol2Index > maxIndex {
			maxIndex = groupingCol2Index
		}
		if aggregationColIndex > maxIndex {
			maxIndex = aggregationColIndex
		}
		if len(row) <= maxIndex {
			continue // Skip incomplete rows
		}

		groupKey := fmt.Sprintf("%s|%s", row[groupingCol1Index], row[groupingCol2Index])

		// Parse aggregation value
		value, err := strconv.ParseFloat(row[aggregationColIndex], 64)
		if err != nil {
			continue // Skip rows with invalid numeric values
		}

		if _, exists := groupedData[groupKey]; !exists {
			groupedData[groupKey] = make([]float64, 0)
			groupKeys = append(groupKeys, groupKey)
		}
		groupedData[groupKey] = append(groupedData[groupKey], value)
	}

	// Perform aggregation
	aggregationResults := make([]map[string]interface{}, 0)
	totalGroups := len(groupKeys)

	for _, groupKey := range groupKeys {
		values := groupedData[groupKey]
		keys := strings.Split(groupKey, "|")

		var aggregatedValue float64
		switch aggregationFunction {
		case "sum":
			aggregatedValue = sum(values)
		case "count":
			aggregatedValue = float64(len(values))
		case "avg":
			aggregatedValue = average(values)
		case "min":
			aggregatedValue = min(values)
		case "max":
			aggregatedValue = max(values...)
		default:
			return nil, fmt.Errorf("unsupported aggregation function: %s", aggregationFunction)
		}

		aggregationResults = append(aggregationResults, map[string]interface{}{
			"group_key":             groupKey,
			"grouping_column_1":     keys[0],
			"grouping_column_2":     keys[1],
			"aggregated_value":      aggregatedValue,
			"original_record_count": len(values),
			"original_values":       values,
		})
	}

	return map[string]interface{}{
		"total_groups":         totalGroups,
		"aggregation_function": aggregationFunction,
		"grouping_columns":     []string{groupingColumn1, groupingColumn2},
		"aggregation_column":   aggregationColumn,
		"aggregation_results":  aggregationResults,
		"summary": map[string]interface{}{
			"total_records_processed":   len(records) - 1,
			"groups_created":            totalGroups,
			"average_records_per_group": float64(len(records)-1) / float64(totalGroups),
		},
	}, nil
}

// updateMasterSourceWithAggregation updates master source with aggregation configuration
func updateMasterSourceWithAggregation(ctx context.Context, masterSourceID, groupingColumn1, groupingColumn2, aggregationColumn, aggregationFunction string) error {
	// Send the complete config structure to match the API response format
	payload := map[string]interface{}{
		"config": map[string]interface{}{
			"skip_top_rows":    0,
			"ingest_to_db":     true,
			"allow_upload":     false,
			"unique_keys":      []string{"EntityID"},
			"is_internal":      false,
			"split_file_basis": "",
			"row_hash_value_based_split_config": map[string]interface{}{
				"header_hash_to_master_source_map": nil,
				"column_joiner":                    "",
			},
			"sub_source_config": map[string]interface{}{
				"aggregate_config": []map[string]interface{}{
					{
						"column":        aggregationColumn,
						"value":         aggregationFunction,
						"operation":     "group_by",
						"operation_val": "1",
					},
				},
				"grouping_columns":           []string{groupingColumn1, groupingColumn2},
				"sub_master_source_id":       "",
				"enable_sub_source_creation": true,
				"amount_key":                 aggregationColumn,
			},
			"extract_distinct_config":  nil,
			"report_enrichment":        true,
			"skip_bottom_rows":         0,
			"skip_row_func":            "",
			"is_header_missing":        false,
			"add_file_details_source":  false,
			"get_file_cycle_from_name": false,
			"add_file_cycle":           false,
			"file_delimiter":           "",
			"get_file_date_from_name": map[string]interface{}{
				"regex_pattern": "",
				"date_format":   "",
			},
			"add_file_date": false,
		},
	}

	fmt.Printf("🔧 DEBUG: Updating Master Source %s with payload: %+v\n", masterSourceID, payload)
	response, err := makeReconSaaSAPICall(ctx, "PATCH", fmt.Sprintf("/v1/admin-recon-saas/sources/update/%s", masterSourceID), payload)
	if err != nil {
		fmt.Printf("❌ DEBUG: Master Source update failed: %v\n", err)
		return err
	}
	fmt.Printf("✅ DEBUG: Master Source update response: %+v\n", response)
	return nil
}

// updateLookupWithAggregation updates lookup with aggregation enablement
func updateLookupWithAggregation(ctx context.Context, lookupID string, enableAggregation bool) error {
	payload := map[string]interface{}{
		"config": []map[string]interface{}{
			{
				"source":  "record_internal",
				"Columns": []string{"EntityID"}, // Fixed: Use "Columns" with capital C
				"aggregation": map[string]interface{}{
					"enabled":    enableAggregation,
					"conditions": nil,
				},
				"advanced_config": map[string]interface{}{
					"enabled":     enableAggregation,
					"cols_config": nil,
				},
			},
		},
	}

	fmt.Printf("🔧 DEBUG: Updating Lookup %s with payload: %+v\n", lookupID, payload)
	response, err := makeReconSaaSAPICall(ctx, "PATCH", fmt.Sprintf("/v1/admin-recon-saas/lookup/%s", lookupID), payload)
	if err != nil {
		fmt.Printf("❌ DEBUG: Lookup update failed: %v\n", err)
		return err
	}
	fmt.Printf("✅ DEBUG: Lookup update response: %+v\n", response)
	return nil
}

// updateMerchantReconProcessWithAggregation updates merchant recon process with aggregation configuration
func updateMerchantReconProcessWithAggregation(ctx context.Context, merchantReconProcessID, groupingColumn1, groupingColumn2, aggregationColumn, aggregationFunction string) error {
	payload := map[string]interface{}{
		"report_config": map[string]interface{}{
			"source_report_config": nil,
			"frontend_cols":        nil,
			"reporting_sources":    nil,
			"report_enrichment":    nil,
			"sub_source_report_config": map[string]interface{}{
				"enabled":              true,
				"source_report_config": nil,
				"recon_status_key":     "",
				"frontend_cols":        nil,
			},
			"report_upload": map[string]interface{}{
				"enabled":        false,
				"s3_bucket":      "",
				"s3_path":        "",
				"s3_file_prefix": "",
			},
			"skip_status":                  false,
			"skip_rows":                    false,
			"skip_rows_recon_state_ids":    nil,
			"report_format":                nil,
			"report_channel":               nil,
			"custom_reports":               nil,
			"advanced_reporting_config_id": "",
		},
		"aggregation_config": map[string]interface{}{
			"enabled":              true,
			"grouping_columns":     []string{groupingColumn1, groupingColumn2},
			"aggregation_column":   aggregationColumn,
			"aggregation_function": aggregationFunction,
		},
	}

	fmt.Printf("🔧 DEBUG: Updating Merchant Recon Process %s with payload: %+v\n", merchantReconProcessID, payload)
	response, err := makeReconSaaSAPICall(ctx, "PATCH", fmt.Sprintf("/v1/admin-recon-saas/recon_process/merchant/%s", merchantReconProcessID), payload)
	if err != nil {
		fmt.Printf("❌ DEBUG: Merchant Recon Process update failed: %v\n", err)
		return err
	}
	fmt.Printf("✅ DEBUG: Merchant Recon Process update response: %+v\n", response)
	return nil
}

// runReconciliation runs reconciliation process to test aggregation
func runReconciliation(ctx context.Context, merchantReconProcessID string) (map[string]interface{}, error) {
	payload := map[string]interface{}{
		"process_id":   merchantReconProcessID,
		"trigger_type": "manual",
	}

	result, err := makeReconSaaSAPICall(ctx, "POST", fmt.Sprintf("/v1/admin-recon-saas/recon_process/merchant/%s/run", merchantReconProcessID), payload)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"status":       "success",
		"process_id":   merchantReconProcessID,
		"execution_id": result["execution_id"],
		"message":      "Reconciliation process started successfully",
	}, nil
}

// Utility functions for aggregation
func sum(values []float64) float64 {
	total := 0.0
	for _, v := range values {
		total += v
	}
	return total
}

func average(values []float64) float64 {
	if len(values) == 0 {
		return 0.0
	}
	return sum(values) / float64(len(values))
}

func min(values []float64) float64 {
	if len(values) == 0 {
		return 0.0
	}
	minVal := values[0]
	for _, v := range values {
		if v < minVal {
			minVal = v
		}
	}
	return minVal
}

func max(values ...float64) float64 {
	if len(values) == 0 {
		return 0.0
	}
	maxVal := values[0]
	for _, v := range values {
		if v > maxVal {
			maxVal = v
		}
	}
	return maxVal
}

// readCSVFile reads a CSV file and returns records
func readCSVFile(filePath string) ([][]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	return records, nil
}

// readCSVHeaders reads only the header row from a CSV file
func readCSVHeaders(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	if len(records) == 0 {
		return nil, fmt.Errorf("CSV file %s is empty", filePath)
	}

	return records[0], nil
}

// readExcelFile reads an Excel file and returns records
func readExcelFile(filePath string) ([][]string, error) {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// Get the first sheet
	sheetName := f.GetSheetName(0)
	if sheetName == "" {
		return nil, fmt.Errorf("no sheets found in Excel file")
	}

	rows, err := f.GetRows(sheetName)
	if err != nil {
		return nil, err
	}

	// Convert to [][]string format
	records := make([][]string, len(rows))
	for i, row := range rows {
		records[i] = row
	}

	return records, nil
}
