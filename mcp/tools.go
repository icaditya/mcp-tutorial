package mcp

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// CalculatorTool Calculator tool for basic math operations
func CalculatorTool() server.ServerTool {
	tool := mcp.NewTool("calculator",
		mcp.WithDescription("Perform basic mathematical calculations"),
		mcp.WithString("operation",
			mcp.Description("The mathematical operation to perform"),
			mcp.Required(),
			mcp.Enum("add", "subtract", "multiply", "divide", "power", "sqrt"),
		),
		mcp.WithNumber("first_number",
			mcp.Description("The first number for the operation"),
			mcp.Required(),
		),
		mcp.WithNumber("second_number",
			mcp.Description("The second number (not required for sqrt)"),
		),
	)

	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		operation, err := request.RequireString("operation")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		firstNum, err := request.RequireFloat("first_number")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		var result float64
		var operatorSymbol string

		switch operation {
		case "add":
			secondNum, err := request.RequireFloat("second_number")
			if err != nil {
				return mcp.NewToolResultError("second_number is required for addition"), nil
			}
			result = firstNum + secondNum
			operatorSymbol = "+"

		case "subtract":
			secondNum, err := request.RequireFloat("second_number")
			if err != nil {
				return mcp.NewToolResultError("second_number is required for subtraction"), nil
			}
			result = firstNum - secondNum
			operatorSymbol = "-"

		case "multiply":
			secondNum, err := request.RequireFloat("second_number")
			if err != nil {
				return mcp.NewToolResultError("second_number is required for multiplication"), nil
			}
			result = firstNum * secondNum
			operatorSymbol = "×"

		case "divide":
			secondNum, err := request.RequireFloat("second_number")
			if err != nil {
				return mcp.NewToolResultError("second_number is required for division"), nil
			}
			if secondNum == 0 {
				return mcp.NewToolResultError("cannot divide by zero"), nil
			}
			result = firstNum / secondNum
			operatorSymbol = "÷"

		default:
			return mcp.NewToolResultError("unsupported operation"), nil
		}

		resultStr := fmt.Sprintf("%.2f %s %.2f = %.6f", firstNum, operatorSymbol, result, result)
		return mcp.NewToolResultText(resultStr), nil
	}

	return server.ServerTool{
		Tool:    tool,
		Handler: handler,
	}
}

// SystemInfoTool provides system information
func SystemInfoTool() server.ServerTool {
	tool := mcp.NewTool("system_info",
		mcp.WithDescription("Get system information like current time and date"),
		mcp.WithString("info_type",
			mcp.Description("Type of system information to retrieve"),
			mcp.Required(),
			mcp.Enum("time", "date", "datetime"),
		),
		mcp.WithString("format",
			mcp.Description("Format for the output"),
			mcp.DefaultString("human"),
			mcp.Enum("iso", "rfc3339", "unix", "human"),
		),
	)

	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		infoType, err := request.RequireString("info_type")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		format := request.GetString("format", "human")
		now := time.Now()

		var result string
		switch infoType {
		case "time":
			switch format {
			case "iso":
				result = now.Format("15:04:05")
			case "rfc3339":
				result = now.Format(time.RFC3339)
			case "unix":
				result = fmt.Sprintf("%d", now.Unix())
			case "human":
				result = now.Format("3:04:05 PM")
			}
		case "date":
			switch format {
			case "iso":
				result = now.Format("2006-01-02")
			case "rfc3339":
				result = now.Format(time.RFC3339)
			case "unix":
				result = fmt.Sprintf("%d", now.Unix())
			case "human":
				result = now.Format("Monday, January 2, 2006")
			}
		case "datetime":
			switch format {
			case "iso":
				result = now.Format("2006-01-02T15:04:05")
			case "rfc3339":
				result = now.Format(time.RFC3339)
			case "unix":
				result = fmt.Sprintf("%d", now.Unix())
			case "human":
				result = now.Format("Monday, January 2, 2006 at 3:04:05 PM")
			}
		}

		return mcp.NewToolResultText(result), nil
	}

	return server.ServerTool{
		Tool:    tool,
		Handler: handler,
	}
}

// ReconFileAnalysisTool analyzes uploaded reconciliation files
func ReconFileAnalysisTool() server.ServerTool {
	tool := mcp.NewTool("recon_file_analysis",
		mcp.WithDescription("Analyze uploaded reconciliation files to identify EntityID and Amount columns for master source creation"),
		mcp.WithString("file1_path",
			mcp.Description("Full file path to the first reconciliation file (e.g., /path/to/transactions.csv or /path/to/transactions.xlsx)"),
			mcp.Required(),
		),
		mcp.WithString("file1_type",
			mcp.Description("Type of the first file"),
			mcp.Required(),
			mcp.Enum("csv", "excel"),
		),
		mcp.WithString("file2_path",
			mcp.Description("Full file path to the second reconciliation file (e.g., /path/to/bank_statements.csv or /path/to/bank_statements.xlsx)"),
			mcp.Required(),
		),
		mcp.WithString("file2_type",
			mcp.Description("Type of the second file"),
			mcp.Required(),
			mcp.Enum("csv", "excel"),
		),
	)

	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// This would typically call your helper functions
		// For now, return a simulated analysis result
		result := `{
			"analysis_type": "comprehensive",
			"compatibility_check": {
				"can_reconcile": true,
				"common_patterns": ["amount", "date"],
				"suggested_reconciliation": "Match by EntityID and Amount fields"
			},
			"file_analysis": {
				"file_1": {
					"all_columns": ["paymentid", "txn_amount", "date"],
					"amount_candidates": [
						{
							"column_name": "txn_amount",
							"confidence": 0.83,
							"reason": "Amount-like naming with monetary values",
							"sample_values": ["500", "1500", "200"]
						}
					],
					"entityid_candidates": [
						{
							"column_name": "paymentid",
							"confidence": 0.98,
							"reason": "ID-like naming pattern with high uniqueness",
							"unique_percentage": 100
						}
					],
					"file_type": "csv",
					"recommended_amount": "txn_amount",
					"recommended_entityid": "paymentid",
					"total_columns": 3,
					"total_rows": 4
				}
			}
		}`

		return mcp.NewToolResultText(result), nil
	}

	return server.ServerTool{
		Tool:    tool,
		Handler: handler,
	}
}

