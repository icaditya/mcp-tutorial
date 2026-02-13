# Recon-SaaS Transformation Functions

Apply transformations ONLY when explicitly requested by the user.

## When to Apply Transformations

**DO apply when user says**:
- "Apply regex on column X to extract EntityID"
- "Concatenate columns A, B, C to create EntityID"
- "Transform the date format from X to Y"
- "I need to apply a transformation on [column]"

**DO NOT apply**:
- During normal master source creation
- During file analysis
- When user doesn't explicitly mention transformation

---

## Transformation Workflow

1. Confirm with user which master source to apply transformation to
2. Identify input column(s) and output column name
3. Get function-specific parameters
4. Apply transformation via PATCH API

### API Update

**PATCH** `/v1/admin-recon-saas/sources/update/{master_source_id}`

```json
{
  "transformation_config": [
    {
      "logic": {
        "function": "regex_exec",
        "columns": ["notes"],
        "params": ["[0-9]{7}"]
      },
      "output_columns": ["EntityID"]
    }
  ],
  "mapping_config": [
    // Updated mappings including new output column
  ]
}
```

---

## Available Functions

### Amount/Number Functions

#### `abs_amount_parsing`
Parse absolute amount (removes commas, handles negatives).

```json
{
  "function": "abs_amount_parsing",
  "columns": ["raw_amount"],
  "params": []
}
```

Input: `"-1,234.56"` → Output: `"1234.56"`

#### `abs_amount_in_paisa`
Convert amount to paisa (multiply by 100).

```json
{
  "function": "abs_amount_in_paisa",
  "columns": ["amount"],
  "params": []
}
```

Input: `"123.45"` → Output: `"12345"`

#### `add_amount_cols`
Sum multiple amount columns.

```json
{
  "function": "add_amount_cols",
  "columns": ["base_amount", "tax", "fee"],
  "params": []
}
```

#### `subtract_amount_cols`
Subtract amounts from base amount.

```json
{
  "function": "subtract_amount_cols",
  "columns": ["gross_amount", "discount", "refund"],
  "params": []
}
```

Result: `gross_amount - discount - refund`

#### `percentage_of_number`
Calculate percentage of a number.

```json
{
  "function": "percentage_of_number",
  "columns": ["base_amount"],
  "params": ["18"]
}
```

Input: `"1000"`, params: `["18"]` → Output: `"180"` (18% of 1000)

#### `extract_amount_from_cols`
Get first non-zero value from multiple columns.

```json
{
  "function": "extract_amount_from_cols",
  "columns": ["primary_amount", "secondary_amount", "fallback_amount"],
  "params": []
}
```

---

### String Functions

#### `append_multiple_columns`
Concatenate multiple columns.

```json
{
  "function": "append_multiple_columns",
  "columns": ["RRN", "TID", "MID"],
  "params": []
}
```

Input: `["ABC", "123", "XYZ"]` → Output: `"ABC123XYZ"`

#### `regex_exec`
Extract using regex pattern.

```json
{
  "function": "regex_exec",
  "columns": ["notes"],
  "params": ["[0-9]{7}"]
}
```

Input: `"Payment for invoice 1234567 completed"` → Output: `"1234567"`

#### `excel_mid`
Extract substring (MID function).

```json
{
  "function": "excel_mid",
  "columns": ["reference"],
  "params": ["3", "5"]
}
```

Params: `[start_position, length]` (1-indexed)
Input: `"ABCDEFGHIJ"`, params: `["3", "5"]` → Output: `"CDEFG"`

#### `excel_left`
Extract from left.

```json
{
  "function": "excel_left",
  "columns": ["code"],
  "params": ["4"]
}
```

Input: `"ABCDEFG"`, params: `["4"]` → Output: `"ABCD"`

#### `excel_right`
Extract from right.

```json
{
  "function": "excel_right",
  "columns": ["code"],
  "params": ["4"]
}
```

Input: `"ABCDEFG"`, params: `["4"]` → Output: `"DEFG"`

#### `split`
Split string and get specific part.

```json
{
  "function": "split",
  "columns": ["reference"],
  "params": ["-", "0"]
}
```

Params: `[delimiter, index]` (0-indexed)
Input: `"ABC-123-XYZ"`, params: `["-", "1"]` → Output: `"123"`

#### `remove_prefix`
Remove specified prefixes.

