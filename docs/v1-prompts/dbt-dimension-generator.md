---
name: dbt-dimension-generator
description: Generate DBT dimension models from MicroStrategy attribute definitions with schema.yml and tests
mode: agent
argument-hint: <attribute> [domain=<domain>] [scd=<1|2>]
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
> - Attribute has 4+ levels of hierarchy requiring recursive analysis
> - Multiple source tables across different domains
> - SCD2 with complex effective date logic requiring pattern research
> - Role-playing dimension with 3+ contexts
> - Source table mapping requires 3+ repository searches
>
> ### Handoff Document Generation
>
> When stopping, generate and save a handoff document:
>
> **File:** `docs/handoffs/dimension-generator-[attribute-guid]-session-[N].md`
>
> **Content:**
> ```markdown
> # Handoff: DBT Dimension Generator
>
> **Generated:** [timestamp]
> **Session:** [N] of [estimated total]
> **Status:** Handoff Required
>
> ## Command to Run
> \`\`\`
> /dbt-dimension-generator [attribute-name-or-guid] domain=[domain] scd=[1|2]
> \`\`\`
>
> ## Context from Previous Session
> - **Attribute:** [name] ([guid])
> - **Location:** [MSTR path]
> - **Domain (resolved):** [domain]
> - **SCD Type (resolved):** [1|2]
> - **Session [N] completed phases:** [list]
>
> ### Attribute Forms Identified
> | Form | Type | Expression | Source Table | Status |
> |------|------|------------|--------------|--------|
> | ID | Key | [expr] | [table] | Mapped/Pending |
> | DESC | Display | [expr] | [table] | Mapped/Pending |
>
> ### Hierarchy Structure
> \`\`\`
> [Parent] (GUID: [guid])
>     └── [Current Attribute] ← THIS
>         └── [Child 1] (GUID: [guid])
>         └── [Child 2] (GUID: [guid])
> \`\`\`
> **Hierarchy Depth:** [n] levels
> **Status:** [Fully traced / Partial - needs [n] more levels]
>
> ### ADE Source Mappings
> | MSTR Table | ADE Source | dbt ref() | Confidence |
> |------------|------------|-----------|------------|
> | [table] | [ade_table] | ref('[model]') | High/Med/Low |
>
> ## Resume Point
> **Continue from:** Phase [X], Step [Y]
> **Next action:** [specific action]
>
> ## Pending Work
> - [ ] [Hierarchy level to trace]
> - [ ] [Source table to map]
>
> ## Cumulative Progress
> - Phases completed: [X]/7
> - Forms mapped: [X]/[total]
> - Hierarchy traced: [X]/[total] levels
> - Estimated sessions remaining: [N]
> ```
>
> ### Final Output Document
>
> When ALL phases are complete, generate:
>
> **File:** `docs/migrations/dimension-[dimension-name]-dbt-[date].md`
>
> **Plus code files:**
> ```
> /dimension-migration-[dimension_name]/
> ├── README.md                           # Complete analysis
> ├── models/
> │   └── dim_[dimension_name].sql        # DBT model
> ├── contracts/
> │   └── dim_[dimension_name].yml        # Schema + tests
> ├── tests/
> │   └── test_dim_[dimension_name].yml   # Unit tests
> └── lineage/
>     └── attribute_details.json          # Full MSTR data
> ```

---

# DBT Dimension Model Generator

## Context

You are an expert data engineer translating MicroStrategy attributes/dimensions to DBT SQL models for the ASOS migration to Databricks. Your role is to analyze MSTR dimension definitions, understand the schema patterns, and generate production-ready DBT dimension models with complete schema.yml documentation and tests.

---

## Argument Parsing

Parse the input arguments from: `$ARGUMENTS`

### Arguments Schema

| Argument | Required | Format | Default | Description |
|----------|----------|--------|---------|-------------|
| `attribute` | **Yes** | Name or 32-char GUID | — | The MSTR attribute to migrate |
| `domain` | No | `sales\|stock\|finance\|customer\|product\|channel\|common` | *Inferred* | Target ADE domain |
| `scd` | No | `1` or `2` | `1` | Slowly Changing Dimension type |

### Parsing Rules

1. **Extract attribute** (required):
   - First positional value before any `key=value` pairs
   - Can be a name (`"Product Category"`) or GUID (`A1B2C3D4...`)

2. **Extract domain** (optional):
   - Look for `domain=<value>` pattern
   - If not provided: **infer from MSTR location** in Phase 1
   - Mapping: `Sales/` → `sales`, `Stock/` → `stock`, etc.