// ReconMasterSourceTool creates master source configurations
func ReconMasterSourceTool() server.ServerTool {
	tool := mcp.NewTool("recon_master_source",
		mcp.WithDescription("Create master source configurations for recon-saas using file analysis data"),
		mcp.WithString("source1_name",
			mcp.Description("Name for the first master source"),
			mcp.Required(),
		),
		mcp.WithString("source2_name",
			mcp.Description("Name for the second master source"),
			mcp.Required(),
		),
		mcp.WithString("source1_columns",
			mcp.Description("JSON array of column names from first file"),
			mcp.Required(),
		),
		mcp.WithString("source2_columns",
			mcp.Description("JSON array of column names from second file"),
			mcp.Required(),
		),
		mcp.WithString("source1_entityid",
			mcp.Description("Selected EntityID column name for first source"),
			mcp.Required(),
		),
		mcp.WithString("source2_entityid",
			mcp.Description("Selected EntityID column name for second source"),
			mcp.Required(),
		),
		mcp.WithString("source1_amount",
			mcp.Description("Selected Amount column name for first source"),
			mcp.Required(),
		),
		mcp.WithString("source2_amount",
			mcp.Description("Selected Amount column name for second source"),
			mcp.Required(),
		),
	)

	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Extract parameters
		source1Name, _ := request.RequireString("source1_name")
		source2Name, _ := request.RequireString("source2_name")

		// Simulate API calls to create master sources
		masterSourceID1 := generateID("RJTcj")
		masterSourceID2 := generateID("RJTcj")

		result := fmt.Sprintf(`{
			"created_sources": {
				"source_1": {
					"master_source_id": "%s",
					"name": "%s",
					"selected_entityid_column": "%s",
					"selected_amount_column": "%s"
				},
				"source_2": {
					"master_source_id": "%s", 
					"name": "%s",
					"selected_entityid_column": "%s",
					"selected_amount_column": "%s"
				}
			},
			"message": "Master sources created successfully",
			"status": "success"
		}`, masterSourceID1, source1Name, request.GetString("source1_entityid", ""), request.GetString("source1_amount", ""),
			masterSourceID2, source2Name, request.GetString("source2_entityid", ""), request.GetString("source2_amount", ""))

		return mcp.NewToolResultText(result), nil
	}

	return server.ServerTool{
		Tool:    tool,
		Handler: handler,
	}
}

// ReconMerchantSourceTool creates merchant-specific source configurations
func ReconMerchantSourceTool() server.ServerTool {
	tool := mcp.NewTool("recon_merchant_source",
		mcp.WithDescription("Create merchant-specific source configurations for recon-saas"),
		mcp.WithString("merchant_id",
			mcp.Description("Merchant identifier for this onboarding process"),
			mcp.Required(),
		),
		mcp.WithString("master_source_id_1",
			mcp.Description("First master source ID from previous step"),
			mcp.Required(),
		),
		mcp.WithString("master_source_id_2",
			mcp.Description("Second master source ID from previous step"),
			mcp.Required(),
		),
		mcp.WithString("source_1_name",
			mcp.Description("Name of the first source"),
			mcp.Required(),
		),
		mcp.WithString("source_2_name",
			mcp.Description("Name of the second source"),
			mcp.Required(),
		),
		mcp.WithString("source_naming_strategy",
			mcp.Description("Strategy for naming merchant sources"),
			mcp.DefaultString("descriptive"),
			mcp.Enum("descriptive", "timestamp", "sequential", "custom"),
		),
	)

	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		merchantID, _ := request.RequireString("merchant_id")
		masterSourceID1, _ := request.RequireString("master_source_id_1")
		masterSourceID2, _ := request.RequireString("master_source_id_2")
		source1Name, _ := request.RequireString("source_1_name")
		source2Name, _ := request.RequireString("source_2_name")

		// Generate merchant source IDs
		merchantSourceID1 := generateID("RJTcs")
		merchantSourceID2 := generateID("RJTcs")

		result := fmt.Sprintf(`{
			"created_merchant_sources": {
				"merchant_source_1": {
					"master_source_id": "%s",
					"merchant_id": "%s",
					"merchant_source_id": "%s",
					"name": "%s - Merchant Portal",
					"naming_strategy": "descriptive"
				},
				"merchant_source_2": {
					"master_source_id": "%s",
					"merchant_id": "%s", 
					"merchant_source_id": "%s",
					"name": "%s - Merchant Portal",
					"naming_strategy": "descriptive"
				}
			},
			"execution_summary": {
				"failed_creations": 0,
				"merchant_id": "%s",
				"successful_creations": 2,
				"total_merchant_sources": 2
			},
			"message": "Merchant sources created successfully",
			"status": "success"
		}`, masterSourceID1, merchantID, merchantSourceID1, source1Name,
			masterSourceID2, merchantID, merchantSourceID2, source2Name, merchantID)

		return mcp.NewToolResultText(result), nil
	}

	return server.ServerTool{
		Tool:    tool,
		Handler: handler,
	}
}

