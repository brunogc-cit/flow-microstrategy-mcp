---
name: mstr-ade-gap-analysis
description: Automated gap analysis comparing MicroStrategy metrics/attributes against ADE schema for migration planning
mode: agent
argument-hint: [metric-or-attribute-name-or-guid]
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
> - Object has 10+ dependencies requiring individual traces
> - Gap analysis spans 3+ ADE domains
> - Approaching 15 ADO API calls limit
> - Batch analysis with 5+ objects
> - Confidence drops below Medium for critical mappings
>
> ### Handoff Document Generation
>
> When stopping, generate and save a handoff document:
>
> **File:** `docs/handoffs/gap-analysis-[object-guid]-session-[N].md`
>
> **Content:**
> ```markdown
> # Handoff: MSTR-ADE Gap Analysis
>
> **Generated:** [timestamp]
> **Session:** [N] of [estimated total]
> **Status:** Handoff Required
>
> ## Command to Run
> \`\`\`
> /mstr-ade-gap-analysis [object-name-or-guid]
> \`\`\`
>
> ## Context from Previous Session
> - **MSTR Object:** [name] ([guid])
> - **Type:** Metric / Attribute
> - **Session [N] completed phases:** [list]
> - **API calls used:** [n]/15
>
> ### Dependencies Traced
> | Type | Count | GUIDs |
> |------|-------|-------|
> | Facts | [n] | [list] |
> | Attributes | [n] | [list] |
> | Tables | [n] | [list] |
>
> ### ADE Searches Completed
> | Repository | Path | Results | Relevance |
> |------------|------|---------|-----------|
> | [repo] | [path] | [n] files | High/Med/Low |
>
> ### Matches Found
> | MSTR Object | ADE Asset | Path | Match Level | Confidence |
> |-------------|-----------|------|-------------|------------|
> | [name] | [model] | [path:line] | Exact/Partial | High/Med/Low |
>
> ### Gaps Identified
> | Gap | Type | Resolution | Confidence |
> |-----|------|------------|------------|
> | [gap] | MISSING/PARTIAL | [action] | High/Med/Low |
>
> ## Resume Point
> **Continue from:** Phase [X], Step [Y]
> **Next action:** [specific action]
>
> ## Pending Work
> - [ ] [Pending search/trace 1]
> - [ ] [Pending search/trace 2]
>
> ## Cumulative Progress
> - Phases completed: [X]/5
> - Dependencies traced: [X]/[total]
> - Estimated sessions remaining: [N]
> ```
>
> ### Final Output Document
>
> When ALL phases are complete, generate:
>
> **File:** `docs/investigations/gap-analysis-[object-name]-[date].md`
>
> This document consolidates ALL findings from ALL sessions into a complete mapping document ready for work item creation.

---

# MSTR to ADE Gap Analysis

Automated comparison of MicroStrategy metrics/attributes against ADE (ASOS Data Environment) schema, generating mapping suggestions and identifying gaps for migration planning.

**Input:** [metric-or-attribute-name-or-guid]

---

## MCP Server Configuration (CRITICAL)

> **⚠️ STOP AND THINK before EVERY MCP call:**
>
> Ask yourself: "Which MCP server am I about to call? What are the correct parameters for this server?"
>
> **THE GOLDEN RULE:**
> - `asos-data-ade-*` repos → **GitHub MCP** (org: `asosteam`)
> - `asos-dataservices-*` repos → **ADO MCP** (project: `DataServices`)

### GitHub MCP (for ADE Core repos)

**Organization:** `asosteam`

**Repos available:** `asos-data-ade-dbt`, `asos-data-ade-powerbi`, `asos-data-ade-processingzone-contracts`, `asos-data-ade-processingzone-processing`

