---
name: parity-validation
description: Generate parity validation SQL comparing ADE against existing Power BI/MSTR models with variance analysis
mode: agent
argument-hint: [metric-name-or-context]
---

> # 🛑 MANDATORY FIRST ACTION — DO NOT SKIP
>
> **BEFORE doing ANYTHING else, you MUST read this file:**
>
> ```
> Read: docs/ade-llm-agent-field-guide.md
> ```
>
> ## WHY THIS IS NON-NEGOTIABLE
>
> This file tells you **which MCP server to use for which repository**:
>
> | Repository Pattern | MCP Server | Example |
> |--------------------|------------|---------|
> | `asos-data-ade-*` | **GitHub** (`mcp__github__search_code`) | dbt, powerbi, contracts, processing |
> | `asos-dataservices-*` | **ADO** (`mcp__ado__search_code`) | ads, edw |
>
> **If you skip this read, you WILL use the wrong MCP server and waste all your API calls with zero results.**
>
> This happened before. Don't repeat it.
>
> ---

> **IMPORTANT: Enterprise-Level Quality Standards**
>
> This is enterprise-level work. The following standards are **MANDATORY** for all outputs:
>
> 1. **No Guessing** — All suggestions must be evidence-based. Never infer or assume.
> 2. **Confidence Levels** — Provide confidence level (High/Medium/Low) for every step, deliverable, and section.
> 3. **Source References** — All file/code references must include `path/file.ext:line_number` format.
> 4. **Pattern References** — Suggested code must point to similar existing code in the repo with proper source paths.
> 5. **Rationales Required** — All key suggestions or recommendations must include rationale and confidence level.
> 6. **No Assumptions** — Only output informed, verified information. Flag uncertainties explicitly.
>
> Failure to follow these standards invalidates the output.

> **SESSION MANAGEMENT: Handoffs and Final Output**
>
> This command supports multi-session execution with proper handoffs.
>
> ### Phase 0: Session Detection (ALWAYS RUN FIRST)
>
> Check if input contains `## Context from Previous Session`:
> - **If YES**: This is a RESUME session. Parse the handoff document and continue from the specified phase.
> - **If NO**: This is a NEW session. Start from Phase 1.
>
> ### Handoff Triggers
>
> **STOP and generate handoff when:**
> - Validation spans 3+ source tables requiring complex joins
> - Schema discovery requires 10+ API calls
> - Multiple metrics require separate validation queries
> - Column mapping ambiguous for 5+ fields
> - Variance thresholds need business confirmation
>
> ### Handoff Document Generation
>
> When stopping, generate and save a handoff document:
>
> **File:** `docs/handoffs/parity-validation-[target]-session-[N].md`
>
> **Content:**
> ```markdown
> # Handoff: Parity Validation
>
> **Generated:** [timestamp]
> **Session:** [N] of [estimated total]
> **Status:** Handoff Required
>
> ## Command to Run
> \`\`\`
> /parity-validation [metric-or-table-name]
> \`\`\`
>
> ## Context from Previous Session
> - **Validation Target:** [name]
> - **Source System:** MSTR / Power BI / EDW
> - **Target System:** ADE ([schema].[table])
> - **Session [N] completed phases:** [list]
> - **API calls used:** [n]/15
>
> ### Source Definition Retrieved
> | Property | Value | Confidence |
> |----------|-------|------------|
> | Formula | [formula] | High/Med/Low |
> | Grain | [grain] | High/Med/Low |
> | Dimensions | [list] | High/Med/Low |
>
> ### ADE Schema Mapping
> | Source Column | ADE Column | Table | Transform | Confidence |
> |---------------|------------|-------|-----------|------------|
> | [col] | [ade_col] | [table] | [none/cast] | High/Med/Low |
>
> ### Queries Generated
> | Query Type | Status | File |
> |------------|--------|------|
> | ADE Aggregation | Complete/Partial | [path] |
> | Source Query | Complete/Partial | [path] |
> | Variance Query | Complete/Pending | [path] |
> | Outlier Query | Pending | — |
>
> ## Resume Point
> **Continue from:** Phase [X], Step [Y]
> **Next action:** [specific action]
>
> ## Pending Work
> - [ ] [Column to map]
> - [ ] [Query to generate]
>
> ## Cumulative Progress
> - Phases completed: [X]/5
> - Columns mapped: [X]/[total]
> - Queries generated: [X]/4
> - Estimated sessions remaining: [N]
> ```
>
> ### Final Output Document
>
> When ALL phases are complete, generate:
>
> **File:** `docs/validations/parity-[target-name]-[date].md`
>
> **Plus query files:**
> ```
> /parity-validation-[target]/
> ├── README.md                    # Complete validation report
> ├── queries/
> │   ├── ade_aggregation.sql     # ADE query
> │   ├── source_query.sql        # Source system query
> │   ├── variance_comparison.sql # Variance analysis
> │   └── outlier_detection.sql   # Outlier identification
> └── results/
>     └── variance_report.md       # Template for results
> ```