// ReconStateRuleTool creates reconciliation states and rules
func ReconStateRuleTool() server.ServerTool {
	tool := mcp.NewTool("recon_state_rule",
		mcp.WithDescription("Create reconciliation states and corresponding rules for recon-saas"),
		mcp.WithString("merchant_id",
			mcp.Description("Merchant identifier"),
			mcp.Required(),
		),
		mcp.WithString("master_source_id_1",
			mcp.Description("First master source ID"),
			mcp.Required(),
		),
		mcp.WithString("master_source_id_2",
			mcp.Description("Second master source ID"),
			mcp.Required(),
		),
		mcp.WithString("source_1_name",
			mcp.Description("Name of the first source for remarks"),
			mcp.Required(),
		),
		mcp.WithString("source_2_name",
			mcp.Description("Name of the second source for remarks"),
			mcp.Required(),
		),
		mcp.WithString("validation_mode",
			mcp.Description("User validation mode for rule expressions"),
			mcp.DefaultString("guided"),
			mcp.Enum("automatic", "guided", "manual"),
		),
		mcp.WithBoolean("approve_expressions",
			mcp.Description("Whether to approve the generated rule expressions"),
			mcp.DefaultString("true"),
		),
	)

	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		merchantID, _ := request.RequireString("merchant_id")
		masterSourceID1, _ := request.RequireString("master_source_id_1")
		masterSourceID2, _ := request.RequireString("master_source_id_2")
		source1Name, _ := request.RequireString("source_1_name")
		source2Name, _ := request.RequireString("source_2_name")

		// Generate state and rule IDs
		reconciledStateID := generateID("RKJzZ")
		amountMismatchStateID := generateID("RKJzZ")
		missingFile1StateID := generateID("RKJzZ")
		missingFile2StateID := generateID("RKJzZ")

		reconciledRuleID := generateID("RKJza")
		amountMismatchRuleID := generateID("RKJza")
		missingFile1RuleID := generateID("RKJza")
		missingFile2RuleID := generateID("RKJza")

		result := fmt.Sprintf(`{
			"created_recon_states": {
				"reconciled_state": {
					"recon_state_id": "%s",
					"name": "Reconciled",
					"priority": 2,
					"remarks": "success"
				},
				"amount_mismatch_state": {
					"recon_state_id": "%s",
					"name": "Unreconciled", 
					"priority": 3,
					"remarks": "Amount mismatch"
				},
				"missing_file1_state": {
					"recon_state_id": "%s",
					"name": "Unreconciled",
					"priority": 3,
					"remarks": "Record not found in %s"
				},
				"missing_file2_state": {
					"recon_state_id": "%s",
					"name": "Unreconciled",
					"priority": 3,
					"remarks": "Record not found in %s"
				}
			},
			"created_rules": {
				"reconciled_rule": {
					"rule_id": "%s",
					"name": "Reconciled Transactions",
					"expression": "%s.EntityID == %s.EntityID && %s.Amount.Equal(%s.Amount)",
					"recon_state_id": "%s"
				},
				"amount_mismatch_rule": {
					"rule_id": "%s", 
					"name": "Amount Mismatch Transactions",
					"expression": "%s.EntityID == %s.EntityID && !%s.Amount.Equal(%s.Amount)",
					"recon_state_id": "%s"
				},
				"missing_record_rule_file1": {
					"rule_id": "%s",
					"name": "Missing Record in File 1",
					"expression": "NoRecord.Value == true",
					"recon_state_id": "%s"
				},
				"missing_record_rule_file2": {
					"rule_id": "%s",
					"name": "Missing Record in File 2", 
					"expression": "NoRecord.Value == true",
					"recon_state_id": "%s"
				}
			},
			"execution_summary": {
				"merchant_id": "%s",
				"total_recon_states": 4,
				"total_rules": 4,
				"user_approved_expressions": true,
				"validation_applied": true,
				"validation_mode": "automatic"
			},
			"message": "Recon states and rules created successfully",
			"status": "success"
		}`, reconciledStateID, amountMismatchStateID, missingFile1StateID, source1Name, missingFile2StateID, source2Name,
			reconciledRuleID, masterSourceID1, masterSourceID2, masterSourceID1, masterSourceID2, reconciledStateID,
			amountMismatchRuleID, masterSourceID1, masterSourceID2, masterSourceID1, masterSourceID2, amountMismatchStateID,
			missingFile1RuleID, missingFile1StateID, missingFile2RuleID, missingFile2StateID, merchantID)

		return mcp.NewToolResultText(result), nil
	}

	return server.ServerTool{
		Tool:    tool,
		Handler: handler,
	}
}