```javascript
// Example: Search dbt models
mcp__github__search_code({
  query: "unit_cost repo:asosteam/asos-data-ade-dbt",
  perPage: 10
})

// Example: Search Power BI measures
mcp__github__search_code({
  query: "UnitCost repo:asosteam/asos-data-ade-powerbi",
  perPage: 10
})

// Example: Search contracts
mcp__github__search_code({
  query: "FactSales repo:asosteam/asos-data-ade-processingzone-contracts",
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
  searchText: "metric_name",
  top: 10
})
```

### msts-explorer MCP (for MicroStrategy data)

This is a local MCP server for MSTR object discovery - no org/project needed:
```javascript
msts_search(query: "metric_name", type: "metric", limit: 5)
msts_metric_get(guid: "<guid>")
msts_trace_metric(guid: "<guid>", depth: 5)
```

---

## Phase 1: Input Resolution

### Step 1.1: Identify Input Type

Determine if the input is:
- **GUID** (32 hex characters): Use directly with MCP tools
- **Name/keyword**: Search first using `msts_search`

### Step 1.2: Retrieve MSTR Object Details

**For Metrics:**
```
msts_metric_get(guid="<resolved_guid>")
```

Extract:
| Field | Purpose |
|-------|---------|
| `name` | Metric name for ADE mapping |
| `formula` | Business logic to replicate |
| `referenced_facts` | Source facts (map to ADE tables) |
| `referenced_attributes` | Dimensions used |
| `location` | MicroStrategy folder path |
| `owner` | Business ownership |

**For Attributes:**
```
msts_attribute_get(guid="<resolved_guid>")
```

Extract:
| Field | Purpose |
|-------|---------|
| `name` | Attribute name for ADE mapping |
| `forms` | ID/DESC forms with source tables |
| `children` / `parents` | Hierarchy relationships |
| `location` | MicroStrategy folder path |

### Step 1.3: Trace Dependencies (Metrics Only)

For comprehensive lineage:
```
msts_trace_metric(guid="<metric_guid>", depth=5)
```

This reveals:
- Underlying facts and their expressions
- Source database tables
- Compound metric dependencies

**Output Phase 1:**
```markdown
## MSTR Object Summary

| Property | Value |
|----------|-------|
| Type | Metric / Attribute |
| Name | [name] |
| GUID | [guid] |
| Location | [path] |
| Owner | [owner] |

### Definition
[formula or form expressions]

### Dependencies
- Facts: [list with GUIDs]
- Attributes: [list with GUIDs]
- Tables: [list]
```

---

## Phase 2: ADE Schema Discovery

### Rate Limit Strategy

**CRITICAL:** Follow these rules to avoid GitHub API rate limits:

1. **Limit directory listings** to 10-15 files max
2. **Use specific paths** — never search from repo root
3. **Batch related searches** — one domain at a time
4. **Cache findings** — don't re-search same paths

### Step 2.1: Identify Target Domain

Map MSTR location to ADE domain:

| MSTR Folder Contains | ADE Domain | dbt Path |
|---------------------|------------|----------|
| Sales, Revenue, Orders | `sales` | `bundles/core_data/models/sales/` |
| Stock, Inventory, Warehouse | `stock` | `bundles/core_data/models/stock/` |
| Finance, Cost, Margin | `finance` | `bundles/core_data/models/finance/` |
| Customer, CRM | `customer` | `bundles/core_data/models/customer/` |
| Product, SKU, Brand | `product` | `bundles/core_data/models/product/` |
| Supply Chain, Logistics | `supply_chain` | `bundles/core_data/models/supply_chain/` |

### Step 2.2: Search ADE Repositories

**Priority order (stop when found):**

#### 2.2.1: dbt Models (Primary)
Repository: `asos-data-ade-dbt` (**GitHub** - use `mcp__github__search_code`)

Search pattern (limit 10 results):
```javascript
mcp__github__search_code({
  query: "[metric_name_keywords] repo:asosteam/asos-data-ade-dbt",
  perPage: 10
})
```