---

# Parity Validation Agent

Generate SQL-based parity validation between ADE (ASOS Data Environment) and existing Power BI/MSTR models, producing variance reports with drill-down capabilities.

**Input:** [metric-name-or-context]

---

## Pre-requisites (Mandatory Read)

Before any analysis, read and understand:
1. `docs/ade-llm-agent-field-guide.md` — ADE repository structure and data flow
2. `docs/github-api-rate-limits-guide.md` — Rate limit best practices (CRITICAL)
3. `docs/stock-sales-metrics-ade-parity-analysis.md` — Example parity analysis patterns

---

## MCP Server Configuration (CRITICAL)

> **⚠️ STOP AND THINK before EVERY MCP call:**
>
> Ask yourself: "Which MCP server am I about to call? What are the correct parameters for this server?"
>
> Check this reference section if unsure. Mixing up servers or parameters causes errors and wastes API calls.

### GitHub MCP (for ADE Core repos)

**Organization:** `asosteam`

**Repos available:** `asos-data-ade-dbt`, `asos-data-ade-powerbi`, `asos-data-ade-processingzone-contracts`, `asos-data-ade-processingzone-processing`

```javascript
// Example: Search dbt models
mcp__github__search_code({
  query: "fact_billed_sale repo:asosteam/asos-data-ade-dbt",
  perPage: 10
})

// Example: Search Power BI measures
mcp__github__search_code({
  query: "metric_name repo:asosteam/asos-data-ade-powerbi",
  perPage: 10
})
```

### Azure DevOps (ADO) MCP (for DataServices repos)

**Project:** `DataServices`

**Repos available:** `asos-dataservices-ads`, `asos-dataservices-edw`, `asos-dataservices-powerbi` (LEGACY)

```javascript
// Example: Search legacy Power BI (ADO)
mcp__ado__search_code({
  project: ["DataServices"],
  repository: ["asos-dataservices-powerbi"],
  searchText: "metric_name",
  top: 5
})

// Example: Search ADS warehouse
mcp__ado__search_code({
  project: ["DataServices"],
  repository: ["asos-dataservices-ads"],
  searchText: "keyword",
  top: 10
})
```

### msts-explorer MCP (for MicroStrategy data)

This is a local MCP server for MSTR object discovery - no org/project needed:
```javascript
msts_metric_get(guid: "<guid>")
msts_trace_metric(guid: "<guid>", depth: 5)
```

---

## Rate Limit Strategy (CRITICAL)

**GitHub API constraints:**
- 5,000 requests/hour (authenticated)
- Max 100 concurrent requests
- 900 points/minute for REST (GET = 1 point)

**Mitigation rules:**
1. **Limit searches** to `top: 10` or less
2. **Use specific paths** — never search from repo root
3. **Cache findings** — document all paths, don't re-search
4. **Batch by domain** — one domain at a time
5. **Stop at 15 ADO calls** — present findings and ask to continue
6. **Use conditional requests** — ETags for repeated checks

---

## Phase 1: Input Analysis

### Step 1.1: Identify Validation Scope

Determine validation type from input:

| Input Type | Validation Approach |
|------------|---------------------|
| Metric name | Single metric comparison |
| Work item ID | Extract metrics from WI requirements |
| Domain (sales/stock) | Domain-wide coverage validation |
| Table name | Table-level row/column validation |

### Step 1.2: Gather Source Definitions

**For MSTR metrics:**
```
msts_metric_get(guid="<guid>")
msts_trace_metric(guid="<guid>", depth=5)
```

Extract:
| Field | Purpose |
|-------|---------|
| `formula` | Business logic to validate |
| `referenced_facts` | Source tables for joins |
| `referenced_attributes` | Dimensions for grouping |
| `usage_count` | Prioritization indicator |