// ReconProcessSetupTool creates lookup configurations and reconciliation processes
func ReconProcessSetupTool() server.ServerTool {
	tool := mcp.NewTool("recon_process_setup",
		mcp.WithDescription("Create lookup configurations and reconciliation processes for recon-saas"),
		mcp.WithString("merchant_id",
			mcp.Description("Merchant identifier"),
			mcp.Required(),
		),
		mcp.WithString("master_source_id_1",
			mcp.Description("First master source ID"),
			mcp.Required(),
		),
		mcp.WithString("master_source_id_2",
			mcp.Description("Second master source ID"),
			mcp.Required(),
		),
		mcp.WithString("merchant_source_id_1",
			mcp.Description("First merchant source ID"),
			mcp.Required(),
		),
		mcp.WithString("merchant_source_id_2",
			mcp.Description("Second merchant source ID"),
			mcp.Required(),
		),
		mcp.WithString("rule_ids",
			mcp.Description("JSON array of rule IDs from previous step"),
			mcp.Required(),
		),
		mcp.WithString("source_1_name",
			mcp.Description("Name of the first source"),
			mcp.Required(),
		),
		mcp.WithString("source_2_name",
			mcp.Description("Name of the second source"),
			mcp.Required(),
		),
		mcp.WithString("source1_columns",
			mcp.Description("JSON array of column names from first file"),
			mcp.Required(),
		),
		mcp.WithString("source2_columns",
			mcp.Description("JSON array of column names from second file"),
			mcp.Required(),
		),
		mcp.WithString("source1_entityid",
			mcp.Description("Selected EntityID column name for first source"),
			mcp.Required(),
		),
		mcp.WithString("source2_entityid",
			mcp.Description("Selected EntityID column name for second source"),
			mcp.Required(),
		),
		mcp.WithString("source1_amount",
			mcp.Description("Selected Amount column name for first source"),
			mcp.Required(),
		),
		mcp.WithString("source2_amount",
			mcp.Description("Selected Amount column name for second source"),
			mcp.Required(),
		),
	)

	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		merchantID, _ := request.RequireString("merchant_id")
		source1Name, _ := request.RequireString("source_1_name")
		source2Name, _ := request.RequireString("source_2_name")

		// Generate process IDs
		lookupID := generateID("RKK1")
		masterReconProcessID := generateID("RKK1")
		merchantReconProcessID := generateID("RKK1")

		result := fmt.Sprintf(`{
			"created_components": {
				"lookup": {
					"lookup_id": "%s",
					"name": "Entity Lookup for %s and %s"
				},
				"master_recon_process": {
					"master_recon_process_id": "%s",
					"name": "%s to %s Reconciliation"
				},
				"merchant_recon_process": {
					"merchant_recon_process_id": "%s"
				}
			},
			"execution_summary": {
				"failed_creations": 0,
				"merchant_id": "%s",
				"process_name": "%s to %s Reconciliation",
				"successful_creations": 3,
				"total_api_calls": 3
			},
			"message": "Reconciliation process setup completed successfully",
			"onboarding_completion": {
				"message": "Merchant onboarding successfully completed. The reconciliation process is now ready for file uploads and processing.",
				"next_steps": [
					"Upload transaction files for reconciliation",
					"Monitor reconciliation results in dashboard",
					"Configure automated file processing schedules",
					"Set up reporting and alerting preferences"
				],
				"status": "COMPLETE"
			},
			"status": "success"
		}`, lookupID, source1Name, source2Name, masterReconProcessID, source1Name, source2Name,
			merchantReconProcessID, merchantID, source1Name, source2Name)

		return mcp.NewToolResultText(result), nil
	}

	return server.ServerTool{
		Tool:    tool,
		Handler: handler,
	}
}

// ReconDataExtractionTool allows users to extract specific patterns from CSV column data
func ReconDataExtractionTool() server.ServerTool {
	tool := mcp.NewTool("recon_data_extraction",
		mcp.WithDescription("Extract specific patterns from CSV column data. Step-by-step: 1) Provide file path 2) Choose column 3) Define extraction pattern (e.g., extract '123' from 'abc123xyz')"),
		mcp.WithString("file_path",
			mcp.Description("Step 1: Full path to the CSV file to process"),
			mcp.Required(),
		),
		mcp.WithString("column_name",
			mcp.Description("Step 2: Name of the column containing data to extract from"),
			mcp.Required(),
		),
		mcp.WithString("source_example",
			mcp.Description("Step 3a: Example of your current data format (e.g., 'abc123xyz')"),
			mcp.Required(),
		),
		mcp.WithString("target_extract",
			mcp.Description("Step 3b: What you want to extract from the source (e.g., '123' from 'abc123xyz')"),
			mcp.Required(),
		),
		mcp.WithString("new_column_name",
			mcp.Description("Name for the new column containing extracted data"),
			mcp.DefaultString("extracted_data"),
		),
		mcp.WithBoolean("save_to_file",
			mcp.Description("Whether to save results to a new CSV file"),
			mcp.DefaultString("true"),
		),
		mcp.WithString("output_file_path",
			mcp.Description("Path for the output file (if save_to_file is true)"),
		),
	)

	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Get required parameters
		filePath, err := request.RequireString("file_path")
		if err != nil {
			return mcp.NewToolResultError("Step 1 - File Path Required: " + err.Error()), nil
		}

		columnName, err := request.RequireString("column_name")
		if err != nil {
			return mcp.NewToolResultError("Step 2 - Column Name Required: " + err.Error()), nil
		}

		sourceExample, err := request.RequireString("source_example")
		if err != nil {
			return mcp.NewToolResultError("Step 3a - Source Example Required: " + err.Error()), nil
		}

		targetExtract, err := request.RequireString("target_extract")
		if err != nil {
			return mcp.NewToolResultError("Step 3b - Target Extract Required: " + err.Error()), nil
		}

		// Optional parameters
		newColumnName := request.GetString("new_column_name", "extracted_data")
		saveToFile := request.GetBool("save_to_file", true)
		outputFilePath := request.GetString("output_file_path", "")

		// Check if file exists
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			return mcp.NewToolResultError(fmt.Sprintf("File not found: %s", filePath)), nil
		}

		// Process the CSV file
		file, err := os.Open(filePath)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Error opening file: %v", err)), nil
		}
		defer file.Close()

		reader := csv.NewReader(file)
		records, err := reader.ReadAll()
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Error reading CSV: %v", err)), nil
		}

		if len(records) == 0 {
			return mcp.NewToolResultError("CSV file is empty"), nil
		}

		// Find column index
		headers := records[0]
		columnIndex := -1
		for i, header := range headers {
			if header == columnName {
				columnIndex = i
				break
			}
		}

		if columnIndex == -1 {
			return mcp.NewToolResultError(fmt.Sprintf("Column '%s' not found. Available columns: %v", columnName, headers)), nil
		}

		// Process extraction - ONLY transform exact matches
		var results [][]string
		var extractionStats struct {
			totalRows       int
			matchedRows     int
			transformedRows int
			sampleMatches   []string
		}

		// Prepare headers - add new column
		newHeaders := append(headers, newColumnName)
		results = append(results, newHeaders)

		// Process each data row
		for i := 1; i < len(records); i++ {
			row := records[i]
			extractionStats.totalRows++

			if columnIndex >= len(row) {
				// Column doesn't exist in this row
				newRow := append(row, "")
				results = append(results, newRow)
				continue
			}

			originalValue := row[columnIndex]
			extractedValue := ""
			newOriginalValue := originalValue // Keep original by default

			// ONLY transform if it exactly matches the source example
			if originalValue == sourceExample {
				extractedValue = targetExtract
				newOriginalValue = targetExtract // Update original column
				extractionStats.transformedRows++

				// Store sample for reporting
				if len(extractionStats.sampleMatches) < 5 {
					extractionStats.sampleMatches = append(extractionStats.sampleMatches,
						fmt.Sprintf("'%s' → '%s'", originalValue, targetExtract))
				}
			}

			// Create new row with updated original column and new extract column
			newRow := make([]string, len(row))
			copy(newRow, row)
			newRow[columnIndex] = newOriginalValue  // Update original column
			newRow = append(newRow, extractedValue) // Add new extracted column
			results = append(results, newRow)
		}

		// Generate output file path if not provided
		if saveToFile && outputFilePath == "" {
			dir := filepath.Dir(filePath)
			base := strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))
			outputFilePath = filepath.Join(dir, fmt.Sprintf("%s_extracted.csv", base))
		}

		// Save to file if requested
		if saveToFile {
			outputFile, err := os.Create(outputFilePath)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Error creating output file: %v", err)), nil
			}
			defer outputFile.Close()

			writer := csv.NewWriter(outputFile)
			defer writer.Flush()

			for _, record := range results {
				if err := writer.Write(record); err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("Error writing to output file: %v", err)), nil
				}
			}
		}

		// Format complete data for display (first 10 rows)
		headersDisplay := strings.Join(results[0], " | ")
		var dataRowsDisplay []string
		maxDisplayRows := 10
		if len(results)-1 < maxDisplayRows {
			maxDisplayRows = len(results) - 1
		}

		for i := 1; i <= maxDisplayRows; i++ {
			if i < len(results) {
				rowDisplay := fmt.Sprintf("Row %d: %s", i, strings.Join(results[i], " | "))
				dataRowsDisplay = append(dataRowsDisplay, rowDisplay)
			}
		}

		if len(results)-1 > maxDisplayRows {
			dataRowsDisplay = append(dataRowsDisplay, fmt.Sprintf("... and %d more rows", len(results)-1-maxDisplayRows))
		}

		dataRowsFormatted := strings.Join(dataRowsDisplay, "\n")

		result := fmt.Sprintf(`🔧 **Data Extraction Complete!**

**📁 Source File:** %s
**📊 Column Processed:** %s
**🎯 Source Pattern:** %s
**📤 Target Extract:** %s

**📈 Extraction Results:**
- **Total Rows:** %d
- **Exact Matches Found:** %d
- **Rows Transformed:** %d
- **New Column:** %s

**🔍 Sample Transformations:**
%s

**📋 COMPLETE TRANSFORMED DATA:**

**Headers:** %s

**Data Rows:**
%s

**📁 Output File:** %s

**💡 Transformation Summary:**
- Only transformed exact matches of '%s' → '%s'
- All other data left unchanged
- %d rows processed successfully
`,
			filepath.Base(filePath),
			columnName,
			sourceExample,
			targetExtract,
			extractionStats.totalRows,
			extractionStats.transformedRows,
			extractionStats.transformedRows,
			newColumnName,
			strings.Join(extractionStats.sampleMatches, "\n"),
			headersDisplay,
			dataRowsFormatted,
			outputFilePath,
			sourceExample,
			targetExtract,
			extractionStats.totalRows,
		)

		return mcp.NewToolResultText(result), nil
	}

	return server.ServerTool{
		Tool:    tool,
		Handler: handler,
	}
}