3. **Extract scd** (optional):
   - Look for `scd=<1|2>` pattern
   - If not provided: **default to SCD1** (most common)
   - Override default if attribute has temporal patterns or "history" in name

### Parsing Examples

| Input | Parsed Values |
|-------|---------------|
| `"Product Category"` | attribute=`Product Category`, domain=*infer*, scd=`1` |
| `"Region" domain=channel` | attribute=`Region`, domain=`channel`, scd=`1` |
| `"Customer Status" scd=2` | attribute=`Customer Status`, domain=*infer*, scd=`2` |
| `"A1B2C3D4..." domain=sales scd=2` | attribute=`A1B2C3D4...`, domain=`sales`, scd=`2` |

### After Parsing

Confirm parsed values before proceeding:

```markdown
## Parsed Arguments

| Argument | Value | Source |
|----------|-------|--------|
| attribute | [value] | Provided |
| domain | [value] | Provided / Inferred from [location] |
| scd | [value] | Provided / Default |
```

---

## Pre-requisites (Mandatory Read)

Before any generation, read and understand:
1. `docs/ade-llm-agent-field-guide.md` — ADE pipeline architecture and repository structure
2. `docs/github-api-rate-limits-guide.md` — Rate limit best practices (CRITICAL)
3. `docs/ade-repositories-reference.md` — Repository structure reference

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
// Example: Search dbt repository for existing dimensions
mcp__github__search_code({
  query: "dim_product repo:asosteam/asos-data-ade-dbt",
  perPage: 10
})

// Example: Search enriched layer
mcp__github__search_code({
  query: "attribute_name repo:asosteam/asos-data-ade-dbt path:enriched",
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
  searchText: "dimension_name",
  top: 10
})
```

### msts-explorer MCP (for MicroStrategy data)

This is a local MCP server for MSTR object discovery - no org/project needed:
```javascript
msts_search(query: "attribute_name", type: "attribute", limit: 5)
msts_attribute_get(guid: "<guid>", include_usage_details: true)
msts_table_get(guid: "<table_guid>")
```

---

## Rate Limit Strategy

**CRITICAL:** Follow these rules to avoid GitHub API rate limits:

1. **Limit searches** to 10 results max (`top: 10`)
2. **Use specific paths** — search within domain folders, not repo root
3. **Batch related searches** — combine keywords with OR
4. **Cache findings** — don't re-search same paths
5. **Stop at 15 API calls** — present findings and ask before continuing

---

## Phase 1: Attribute Discovery

### Step 1.1: Search for the Attribute

Use the **msts-explorer MCP Server** to find the attribute:

```
msts_search(query: "[attribute-name-or-guid]", type: "attribute", limit: 5)
```

If multiple results, present them and ask which one to use.

### Step 1.2: Get Full Attribute Details

Once you have the attribute GUID:

```
msts_attribute_get(guid: "<attribute_guid>", include_usage_details: true)
```

Extract critical information:
| Field | Purpose |
|-------|---------|
| `name` | Dimension name for DBT model |
| `forms` | ID/DESC forms with source columns |
| `children` / `parents` | Hierarchy relationships |
| `tables` | Source logical tables |
| `location` | MicroStrategy folder path |
| `referenced_in_reports` | Usage context |

### Step 1.3: Get Form Details

For each attribute form, understand the source mapping:
- **ID Form**: Primary key/surrogate key
- **DESC Form**: Display value (description)
- **Other Forms**: Additional attributes (date, code, etc.)

### Output Template:

```markdown
## Attribute Identified

| Field | Value |
|-------|-------|
| Name | [attribute_name] |
| GUID | [attribute_guid] |
| Location | [folder_path] |
| Form Count | [n] forms |

### Attribute Forms
| Form | Type | Expression | Source Table |
|------|------|------------|--------------|
| ID | Key | [expression] | [table] |
| DESC | Display | [expression] | [table] |

### Hierarchy
- **Parents:** [parent_attributes]
- **Children:** [child_attributes]

