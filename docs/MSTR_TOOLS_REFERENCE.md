# MicroStrategy MCP Tools - Reference

This document provides a complete reference for the MicroStrategy MCP tools, including Cypher queries, input parameters, and LLM-facing descriptions.

For the specification document, see [103-mcp-tools-reference.md](./flowdash-queries/103-mcp-tools-reference.md).

## Table of Contents

1. [Tools Overview](#tools-overview)
2. [Design Principles](#design-principles)
3. [Tool 1: search-metrics](#tool-1-search-metrics)
4. [Tool 2: search-attributes](#tool-2-search-attributes)
5. [Tool 3: trace-metric](#tool-3-trace-metric)
6. [Tool 4: trace-attribute](#tool-4-trace-attribute)
7. [Key Differences Between Tools](#key-differences-between-tools)
8. [Common Workflows](#common-workflows)
9. [Graph Traversal Rules](#graph-traversal-rules)
10. [Database Schema](#database-schema)
11. [Migration from Previous Tools](#migration-from-previous-tools)

---

## Tools Overview

The system has **4 MicroStrategy tools** organised into two categories:

### Search Tools (Find Objects)
| Tool | Description | Input |
|------|-------------|-------|
| `search-metrics` | Find Metrics by GUID or name | GUID (full/partial) or name |
| `search-attributes` | Find Attributes by GUID or name | GUID (full/partial) or name |

### Trace Tools (Explore Lineage)
| Tool | Description | Input |
|------|-------------|-------|
| `trace-metric` | Trace Metric lineage by direction | Full GUID + direction |
| `trace-attribute` | Trace Attribute lineage by direction | Full GUID + direction |

---

## Design Principles

### 1. Return All Matching Objects (Including Unprioritized/Unmapped)

The tools return **ALL** Metrics/Attributes matching the query, including those without parity mapping or priority assignment. This design decision ensures:

- Complete visibility into the MicroStrategy object inventory
- No objects are hidden due to missing parity matrix entries
- Users can discover objects that need to be added to parity tracking

**Status Values:**
| Status | Meaning |
|--------|---------|
| `Complete` | Migration/implementation is complete |
| `Planned` | Migration is planned |
| `Not Planned` | Object is not planned for migration |
| `Drop` | Object will be deprecated |
| `No Status` | Object is not in the parity matrix (unmapped) |

### 2. Updated Properties Take Precedence

Properties with the `updated_` prefix override their base counterparts. This allows manual corrections from ADO backlog sync to take precedence over computed values.

| Property | Updated Version | Behavior |
|----------|-----------------|----------|
| `parity_status` | `updated_parity_status` | `COALESCE(updated_parity_status, parity_status, 'No Status')` |
| `parity_notes` | `updated_parity_notes` | `COALESCE(updated_parity_notes, parity_notes)` |
| `db_raw` | `updated_db_raw` | `COALESCE(updated_db_raw, db_raw)` |
| `db_serve` | `updated_db_serve` | `COALESCE(updated_db_serve, db_serve)` |
| `edw_table` | `updated_edw_table` | `COALESCE(updated_edw_table, edw_table)` |
| `ade_db_table` | `updated_ade_db_table` | `COALESCE(updated_ade_db_table, ade_db_table)` |
| `ado_link` | `updated_ado_link` | `COALESCE(updated_ado_link, ado_link)` |

**Example:** If an object has `parity_status = "Planned"` but `updated_parity_status = "Complete"`, the effective status returned is `"Complete"`.

### 3. Directional Lineage Tracing

Trace tools require a `direction` parameter to avoid returning too much data in a single call:

| Direction | Description | Returns | Graph Traversal |
|-----------|-------------|---------|-----------------|
| `downstream` | Who uses this M/A? | reports[] | M/A → Reports (reverse dependency) |
| `upstream` | Where does data come from? | tables[], dependencies[] | M/A → Facts → Tables (forward dependency) |

**Why Split?**
- High-connectivity objects (e.g., "Retail Sales Value" with 6,000+ reports) would return excessive data
- LLM queries typically need one direction at a time
- Reduces response size and improves clarity

```
UPSTREAM (sources)                                DOWNSTREAM (consumers)
    Tables ──→ Facts ──→ Metrics/Attributes ──→ Reports
              ←─────── trace(direction="upstream") ───────
              ─────── trace(direction="downstream") ─────→
```

### 4. Full Property Set Returned

All tools return the complete set of 21 properties per object, enabling comprehensive analysis without additional queries:

| Category | Properties |
|----------|------------|
| **Identity** | `type`, `guid`, `name` |
| **Parity** | `status`, `priority`, `notes` |
| **Definition** | `formula` (Metrics) or `forms_json` (Attributes) |
| **Databricks Mapping** | `raw`, `serve`, `semantic` |
| **EDW Mapping** | `edwTable`, `edwColumn` |
| **ADE Mapping** | `adeTable`, `adeColumn` |
| **Power BI Mapping** | `semanticName`, `semanticModel` |
| **Essential Flags** | `dbEssential`, `pbEssential` |
| **Lineage Counts** | `reportCount`, `tableCount` |
| **Integration** | `ado_link` |

---

## Tool 1: search-metrics

**Source:** `internal/tools/mstr/search_metrics.go`

### Input Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `query` | string | Yes | — | GUID (full or partial 8+ chars) or name search term |
| `status` | []string | No | null | Filter by parity status: `["Complete", "Planned", "Not Planned"]` |
| `offset` | int | No | 0 | Skip first N results for pagination |

### LLM-Facing Description

```
Find Metrics by GUID or name. Accepts full GUIDs, partial GUIDs (8+ chars), or name search terms.

CORRECT USAGE:
- search-metrics(query="2F00974D44E1D0D24CA344ABD872806A") - full GUID
- search-metrics(query="2F00974D") - partial GUID (8+ chars)
- search-metrics(query="Retail Sales") - name search
- search-metrics(query="sales", status=["Complete"]) - with filter

INCORRECT USAGE:
- DON'T use partial GUID < 8 chars (too ambiguous)
- DON'T use for lineage - use trace-metric instead
- DON'T search for Attributes here - use search-attributes

PAGINATION: Returns 100 results. If moreResults=true, call again with offset+100.
```

### Cypher Query

```cypher
// Search for Metrics by GUID or name
// $query: GUID (full/partial) or name search term
// $status: optional parity status filter
// $offset: pagination offset (0, 100, 200, ...)

// Determine if query looks like a GUID (hex chars, 8+ length)
WITH $query as query,
     $query =~ '^[A-Fa-f0-9]{8,}$' as isGuidLike

MATCH (n:Metric)
WHERE n.guid IS NOT NULL
  AND (
    // GUID match: exact or partial (starts with)
    (isGuidLike AND (n.guid = query OR n.guid STARTS WITH toUpper(query)))
    OR
    // Name match: case-insensitive contains
    (NOT isGuidLike AND toLower(n.name) CONTAINS toLower(query))
  )
  AND ($status IS NULL OR COALESCE(n.updated_parity_status, n.parity_status) IN $status)

// Compute effective status (updated takes precedence)
WITH n,
     COALESCE(n.updated_parity_status, n.parity_status, 'No Status') as effectiveStatus

ORDER BY n.name ASC
SKIP $offset
LIMIT 101  // Fetch 101 to determine if more results exist

// Collect results
WITH collect({
  type: 'Metric',
  guid: n.guid,
  name: n.name,
  status: effectiveStatus,
  formula: n.formula,
  ado_link: n.ado_link
}) as fetched

// Return first 100; moreResults=true if 101st exists
RETURN 
  fetched[0..100] as results,
  size(fetched) > 100 as moreResults
```

### Response Structure

```json
{
  "results": [
    {
      "type": "Metric",
      "guid": "E50193A144D3BBAA4B8C938EBFBDA01A",
      "name": "Retail Sales Value",
      "status": "Complete",
      "priority": 0,
      "formula": "( Sum ( Billed Sales Value Inc Tax ) - Sum ( Billed Sales Checkout Tax Amount ) )",
      "notes": "+ afs, pf in ade",
      "raw": "Y",
      "serve": "Y",
      "semantic": "Y",
      "edwTable": "[presentation].[vwFactSales]",
      "edwColumn": "BilledSalesAmountIncTax-BilledSalesCheckoutTaxAmount",
      "adeTable": "sales.serve.fact_billed_sale_v1",
      "adeColumn": "retail_value",
      "semanticName": "Retail Sales Value",
      "semanticModel": "Trade",
      "dbEssential": "Y",
      "pbEssential": "Y",
      "reportCount": 888,
      "tableCount": 1,
      "ado_link": "https://dev.azure.com/org/project/_workitems/edit/12345"
    }
  ],
  "moreResults": true
}
```

---

## Tool 2: search-attributes

**Source:** `internal/tools/mstr/search_attributes.go`

### Input Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `query` | string | Yes | — | GUID (full or partial 8+ chars) or name search term |
| `status` | []string | No | null | Filter by parity status: `["Complete", "Planned", "Not Planned"]` |
| `offset` | int | No | 0 | Skip first N results for pagination |

### LLM-Facing Description

```
Find Attributes by GUID or name. Accepts full GUIDs, partial GUIDs (8+ chars), or name search terms.

CORRECT USAGE:
- search-attributes(query="BC105EDE477D7CEF3296FFA6E4D26797") - full GUID
- search-attributes(query="BC105EDE") - partial GUID (8+ chars)
- search-attributes(query="Product Category") - name search
- search-attributes(query="product", status=["Complete"]) - with filter

INCORRECT USAGE:
- DON'T use partial GUID < 8 chars (too ambiguous)
- DON'T use for lineage - use trace-attribute instead
- DON'T search for Metrics here - use search-metrics

PAGINATION: Returns 100 results. If moreResults=true, call again with offset+100.
```

### Cypher Query

```cypher
// Search for Attributes by GUID or name
// $query: GUID (full/partial) or name search term
// $status: optional parity status filter
// $offset: pagination offset (0, 100, 200, ...)

// Determine if query looks like a GUID (hex chars, 8+ length)
WITH $query as query,
     $query =~ '^[A-Fa-f0-9]{8,}$' as isGuidLike

MATCH (n:Attribute)
WHERE n.guid IS NOT NULL
  AND (
    // GUID match: exact or partial (starts with)
    (isGuidLike AND (n.guid = query OR n.guid STARTS WITH toUpper(query)))
    OR
    // Name match: case-insensitive contains
    (NOT isGuidLike AND toLower(n.name) CONTAINS toLower(query))
  )
  AND ($status IS NULL OR COALESCE(n.updated_parity_status, n.parity_status) IN $status)

// Compute effective status (updated takes precedence)
WITH n,
     COALESCE(n.updated_parity_status, n.parity_status, 'No Status') as effectiveStatus

ORDER BY n.name ASC
SKIP $offset
LIMIT 101  // Fetch 101 to determine if more results exist

// Collect results
WITH collect({
  type: 'Attribute',
  guid: n.guid,
  name: n.name,
  status: effectiveStatus,
  forms_json: n.forms_json,
  ado_link: n.ado_link
}) as fetched

// Return first 100; moreResults=true if 101st exists
RETURN 
  fetched[0..100] as results,
  size(fetched) > 100 as moreResults
```

### Response Structure

```json
{
  "results": [
    {
      "type": "Attribute",
      "guid": "BC105EDE477D7CEF3296FFA6E4D26797",
      "name": "Product Category",
      "status": "Complete",
      "priority": 0,
      "forms_json": "{\"ID\": \"product_category_id\", \"DESC\": \"product_category_desc\"}",
      "notes": null,
      "raw": "Y",
      "serve": "Y",
      "semantic": "Y",
      "edwTable": "presentation.vwDimProduct",
      "edwColumn": "ProductCategorySKey",
      "adeTable": "product.serve.dim_product_v1",
      "adeColumn": "category_id",
      "semanticName": "Product Category",
      "semanticModel": "Trade",
      "dbEssential": "Y",
      "pbEssential": "Y",
      "reportCount": 450,
      "tableCount": 2,
      "ado_link": "https://dev.azure.com/org/project/_workitems/edit/23456"
    }
  ],
  "moreResults": false
}
```

---

## Tool 3: trace-metric

**Source:** `internal/tools/mstr/trace_metric.go`

### Input Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `guid` | string | Yes | Full GUID of the Metric to trace (32-char hex) |
| `direction` | string | Yes | `"downstream"` (toward reports) or `"upstream"` (toward tables) |

### LLM-Facing Description

```
Trace lineage of a Metric in a specific direction.

DIRECTION:
- 'downstream': Find reports that USE this metric (M/A → Reports)
- 'upstream': Find source tables and dependencies (M/A → Tables)

WHY SPLIT? High-connectivity metrics (e.g., 'Retail Sales Value' with 6000+ reports)
would return too much data in a single call. Choose the direction relevant to your query.

CORRECT USAGE:
- First search: search-metrics(query="Retail Sales")
- Then trace downstream: trace-metric(guid="2F00974D...", direction="downstream")
- Or trace upstream: trace-metric(guid="2F00974D...", direction="upstream")

RETURNS:
- downstream: metric details + reports[] (max 50)
- upstream: metric details + tables[] (max 50) + dependencies[] (max 50)
```

### Cypher Query (downstream)

```cypher
// Trace DOWNSTREAM lineage for a Metric (toward reports)
// $guid: Full GUID of the Metric

MATCH (n:Metric {guid: $guid})

WITH n,
     COALESCE(n.updated_parity_status, n.parity_status, 'No Status') as effectiveStatus

// Get reports using this metric (from pre-computed lineage)
OPTIONAL MATCH (r:MSTRObject)
WHERE r.guid IN COALESCE(n.lineage_used_by_reports, [])
WITH n, effectiveStatus, collect(DISTINCT {
  name: r.name,
  guid: r.guid,
  type: r.type,
  priority: r.priority_level,
  area: r.usage_area
})[0..50] as reports

RETURN {
  metric: { ... 21 properties ... },
  direction: 'downstream',
  reports: reports
} as result
```

### Cypher Query (upstream)

```cypher
// Trace UPSTREAM lineage for a Metric (toward source tables)
// $guid: Full GUID of the Metric

MATCH (n:Metric {guid: $guid})

WITH n,
     COALESCE(n.updated_parity_status, n.parity_status, 'No Status') as effectiveStatus

// Get source tables (from pre-computed lineage)
OPTIONAL MATCH (t:MSTRObject)
WHERE t.guid IN COALESCE(n.lineage_source_tables, [])
WITH n, effectiveStatus, collect(DISTINCT {
  name: t.name,
  guid: t.guid,
  type: t.type,
  physicalTable: t.physical_table_name,
  database: t.database_instance
})[0..50] as tables

// Get direct dependencies (1-hop toward sources)
OPTIONAL MATCH (n)-[:DEPENDS_ON]->(dep:MSTRObject)
WITH n, effectiveStatus, tables, collect(DISTINCT {
  name: dep.name,
  guid: dep.guid,
  type: dep.type,
  formula: dep.formula
})[0..50] as dependencies

RETURN {
  metric: { ... 21 properties ... },
  direction: 'upstream',
  tables: tables,
  dependencies: dependencies
} as result
```

### Response Structure (downstream)

```json
{
  "result": {
    "metric": {
      "type": "Metric",
      "guid": "E50193A144D3BBAA4B8C938EBFBDA01A",
      "name": "Retail Sales Value",
      "status": "Complete",
      "priority": 0,
      "formula": "...",
      "notes": "...",
      "raw": "Y", "serve": "Y", "semantic": "Y",
      "edwTable": "...", "edwColumn": "...",
      "adeTable": "...", "adeColumn": "...",
      "semanticName": "...", "semanticModel": "...",
      "dbEssential": "Y", "pbEssential": "Y",
      "reportCount": 888, "tableCount": 1,
      "ado_link": "..."
    },
    "direction": "downstream",
    "reports": [
      { "name": "Daily Sales Dashboard", "guid": "ABC123...", "type": "Report", "priority": 1, "area": "Customer Commercial" }
    ]
  }
}
```

### Response Structure (upstream)

```json
{
  "result": {
    "metric": { ... same 21 properties ... },
    "direction": "upstream",
    "tables": [
      { "name": "FACT_SALES", "guid": "GHI789...", "type": "LogicalTable", "physicalTable": "dbo.FACT_SALES_DAILY", "database": "EDW_PROD" }
    ],
    "dependencies": [
      { "name": "Sales Fact", "guid": "FACT123...", "type": "Fact", "formula": null }
    ]
  }
}
```

---

## Tool 4: trace-attribute

**Source:** `internal/tools/mstr/trace_attribute.go`

### Input Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `guid` | string | Yes | Full GUID of the Attribute to trace (32-char hex) |
| `direction` | string | Yes | `"downstream"` (toward reports) or `"upstream"` (toward tables) |

### LLM-Facing Description

```
Trace lineage of an Attribute in a specific direction.

DIRECTION:
- 'downstream': Find reports that USE this attribute (M/A → Reports)
- 'upstream': Find source tables and dependencies (M/A → Tables)

WHY SPLIT? High-connectivity attributes (e.g., 'Product' with thousands of reports)
would return too much data in a single call. Choose the direction relevant to your query.

CORRECT USAGE:
- First search: search-attributes(query="Product Category")
- Then trace downstream: trace-attribute(guid="BC105EDE...", direction="downstream")
- Or trace upstream: trace-attribute(guid="BC105EDE...", direction="upstream")

RETURNS:
- downstream: attribute details + reports[] (max 50)
- upstream: attribute details + tables[] (max 50) + dependencies[] (max 50)
```

### Cypher Query (downstream)

```cypher
// Trace DOWNSTREAM lineage for an Attribute (toward reports)
// $guid: Full GUID of the Attribute

MATCH (n:Attribute {guid: $guid})

WITH n,
     COALESCE(n.updated_parity_status, n.parity_status, 'No Status') as effectiveStatus

// Get reports using this attribute (from pre-computed lineage)
OPTIONAL MATCH (r:MSTRObject)
WHERE r.guid IN COALESCE(n.lineage_used_by_reports, [])
WITH n, effectiveStatus, collect(DISTINCT {
  name: r.name,
  guid: r.guid,
  type: r.type,
  priority: r.priority_level,
  area: r.usage_area
})[0..50] as reports

RETURN {
  attribute: { ... 21 properties ... },
  direction: 'downstream',
  reports: reports
} as result
```

### Cypher Query (upstream)

```cypher
// Trace UPSTREAM lineage for an Attribute (toward source tables)
// $guid: Full GUID of the Attribute

MATCH (n:Attribute {guid: $guid})

WITH n,
     COALESCE(n.updated_parity_status, n.parity_status, 'No Status') as effectiveStatus

// Get source tables (from pre-computed lineage)
OPTIONAL MATCH (t:MSTRObject)
WHERE t.guid IN COALESCE(n.lineage_source_tables, [])
WITH n, effectiveStatus, collect(DISTINCT {
  name: t.name,
  guid: t.guid,
  type: t.type,
  physicalTable: t.physical_table_name,
  database: t.database_instance
})[0..50] as tables

// Get direct dependencies (1-hop toward sources)
OPTIONAL MATCH (n)-[:DEPENDS_ON]->(dep:MSTRObject)
WITH n, effectiveStatus, tables, collect(DISTINCT {
  name: dep.name,
  guid: dep.guid,
  type: dep.type
})[0..50] as dependencies

RETURN {
  attribute: { ... 21 properties ... },
  direction: 'upstream',
  tables: tables,
  dependencies: dependencies
} as result
```

### Response Structure (downstream)

```json
{
  "result": {
    "attribute": {
      "type": "Attribute",
      "guid": "BC105EDE477D7CEF3296FFA6E4D26797",
      "name": "Product Category",
      "status": "Complete",
      "priority": 0,
      "forms_json": "...",
      "notes": null,
      "raw": "Y", "serve": "Y", "semantic": "Y",
      "edwTable": "...", "edwColumn": "...",
      "adeTable": "...", "adeColumn": "...",
      "semanticName": "...", "semanticModel": "...",
      "dbEssential": "Y", "pbEssential": "Y",
      "reportCount": 450, "tableCount": 2,
      "ado_link": "..."
    },
    "direction": "downstream",
    "reports": [
      { "name": "Category Performance", "guid": "RPT123...", "type": "Report", "priority": 1, "area": "Fashion Product" }
    ]
  }
}
```

### Response Structure (upstream)

```json
{
  "result": {
    "attribute": { ... same 21 properties ... },
    "direction": "upstream",
    "tables": [
      { "name": "LU_PRODUCT", "guid": "MNO345...", "type": "LogicalTable", "physicalTable": "dbo.DIM_PRODUCT", "database": "EDW_PROD" }
    ],
    "dependencies": [
      { "name": "product_category_id", "guid": "DEP123...", "type": "Fact" }
    ]
  }
}
```

---

## Key Differences Between Tools

| Aspect | search-metrics | search-attributes | trace-metric | trace-attribute |
|--------|----------------|-------------------|--------------|-----------------|
| **Node Label** | `:Metric` | `:Attribute` | `:Metric` | `:Attribute` |
| **Type-specific field** | `formula` | `forms_json` | `formula` | `forms_json` |
| **Properties returned** | 21 fields | 21 fields | 21 fields + lineage | 21 fields + lineage |
| **Dependencies include** | — | — | `formula` | (no formula) |
| **Reports include** | — | — | `priority`, `area` | `priority`, `area` |
| **Query mode** | GUID or name | GUID or name | Full GUID only | Full GUID only |
| **Returns** | `results[]`, `moreResults` | `results[]`, `moreResults` | Single object with lineage | Single object with lineage |

---

## Common Workflows

### Find and Trace a Metric

```
Step 1: Search for the metric
─────────────────────────────
search-metrics(query="Retail Sales")
→ Returns list of matching metrics with GUIDs

Step 2: Trace the specific metric
─────────────────────────────────
trace-metric(guid="2F00974D44E1D0D24CA344ABD872806A")
→ Returns reports, tables, and dependencies
```

### Find and Trace an Attribute

```
Step 1: Search for the attribute
────────────────────────────────
search-attributes(query="Product")
→ Returns list of matching attributes with GUIDs

Step 2: Trace the specific attribute
────────────────────────────────────
trace-attribute(guid="BC105EDE477D7CEF3296FFA6E4D26797")
→ Returns reports, tables, and dependencies
```

### Browse by Status

```
# Find all "Not Planned" metrics
search-metrics(query="*", status=["Not Planned"])

# Paginate through results
search-metrics(query="*", status=["Not Planned"], offset=100)
```

---

## Graph Traversal Rules

**Reference:** [90-symmetric-bfs-traversal.md](./flowdash-queries/90-symmetric-bfs-traversal.md)

The trace tools use pre-computed lineage arrays that were built following these traversal rules.

### Core Concept: Capture vs Traverse

| Behavior | Types | What Happens |
|----------|-------|--------------|
| **TRAVERSE** | `Prompt`, `Filter` | BFS follows their edges to find more objects |
| **CAPTURE** | `Metric`, `Attribute`, `DerivedMetric`, `Transformation` | Added to results, but BFS stops here |

### Traversal Directions

| Direction | Purpose | Intermediate Types | Target Types |
|-----------|---------|-------------------|--------------|
| **Inbound** (Reports → M/A) | Find reports using an object | `[Prompt, Filter]` | `[Report, GridReport, Document]` |
| **Outbound** (M/A → Tables) | Find source tables | `[Fact, Metric, Attribute]` | `[LogicalTable, Table]` |

### Pre-computed Lineage

The trace tools use pre-computed arrays for performance:
- `lineage_used_by_reports` — Array of report GUIDs that use this M/A
- `lineage_source_tables` — Array of table GUIDs this M/A depends on

**Why Pre-computed:** Runtime BFS traversal was timing out (>30 seconds) for high-connectivity objects (e.g., "Retail Sales Value" with 6,000+ reports). Pre-computed arrays reduce query time to <100ms.

---

## Database Schema

### Node Labels

| Label | Description |
|-------|-------------|
| `Metric` | Metrics |
| `Attribute` | Attributes |
| `MSTRObject` | Generic label for all MicroStrategy objects |
| `Fact` | Facts |
| `LogicalTable` | Logical tables |
| `Report` | Reports |
| `GridReport` | Grid Reports |
| `Document` | Documents |

### Key Properties

#### On Metric

| Property | Description |
|----------|-------------|
| `guid` | Unique identifier |
| `name` | Object name |
| `formula` | Metric formula (text) |
| `parity_status` | Original parity status |
| `updated_parity_status` | Updated parity status (takes precedence) |
| `ado_link` | ADO work item URL |
| `lineage_used_by_reports` | Pre-computed report GUIDs array |
| `lineage_source_tables` | Pre-computed table GUIDs array |

#### On Attribute

| Property | Description |
|----------|-------------|
| `guid` | Unique identifier |
| `name` | Object name |
| `forms_json` | Attribute forms in JSON |
| `parity_status` | Original parity status |
| `updated_parity_status` | Updated parity status (takes precedence) |
| `ado_link` | ADO work item URL |
| `lineage_used_by_reports` | Pre-computed report GUIDs array |
| `lineage_source_tables` | Pre-computed table GUIDs array |

#### On LogicalTable/Table

| Property | Description |
|----------|-------------|
| `guid` | Unique identifier |
| `name` | Logical table name |
| `physical_table_name` | Physical table name in DB |
| `database_instance` | Database instance |

---

## Migration from Previous Tools

| Old Tool | New Tool |
|----------|----------|
| `get-metric-by-guid` | `search-metrics` (use GUID as query) |
| `get-attribute-by-guid` | `search-attributes` (use GUID as query) |
| `get-reports-using-metric` | `trace-metric` (returns reports in response) |
| `get-reports-using-attribute` | `trace-attribute` (returns reports in response) |
| `get-metric-source-tables` | `trace-metric` (returns tables in response) |
| `get-attribute-source-tables` | `trace-attribute` (returns tables in response) |
| `get-metric-dependencies` | `trace-metric` (returns dependencies in response) |
| `get-attribute-dependencies` | `trace-attribute` (returns dependencies in response) |

**Removed tools:** `get-metrics-stats`, `get-attributes-stats`, `get-object-stats`, `get-metric-dependents`, `get-attribute-dependents`

### Removed Fields

The following fields have been removed from responses:
- `parity_group` / `Group`
- `parity_subgroup` / `SubGroup`
- `lineage_used_by_reports_count` / `reportCount`
- `lineage_source_tables_count` / `tableCount`
- `team` (NeoDash-specific)
- `priority` / `inherited_priority_level` (NeoDash-specific)

---

## Update History

| Date | Version | Change |
|------|---------|--------|
| 2026-02-03 | 2.0.0 | Major refactor: Consolidated 15 tools into 4 unified tools; Added full Cypher queries and implementation details |
| 2026-01-30 | 1.0.0 | Initial document with 12 MSTR tools |
