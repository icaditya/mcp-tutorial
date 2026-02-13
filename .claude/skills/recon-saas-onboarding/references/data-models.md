# Recon-SaaS Data Models

Entity structures and relationships for the reconciliation platform.

## Entity Relationships

```
┌─────────────────┐     ┌───────────────────┐
│  Master Source  │────▶│  Merchant Source  │
└────────┬────────┘     └─────────┬─────────┘
         │                        │
         │                        │
         ▼                        ▼
┌─────────────────┐     ┌───────────────────┐
│  Master Recon   │────▶│  Merchant Recon   │
│    Process      │     │     Process       │
└────────┬────────┘     └───────────────────┘
         │
    ┌────┴────┐
    │         │
    ▼         ▼
┌───────┐  ┌────────┐
│ Rules │  │ Lookup │
└───┬───┘  └────────┘
    │
    ▼
┌─────────────┐
│ Recon State │
└─────────────┘
```

---

## Master Source

Defines the schema and mapping configuration for a data source type.

```go
type MasterSource struct {
    ID                       string                    // Unique identifier
    Name                     string                    // Display name
    Config                   MasterConfig              // Configuration settings
    SourceSchema             []SourceSchema            // Column definitions
    MappingConfig            []MappingConfig           // Column mappings
    TransformationConfig     []TransformationConfig    // Data transformations
    ValidationConfig         ValidationConfig          // Validation rules
    MetadataExtractionConfig MetadataExtractionConfig  // Metadata extraction
}

type MasterConfig struct {
    SkipTopRows                  int      // Rows to skip at file start
    IngestToDb                   bool     // Enable database ingestion
    AllowUpload                  bool     // Enable file uploads
    UniqueKeys                   []string // Unique identifier columns (e.g., ["EntityID"])
    IsInternal                   bool     // Internal source flag
    SplitFileBasis               string   // File splitting basis
    ExtractDistinctConfig        []string // Distinct extraction columns
    ReportEnrichment             bool     // Enable report enrichment
    SkipBottomRows               int      // Rows to skip at file end
    SkipRowFunc                  string   // Row skip function
    IsHeaderMissing              bool     // Header missing flag
}

type SourceSchema struct {
    Name string  // Column name
    Type string  // Always "string"
}

type MappingConfig struct {
    Source      string  // Original column name
    Destination string  // Target column name (snake_case, EntityID, Amount)
    Value       string  // Always empty ""
}
```

---

## Merchant Source

Merchant-specific source configuration linked to a master source.

```go
type MerchantSource struct {
    ID                       string          // Unique identifier
    Name                     string          // Display name
    MerchantID               string          // Merchant identifier
    MasterSourceID           string          // Linked master source
    Config                   MerchantConfig  // Merchant-specific config
    SourceSchema             []SourceSchema  // Override schema (usually null)
    MappingConfig            []MappingConfig // Override mappings (usually null)
}

type MerchantConfig struct {
    AllowUpload             bool                        // Enable uploads
    ReportingEmails         []string                    // Email recipients
    CCEmails                []string                    // CC recipients
    BCCEmails               []string                    // BCC recipients
    BeamSFTPPushJob         string                      // SFTP job config
    SplitFileBasis          string                      // Split basis
    SlackNotificationConfig SlackNotificationConfig     // Slack alerts
}

type SlackNotificationConfig struct {
    ReconPercentageThreshold float64  // Alert threshold (default: 90)
    FileAlertEnabled         bool     // Enable file alerts
    FileAlertCron            string   // Alert cron schedule
    NumberOfFilesToCheck     int      // Files to check
}
```

---

## Recon State

Defines reconciliation outcome states.

```go
type ReconState struct {
    ID         string  // Unique identifier
    MerchantID string  // Merchant identifier
    Name       string  // State name (Reconciled, Unreconciled)
    Priority   int8    // Processing priority (lower = higher priority)
    Remarks    string  // State description
}
```

**Standard States**:
| Name | Priority | Remarks |
|------|----------|---------|
| Reconciled | 2 | success |
| Unreconciled | 3 | Amount mismatch |
| Unreconciled | 3 | Record not found in Source A |
| Unreconciled | 3 | Record not found in Source B |

---

## Rule

Defines matching logic for reconciliation.