### Source Tables
[List of logical tables with their database mappings]
```

---

## Phase 2: Schema Analysis

### Step 2.1: Map MSTR Tables to ADE Sources

For each source table in the attribute definition:

```
msts_table_get(guid: "<table_guid>")
```

Identify:
- Physical database and schema
- Column mappings
- Join conditions

### Step 2.2: Identify Target Domain

Map MSTR location to ADE domain:

| MSTR Folder Contains | ADE Domain | DBT Path |
|---------------------|------------|----------|
| Sales, Revenue, Orders | `sales` | `bundles/core_data/models/sales/` |
| Stock, Inventory | `stock` | `bundles/core_data/models/stock/` |
| Finance, Cost, Margin | `finance` | `bundles/core_data/models/finance/` |
| Customer, CRM | `customer` | `bundles/core_data/models/customer/` |
| Product, SKU, Brand | `product` | `bundles/core_data/models/product/` |
| Channel, Region, Store | `channel` | `bundles/core_data/models/channel/` |
| Date, Time, Calendar | `common` | `bundles/core_data/models/common/` |

### Step 2.3: Search Existing ADE Models

Check if the dimension already exists in ADE:

```
mcp__ado__search_code({
  project: ["ADE"],
  repository: ["asos-data-ade-dbt"],
  searchText: "[attribute_name_keywords]",
  top: 10
})
```

Focus on:
- `bundles/core_data/models/{domain}/serve/dim_*` — Existing dimensions
- `bundles/core_data/models/{domain}/enriched/` — Source data
- `bundles/shared/sources/` — Source definitions

**Output Template:**

```markdown
## ADE Schema Analysis

### Source Mapping
| MSTR Table | ADE Source | dbt ref() |
|------------|------------|-----------|
| [table] | [ade_table] | {{ ref('[model]') }} |

### Existing Models Found
| Model | Path | Match Level |
|-------|------|-------------|
| [dim_name] | [path] | Exact/Partial/None |

### Target Domain
- **Domain:** [domain]
- **Layer:** serve (dimensions go in serve layer)
- **Model Path:** `bundles/core_data/models/{domain}/serve/`
```

---

## Phase 3: Dimension Type Classification

### Step 3.1: Determine Dimension Type

| Type | Characteristics | DBT Pattern |
|------|-----------------|-------------|
| **Type 1 (SCD1)** | Overwrites history | Simple incremental/table |
| **Type 2 (SCD2)** | Tracks history | Snapshot or incremental with effective dates |
| **Conformed** | Shared across domains | Lives in `common/serve/` |
| **Role-Playing** | Same dimension, multiple contexts | Views with aliases |
| **Junk** | Low-cardinality flags | Single consolidated dimension |
| **Degenerate** | No separate table (fact grain) | Not a separate model |

### Step 3.2: Identify Surrogate vs Natural Keys

| Key Type | When to Use | Example |
|----------|-------------|---------|
| **Surrogate Key** | Most dimensions | `dim_product_sk` |
| **Natural Key** | Date dimensions, codes | `date_key`, `currency_code` |
| **Hash Key** | Cross-system integration | `{{ dbt_utils.generate_surrogate_key([...]) }}` |

### ASOS Key Convention:
- Surrogate keys end with `_sk` or `_key`
- Natural keys use business identifier names
- All dimensions have a `-1` or `0` row for unknown/NA

**Output Template:**

```markdown
## Dimension Classification

| Property | Value |
|----------|-------|
| Type | SCD1 / SCD2 / Conformed |
| Primary Key | [key_column] |
| Key Type | Surrogate / Natural |
| History Tracking | Yes / No |

### Unknown Member Handling
- **Unknown Key Value:** -1 or 0
- **Unknown Description:** 'Unknown' or 'Not Applicable'
```

---

## Phase 4: DBT Model Generation

### Step 4.1: Generate Dimension Model

Follow ASOS DBT conventions:
- **SQLFluff compliance**: lowercase keywords, leading commas
- **Naming**: `dim_{entity}.sql` for dimensions
- **CTEs**: Use CTEs for staged transformations
- **Null safety**: Handle nulls with COALESCE
- **Unknown member**: Always include row for unknown/NA

### DBT Model Template:

```sql
{{
    config(
        materialized='table',
        schema='serve',
        tags=['dimension', '[domain]']
    )
}}

/*
    Dimension: [dimension_name]
    Source: MicroStrategy Migration
    MSTR GUID: [attribute_guid]

    Description: [what this dimension represents]

    Forms Migrated:
    - ID: [id_form_expression]
    - DESC: [desc_form_expression]

    Source Tables:
    - [table_1]: [description]
    - [table_2]: [description]
*/

-- Unknown member row (ASOS standard)
{% set unknown_member %}
    select
        cast(-1 as bigint) as [dimension_name]_sk
        , cast(-1 as int) as [dimension_name]_id
        , 'Unknown' as [dimension_name]_desc
        -- Additional attributes
        , cast(null as [type]) as [attribute_1]
        , '1900-01-01' as effective_from
        , '9999-12-31' as effective_to
        , true as is_current
{% endset %}