**For Power BI metrics:**
Search ADO repository:
```
mcp__ado__search_code({
  project: ["DataServices"],
  repository: ["asos-dataservices-powerbi"],
  searchText: "[metric_name]",
  top: 5
})
```

### Output Phase 1:
```markdown
## Validation Scope

| Property | Value |
|----------|-------|
| Metric/Entity | [name] |
| Source System | MSTR / Power BI / EDW |
| Target System | ADE |
| Definition | [formula] |
| Key Dimensions | [list] |
| Expected Grain | [daily/weekly/monthly] |
```

---

## Phase 2: ADE Schema Discovery

### Step 2.1: Map Domain to ADE Path

| Domain | ADE dbt Path | Primary Tables |
|--------|--------------|----------------|
| Sales | `bundles/core_data/models/sales/serve/` | `fact_billed_sale_v1`, `fact_order_v1` |
| Stock | `bundles/core_data/models/supply_chain/serve/` | `fact_inventory_daily_snapshot_v2` |
| Finance | `bundles/core_data/models/finance/serve/` | `fact_journal_entry_v1` |
| Customer | `bundles/core_data/models/customer/serve/` | `dim_customer_v1` |
| Product | `bundles/core_data/models/product/serve/` | `dim_product_v1` |

### Step 2.2: Search ADE Repository

Search dbt models (limit 10 results):
```
mcp__ado__search_code({
  project: ["ADE"],
  repository: ["asos-data-ade-dbt"],
  searchText: "[metric_keywords]",
  top: 10
})
```

Focus paths (priority order):
1. `bundles/core_data/models/{domain}/serve/` — Final semantic models
2. `bundles/core_data/models/{domain}/curated/` — Aggregated calculations
3. `bundles/core_data/models/{domain}/enriched/` — Transformed data

### Step 2.3: Extract Column Mappings

From dbt contracts (`_contracts/*.yml`):
| MSTR Column | ADE Column | Data Type | Transform |
|-------------|------------|-----------|-----------|
| [source] | [target] | [type] | [any conversion] |

### Output Phase 2:
```markdown
## ADE Schema Mapping

### Target Table
- **Catalog**: `hive_metastore`
- **Schema**: `{domain}.serve`
- **Table**: `{table_name}`
- **Grain**: [grain description]

### Column Mapping
| Source (MSTR/PBI) | Target (ADE) | Transform |
|-------------------|--------------|-----------|
| [col1] | [col1_ade] | [none/cast/etc] |

### API Calls Used: [n]/15
```

---

## Phase 3: Validation SQL Generation

### Step 3.1: Aggregation Query Template

Generate SQL for both source and target systems:

**ADE Query (Databricks SQL):**
```sql
-- Parity Validation: [Metric Name]
-- Generated: [timestamp]
-- Purpose: Compare ADE values against [Source System]

WITH ade_aggregation AS (
    SELECT
        -- Grouping dimensions
        [dim_date.calendar_date] AS validation_date,
        [dim_product.product_code] AS product_key,
        [dim_warehouse.warehouse_code] AS location_key,

        -- Measures
        SUM([measure_column]) AS ade_value,
        COUNT(*) AS ade_row_count
    FROM hive_metastore.[schema].[table] fact
    LEFT JOIN hive_metastore.[schema].dim_date d
        ON fact.dim_date_sk = d.dim_date_sk
    LEFT JOIN hive_metastore.[schema].dim_product p
        ON fact.dim_product_sk = p.dim_product_sk
    -- Additional dimension joins
    WHERE [date_filter]
    GROUP BY 1, 2, 3
)
SELECT
    validation_date,
    product_key,
    location_key,
    ade_value,
    ade_row_count
FROM ade_aggregation
ORDER BY validation_date DESC, ade_value DESC
LIMIT 1000;
```

**Source Query (MSTR/EDW):**
```sql
-- Source System Query: [Metric Name]
-- System: [MSTR/EDW/Power BI]

SELECT
    -- Matching dimensions
    [DATE_COLUMN] AS validation_date,
    [PRODUCT_KEY] AS product_key,
    [LOCATION_KEY] AS location_key,

    -- Measures (matching ADE calculation)
    SUM([MEASURE]) AS source_value,
    COUNT(*) AS source_row_count
FROM [SOURCE_TABLE]
WHERE [date_filter]
GROUP BY 1, 2, 3
ORDER BY validation_date DESC, source_value DESC;
```