Focus paths:
- `bundles/core_data/models/{domain}/serve/` — Final semantic models
- `bundles/core_data/models/{domain}/curated/` — Business-ready aggregations
- `bundles/core_data/models/{domain}/enriched/` — Transformed data

#### 2.2.2: Power BI Semantic Models (Secondary)
Repository: `asos-data-ade-powerbi` (**GitHub** - use `mcp__github__search_code`)

Search pattern:
```javascript
mcp__github__search_code({
  query: "[metric_name_keywords] repo:asosteam/asos-data-ade-powerbi",
  perPage: 10
})
```

Focus paths:
- `powerbi/scripts/*.csx` — Tabular Editor scripts
- `powerbi/datadictionary/` — Measure definitions

#### 2.2.3: Contracts (Source Schema)
Repository: `asos-data-ade-processingzone-contracts` (**GitHub** - use `mcp__github__search_code`)

Search pattern:
```javascript
mcp__github__search_code({
  query: "[source_table_name] repo:asosteam/asos-data-ade-processingzone-contracts",
  perPage: 10
})
```

Focus paths:
- `config/Contracts/` — Schema definitions
- `config/Transforms/` — Transformation logic

**Output Phase 2:**
```markdown
## ADE Schema Discovery

### Search Strategy
- Domain: [identified_domain]
- Keywords: [search_terms_used]
- API calls made: [count]

### Matches Found

#### dbt Models
| Model | Path | Layer | Relevance |
|-------|------|-------|-----------|
| [name] | [path] | serve/curated/enriched | High/Medium/Low |

#### Power BI Measures
| Measure | Path | Relevance |
|---------|------|-----------|
| [name] | [path] | High/Medium/Low |

#### Source Contracts
| Contract | Path | Relevance |
|----------|------|-----------|
| [name] | [path] | High/Medium/Low |
```

---

## Phase 3: Gap Analysis

### Step 3.1: Semantic Comparison

Compare MSTR definition against ADE findings:

| Aspect | MSTR Definition | ADE Equivalent | Gap? |
|--------|-----------------|----------------|------|
| Metric Name | [name] | [ade_name] | Y/N |
| Business Logic | [formula] | [dbt_sql/DAX] | Y/N |
| Granularity | [attributes] | [dimensions] | Y/N |
| Source Tables | [mstr_tables] | [ade_sources] | Y/N |
| Aggregation | [sum/avg/etc] | [ade_agg] | Y/N |

### Step 3.2: Classify Gaps

| Gap Type | Description | Example |
|----------|-------------|---------|
| **MISSING** | No equivalent in ADE | Metric doesn't exist |
| **PARTIAL** | Exists but incomplete | Missing dimensions |
| **DIFFERENT** | Logic differs | Different calculation |
| **NAMING** | Same logic, different name | Terminology mismatch |
| **NONE** | Exact match exists | Ready to migrate |

### Step 3.3: Generate Mapping Suggestions

For each gap, suggest resolution:

| Gap | Suggested Action | Effort |
|-----|------------------|--------|
| MISSING | Create new dbt model in `{domain}/serve/` | Medium |
| PARTIAL | Extend existing model `{model_name}` | Low |
| DIFFERENT | Review business logic with stakeholders | High |
| NAMING | Add alias/synonym in data dictionary | Low |

**Output Phase 3:**
```markdown
## Gap Analysis Results

### Summary
| Category | Count |
|----------|-------|
| Exact Match | [n] |
| Partial Match | [n] |
| Missing | [n] |
| Different Logic | [n] |

### Detailed Gaps

#### [Gap 1: MISSING - Metric Name]
- **MSTR Definition:** [formula]
- **ADE Status:** Not found
- **Recommendation:** Create `{domain}_serve_{metric_name}.sql` in `bundles/core_data/models/{domain}/serve/`
- **Dependencies to Create First:** [list]
- **Estimated Effort:** [Low/Medium/High]

#### [Gap 2: PARTIAL - Attribute Name]
...
```