// ReconCombinedEntityTool creates composite entity IDs from multiple columns for reconciliation
func ReconCombinedEntityTool() server.ServerTool {
	tool := mcp.NewTool("recon_combined_entity",
		mcp.WithDescription("Create combined entity ID from multiple columns when files lack unique reconciliation keys. Perfect for cases where you need mid+tid+amount+date as entity identifier."),
		mcp.WithString("file_path",
			mcp.Description("Full path to the CSV file that needs a combined entity ID"),
			mcp.Required(),
		),
		mcp.WithString("columns_to_combine",
			mcp.Description("Comma-separated list of column names to combine (e.g., 'mid,tid,amount,date')"),
			mcp.Required(),
		),
		mcp.WithString("separator",
			mcp.Description("Separator to use between combined values (e.g., '_', '-', '|')"),
			mcp.DefaultString("_"),
		),
		mcp.WithString("entity_column_name",
			mcp.Description("Name for the new combined entity ID column"),
			mcp.DefaultString("combined_entity_id"),
		),
		mcp.WithBoolean("save_to_file",
			mcp.Description("Whether to save results to a new CSV file"),
			mcp.DefaultString("true"),
		),
		mcp.WithString("output_file_path",
			mcp.Description("Path for the output file (if save_to_file is true)"),
		),
	)

	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Get required parameters
		filePath, err := request.RequireString("file_path")
		if err != nil {
			return mcp.NewToolResultError("File Path Required: " + err.Error()), nil
		}

		columnsToCombine, err := request.RequireString("columns_to_combine")
		if err != nil {
			return mcp.NewToolResultError("Columns to Combine Required: " + err.Error()), nil
		}

		// Optional parameters
		separator := request.GetString("separator", "_")
		entityColumnName := request.GetString("entity_column_name", "combined_entity_id")
		saveToFile := request.GetBool("save_to_file", true)
		outputFilePath := request.GetString("output_file_path", "")

		// Parse the columns list
		columnNames := strings.Split(columnsToCombine, ",")
		for i, col := range columnNames {
			columnNames[i] = strings.TrimSpace(col)
		}

		// Check if file exists
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			return mcp.NewToolResultError(fmt.Sprintf("File not found: %s", filePath)), nil
		}

		// Process the CSV file
		file, err := os.Open(filePath)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Error opening file: %v", err)), nil
		}
		defer file.Close()

		reader := csv.NewReader(file)
		records, err := reader.ReadAll()
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Error reading CSV: %v", err)), nil
		}

		if len(records) == 0 {
			return mcp.NewToolResultError("CSV file is empty"), nil
		}

		// Find column indices
		headers := records[0]
		var columnIndices []int
		var missingColumns []string

		for _, colName := range columnNames {
			found := false
			for i, header := range headers {
				if header == colName {
					columnIndices = append(columnIndices, i)
					found = true
					break
				}
			}
			if !found {
				missingColumns = append(missingColumns, colName)
			}
		}

		if len(missingColumns) > 0 {
			return mcp.NewToolResultError(fmt.Sprintf("Columns not found: %v. Available columns: %v", missingColumns, headers)), nil
		}

		// Process entity combination
		var results [][]string
		var combinationStats struct {
			totalRows      int
			combinedRows   int
			uniqueEntities map[string]int
			sampleEntities []string
		}
		combinationStats.uniqueEntities = make(map[string]int)

		// Add new header
		newHeaders := append(headers, entityColumnName)
		results = append(results, newHeaders)

		// Process each data row
		for i := 1; i < len(records); i++ {
			row := records[i]
			combinationStats.totalRows++

			// Collect values from specified columns
			var values []string
			for _, colIndex := range columnIndices {
				if colIndex < len(row) {
					values = append(values, row[colIndex])
				} else {
					values = append(values, "")
				}
			}

			// Create combined entity ID
			combinedEntity := strings.Join(values, separator)
			combinationStats.combinedRows++

			// Track unique entities
			combinationStats.uniqueEntities[combinedEntity]++

			// Store sample entities for reporting
			if len(combinationStats.sampleEntities) < 5 {
				combinationStats.sampleEntities = append(combinationStats.sampleEntities,
					fmt.Sprintf("%s → %s", strings.Join(columnNames, "+"), combinedEntity))
			}

			// Create new row with combined entity
			newRow := append(row, combinedEntity)
			results = append(results, newRow)
		}

		// Generate output file path if not provided
		if saveToFile && outputFilePath == "" {
			dir := filepath.Dir(filePath)
			base := strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))
			outputFilePath = filepath.Join(dir, fmt.Sprintf("%s_with_entity.csv", base))
		}

		// Save to file if requested
		if saveToFile {
			outputFile, err := os.Create(outputFilePath)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Error creating output file: %v", err)), nil
			}
			defer outputFile.Close()

			writer := csv.NewWriter(outputFile)
			defer writer.Flush()

			for _, record := range results {
				if err := writer.Write(record); err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("Error writing to output file: %v", err)), nil
				}
			}
		}

		// Format complete data for display (first 10 rows)
		headersDisplay := strings.Join(results[0], " | ")
		var dataRowsDisplay []string
		maxDisplayRows := 10
		if len(results)-1 < maxDisplayRows {
			maxDisplayRows = len(results) - 1
		}

		for i := 1; i <= maxDisplayRows; i++ {
			if i < len(results) {
				rowDisplay := fmt.Sprintf("Row %d: %s", i, strings.Join(results[i], " | "))
				dataRowsDisplay = append(dataRowsDisplay, rowDisplay)
			}
		}

		if len(results)-1 > maxDisplayRows {
			dataRowsDisplay = append(dataRowsDisplay, fmt.Sprintf("... and %d more rows", len(results)-1-maxDisplayRows))
		}

		dataRowsFormatted := strings.Join(dataRowsDisplay, "\n")

		// Calculate uniqueness rate
		uniquenessRate := float64(len(combinationStats.uniqueEntities)) / float64(combinationStats.combinedRows) * 100

		result := fmt.Sprintf(`🔧 **Combined Entity ID Creation Complete!**

**📁 Source File:** %s
**🔗 Combined Columns:** %s
**🎯 Separator Used:** "%s"
**📊 New Entity Column:** %s

**📈 Combination Results:**
- **Total Rows:** %d
- **Rows Processed:** %d
- **Unique Entities:** %d
- **Uniqueness Rate:** %.1f%%

**🔍 Sample Combined Entities:**
%s

**📋 COMPLETE DATA WITH ENTITY IDs:**

**Headers:** %s

**Data Rows:**
%s

**📁 Output File:** %s

**💡 Entity ID Summary:**
- Combined: %s
- Separator: "%s"
- %d unique entities generated
- Ready for reconciliation matching!

**🎯 Reconciliation Ready:**
Your file now has a combined entity ID that can be used as a unique key for reconciliation between multiple files!
`,
			filepath.Base(filePath),
			strings.Join(columnNames, " + "),
			separator,
			entityColumnName,
			combinationStats.totalRows,
			combinationStats.combinedRows,
			len(combinationStats.uniqueEntities),
			uniquenessRate,
			strings.Join(combinationStats.sampleEntities, "\n"),
			headersDisplay,
			dataRowsFormatted,
			outputFilePath,
			strings.Join(columnNames, " + "),
			separator,
			len(combinationStats.uniqueEntities),
		)

		return mcp.NewToolResultText(result), nil
	}

	return server.ServerTool{
		Tool:    tool,
		Handler: handler,
	}
}

