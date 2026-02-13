# Recon-SaaS API Reference

Complete API documentation for recon-saas merchant onboarding.

## Authentication

All API calls require Basic Authentication:
```
Authorization: Basic cmVjb24tc2FhczpyZWNvbi1zYWFz
Content-Type: application/json
```

## Base URLs

| Environment | URL |
|-------------|-----|
| Local | http://localhost:9400 |
| Dev | https://recon-saas.dev.razorpay.in |
| Prod | https://recon-saas.concierge.razorpay.com |

---

## Master Source APIs

### Create Master Source
**POST** `/v1/admin-recon-saas/sources/create`

```json
{
  "name": "POS Transaction Source",
  "skip_top_rows": 0,
  "ingest_to_db": true,
  "allow_upload": true,
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
  ],
  "transformation_config": [],
  "validation_config": {"logics": []},
  "sub_source_config": {},
  "extract_distinct_config": [],
  "report_enrichment": false,
  "is_header_missing": false,
  "metadata_extraction_config": {"logics": []},
  "skip_bottom_rows": 0,
  "skip_row_func": ""
}
```

**Response**:
```json
{
  "id": "master_source_id_abc123",
  "name": "POS Transaction Source",
  "created_at": "2024-01-15T10:30:00Z"
}
```

### Get Master Source
**GET** `/v1/admin-recon-saas/sources/get/{id}`

**Response**: Full master source object with all configurations.

### Update Master Source
**PATCH** `/v1/admin-recon-saas/sources/update/{id}`

Send only fields to update:
```json
{
  "name": "Updated Source Name",
  "unique_keys": ["EntityID", "EntityIdentifier"]
}
```

---

## Merchant Source APIs

### Create Merchant Source
**POST** `/v1/admin-recon-saas/sources/create_merchant`

```json
{
  "name": "POS Transaction Source - Merchant Portal",
  "merchant_id": "merchant_123",
  "master_source_id": "master_source_id_abc123",
  "allow_upload": true,
  "reporting_emails": null,
  "cc_emails": null,
  "bcc_emails": null,
  "source_schema": null,
  "mapping_config": null,
  "split_file_basis": "",
  "beam_sftp_push_job": "",
  "row_hash_value_based_split_config": {
    "header_hash_to_master_source_map": {},
    "column_joiner": ""
  }
}
```

**Response**:
```json
{
  "id": "merchant_source_id_xyz789",
  "name": "POS Transaction Source - Merchant Portal",
  "merchant_id": "merchant_123",
  "master_source_id": "master_source_id_abc123"
}
```

### Update Merchant Source
**PATCH** `/v1/admin-recon-saas/sources/update_merchant/{id}`

```json
{
  "reporting_emails": ["reports@company.com"],
  "cc_emails": ["manager@company.com"],
  "allow_upload": true,
  "slack_notification_config": {
    "recon_percentage_threshold": 90,
    "file_alert_enabled": true
  }
}
```

---

## Recon State APIs

### Create Recon State
**POST** `/v1/admin-recon-saas/recon_state`

```json
{
  "merchant_id": "merchant_123",
  "name": "Reconciled",
  "priority": 2,
  "remarks": "success"
}
```

**Response**:
```json
{
  "id": "recon_state_id_123",
  "merchant_id": "merchant_123",
  "name": "Reconciled",
  "priority": 2,
  "remarks": "success"
}
```

### Update Recon State
**PATCH** `/v1/admin-recon-saas/recon_state/{id}`

```json
{
  "priority": 1,
  "remarks": "Updated remarks"
}
```

---

## Rule APIs

### Create Rule
**POST** `/v1/admin-recon-saas/rule`

```json
{
  "merchant_id": "merchant_123",
  "name": "Reconciled Rule",
  "type": "reconciliation",
  "expression": "source1_id.EntityID == source2_id.EntityID && source1_id.Amount.Equal(source2_id.Amount)",
  "sources": ["master_source_id_1", "master_source_id_2"],
  "recon_state_id": "recon_state_id_123"
}
```