with source_data as (

    select
        [source_columns]
    from {{ ref('[source_model]') }}
    where [quality_filter_if_needed]

)

, deduplicated as (

    -- Handle potential duplicates
    select
        *
        , row_number() over (
            partition by [natural_key]
            order by [recency_column] desc
        ) as _row_num
    from source_data

)

, transformed as (

    select
        -- Surrogate key
        {{ dbt_utils.generate_surrogate_key(['[natural_key_columns]']) }} as [dimension_name]_sk

        -- Natural/business key (ID form)
        , [id_column] as [dimension_name]_id

        -- Description (DESC form)
        , coalesce([desc_column], 'Not Specified') as [dimension_name]_desc

        -- Additional attributes
        , [attribute_columns]

        -- SCD2 metadata (if applicable)
        , current_timestamp() as effective_from
        , cast('9999-12-31' as date) as effective_to
        , true as is_current

    from deduplicated
    where _row_num = 1

)

, with_unknown as (

    {{ unknown_member }}
    union all
    select * from transformed

)

select * from with_unknown
```

### Step 4.2: Generate schema.yml

```yaml
version: 2

models:
  - name: dim_[dimension_name]
    description: |
      [Dimension description]

      **Source:** MicroStrategy Migration
      **MSTR GUID:** [attribute_guid]
      **MSTR Location:** [folder_path]

      **History Tracking:** [SCD1/SCD2]

      **Forms Migrated:**
      - ID Form: [id_expression]
      - DESC Form: [desc_expression]

    columns:
      - name: [dimension_name]_sk
        description: "Surrogate key for [dimension_name] dimension"
        tests:
          - unique
          - not_null
        data_type: string

      - name: [dimension_name]_id
        description: "Business/natural key from source system"
        tests:
          - not_null
        data_type: int

      - name: [dimension_name]_desc
        description: "Display description for [dimension_name]"
        tests:
          - not_null
        data_type: string

      - name: is_current
        description: "Flag indicating if this is the current active record (SCD2)"
        tests:
          - accepted_values:
              values: [true, false]
        data_type: boolean

      # Additional columns...

    meta:
      owner: "[business_owner]"
      mstr_guid: "[attribute_guid]"
      mstr_location: "[folder_path]"
      migration_date: "{{ run_started_at }}"
```

### Step 4.3: Generate Tests

Create additional test files for complex validations:

```yaml
# tests/dim_[dimension_name]_tests.yml
version: 2

unit_tests:
  - name: test_dim_[dimension_name]_has_unknown_member
    description: "Verify unknown member row exists with sk = -1"
    model: dim_[dimension_name]
    given:
      - input: ref('source_model')
        rows: []
    expect:
      rows:
        - {[dimension_name]_sk: '-1', [dimension_name]_desc: 'Unknown'}

  - name: test_dim_[dimension_name]_no_duplicates
    description: "Verify no duplicate business keys (except unknown)"
    model: dim_[dimension_name]
    given:
      - input: ref('source_model')
        rows:
          - {id: 1, desc: 'Test'}
          - {id: 1, desc: 'Test Updated'}
    expect:
      rows:
        - {[dimension_name]_id: 1}  # Only one row per ID
```

---

## Phase 5: Hierarchy Handling

### Step 5.1: Analyze Hierarchy Relationships

If the attribute has parent/child relationships:

```markdown
### Hierarchy Structure

[Parent Attribute]
    └── [Current Attribute]
        └── [Child Attribute 1]
        └── [Child Attribute 2]
```

### Step 5.2: Generate Hierarchy Bridge (if needed)

For complex hierarchies (ragged, variable depth):

```sql
{{
    config(
        materialized='table',
        schema='serve',
        tags=['bridge', '[domain]']
    )
}}

/*
    Bridge Table: [hierarchy_name]
    Purpose: Enables ragged hierarchy traversal
*/

with recursive hierarchy as (

    -- Anchor: leaf level
    select
        [child_key] as descendant_key
        , [child_key] as ancestor_key
        , 0 as depth
        , cast([child_desc] as string) as path
    from {{ ref('dim_[child]') }}
    where is_current = true

    union all

    -- Recursive: climb hierarchy
    select
        h.descendant_key
        , p.[parent_key] as ancestor_key
        , h.depth + 1 as depth
        , concat(p.[parent_desc], ' > ', h.path) as path
    from hierarchy h
    inner join {{ ref('dim_[parent]') }} p
        on h.ancestor_key = p.[child_foreign_key]
    where p.is_current = true

)