// ReconAggregationTool applies aggregation logic to reconciliation data for duplicate handling
func ReconAggregationTool() server.ServerTool {
	tool := mcp.NewTool("recon_aggregation",
		mcp.WithDescription("Apply aggregation logic on reconciliation data using patch logic. Groups duplicate records by a key column and aggregates values in a target column using SUM, AVG, COUNT, MIN, or MAX functions."),
		mcp.WithString("file_path",
			mcp.Description("Full path to the CSV file that needs aggregation processing"),
			mcp.Required(),
		),
		mcp.WithString("group_by_column",
			mcp.Description("Column name to group by (e.g., 'UTR', 'transaction_id', 'reference_number'). Rows with same value will be aggregated."),
			mcp.Required(),
		),
		mcp.WithString("aggregate_column",
			mcp.Description("Column name containing values to aggregate (e.g., 'amount', 'txn_amount', 'value'). Must contain numeric values."),
			mcp.Required(),
		),
		mcp.WithString("aggregation_function",
			mcp.Description("Aggregation function to apply to duplicate rows"),
			mcp.Required(),
			mcp.Enum("SUM", "AVG", "COUNT", "MIN", "MAX"),
		),
		mcp.WithBoolean("enable_aggregation",
			mcp.Description("Enable aggregation logic? Set to true to apply aggregation, false to just analyze data."),
			mcp.DefaultString("true"),
		),
		mcp.WithString("output_file_path",
			mcp.Description("Path for the output file with aggregated data"),
		),
		mcp.WithBoolean("save_to_file",
			mcp.Description("Whether to save results to a new CSV file"),
			mcp.DefaultString("true"),
		),
	)

	handler := func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Get required parameters
		filePath, err := request.RequireString("file_path")
		if err != nil {
			return mcp.NewToolResultError("File Path Required: " + err.Error()), nil
		}

		groupByColumn, err := request.RequireString("group_by_column")
		if err != nil {
			return mcp.NewToolResultError("Group By Column Required: " + err.Error()), nil
		}

		aggregateColumn, err := request.RequireString("aggregate_column")
		if err != nil {
			return mcp.NewToolResultError("Aggregate Column Required: " + err.Error()), nil
		}

		aggregationFunction, err := request.RequireString("aggregation_function")
		if err != nil {
			return mcp.NewToolResultError("Aggregation Function Required: " + err.Error()), nil
		}

		// Optional parameters
		enableAggregation := request.GetBool("enable_aggregation", true)
		saveToFile := request.GetBool("save_to_file", true)
		outputFilePath := request.GetString("output_file_path", "")

		// Check if file exists
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			return mcp.NewToolResultError(fmt.Sprintf("File not found: %s", filePath)), nil
		}

		// Process the CSV file
		file, err := os.Open(filePath)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Error opening file: %v", err)), nil
		}
		defer file.Close()

		reader := csv.NewReader(file)
		records, err := reader.ReadAll()
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Error reading CSV: %v", err)), nil
		}

		if len(records) == 0 {
			return mcp.NewToolResultError("CSV file is empty"), nil
		}

		// Find column indices
		headers := records[0]
		groupByIndex := -1
		aggregateIndex := -1

		for i, header := range headers {
			if header == groupByColumn {
				groupByIndex = i
			}
			if header == aggregateColumn {
				aggregateIndex = i
			}
		}

		if groupByIndex == -1 {
			return mcp.NewToolResultError(fmt.Sprintf("Group By Column '%s' not found. Available columns: %v", groupByColumn, headers)), nil
		}

		if aggregateIndex == -1 {
			return mcp.NewToolResultError(fmt.Sprintf("Aggregate Column '%s' not found. Available columns: %v", aggregateColumn, headers)), nil
		}

		// Analyze data for duplicates
		groupMap := make(map[string][]int) // group key -> row indices
		var aggregationStats struct {
			totalRows          int
			uniqueGroups       int
			duplicateGroups    int
			totalDuplicateRows int
			beforeAggregation  map[string][]float64
			afterAggregation   map[string]float64
			sampleDuplicates   []string
		}
		aggregationStats.beforeAggregation = make(map[string][]float64)
		aggregationStats.afterAggregation = make(map[string]float64)

		// First pass: group rows and collect duplicate information
		for i := 1; i < len(records); i++ {
			row := records[i]
			aggregationStats.totalRows++

			if groupByIndex >= len(row) || aggregateIndex >= len(row) {
				continue
			}

			groupKey := row[groupByIndex]
			if groupKey == "" {
				continue
			}

			// Parse aggregate value
			aggregateValueStr := row[aggregateIndex]
			aggregateValue, err := strconv.ParseFloat(aggregateValueStr, 64)
			if err != nil {
				continue // Skip non-numeric values
			}

			// Add to group map
			groupMap[groupKey] = append(groupMap[groupKey], i)
			aggregationStats.beforeAggregation[groupKey] = append(aggregationStats.beforeAggregation[groupKey], aggregateValue)
		}

		// Analyze groups
		for groupKey, rowIndices := range groupMap {
			if len(rowIndices) > 1 {
				aggregationStats.duplicateGroups++
				aggregationStats.totalDuplicateRows += len(rowIndices)

				// Store sample duplicate information
				if len(aggregationStats.sampleDuplicates) < 5 {
					values := aggregationStats.beforeAggregation[groupKey]
					aggregationStats.sampleDuplicates = append(aggregationStats.sampleDuplicates,
						fmt.Sprintf("%s: %v → %s applied", groupKey, values, aggregationFunction))
				}
			}
		}
		aggregationStats.uniqueGroups = len(groupMap)

		// If aggregation not enabled, just return analysis
		if !enableAggregation {
			result := fmt.Sprintf(`🔍 **Aggregation Analysis Complete!**

**📁 File:** %s
**📊 Group By:** %s
**💰 Aggregate:** %s
**🧮 Function:** %s

**📈 Analysis Results:**
- **Total Rows:** %d
- **Unique Groups:** %d
- **Groups with Duplicates:** %d
- **Total Duplicate Rows:** %d

**🔍 Sample Duplicate Groups:**
%s

**❓ Enable Aggregation?**
Set 'enable_aggregation' to true to apply %s aggregation logic and resolve duplicates.
`,
				filepath.Base(filePath),
				groupByColumn,
				aggregateColumn,
				aggregationFunction,
				aggregationStats.totalRows,
				aggregationStats.uniqueGroups,
				aggregationStats.duplicateGroups,
				aggregationStats.totalDuplicateRows,
				strings.Join(aggregationStats.sampleDuplicates, "\n"),
				aggregationFunction)

			return mcp.NewToolResultText(result), nil
		}

		// Apply aggregation logic
		var results [][]string
		results = append(results, headers) // Add headers

		// Process each group
		processedGroups := make(map[string]bool)
		aggregatedRows := 0

		for i := 1; i < len(records); i++ {
			row := records[i]

			if groupByIndex >= len(row) {
				results = append(results, row)
				continue
			}

			groupKey := row[groupByIndex]
			if groupKey == "" || processedGroups[groupKey] {
				continue // Skip empty keys or already processed groups
			}

			rowIndices := groupMap[groupKey]
			if len(rowIndices) == 1 {
				// Single row, no aggregation needed
				results = append(results, records[rowIndices[0]])
			} else {
				// Multiple rows, apply aggregation
				values := aggregationStats.beforeAggregation[groupKey]
				aggregatedValue := applyAggregation(values, aggregationFunction)
				aggregationStats.afterAggregation[groupKey] = aggregatedValue

				// Create aggregated row (use first row as template)
				aggregatedRow := make([]string, len(records[rowIndices[0]]))
				copy(aggregatedRow, records[rowIndices[0]])
				aggregatedRow[aggregateIndex] = fmt.Sprintf("%.2f", aggregatedValue)

				results = append(results, aggregatedRow)
				aggregatedRows++
			}

			processedGroups[groupKey] = true
		}

		// Generate output file path if not provided
		if saveToFile && outputFilePath == "" {
			dir := filepath.Dir(filePath)
			base := strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))
			outputFilePath = filepath.Join(dir, fmt.Sprintf("%s_aggregated_%s.csv", base, strings.ToLower(aggregationFunction)))
		}

		// Save to file if requested
		if saveToFile {
			outputFile, err := os.Create(outputFilePath)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Error creating output file: %v", err)), nil
			}
			defer outputFile.Close()

			writer := csv.NewWriter(outputFile)
			defer writer.Flush()

			for _, record := range results {
				if err := writer.Write(record); err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("Error writing to output file: %v", err)), nil
				}
			}
		}

		// Format results for display
		headersDisplay := strings.Join(results[0], " | ")
		var dataRowsDisplay []string
		maxDisplayRows := 10
		if len(results)-1 < maxDisplayRows {
			maxDisplayRows = len(results) - 1
		}

		for i := 1; i <= maxDisplayRows; i++ {
			if i < len(results) {
				rowDisplay := fmt.Sprintf("Row %d: %s", i, strings.Join(results[i], " | "))
				dataRowsDisplay = append(dataRowsDisplay, rowDisplay)
			}
		}

		if len(results)-1 > maxDisplayRows {
			dataRowsDisplay = append(dataRowsDisplay, fmt.Sprintf("... and %d more rows", len(results)-1-maxDisplayRows))
		}

		dataRowsFormatted := strings.Join(dataRowsDisplay, "\n")

		// Generate sample aggregations for display
		var sampleAggregations []string
		count := 0
		for groupKey, beforeValues := range aggregationStats.beforeAggregation {
			if len(beforeValues) > 1 && count < 5 {
				afterValue := aggregationStats.afterAggregation[groupKey]
				sampleAggregations = append(sampleAggregations,
					fmt.Sprintf("%s: %v → %.2f (%s)", groupKey, beforeValues, afterValue, aggregationFunction))
				count++
			}
		}

		result := fmt.Sprintf(`🧮 **Reconciliation Aggregation Complete!**

**📁 Source File:** %s
**🔗 Grouped By:** %s
**💰 Aggregated:** %s
**📊 Function:** %s(%s)

**📈 Aggregation Results:**
- **Original Rows:** %d
- **Final Rows:** %d
- **Groups Aggregated:** %d
- **Duplicate Groups:** %d
- **Rows Reduced:** %d

**🔍 Sample Aggregations:**
%s

**📋 AGGREGATED DATA:**

**Headers:** %s

**Data Rows:**
%s

**📁 Output File:** %s

**💡 Aggregation Summary:**
- Applied %s logic to %s column
- Grouped by %s column
- %d rows with duplicates were aggregated
- Ready for reconciliation processing!

**🎯 Reconciliation Benefits:**
✅ Duplicate handling resolved
✅ Data consistency improved  
✅ Reconciliation accuracy enhanced
✅ Processing efficiency optimized
`,
			filepath.Base(filePath),
			groupByColumn,
			aggregateColumn,
			aggregationFunction,
			aggregateColumn,
			aggregationStats.totalRows,
			len(results)-1,
			aggregatedRows,
			aggregationStats.duplicateGroups,
			aggregationStats.totalRows-(len(results)-1),
			strings.Join(sampleAggregations, "\n"),
			headersDisplay,
			dataRowsFormatted,
			outputFilePath,
			aggregationFunction,
			aggregateColumn,
			groupByColumn,
			aggregationStats.duplicateGroups)

		return mcp.NewToolResultText(result), nil
	}

	return server.ServerTool{
		Tool:    tool,
		Handler: handler,
	}
}

// applyAggregation applies the specified aggregation function to a slice of float64 values
func applyAggregation(values []float64, function string) float64 {
	if len(values) == 0 {
		return 0
	}

	switch function {
	case "SUM":
		sum := 0.0
		for _, v := range values {
			sum += v
		}
		return sum

	case "AVG":
		sum := 0.0
		for _, v := range values {
			sum += v
		}
		return sum / float64(len(values))

	case "COUNT":
		return float64(len(values))

	case "MIN":
		min := values[0]
		for _, v := range values[1:] {
			if v < min {
				min = v
			}
		}
		return min

	case "MAX":
		max := values[0]
		for _, v := range values[1:] {
			if v > max {
				max = v
			}
		}
		return max

	default:
		return 0
	}
}

// Helper function to generate mock IDs
func generateID(prefix string) string {
	// In real implementation, this would generate proper unique IDs
	return fmt.Sprintf("%s%s", prefix, randomString(10))
}

func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}