**Response**:
```json
{
  "id": "rule_id_abc",
  "merchant_id": "merchant_123",
  "name": "Reconciled Rule",
  "expression": "..."
}
```

### Update Rule
**PATCH** `/v1/admin-recon-saas/rule/{id}`

```json
{
  "expression": "updated_expression",
  "recon_state_id": "new_state_id"
}
```

---

## Lookup APIs

### Create Lookup
**POST** `/v1/admin-recon-saas/lookup`

```json
{
  "merchant_id": "merchant_123",
  "name": "POS to Bank Lookup",
  "config": [
    {
      "source": "master_source_id_1",
      "Columns": ["EntityID"],
      "aggregation": {
        "enabled": false,
        "conditions": null
      },
      "advanced_config": {
        "enabled": false,
        "cols_config": []
      },
      "lookback_config": {
        "enabled": false,
        "days": 0,
        "direction": "",
        "date_format": "",
        "date_column": ""
      }
    },
    {
      "source": "master_source_id_2",
      "Columns": ["EntityID"],
      "aggregation": {
        "enabled": false,
        "conditions": null
      }
    }
  ]
}
```

**Response**:
```json
{
  "id": "lookup_id_123",
  "merchant_id": "merchant_123",
  "name": "POS to Bank Lookup"
}
```

### Update Lookup
**PATCH** `/v1/admin-recon-saas/lookup/{id}`

```json
{
  "name": "Updated Lookup Name",
  "config": [...]
}
```

---

## Recon Process APIs

### Create Master Recon Process
**POST** `/v1/admin-recon-saas/recon_process/master`

```json
{
  "name": "POS to Bank Statement Reconciliation",
  "product_id": "POS_BANK",
  "lookup_config": [
    {
      "streaming_source_id": "master_source_id_1",
      "config": {
        "master_source_id_2": "lookup_id_123"
      }
    }
  ],
  "rules": {
    "rule_ids": ["rule_id_1", "rule_id_2", "rule_id_3", "rule_id_4"],
    "default_rule": {
      "master_source_id_1": "rule_id_3",
      "master_source_id_2": "rule_id_4"
    }
  },
  "sources": ["master_source_id_1", "master_source_id_2"],
  "sequence": [
    {"master_source_id": "master_source_id_1", "priority": 1},
    {"master_source_id": "master_source_id_2", "priority": 2}
  ],
  "report_config": {
    "source_report_config": [
      {
        "master_source_id": "master_source_id_1",
        "column_map": [
          {"source_column": "EntityID", "report_column": "Transaction ID", "type": "", "id": ""}
        ],
        "report_name": "POS Report"
      }
    ],
    "frontend_cols": ["transaction_id", "amount", "date", "status"]
  }
}
```

**Response**:
```json
{
  "id": "master_process_id_123",
  "name": "POS to Bank Statement Reconciliation",
  "product_id": "POS_BANK"
}
```

### Create Merchant Recon Process
**POST** `/v1/admin-recon-saas/recon_process/merchant`

```json
{
  "merchant_id": "merchant_123",
  "master_recon_process_id": "master_process_id_123",
  "sources": ["merchant_source_id_1", "merchant_source_id_2"],
  "status": "approved",
  "report_config": null
}
```

**Response**:
```json
{
  "id": "merchant_process_id_456",
  "merchant_id": "merchant_123",
  "master_recon_process_id": "master_process_id_123",
  "status": "approved"
}
```

### Update Master Recon Process
**PATCH** `/v1/admin-recon-saas/recon_process/master/{id}`

### Update Merchant Recon Process
**PATCH** `/v1/admin-recon-saas/recon_process/merchant/{id}`

```json
{
  "status": "approved",
  "report_config": {...}
}
```

---

## Error Codes

| Code | Meaning |
|------|---------|
| 400 | Bad Request - Invalid payload structure |
| 401 | Unauthorized - Check authentication |
| 404 | Not Found - Entity doesn't exist |
| 409 | Conflict - Duplicate name |
| 422 | Unprocessable Entity - Business logic error |
| 500 | Internal Server Error |
