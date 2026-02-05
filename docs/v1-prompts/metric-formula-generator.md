---
name: metric-formula-generator
description: Generate DBT SQL from MicroStrategy metric formulas using lineage tracing
mode: agent
argument-hint: [metric-name-or-guid]
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
> - Metric has 5+ nested dependencies requiring separate traces
> - ADE table mapping requires 3+ repository searches
> - Compound metric with circular or ambiguous references
> - Formula contains unsupported MSTR functions requiring research
> - GBP conversion logic requires currency table investigation
>
> ### Handoff Document Generation
>
> When stopping, generate and save a handoff document:
>
> **File:** `docs/handoffs/metric-generator-[metric-guid]-session-[N].md`
>
> **Content:**
> ```markdown
> # Handoff: Metric Formula Generator
>
> **Generated:** [timestamp]
> **Session:** [N] of [estimated total]
> **Status:** Handoff Required
>
> ## Command to Run
> \`\`\`
> /metric-formula-generator [metric-name-or-guid]
> \`\`\`
>
> ## Context from Previous Session
> - **Metric:** [name] ([guid])
> - **Type:** Simple / Compound
> - **Formula:** [original formula]
> - **Session [N] completed phases:** [list]
>
> ### Dependency Tree (Traced)
> | Level | Object | Type | GUID | Status |
> |-------|--------|------|------|--------|
> | 0 | [metric] | Metric | [guid] | Traced |
> | 1 | [fact] | Fact | [guid] | Traced/Pending |
>
> ### Formula Components Parsed
> | Component | MSTR Function | SQL Translation | Confidence |
> |-----------|---------------|-----------------|------------|
> | [comp] | [func] | [sql] | High/Med/Low |
>
> ### ADE Table Mappings
> | MSTR Fact | ADE Table | Column | Status | Confidence |
> |-----------|-----------|--------|--------|------------|
> | [fact] | [table] | [col] | Mapped/Pending | High/Med/Low |
>
> ## Resume Point
> **Continue from:** Phase [X], Step [Y]
> **Next action:** [specific action]
>
> ## Pending Work
> - [ ] [Fact to trace]
> - [ ] [Table to map]
>
> ## Cumulative Progress
> - Phases completed: [X]/6
> - Dependencies traced: [X]/[total]
> - SQL translation: [%] complete
> - Estimated sessions remaining: [N]
> ```
>
> ### Final Output Document
>
> When ALL phases are complete, generate:
>
> **File:** `docs/migrations/metric-[metric-name]-dbt-[date].md`
>
> **Plus code files:**
> ```
> /metric-migration-[metric_name]/
> ├── README.md                    # Complete analysis
> ├── models/
> │   └── serve_[metric].sql      # DBT model
> ├── contracts/
> │   └── serve_[metric].yml      # Tests + docs
> └── lineage/
>     └── trace_output.json        # Full trace for reference
> ```

---

# Metric Formula Generator: [metric-name-or-guid]

## Context

You are an expert data engineer translating MicroStrategy metric formulas to DBT SQL for the ASOS migration to Databricks. Your role is to trace metric dependencies, understand the MSTR formula, and generate production-ready DBT model code.

---

## Pre-requisites (Read First)

Before any generation, read and understand:
1. `docs/ade-llm-agent-field-guide.md` - ADE pipeline architecture and repository structure
2. `docs/github-api-rate-limits-guide.md` - Rate limit best practices (CRITICAL)
3. `docs/ASOS_Data_Pipeline_Summary.md` - End-to-end data flow

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
// Example: Search dbt models for fact tables
mcp__github__search_code({
  query: "fact_name repo:asosteam/asos-data-ade-dbt path:bundles/core_data/models",
  perPage: 10
})

// Example: Search contracts
mcp__github__search_code({
  query: "table_name repo:asosteam/asos-data-ade-processingzone-contracts",
  perPage: 10
})
```

### Azure DevOps (ADO) MCP (for DataServices repos)

**Project:** `DataServices`

**Repos available:** `asos-dataservices-ads`, `asos-dataservices-edw`, `asos-dataservices-powerbi` (LEGACY)

```javascript
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
msts_search(query: "metric_name", type: "metric", limit: 5)
msts_metric_get(guid: "<guid>")
msts_trace_metric(guid: "<guid>", depth: 5)
msts_fact_get(guid: "<fact_guid>", include_usage_details: true)
```

---

## Phase 1: Metric Discovery

### Step 1.1: Search for the Metric

Use the **msts-explorer MCP Server** to find the metric:

```
msts_search(query: "[metric-name-or-guid]", type: "metric", limit: 5)
```

If multiple results, present them and ask which one to trace.

### Step 1.2: Trace Metric Dependencies

Once you have the metric GUID, run a full trace:

```
msts_trace_metric(guid: "<metric_guid>", depth: 5)
```

### Output Template:

```markdown
## Metric Identified