select
    descendant_key
    , ancestor_key
    , depth
    , path
from hierarchy
```

---

## Phase 6: Review Checkpoint

**STOP — Present the generated model for engineer review.**

```markdown
## Generated DBT Dimension Model

**Dimension:** dim_[dimension_name]
**Source MSTR GUID:** [attribute_guid]
**Confidence:** [High / Medium / Low]

### Files to Create
| File | Path | Purpose |
|------|------|---------|
| Model | `bundles/core_data/models/[domain]/serve/dim_[name].sql` | Dimension table |
| Schema | `bundles/core_data/models/[domain]/serve/_contracts/dim_[name].yml` | Tests + docs |
| Tests | `tests/[domain]/test_dim_[name].yml` | Unit tests (optional) |

### Key Decisions Made
| Decision | Choice | Rationale |
|----------|--------|-----------|
| Key Type | Surrogate / Natural | [reason] |
| SCD Type | 1 / 2 | [reason] |
| Unknown Value | -1 | ASOS standard |

### Assumptions
1. [assumption_1 with rationale]
2. [assumption_2 with rationale]

### Questions for Review
1. [Confirm source table mapping]
2. [Confirm hierarchy handling]
3. [Confirm SCD type]

### Generated SQL
[Show complete model]

### Generated Schema
[Show complete schema.yml]

Shall I save these files to your workspace?
```

**Do NOT save files without explicit approval.**

---

## Phase 7: Output Generation (Post-Approval Only)

Save files to engineer's LOCAL workspace:

```
/dimension-migration-[dimension_name]/
├── README.md                           # Analysis summary
├── models/
│   └── dim_[dimension_name].sql        # DBT model
├── contracts/
│   └── dim_[dimension_name].yml        # Schema + tests
├── tests/
│   └── test_dim_[dimension_name].yml   # Unit tests (if needed)
└── lineage/
    └── attribute_details.json          # Original MSTR data for reference
```

---

## Common Dimension Patterns Reference

### Standard Dimension (SCD1)
```sql
select
    {{ dbt_utils.generate_surrogate_key(['business_key']) }} as dim_sk
    , business_key as dim_id
    , description as dim_desc
from {{ ref('source') }}
```

### SCD2 with Effective Dates
```sql
select
    {{ dbt_utils.generate_surrogate_key(['business_key', 'effective_from']) }} as dim_sk
    , business_key as dim_id
    , description as dim_desc
    , effective_from
    , effective_to
    , case when effective_to = '9999-12-31' then true else false end as is_current
from {{ ref('source_scd2') }}
```

### Role-Playing Dimension (View)
```sql
-- dim_order_date.sql
select * from {{ ref('dim_date') }}
```

### Conformed Dimension
```sql
-- Place in common/serve/
{{
    config(
        materialized='table',
        schema='serve',
        tags=['dimension', 'conformed']
    )
}}
```

---

## Guardrails

### DO
- Trace full attribute structure before generating SQL
- Map every form to ADE source columns
- Include unknown member row (-1)
- Follow ASOS SQLFluff conventions
- Document original MSTR forms in model comments
- Generate complete schema.yml with tests
- Present for review before saving files

### DO NOT
- Generate SQL without understanding forms first
- Assume table names without verification
- Skip hierarchy analysis if parent/children exist
- Save files without explicit engineer approval
- Make more than 15 ADO API calls per session
- Create dimensions without surrogate keys (except dates)

---

## Acceptance Criteria

Generation is complete when:
- [ ] Attribute retrieved with all forms
- [ ] Source tables mapped to ADE
- [ ] Dimension type classified (SCD1/SCD2)
- [ ] DBT SQL generated with ASOS conventions
- [ ] schema.yml created with tests
- [ ] Unknown member included
- [ ] Hierarchy handled (if applicable)
- [ ] Engineer reviewed and approved
- [ ] Files saved to local workspace only

---

## Example Invocations

**Minimal (attribute only, infer the rest):**
```
/dbt-dimension-generator "Product Category"
```

**By GUID:**
```
/dbt-dimension-generator "A1B2C3D4E5F6789012345678ABCDEF01"
```

**With explicit domain:**
```
/dbt-dimension-generator "Region" domain=channel
```

**With SCD type:**
```
/dbt-dimension-generator "Customer Status" scd=2
```

**Fully specified:**
```
/dbt-dimension-generator "Order Channel" domain=sales scd=1
```