### Step 3.2: Variance Calculation Query

```sql
-- Parity Variance Analysis: [Metric Name]
-- Compares: ADE vs [Source System]
-- Date Range: [start] to [end]

WITH ade_data AS (
    -- [ADE aggregation query from 3.1]
    SELECT
        validation_date,
        product_key,
        location_key,
        ade_value,
        ade_row_count
    FROM [ade_query]
),

source_data AS (
    -- [Source aggregation - replace with actual values from source export]
    SELECT
        validation_date,
        product_key,
        location_key,
        source_value,
        source_row_count
    FROM [source_query_or_import_table]
)

SELECT
    COALESCE(a.validation_date, s.validation_date) AS validation_date,
    COALESCE(a.product_key, s.product_key) AS product_key,
    COALESCE(a.location_key, s.location_key) AS location_key,

    -- Values
    s.source_value,
    a.ade_value,

    -- Absolute variance
    (a.ade_value - s.source_value) AS absolute_variance,

    -- Percentage variance
    CASE
        WHEN s.source_value = 0 THEN NULL
        ELSE ROUND(((a.ade_value - s.source_value) / s.source_value) * 100, 2)
    END AS pct_variance,

    -- Variance classification
    CASE
        WHEN a.ade_value IS NULL THEN 'MISSING_IN_ADE'
        WHEN s.source_value IS NULL THEN 'MISSING_IN_SOURCE'
        WHEN ABS((a.ade_value - s.source_value) / NULLIF(s.source_value, 0)) <= 0.001 THEN 'MATCH'
        WHEN ABS((a.ade_value - s.source_value) / NULLIF(s.source_value, 0)) <= 0.01 THEN 'MINOR_VARIANCE'
        WHEN ABS((a.ade_value - s.source_value) / NULLIF(s.source_value, 0)) <= 0.05 THEN 'MODERATE_VARIANCE'
        ELSE 'SIGNIFICANT_VARIANCE'
    END AS variance_status,

    -- Row count comparison
    s.source_row_count,
    a.ade_row_count,
    (a.ade_row_count - s.source_row_count) AS row_count_diff

FROM source_data s
FULL OUTER JOIN ade_data a
    ON s.validation_date = a.validation_date
    AND s.product_key = a.product_key
    AND s.location_key = a.location_key
ORDER BY
    CASE variance_status
        WHEN 'SIGNIFICANT_VARIANCE' THEN 1
        WHEN 'MODERATE_VARIANCE' THEN 2
        WHEN 'MINOR_VARIANCE' THEN 3
        WHEN 'MISSING_IN_ADE' THEN 4
        WHEN 'MISSING_IN_SOURCE' THEN 5
        ELSE 6
    END,
    ABS(pct_variance) DESC NULLS LAST;
```

### Step 3.3: Outlier Detection Query

```sql
-- Outlier Detection: [Metric Name]
-- Identifies records requiring investigation

WITH variance_data AS (
    -- [Variance query from 3.2]
    SELECT * FROM [variance_query]
),

stats AS (
    SELECT
        AVG(pct_variance) AS avg_variance,
        STDDEV(pct_variance) AS stddev_variance,
        PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY ABS(pct_variance)) AS p95_variance
    FROM variance_data
    WHERE variance_status NOT IN ('MISSING_IN_ADE', 'MISSING_IN_SOURCE')
)

SELECT
    v.*,
    s.avg_variance,
    s.stddev_variance,
    s.p95_variance,

    -- Outlier flags
    CASE
        WHEN ABS(v.pct_variance) > (s.avg_variance + 3 * s.stddev_variance) THEN 'STATISTICAL_OUTLIER'
        WHEN ABS(v.pct_variance) > s.p95_variance THEN 'TOP_5_PCT'
        WHEN v.variance_status = 'SIGNIFICANT_VARIANCE' THEN 'NEEDS_REVIEW'
        ELSE 'ACCEPTABLE'
    END AS outlier_category,

    -- Investigation priority
    CASE
        WHEN ABS(v.absolute_variance) > 1000000 THEN 'P1_HIGH_VALUE'
        WHEN v.variance_status = 'MISSING_IN_ADE' THEN 'P2_DATA_GAP'
        WHEN ABS(v.pct_variance) > 10 THEN 'P3_LARGE_DEVIATION'
        ELSE 'P4_MONITOR'
    END AS investigation_priority

FROM variance_data v
CROSS JOIN stats s
WHERE v.variance_status != 'MATCH'
ORDER BY
    investigation_priority,
    ABS(absolute_variance) DESC;
```