| Field | Value |
|-------|-------|
| Name | [metric_name] |
| GUID | [metric_guid] |
| Type | Simple / Compound |
| Formula | [formula] |

### Dependency Tree
[Paste the trace result showing parent metrics, facts, and tables]
```

---

## Phase 2: Formula Analysis

### Step 2.1: Parse the MSTR Formula

MicroStrategy formulas use specific syntax. Common patterns:

| MSTR Pattern | Description | DBT SQL Equivalent |
|--------------|-------------|-------------------|
| `Sum(Fact)` | Simple aggregation | `SUM(fact_column)` |
| `Sum(Fact){~+}` | Subtotal enabled | `SUM(fact_column)` with window function |
| `[Metric1] + [Metric2]` | Compound metric | CTE combining metrics |
| `ApplySimple("#0/#1", M1, M2)` | Division | `metric_1 / NULLIF(metric_2, 0)` |
| `NullToZero(Metric)` | Null handling | `COALESCE(metric, 0)` |
| `Avg(Fact)` | Average | `AVG(fact_column)` |
| `Count(Attribute)` | Distinct count | `COUNT(DISTINCT attribute)` |
| `Max/Min(Fact)` | Min/Max | `MAX/MIN(fact_column)` |

### Step 2.2: Identify GBP Currency Conversions

ASOS frequently requires GBP currency conversion. Look for:

- Metrics with "GBP" in name or formula
- References to exchange rate facts
- `ApplySimple` with currency logic

**Standard GBP Conversion Pattern:**
```sql
-- ASOS standard currency conversion to GBP
original_value * COALESCE(exchange_rate.rate_to_gbp, 1) AS value_gbp
```

### Output Template:

```markdown
## Formula Breakdown

### Original MSTR Formula
```
[raw_formula]
```

### Components Identified
| Component | Type | Purpose |
|-----------|------|---------|
| [component_1] | Metric/Fact/Function | [what it does] |

### GBP Conversion Required
- [ ] Yes - [describe conversion pattern]
- [ ] No

### SQL Translation Notes
[Explain how each component translates to SQL]
```

---

## Phase 3: Source Table Mapping

### Step 3.1: Map Facts to ADE Tables

For each fact identified in the trace:

```
msts_fact_get(guid: "<fact_guid>", include_usage_details: true)
```

Then search the DBT repository for existing models:

**Rate Limit Strategy - Use Targeted Searches:**
```
1. Search specific domain folder first (limit: 10)
2. Search serve layer for final tables (limit: 5)
3. Only read files directly matching the fact name
```

### ADE Layer Mapping

| MSTR Layer | ADE Layer | DBT Model Path |
|------------|-----------|----------------|
| Fact Table | curated/serve | `bundles/core_data/models/{domain}/serve/` |
| Attribute | conformed/enriched | `bundles/core_data/models/{domain}/enriched/` |
| Logical Table | raw/conformed | `bundles/core_data/models/{domain}/conformed/` |

### Output Template:

```markdown
## Source Table Mapping

### Facts Required
| MSTR Fact | GUID | ADE Table | DBT Model |
|-----------|------|-----------|-----------|
| [fact_name] | [guid] | [ade_table_name] | [model_path.sql] |

### Attributes Required
| MSTR Attribute | GUID | ADE Column | Source |
|----------------|------|------------|--------|
| [attr_name] | [guid] | [column_name] | [table.column] |

### Join Paths
[Describe how tables connect, typically through date/product/channel keys]
```

---

## Phase 4: DBT Model Generation

### Step 4.1: Generate the DBT SQL

Follow ASOS DBT conventions:
- **SQLFluff compliance**: lowercase keywords, leading commas
- **Naming**: `{layer}_{entity}_{version}.sql`
- **CTEs**: Use CTEs for intermediate calculations
- **Null safety**: Always use `NULLIF` for divisions
- **Currency**: Apply GBP conversion where needed

### DBT Model Template:

```sql
{{
    config(
        materialized='table',
        schema='serve',
        tags=['metrics', '[domain]']
    )
}}

/*
    Metric: [metric_name]
    Source: MicroStrategy Migration
    MSTR GUID: [metric_guid]
    Original Formula: [mstr_formula]

    Description: [what this metric calculates]

    Dependencies:
    - [fact_1]: [description]
    - [fact_2]: [description]
*/

with source_facts as (

    select
        -- Grain columns (typically date, product, channel)
        date_key
        , product_key
        , channel_key
        -- Fact columns
        , [fact_column_1]
        , [fact_column_2]
    from {{ ref('[source_model]') }}

)

, currency_rates as (

    -- Only include if GBP conversion needed
    select
        date_key
        , currency_code
        , rate_to_gbp
    from {{ ref('dim_currency_rates') }}

)