---

## Phase 4: Mapping Document Generation

### Output Format

Generate a mapping document suitable for work item creation:

```markdown
# MSTR to ADE Mapping: [Object Name]

## Source (MicroStrategy)
- **Name:** [name]
- **GUID:** [guid]
- **Type:** Metric / Attribute
- **Formula/Definition:** [definition]
- **Location:** [path]
- **Owner:** [owner]

## Target (ADE)

### Existing Assets
| Asset | Type | Path | Match Level |
|-------|------|------|-------------|
| [name] | dbt model | [path] | Exact/Partial |

### Gaps Identified
| Gap | Type | Resolution | Effort |
|-----|------|------------|--------|
| [description] | MISSING/PARTIAL | [action] | Low/Med/High |

## Implementation Plan

### Prerequisites
1. [Dependency 1]
2. [Dependency 2]

### Steps
1. **[Repository]:** [Action] at `[path]`
2. **[Repository]:** [Action] at `[path]`

### Testing
- [ ] Unit test in `tests/{domain}/`
- [ ] dbt build --select [model]
- [ ] Validate against MSTR output

## Stakeholders
- **Business Owner:** [from MSTR owner]
- **Technical Owner:** Team Solero (ADE)

## References
- MSTR Report using this: [if applicable]
- Related Work Items: [if known]
```

---

## Phase 5: Batch Analysis Mode

When analyzing multiple objects, optimize API usage:

### Step 5.1: Group by Domain
```
Objects grouped by ADE domain:
- sales: [list]
- stock: [list]
- finance: [list]
```

### Step 5.2: Batch Search per Domain
One search per domain covering all related keywords:
```
searchText: "keyword1 OR keyword2 OR keyword3"
```

### Step 5.3: Cross-Reference Results
Map all found ADE assets to MSTR objects in a single pass.

### Output Batch Mode:
```markdown
## Batch Gap Analysis Summary

| MSTR Object | Type | ADE Status | Gap Type | Action |
|-------------|------|------------|----------|--------|
| [name 1] | Metric | Found | NAMING | Map alias |
| [name 2] | Attribute | Partial | PARTIAL | Extend model |
| [name 3] | Metric | Missing | MISSING | Create new |

### Total API Calls: [n]
### Rate Limit Used: [n]/5000
```

---

## Guardrails

### DO
- Read pre-requisite docs before starting
- Use `msts_*` MCP tools for MSTR data
- Limit ADO searches to 10 results max
- Document all search paths used
- Flag ambiguities for engineer review
- Generate actionable mapping documents

### DO NOT
- Search entire repositories (rate limits!)
- Assume business logic without evidence
- Skip dependency tracing for metrics
- Make more than 15 ADO API calls per session
- Proceed without confirming MSTR object details

---

## Rate Limit Checkpoint

Before each ADO API call, verify:
- [ ] Search is scoped to specific repository
- [ ] Using `top: 10` or less
- [ ] Path is narrowed to domain folder
- [ ] Not duplicating previous search

If approaching 15 calls, STOP and present findings so far.

---

## Acceptance Criteria

Analysis is complete when:
- [ ] MSTR object fully retrieved with dependencies
- [ ] ADE repositories searched (rate limit compliant)
- [ ] Gaps classified and quantified
- [ ] Mapping suggestions generated
- [ ] Implementation steps outlined
- [ ] All sources documented
- [ ] Engineer reviewed findings

---

## Example Invocation

**Single metric:**
```
/mstr-ade-gap-analysis "Net Sales Value"
```

**By GUID:**
```
/mstr-ade-gap-analysis "05EB47444E0D0C30724D9E98FA1581D6"
```

**Batch (comma-separated):**
```
/mstr-ade-gap-analysis "Gross Margin, Net Sales, Units Sold"
```