### Output Phase 3:
```markdown
## Generated SQL Queries

### 1. ADE Aggregation Query
\`\`\`sql
[ADE query from 3.1]
\`\`\`

### 2. Source System Query
\`\`\`sql
[Source query from 3.1]
\`\`\`

### 3. Variance Comparison Query
\`\`\`sql
[Variance query from 3.2]
\`\`\`

### 4. Outlier Detection Query
\`\`\`sql
[Outlier query from 3.3]
\`\`\`

### Execution Instructions
1. Run ADE query in Databricks SQL Warehouse
2. Export source data from [MSTR/EDW] to temp table or CSV
3. Run variance query with imported source data
4. Review outliers and prioritize investigation
```

---

## Phase 4: Variance Report Template

### Step 4.1: Summary Statistics

```markdown
## Parity Validation Report

**Metric**: [Metric Name]
**Validation Date**: [YYYY-MM-DD]
**Date Range**: [start] to [end]
**Source**: [MSTR/Power BI/EDW]
**Target**: ADE ([schema].[table])

### Executive Summary

| Metric | Value |
|--------|-------|
| Total Records (Source) | [n] |
| Total Records (ADE) | [n] |
| Match Rate | [%] |
| Significant Variances | [n] |
| Missing in ADE | [n] |
| Missing in Source | [n] |

### Variance Distribution

| Category | Count | % of Total | Action |
|----------|-------|------------|--------|
| MATCH (≤0.1%) | [n] | [%] | None |
| MINOR (0.1-1%) | [n] | [%] | Monitor |
| MODERATE (1-5%) | [n] | [%] | Review |
| SIGNIFICANT (>5%) | [n] | [%] | Investigate |
| MISSING_IN_ADE | [n] | [%] | Gap analysis |
| MISSING_IN_SOURCE | [n] | [%] | Validate source |
```

### Step 4.2: Drill-Down Analysis

```markdown
### Top 10 Variances by Absolute Value

| Date | Product | Location | Source | ADE | Variance | % | Status |
|------|---------|----------|--------|-----|----------|---|--------|
| [date] | [prod] | [loc] | [val] | [val] | [diff] | [%] | [status] |

### Variance Trends (by Date)

| Date | Total Source | Total ADE | Variance | % |
|------|--------------|-----------|----------|---|
| [date] | [sum] | [sum] | [diff] | [%] |

### Variance by Dimension

**By Product Category:**
| Category | Match Rate | Avg Variance | Max Variance |
|----------|------------|--------------|--------------|

**By Location:**
| Warehouse | Match Rate | Avg Variance | Max Variance |
|-----------|------------|--------------|--------------|
```

### Step 4.3: Investigation Checklist

```markdown
### Investigation Items

| # | Priority | Issue | Potential Cause | Recommended Action |
|---|----------|-------|-----------------|-------------------|
| 1 | P1 | [description] | [hypothesis] | [action] |
| 2 | P2 | [description] | [hypothesis] | [action] |

### Common Variance Causes

- [ ] **Timing differences**: Source snapshot vs ADE ETL timing
- [ ] **Filter logic**: Different exclusion criteria (returns, cancellations)
- [ ] **Rounding**: Decimal precision differences
- [ ] **Currency conversion**: FX rate timing
- [ ] **Aggregation level**: Different grain or grouping
- [ ] **Data type mismatch**: INT vs DECIMAL truncation
- [ ] **Null handling**: COALESCE vs NULL treatment
- [ ] **Historical restatements**: Source data corrections not in ADE
```

---

## Phase 5: Automation Suggestions

### Step 5.1: Scheduled Validation Job

```yaml
# dbt job config suggestion for automated parity checks
# Location: bundles/core_data/job_config/parity_validation.yml

parity_validation_job:
  name: "Parity Validation - [Domain]"
  schedule:
    quartz_cron_expression: "0 0 8 * * ?"  # Daily at 8 AM
  tasks:
    - task_key: validate_[metric]
      dbt_task:
        commands:
          - "dbt run --select tag:parity_check"
          - "dbt test --select tag:parity_check"
  email_notifications:
    on_failure:
      - data-engineering@asos.com
```