, calculated_metric as (

    select
        sf.date_key
        , sf.product_key
        , sf.channel_key
        -- Metric calculation
        , [SQL_TRANSLATION_OF_MSTR_FORMULA] as [metric_name]
        -- GBP conversion (if needed)
        , [metric_name] * coalesce(cr.rate_to_gbp, 1) as [metric_name]_gbp
    from source_facts sf
    left join currency_rates cr
        on sf.date_key = cr.date_key
        and sf.currency_code = cr.currency_code

)

select * from calculated_metric
```

### Step 4.2: Generate Tests

```yaml
# _contracts/[metric_name].yml
version: 2

models:
  - name: serve_[metric_name]
    description: "[metric_description] - Migrated from MSTR"
    columns:
      - name: [metric_name]
        description: "[original_mstr_formula]"
        tests:
          - not_null:
              severity: warn
      - name: [metric_name]_gbp
        description: "GBP converted value"
        tests:
          - not_null:
              severity: warn
```

---

## Phase 5: Review Checkpoint

**STOP - Present the generated model for engineer review.**

```markdown
## Generated DBT Model

**Metric:** [metric_name]
**Source MSTR GUID:** [metric_guid]
**Confidence:** [High / Medium / Low]

### Files to Create
| File | Path | Purpose |
|------|------|---------|
| Model | `bundles/core_data/models/[domain]/serve/serve_[metric].sql` | Metric calculation |
| Contract | `bundles/core_data/models/[domain]/serve/_contracts/serve_[metric].yml` | Tests + docs |

### Assumptions Made
1. [assumption_1 with rationale]
2. [assumption_2 with rationale]

### Questions for Review
1. [Confirm source table mapping]
2. [Confirm GBP conversion approach]
3. [Confirm aggregation grain]

### Generated SQL
[Show the complete generated model]

Shall I save these files to your workspace?
```

**Do NOT save files without explicit approval.**

---

## Phase 6: Output Generation (Post-Approval Only)

Save files to engineer's LOCAL workspace:

```
/metric-migration-[metric_name]/
├── README.md                    # Analysis summary
├── models/
│   └── serve_[metric].sql      # DBT model
├── contracts/
│   └── serve_[metric].yml      # Tests + docs
└── lineage/
    └── trace_output.json        # Original trace for reference
```

---

## Common MSTR Formula Patterns Reference

### Aggregation Functions
| MSTR | DBT SQL |
|------|---------|
| `Sum(Fact)` | `SUM(fact_column)` |
| `Avg(Fact)` | `AVG(fact_column)` |
| `Count(Attr)` | `COUNT(DISTINCT attr)` |
| `Max(Fact)` | `MAX(fact_column)` |
| `Min(Fact)` | `MIN(fact_column)` |

### Mathematical Functions
| MSTR | DBT SQL |
|------|---------|
| `ApplySimple("#0/#1", M1, M2)` | `m1 / NULLIF(m2, 0)` |
| `ApplySimple("#0*#1", M1, M2)` | `m1 * m2` |
| `ApplySimple("#0-#1", M1, M2)` | `m1 - m2` |
| `Abs(Metric)` | `ABS(metric)` |
| `Round(Metric, n)` | `ROUND(metric, n)` |

### Conditional Functions
| MSTR | DBT SQL |
|------|---------|
| `NullToZero(Metric)` | `COALESCE(metric, 0)` |
| `ZeroToNull(Metric)` | `NULLIF(metric, 0)` |
| `If(Cond, Then, Else)` | `CASE WHEN cond THEN val1 ELSE val2 END` |

### Time Functions
| MSTR | DBT SQL |
|------|---------|
| `RunningSum(Metric)` | `SUM(metric) OVER (ORDER BY date_key)` |
| `MovingAvg(Metric, n)` | `AVG(metric) OVER (ORDER BY date_key ROWS n-1 PRECEDING)` |
| `Lag(Metric, n)` | `LAG(metric, n) OVER (ORDER BY date_key)` |

### Level/Dimensionality
| MSTR Pattern | DBT SQL |
|--------------|---------|
| `{@}` (report level) | Standard aggregation |
| `{~+}` (subtotal) | Window function with PARTITION BY |
| `{Attr}` (at attribute level) | GROUP BY attribute |

---

## Guardrails

### DO
- Trace full dependency tree before generating SQL
- Map every fact/attribute to ADE source
- Include null safety (`NULLIF` for divisions)
- Follow ASOS SQLFluff conventions
- Document original MSTR formula in model comments
- Present for review before saving files

### DO NOT
- Generate SQL without tracing dependencies first
- Assume table names without verification
- Skip currency conversion when metrics involve money
- Save files without explicit engineer approval
- Search entire repositories (rate limit aware)

---

## Acceptance Criteria

Generation is complete when:
- [ ] Metric traced with full dependency tree
- [ ] Formula parsed and documented
- [ ] All facts mapped to ADE tables
- [ ] DBT SQL generated with ASOS conventions
- [ ] Tests/contracts created
- [ ] GBP conversion included (if monetary)
- [ ] Engineer reviewed and approved
- [ ] Files saved to local workspace only
