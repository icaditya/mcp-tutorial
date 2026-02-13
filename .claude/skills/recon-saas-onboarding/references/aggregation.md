# Recon-SaaS Aggregation Configuration

Enable aggregation ONLY when explicitly requested by the user.

## When to Use Aggregation

Aggregation is needed when:
- Multiple records share the same EntityID
- Records need grouping by a secondary identifier (EntityIdentifier)
- Amounts need to be summed across related records
- Parent-child reconciliation is required

**Example scenario**: Multiple line items per invoice, where each line item has the same invoice_id but different line_item_id. You want to sum all line items and reconcile against the invoice total.

---

## Aggregation Workflow

### Step 1: Identify Requirements

Confirm with user:
1. Which master source needs aggregation?
2. Which column should be EntityIdentifier?
3. What is the lookup_id for this reconciliation?

### Step 2: Update Master Source

**PATCH** `/v1/admin-recon-saas/sources/update/{master_source_id}`

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

**Important**: 
- APPEND "EntityIdentifier" to existing unique_keys (don't replace)
- Update mapping_config to map the user's column to "EntityIdentifier"

### Step 3: Update Lookup

**PATCH** `/v1/admin-recon-saas/lookup/{lookup_id}`

Enable aggregation on the config item containing EntityID:

```json
{
  "config": [
    {
      "source": "master_source_id",
      "Columns": ["EntityID"],
      "aggregation": {
        "enabled": true,
        "conditions": null
      }
    }
  ]
}
```

---

## API Details

### Get Current Configuration

First, fetch current configs:

**GET** `/v1/admin-recon-saas/sources/get/{master_source_id}`
**GET** `/v1/admin-recon-saas/lookup/{lookup_id}`

### Update Master Source

**PATCH** `/v1/admin-recon-saas/sources/update/{master_source_id}`

```json
{
  "unique_keys": ["EntityID", "EntityIdentifier"],
  "mapping_config": [
    // Include ALL existing mappings
    {"source": "existing_col1", "destination": "existing_col1", "value": ""},
    {"source": "existing_col2", "destination": "existing_col2", "value": ""},
    // Update the EntityIdentifier column
    {"source": "line_item_id", "destination": "EntityIdentifier", "value": ""}
  ]
}
```

### Update Lookup

**PATCH** `/v1/admin-recon-saas/lookup/{lookup_id}`

```json
{
  "config": [
    {
      "source": "master_source_id_with_aggregation",
      "Columns": ["EntityID"],
      "aggregation": {
        "enabled": true,
        "conditions": null
      },
      "advanced_config": {
        "enabled": false,
        "cols_config": []
      }
    },
    {
      "source": "other_master_source_id",
      "Columns": ["EntityID"],
      "aggregation": {
        "enabled": false,
        "conditions": null
      }
    }
  ]
}
```

---

## Aggregation with Conditions

For conditional aggregation (e.g., multiply amount by quantity):

```json
{
  "aggregation": {
    "enabled": true,
    "conditions": [
      {
        "column": "Amount",
        "value": "quantity",
        "operation": "mul",
        "operation_val": null
      }
    ]
  }
}
```

**Operations**:
- `mul` - Multiply
- `div` - Divide
- `add` - Add
- `sub` - Subtract

---

## Example: Invoice Line Items

### Scenario
- Source A: Invoices with total amount
- Source B: Line items with individual amounts

Source B needs aggregation to sum line item amounts per invoice.

### Source B Data
| invoice_id | line_item_id | amount |
|------------|--------------|--------|
| INV001     | LI001        | 100    |
| INV001     | LI002        | 200    |
| INV001     | LI003        | 50     |
| INV002     | LI004        | 500    |

### Master Source B Configuration

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

### Lookup Configuration

```json
{
  "config": [
    {
      "source": "source_a_id",
      "Columns": ["EntityID"],
      "aggregation": {"enabled": false, "conditions": null}
    },
    {
      "source": "source_b_id",
      "Columns": ["EntityID"],
      "aggregation": {"enabled": true, "conditions": null}
    }
  ]
}
```

### Result
- Line items are grouped by invoice_id
- Amounts are summed: INV001 = 350, INV002 = 500
- Reconciliation compares aggregated totals

---

## Conversation Examples

### User requests aggregation

**User**: "Enable aggregation on the invoice_number column for Source A"

**Assistant should**:
1. Check context for master_source_id of Source A
2. Ask for lookup_id if not available
3. Call aggregation API updates

### User provides all information

**User**: "Aggregate on line_item_id for master source ABC123 with lookup LKP456"

**Assistant should**:
1. GET current master source config
2. PATCH master source with updated unique_keys and mapping_config
3. GET current lookup config
4. PATCH lookup with aggregation enabled
