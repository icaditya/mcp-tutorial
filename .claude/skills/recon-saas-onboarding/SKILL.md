---
name: recon-saas-onboarding
description: Onboard merchants to the recon-saas reconciliation platform. Use when the user wants to set up reconciliation, create master sources, merchant sources, rules, recon states, lookups, or recon processes. Handles file analysis, column mapping, transformation configuration, and complete merchant onboarding workflows.
version: 1.0.0
tags: [recon, reconciliation, onboarding, merchant, api, razorpay]
context: codebase
---

# Recon-SaaS Merchant Onboarding

Guide for onboarding merchants to the recon-saas reconciliation platform. This skill enables complete merchant setup including file analysis, source creation, rule configuration, and process setup.

## Quick Start

### What is Recon-SaaS?
A reconciliation platform that matches records between two data sources (e.g., POS transactions vs bank statements) using EntityID and Amount fields.

### Onboarding Flow (6 Steps)

1. **File Analysis** - Analyze files, identify columns, recommend EntityID/Amount
2. **Master Source Creation** - Create master source configs with column schemas
3. **Merchant Source Creation** - Create merchant-specific sources linked to master sources
4. **Recon State Creation** - Define reconciliation outcomes (Reconciled, Unreconciled)
5. **Rule Creation** - Define matching logic expressions
6. **Process Setup** - Create lookup, master process, and merchant process

## Core Concepts

### EntityID
Unique identifier column for matching records between sources (e.g., transaction_id, reference_number).

### Amount
Monetary value column for reconciliation comparison.

### EntityIdentifier (Optional)
Secondary grouping column for aggregation scenarios (multiple line items per transaction).

### Column Mapping
- **source**: Original column name from file
- **destination**: Target column name (snake_case, except EntityID/Amount)
- **value**: Always empty string ""

## API Configuration

### Environments
| Environment | Base URL |
|-------------|----------|
| local | http://localhost:9400 |
| dev | https://recon-saas.dev.razorpay.in |
| prod | https://recon-saas.concierge.razorpay.com |

### Authentication
```
Authorization: Basic cmVjb24tc2FhczpyZWNvbi1zYWFz
Content-Type: application/json
```

## Onboarding Workflow Details

### Step 1: File Analysis

Analyze both reconciliation files to:
- Extract column names
- Identify EntityID candidates (unique identifiers)
- Identify Amount candidates (monetary values)
- Recommend mappings

**EntityID Candidates** (priority order):
1. Columns named: transaction_id, entity_id, id, reference_number, ref_no
2. Columns with 95%+ unique values
3. Alphanumeric identifiers with consistent patterns

**Amount Candidates**:
- Columns named: amount*, *amount*, value*, total*, balance*, price*
- Columns with numerical monetary values

### Step 2: Master Source Creation

**Endpoint**: POST `/v1/admin-recon-saas/sources/create`

**Key Rules**:
- All columns in source_schema must have type: "string"
- EntityID column maps to destination "EntityID"
- Amount column maps to destination "Amount"
- All other columns: destination = snake_case(source)

See [references/api-reference.md](references/api-reference.md) for complete payload structure.

### Step 3: Merchant Source Creation

**Endpoint**: POST `/v1/admin-recon-saas/sources/create_merchant`

Create merchant-specific sources linked to master sources with merchant_id.

### Step 4: Recon State Creation

**Endpoint**: POST `/v1/admin-recon-saas/recon_state`

Create 4 states:

| State | Priority | Remarks |
|-------|----------|---------|
| Reconciled | 2 | success |
| Unreconciled | 3 | Amount mismatch |
| Unreconciled | 3 | Record not found in [Source A] |
| Unreconciled | 3 | Record not found in [Source B] |

### Step 5: Rule Creation

**Endpoint**: POST `/v1/admin-recon-saas/rule`

**Rule Expressions**:

1. **Reconciled**:
   ```
   {master_source_id_1}.EntityID == {master_source_id_2}.EntityID && {master_source_id_1}.Amount.Equal({master_source_id_2}.Amount)
   ```

2. **Amount Mismatch**:
   ```
   {master_source_id_1}.EntityID == {master_source_id_2}.EntityID && !{master_source_id_1}.Amount.Equal({master_source_id_2}.Amount)
   ```

3. **Missing Record**:
   ```
   NoRecord.Value == true
   ```

### Step 6: Process Setup

Create in sequence:
1. **Lookup** - POST `/v1/admin-recon-saas/lookup`
2. **Master Recon Process** - POST `/v1/admin-recon-saas/recon_process/master`
3. **Merchant Recon Process** - POST `/v1/admin-recon-saas/recon_process/merchant`

## Transformations

Apply transformations ONLY when explicitly requested by the user.

Common transformations:
- `regex_exec`: Extract using regex pattern
- `append_multiple_columns`: Concatenate columns
- `change_date_format`: Convert date formats
- `abs_amount_parsing`: Parse amounts (remove commas, handle negatives)

See [references/transformations.md](references/transformations.md) for complete list.

## Aggregation

Enable aggregation when multiple records share the same EntityID but differ by EntityIdentifier.

See [references/aggregation.md](references/aggregation.md) for configuration details.

## Entity Updates

**PATCH Endpoints**:
- Master Source: `/v1/admin-recon-saas/sources/update/{id}`
- Merchant Source: `/v1/admin-recon-saas/sources/update_merchant/{id}`
- Rule: `/v1/admin-recon-saas/rule/{id}`
- Recon State: `/v1/admin-recon-saas/recon_state/{id}`
- Lookup: `/v1/admin-recon-saas/lookup/{id}`
- Master Process: `/v1/admin-recon-saas/recon_process/master/{id}`
- Merchant Process: `/v1/admin-recon-saas/recon_process/merchant/{id}`

## Additional Resources

- [API Reference](references/api-reference.md) - Complete API endpoints and payloads
- [Data Models](references/data-models.md) - Entity structures and relationships
- [Transformations](references/transformations.md) - Available transformation functions
- [Aggregation](references/aggregation.md) - Aggregation configuration guide
- [Examples](references/examples.md) - Common onboarding scenarios

## Backend Code Reference

The recon-saas backend code is available at `recon-saas/` in this repository. Key directories:
- `internal/source/` - Master and Merchant source models
- `internal/reconprocess/` - Recon process models
- `internal/rule/` - Rule models
- `internal/reconstate/` - Recon state models
- `internal/lookup/` - Lookup models
- `internal/common/` - Shared configurations