### Step 5.2: dbt Test Integration

```yaml
# Suggested dbt test for continuous parity monitoring
# Location: bundles/core_data/models/{domain}/serve/_contracts/parity_tests.yml

version: 2

models:
  - name: serve_fact_[table]_v1
    tests:
      - dbt_utils.expression_is_true:
          name: parity_check_[metric]
          expression: |
            ABS(SUM(ade_value) - [expected_source_total]) / NULLIF([expected_source_total], 0) < 0.01
          config:
            severity: warn
            tags: ['parity_check']
```

### Output Phase 5:
```markdown
## Automation Recommendations

### Immediate
- [ ] Save validation queries as dbt macros
- [ ] Add parity tests to CI/CD pipeline
- [ ] Configure alerting for >5% variance

### Medium-term
- [ ] Create automated daily parity job
- [ ] Build variance trend dashboard in Power BI
- [ ] Integrate with data quality framework

### Files to Create
| File | Location | Purpose |
|------|----------|---------|
| `parity_[metric].sql` | `macros/parity/` | Reusable validation macro |
| `parity_tests.yml` | `_contracts/` | Automated test definitions |
| `parity_validation.yml` | `job_config/` | Scheduled validation job |
```

---

## Guardrails

### DO
- Read pre-requisite docs before starting
- Use specific repository paths (not root searches)
- Limit ADO searches to 10 results
- Document all column mappings explicitly
- Generate parameterized SQL for reuse
- Include variance thresholds in queries
- Flag ambiguities for engineer review
- Present queries for review before execution

### DO NOT
- Search entire repositories (rate limits!)
- Assume column mappings without verification
- Execute queries without engineer approval
- Generate queries without date filters
- Skip grain/aggregation validation
- Make more than 15 ADO API calls per session
- Proceed without confirming source definitions

---

## Rate Limit Checkpoint

Before each ADO API call, verify:
- [ ] Search is scoped to specific repository
- [ ] Using `top: 10` or less
- [ ] Path is narrowed to domain folder
- [ ] Not duplicating previous search

**If approaching 15 calls, STOP and present findings so far.**

---

## Acceptance Criteria

Validation is complete when:
- [ ] Source system definition retrieved and documented
- [ ] ADE schema mapped with column-level detail
- [ ] Aggregation queries generated for both systems
- [ ] Variance calculation query with thresholds
- [ ] Outlier detection query with prioritization
- [ ] Report template populated with placeholders
- [ ] Investigation checklist created
- [ ] All sources documented with paths
- [ ] Engineer reviewed and approved queries
- [ ] Automation suggestions provided

---

## Example Invocations

**Single metric:**
```
/parity-validation "Saleable Stock Units"
```

**By work item:**
```
/parity-validation "WI-12345: Stock metrics parity check"
```

**Domain-wide:**
```
/parity-validation "sales domain weekly totals"
```

**Table-level:**
```
/parity-validation "fact_billed_sale_v1 vs EDW.dbo.FactSales"
```

---

## Quick Reference

### Variance Thresholds
| Threshold | Classification | Action |
|-----------|----------------|--------|
| ≤0.1% | MATCH | None required |
| 0.1-1% | MINOR | Monitor trends |
| 1-5% | MODERATE | Review logic |
| >5% | SIGNIFICANT | Investigate |

### Common ADE Tables for Parity
| Domain | ADE Table | Typical Source |
|--------|-----------|----------------|
| Sales | `sales.serve.fact_billed_sale_v1` | EDW.dbo.FactSales, MSTR Sales cube |
| Stock | `supply_chain.serve.fact_inventory_daily_snapshot_v2` | MSTR Stock cube, Power BI Stock Health |
| Finance | `finance.serve.fact_journal_entry_v1` | EDW.dbo.FactJournal |

### Dimension Join Keys
| Dimension | ADE SK Column | Typical Business Key |
|-----------|---------------|---------------------|
| Date | `dim_date_sk` | `calendar_date` |
| Product | `dim_product_sk` | `product_code`, `sku` |
| Warehouse | `dim_warehouse_sk` | `warehouse_code` |
| Customer | `dim_customer_sk` | `customer_id` |