```go
type Rule struct {
    ID           string    // Unique identifier
    MerchantID   string    // Merchant identifier
    Name         string    // Rule name
    Type         string    // Rule type ("reconciliation")
    Expression   string    // Matching expression
    Sources      []string  // Master source IDs involved
    ReconStateID string    // Outcome state ID
    CaseRule     bool      // Case management flag
    AssignedTo   *string   // Assignee (optional)
}
```

**Expression Syntax**:
- `{source_id}.{column}` - Access column value
- `==` - Equality comparison
- `&&` - Logical AND
- `!` - Logical NOT
- `.Equal()` - Amount comparison method
- `NoRecord.Value == true` - Missing record check

**Examples**:
```
// Exact match
source1.EntityID == source2.EntityID && source1.Amount.Equal(source2.Amount)

// Amount mismatch
source1.EntityID == source2.EntityID && !source1.Amount.Equal(source2.Amount)

// Missing record
NoRecord.Value == true
```

---

## Lookup

Defines lookup configuration for record matching.

```go
type Lookup struct {
    ID         string          // Unique identifier
    MerchantID string          // Merchant identifier
    Name       string          // Lookup name
    Config     []LookupConfig  // Lookup configurations
}

type LookupConfig struct {
    Source         string               // Master source ID
    Columns        []string             // Lookup columns (e.g., ["EntityID"])
    Aggregation    Aggregation          // Aggregation settings
    AdvancedConfig AdvancedLookupConfig // Advanced lookup config
    LookbackConfig LookbackConfig       // Date range lookup
}

type Aggregation struct {
    Enabled    bool              // Enable aggregation
    Conditions []AggregationCond // Aggregation conditions
}

type AggregationCond struct {
    Column       string  // Column to aggregate
    Value        string  // Value reference
    Operation    string  // Operation (mul, div, add, sub)
    OperationVal decimal.Decimal  // Operation value
}
```

---

## Master Recon Process

Master configuration for reconciliation processing.

```go
type MasterReconProcess struct {
    ID             string                // Unique identifier
    Name           string                // Process name
    ProductID      string                // Product identifier
    Type           string                // Process type
    LookupConfig   []LookupReconConfig   // Lookup configurations
    Rules          Rules                 // Rule configurations
    Sources        []string              // Master source IDs
    Sequence       []Sequence            // Processing sequence
    ReportConfig   ReportReconConfig     // Report configuration
    WorkflowConfig WorkflowConfig        // Workflow configuration
}

type LookupReconConfig struct {
    StreamingSourceID string            // Streaming source ID
    Config            map[string]string // Source ID to Lookup ID mapping
}

type Rules struct {
    RuleIDs     []string          // All rule IDs
    DefaultRule map[string]string // Default rules per source
}

type Sequence struct {
    MasterSourceID string  // Master source ID
    Priority       int     // Processing priority
}
```

---

## Merchant Recon Process

Merchant-specific reconciliation process.

```go
type MerchantReconProcess struct {
    ID                   string            // Unique identifier
    MerchantID           string            // Merchant identifier
    MasterReconProcessID string            // Linked master process
    Sources              []string          // Merchant source IDs
    ReportConfig         ReportReconConfig // Report configuration
    Status               string            // Process status (approved, pending)
}
```

---

## Report Configuration

```go
type ReportReconConfig struct {
    ReportConfig          []SourceReportConfig  // Per-source report config
    FrontendCols          []string              // Display columns
    ReportingSources      []string              // Sources for reports
    ReportEnrichment      []ReportEnrichment    // Enrichment config
    ReportFormat          []string              // Formats (csv, xlsx, tsv)
    ReportChannel         []string              // Channels (email, sftp)
    SkipStatus            bool                  // Skip status column
    SkipRows              bool                  // Skip rows flag
    SkipRowsReconStateIDs []string              // States to skip
}

type SourceReportConfig struct {
    MasterSourceID string            // Master source ID
    ColumnMap      []ReportColumnMap // Column mappings
    ReportName     string            // Report name
}

type ReportColumnMap struct {
    SourceColumn string  // Source column name
    ReportColumn string  // Report display name
    Type         string  // Column type
    ID           string  // Column ID
}
```

---

## Transformation Configuration

```go
type TransformationConfig struct {
    Logic         map[string]interface{}  // Transformation logic
    OutputColumns []string                // Output column names
}
```

**Logic Structure**:
```json
{
  "function": "regex_exec",
  "columns": ["notes"],
  "params": ["[0-9]{7}"]
}
```
