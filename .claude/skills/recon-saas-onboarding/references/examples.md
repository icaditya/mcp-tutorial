# Recon-SaaS Onboarding Examples

Common merchant onboarding scenarios and workflows.

---

## Example 1: Basic Two-Source Reconciliation

### Scenario
Reconcile POS transactions against bank statements using transaction ID.

### Files
**File 1 (POS Transactions)**:
| transaction_id | amount | date | merchant_name |
|----------------|--------|------|---------------|
| TXN001 | 1000.00 | 2024-01-15 | Store A |
| TXN002 | 500.50 | 2024-01-15 | Store B |

**File 2 (Bank Statement)**:
| ref_number | credit_amount | transaction_date | description |
|------------|---------------|------------------|-------------|
| TXN001 | 1000.00 | 2024-01-15 | POS Credit |
| TXN002 | 500.50 | 2024-01-15 | POS Credit |

### Step 1: File Analysis

Identify columns:
- **File 1**: EntityID = `transaction_id`, Amount = `amount`
- **File 2**: EntityID = `ref_number`, Amount = `credit_amount`

### Step 2: Create Master Sources

**Source 1 (POS)**:
```json
{
  "name": "POS Transaction Source",
  "unique_keys": ["EntityID"],
  "source_schema": [
    {"name": "transaction_id", "type": "string"},
    {"name": "amount", "type": "string"},
    {"name": "date", "type": "string"},
    {"name": "merchant_name", "type": "string"}
  ],
  "mapping_config": [
    {"source": "transaction_id", "destination": "EntityID", "value": ""},
    {"source": "amount", "destination": "Amount", "value": ""},
    {"source": "date", "destination": "date", "value": ""},
    {"source": "merchant_name", "destination": "merchant_name", "value": ""}
  ]
}
```

**Source 2 (Bank)**:
```json
{
  "name": "Bank Statement Source",
  "unique_keys": ["EntityID"],
  "source_schema": [
    {"name": "ref_number", "type": "string"},
    {"name": "credit_amount", "type": "string"},
    {"name": "transaction_date", "type": "string"},
    {"name": "description", "type": "string"}
  ],
  "mapping_config": [
    {"source": "ref_number", "destination": "EntityID", "value": ""},
    {"source": "credit_amount", "destination": "Amount", "value": ""},
    {"source": "transaction_date", "destination": "transaction_date", "value": ""},
    {"source": "description", "destination": "description", "value": ""}
  ]
}
```

### Step 3: Create Merchant Sources

```json
{
  "name": "POS Transaction Source - Merchant Portal",
  "merchant_id": "merchant_xyz",
  "master_source_id": "{source1_id}",
  "allow_upload": true
}
```

### Step 4: Create Recon States

Create 4 states for: Reconciled, Amount Mismatch, Missing from POS, Missing from Bank.

### Step 5: Create Rules

**Reconciled Rule**:
```json
{
  "expression": "{source1_id}.EntityID == {source2_id}.EntityID && {source1_id}.Amount.Equal({source2_id}.Amount)",
  "recon_state_id": "{reconciled_state_id}"
}
```

### Step 6: Create Process

Create lookup, master process, and merchant process.

---

## Example 2: Transformation - Extract EntityID from Notes

### Scenario
Transaction ID is embedded in a notes column: "Payment for order 1234567 completed"

### Solution

Apply regex transformation to extract the 7-digit number.