```json
{
  "function": "remove_prefix",
  "columns": ["transaction_id"],
  "params": ["TXN_", "TX_", "T_"]
}
```

Input: `"TXN_12345"` → Output: `"12345"`

#### `remove_suffix`
Remove specified suffixes.

```json
{
  "function": "remove_suffix",
  "columns": ["reference"],
  "params": ["_OLD", "_LEGACY"]
}
```

#### `add_padding_prefix`
Zero-pad to specified length.

```json
{
  "function": "add_padding_prefix",
  "columns": ["id"],
  "params": ["10"]
}
```

Input: `"123"`, params: `["10"]` → Output: `"0000000123"`

#### `replace_blank_string`
Trim whitespace.

```json
{
  "function": "replace_blank_string",
  "columns": ["name"],
  "params": []
}
```

#### `remove_single_quotes`
Remove single quotes from string.

```json
{
  "function": "remove_single_quotes",
  "columns": ["value"],
  "params": []
}
```

Input: `"'ABC'"` → Output: `"ABC"`

#### `remove_double_quotes`
Remove double quotes from string.

```json
{
  "function": "remove_double_quotes",
  "columns": ["value"],
  "params": []
}
```

---

### Date Functions

#### `change_date_format`
Convert between date formats.

```json
{
  "function": "change_date_format",
  "columns": ["transaction_date"],
  "params": ["%m/%d/%Y", "%Y-%m-%d"]
}
```

Params: `[from_format, to_format]`
Input: `"12/25/2024"` → Output: `"2024-12-25"`

**Common Format Codes**:
- `%Y` - 4-digit year (2024)
- `%y` - 2-digit year (24)
- `%m` - Month (01-12)
- `%d` - Day (01-31)
- `%H` - Hour 24h (00-23)
- `%I` - Hour 12h (01-12)
- `%M` - Minute (00-59)
- `%S` - Second (00-59)
- `%p` - AM/PM

#### `date_normalization`
Normalize to YYYY-MM-DD format.

```json
{
  "function": "date_normalization",
  "columns": ["date"],
  "params": ["%d-%m-%Y"]
}
```

Params: `[current_format]`
Input: `"25-12-2024"` → Output: `"2024-12-25"`

#### `txn_date_extraction_generic`
Parse various date formats automatically.

```json
{
  "function": "txn_date_extraction_generic",
  "columns": ["raw_date"],
  "params": []
}
```

#### `excel_to_datetime`
Convert Excel serial date to datetime.

```json
{
  "function": "excel_to_datetime",
  "columns": ["excel_date"],
  "params": []
}
```

Input: `"45292"` → Output: `"2024-01-15"`

#### `subtract_date`
Subtract days from date.

```json
{
  "function": "subtract_date",
  "columns": ["date"],
  "params": ["7"]
}
```

#### `add_date`
Add days to date.

```json
{
  "function": "add_date",
  "columns": ["date"],
  "params": ["7"]
}
```

---

### Other Functions

#### `hard_code_value`
Set a constant value.

```json
{
  "function": "hard_code_value",
  "columns": [],
  "params": ["CONSTANT_VALUE"]
}
```

#### `settlement_amount_from_debit_credit_cols`
Derive settlement amount from debit/credit columns.

```json
{
  "function": "settlement_amount_from_debit_credit_cols",
  "columns": ["debit", "credit"],
  "params": []
}
```

#### `get_field_from_notes`
Extract field from JSON notes.

```json
{
  "function": "get_field_from_notes",
  "columns": ["json_notes"],
  "params": ["payment_id", "order_id"]
}
```

Extracts first matching field from JSON.

---

## Mapping Config Updates

When applying transformation with special output columns (EntityID, Amount, EntityStatus, EntityIdentifier):

### Before Transformation
```json
{
  "mapping_config": [
    {"source": "notes", "destination": "EntityID", "value": ""},
    {"source": "amount", "destination": "Amount", "value": ""}
  ]
}
```

### After Transformation (output_column = "EntityID")
```json
{
  "mapping_config": [
    {"source": "notes", "destination": "notes", "value": ""},
    {"source": "amount", "destination": "Amount", "value": ""},
    {"source": "EntityID", "destination": "EntityID", "value": ""}
  ]
}
```

**Logic**:
1. Find existing mapping where destination = special column
2. Change that mapping's destination to snake_case of its source
3. Add new mapping: source = output_column, destination = output_column