**Transformation Config**:
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
  ]
}
```

**Updated Mapping Config**:
```json
{
  "mapping_config": [
    {"source": "notes", "destination": "notes", "value": ""},
    {"source": "amount", "destination": "Amount", "value": ""},
    {"source": "EntityID", "destination": "EntityID", "value": ""}
  ]
}
```

---

## Example 3: Concatenate Columns for EntityID

### Scenario
Unique identifier requires combining RRN + TID + MID.

### Solution

Use `append_multiple_columns` transformation.

**Transformation Config**:
```json
{
  "transformation_config": [
    {
      "logic": {
        "function": "append_multiple_columns",
        "columns": ["RRN", "TID", "MID"],
        "params": []
      },
      "output_columns": ["EntityID"]
    }
  ]
}
```

---

## Example 4: Aggregation - Line Items

### Scenario
Reconcile invoice totals against individual line items.

**Source A (Invoices)**:
| invoice_id | total_amount |
|------------|--------------|
| INV001 | 350 |
| INV002 | 500 |

**Source B (Line Items)**:
| invoice_id | line_item_id | amount |
|------------|--------------|--------|
| INV001 | LI001 | 100 |
| INV001 | LI002 | 200 |
| INV001 | LI003 | 50 |
| INV002 | LI004 | 500 |

### Solution

Enable aggregation on Source B to sum amounts per invoice_id.

**Source B Master Source**:
```json
{
  "unique_keys": ["EntityID", "EntityIdentifier"],
  "mapping_config": [
    {"source": "invoice_id", "destination": "EntityID", "value": ""},
    {"source": "line_item_id", "destination": "EntityIdentifier", "value": ""},
    {"source": "amount", "destination": "Amount", "value": ""}
  ]
}
```

**Lookup Config**:
```json
{
  "config": [
    {
      "source": "{source_a_id}",
      "Columns": ["EntityID"],
      "aggregation": {"enabled": false, "conditions": null}
    },
    {
      "source": "{source_b_id}",
      "Columns": ["EntityID"],
      "aggregation": {"enabled": true, "conditions": null}
    }
  ]
}
```

---

## Example 5: Date Format Conversion

### Scenario
Source A has dates as "MM/DD/YYYY", Source B has "YYYY-MM-DD". Need normalization.

### Solution

Apply date format transformation to Source A.

**Transformation Config**:
```json
{
  "transformation_config": [
    {
      "logic": {
        "function": "change_date_format",
        "columns": ["transaction_date"],
        "params": ["%m/%d/%Y", "%Y-%m-%d"]
      },
      "output_columns": ["normalized_date"]
    }
  ]
}
```

---

## Example 6: Amount in Paisa Conversion

### Scenario
Source A has amounts in rupees, Source B has amounts in paisa. Need conversion.

### Solution

Apply amount conversion to Source A.

**Transformation Config**:
```json
{
  "transformation_config": [
    {
      "logic": {
        "function": "abs_amount_in_paisa",
        "columns": ["amount"],
        "params": []
      },
      "output_columns": ["Amount"]
    }
  ]
}
```

---

## Example 7: Complex Entity ID Extraction

### Scenario
Extract transaction ID from complex reference: "REF-2024-TXN-1234567-SETTLED"

### Solution

Use regex to extract the 7-digit number.

**Transformation Config**:
```json
{
  "transformation_config": [
    {
      "logic": {
        "function": "regex_exec",
        "columns": ["reference"],
        "params": ["TXN-([0-9]{7})"]
      },
      "output_columns": ["EntityID"]
    }
  ]
}
```

Or use split and take specific part:

```json
{
  "transformation_config": [
    {
      "logic": {
        "function": "split",
        "columns": ["reference"],
        "params": ["-", "3"]
      },
      "output_columns": ["EntityID"]
    }
  ]
}
```

---

## Example 8: Three-Way Reconciliation Setup

### Scenario
Reconcile POS transactions, bank statements, and payment gateway records.

### Solution

Create three master sources and configure:
1. POS ↔ Bank lookup
2. POS ↔ Gateway lookup
3. Configure rules for all three-way matching scenarios

**Lookup Config**:
```json
{
  "config": [
    {
      "source": "{pos_source_id}",
      "Columns": ["EntityID"],
      "aggregation": {"enabled": false, "conditions": null}
    },
    {
      "source": "{bank_source_id}",
      "Columns": ["EntityID"],
      "aggregation": {"enabled": false, "conditions": null}
    },
    {
      "source": "{gateway_source_id}",
      "Columns": ["EntityID"],
      "aggregation": {"enabled": false, "conditions": null}
    }
  ]
}
```

**Rule Expressions**:
```
// All three match
{pos}.EntityID == {bank}.EntityID && {pos}.EntityID == {gateway}.EntityID && {pos}.Amount.Equal({bank}.Amount) && {pos}.Amount.Equal({gateway}.Amount)

// POS-Bank match, Gateway mismatch
{pos}.EntityID == {bank}.EntityID && {pos}.Amount.Equal({bank}.Amount) && (!{pos}.Amount.Equal({gateway}.Amount) || NoRecord.Value == true)
```

---

## Conversation Flow Template

### User Request
"I want to onboard a new merchant for POS reconciliation"

### Assistant Response Flow

1. **Ask for files**: "Please provide the two reconciliation files you want to analyze."

2. **Analyze files**: Run file analysis to identify columns and recommend EntityID/Amount.

3. **Confirm selections**: "Based on analysis, I recommend:
   - File 1: EntityID = `transaction_id`, Amount = `amount`
   - File 2: EntityID = `ref_number`, Amount = `credit_amount`
   
   Should I proceed with these selections?"

4. **Get merchant_id**: "What merchant_id should I use for this setup?"

5. **Execute onboarding**: Create all entities in sequence:
   - Master Sources (capture IDs)
   - Merchant Sources (capture IDs)
   - Recon States (capture IDs)
   - Rules (capture IDs)
   - Lookup (capture ID)
   - Master Process (capture ID)
   - Merchant Process (capture ID)

6. **Confirm completion**: "Merchant onboarding complete! Summary:
   - Master Source 1: {id}
   - Master Source 2: {id}
   - Merchant Process: {id}
   
   The merchant can now upload files for reconciliation."
