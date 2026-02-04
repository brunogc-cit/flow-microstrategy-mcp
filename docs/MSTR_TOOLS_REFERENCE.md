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
12. [Stress Testing](#stress-testing)

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
// $status: optional parity status filter (applies to effective status)
// $offset: pagination offset (0, 100, 200, ...)
//
// Design Decision: Returns ALL Metrics matching the query, including those without
// parity mapping. Objects not in the parity matrix get status "No Status".
// The updated_parity_status property (from ADO backlog sync) takes precedence
// over the computed parity_status.

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

// Compute effective values (updated_ properties take precedence)
WITH n,
     COALESCE(n.updated_parity_status, n.parity_status, 'No Status') as effectiveStatus

ORDER BY n.name ASC
SKIP $offset
LIMIT 101  // Fetch 101 to determine if more results exist

// Collect results with all properties (updated_ values take precedence)
WITH collect({
  type: 'Metric',
  guid: n.guid,
  name: n.name,
  status: effectiveStatus,
  priority: n.inherited_priority_level,
  formula: n.formula,
  notes: COALESCE(n.updated_parity_notes, n.parity_notes),
  raw: COALESCE(n.updated_db_raw, n.db_raw),
  serve: COALESCE(n.updated_db_serve, n.db_serve),
  semantic: n.pb_semantic,
  edwTable: COALESCE(n.updated_edw_table, n.edw_table),
  edwColumn: n.edw_column,
  adeTable: COALESCE(n.updated_ade_db_table, n.ade_db_table),
  adeColumn: n.ade_db_column,
  semanticName: n.pb_semantic_name,
  semanticModel: n.pb_semantic_model,
  dbEssential: n.db_essential,
  pbEssential: n.pb_essential,
  reportCount: COALESCE(n.lineage_used_by_reports_count, 0),
  tableCount: COALESCE(n.lineage_source_tables_count, 0),
  ado_link: COALESCE(n.updated_ado_link, n.ado_link)
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
// $status: optional parity status filter (applies to effective status)
// $offset: pagination offset (0, 100, 200, ...)
//
// Design Decision: Returns ALL Attributes matching the query, including those without
// parity mapping. Objects not in the parity matrix get status "No Status".
// The updated_parity_status property (from ADO backlog sync) takes precedence
// over the computed parity_status.

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

// Compute effective values (updated_ properties take precedence)
WITH n,
     COALESCE(n.updated_parity_status, n.parity_status, 'No Status') as effectiveStatus

ORDER BY n.name ASC
SKIP $offset
LIMIT 101  // Fetch 101 to determine if more results exist

// Collect results with all properties (updated_ values take precedence)
WITH collect({
  type: 'Attribute',
  guid: n.guid,
  name: n.name,
  status: effectiveStatus,
  priority: n.inherited_priority_level,
  forms_json: n.forms_json,
  notes: COALESCE(n.updated_parity_notes, n.parity_notes),
  raw: COALESCE(n.updated_db_raw, n.db_raw),
  serve: COALESCE(n.updated_db_serve, n.db_serve),
  semantic: n.pb_semantic,
  edwTable: COALESCE(n.updated_edw_table, n.edw_table),
  edwColumn: n.edw_column,
  adeTable: COALESCE(n.updated_ade_db_table, n.ade_db_table),
  adeColumn: n.ade_db_column,
  semanticName: n.pb_semantic_name,
  semanticModel: n.pb_semantic_model,
  dbEssential: n.db_essential,
  pbEssential: n.pb_essential,
  reportCount: COALESCE(n.lineage_used_by_reports_count, 0),
  tableCount: COALESCE(n.lineage_source_tables_count, 0),
  ado_link: COALESCE(n.updated_ado_link, n.ado_link)
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

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `guid` | string | Yes | — | Full GUID of the Metric to trace (32-char hex) |
| `direction` | string | Yes | — | `"downstream"` (toward reports) or `"upstream"` (toward tables) |
| `offset` | int | No | 0 | Skip first N results for pagination |

### LLM-Facing Description

```
Trace lineage of a Metric in a specific direction using live graph traversal.

DIRECTION:
- 'downstream': Find reports that USE this metric (live BFS traversal)
- 'upstream': Find source tables and dependencies (live BFS traversal)

WHY SPLIT? High-connectivity metrics would return too much data in a single call.
Choose the direction relevant to your query.

CORRECT USAGE:
- First search: search-metrics(query="Retail Sales")
- Then trace downstream: trace-metric(guid="2F00974D...", direction="downstream")
- Or trace upstream: trace-metric(guid="2F00974D...", direction="upstream")

PAGINATION: Returns 100 results. If moreResults=true, call again with offset+100.
```

### Cypher Query (downstream)

```cypher
// Trace DOWNSTREAM lineage for a Metric (toward reports)
// $guid: Full GUID of the Metric
// $offset: Pagination offset
//
// LIVE TRAVERSAL: Follows incoming DEPENDS_ON relationships to find consumers.
// Traverses up to 10 hops to find Reports/GridReports/Documents that use this metric.

MATCH (n:Metric {guid: $guid})

// Get effective status (updated takes precedence)
WITH n,
     COALESCE(n.updated_parity_status, n.parity_status, 'No Status') as effectiveStatus

// Find prioritized reports that depend on this metric (live traversal)
// Filter: Only prioritized reports (priority_level IS NOT NULL)
OPTIONAL MATCH (report)-[:DEPENDS_ON*1..10]->(n)
WHERE report.type IN ['Report', 'GridReport', 'Document']
  AND report.priority_level IS NOT NULL

WITH n, effectiveStatus, report
ORDER BY report.name ASC
SKIP $offset
LIMIT 101

// Collect paginated reports
WITH n, effectiveStatus, collect(DISTINCT {
  name: report.name,
  guid: report.guid,
  type: report.type,
  priority: report.priority_level,
  area: report.usage_area
}) as fetched

// Return metric with all properties (updated_ values take precedence)
RETURN {
  metric: {
    type: 'Metric',
    guid: n.guid,
    name: n.name,
    status: effectiveStatus,
    priority: n.inherited_priority_level,
    formula: n.formula,
    notes: COALESCE(n.updated_parity_notes, n.parity_notes),
    raw: COALESCE(n.updated_db_raw, n.db_raw),
    serve: COALESCE(n.updated_db_serve, n.db_serve),
    semantic: n.pb_semantic,
    edwTable: COALESCE(n.updated_edw_table, n.edw_table),
    edwColumn: n.edw_column,
    adeTable: COALESCE(n.updated_ade_db_table, n.ade_db_table),
    adeColumn: n.ade_db_column,
    semanticName: n.pb_semantic_name,
    semanticModel: n.pb_semantic_model,
    dbEssential: n.db_essential,
    pbEssential: n.pb_essential,
    ado_link: COALESCE(n.updated_ado_link, n.ado_link)
  },
  direction: 'downstream',
  reports: fetched[0..100],
  moreResults: size(fetched) > 100
} as result
```

### Cypher Query (upstream)

```cypher
// Trace UPSTREAM lineage for a Metric (toward source tables)
// $guid: Full GUID of the Metric
// $offset: Pagination offset
//
// LIVE TRAVERSAL: Follows outgoing DEPENDS_ON relationships to find data sources.
// Tables are reached via Facts; Dependencies are direct DEPENDS_ON targets.

MATCH (n:Metric {guid: $guid})

// Get effective status (updated takes precedence)
WITH n,
     COALESCE(n.updated_parity_status, n.parity_status, 'No Status') as effectiveStatus

// Find source tables via live traversal (Metric -> ... -> LogicalTable)
OPTIONAL MATCH (n)-[:DEPENDS_ON*1..10]->(t)
WHERE t.type IN ['LogicalTable', 'Table']

WITH n, effectiveStatus, t
ORDER BY t.name ASC
SKIP $offset
LIMIT 101

// Collect paginated tables
WITH n, effectiveStatus, collect(DISTINCT {
  name: t.name,
  guid: t.guid,
  type: t.type,
  physicalTable: t.physical_table_name,
  database: t.database_instance
}) as fetchedTables

// Get direct dependencies (depth 1-2 for immediate dependencies)
OPTIONAL MATCH (n)-[:DEPENDS_ON*1..2]->(dep)
WHERE dep.type IN ['Fact', 'Metric', 'Attribute', 'DerivedMetric', 'Column', 'Transformation']

WITH n, effectiveStatus, fetchedTables, collect(DISTINCT {
  name: dep.name,
  guid: dep.guid,
  type: dep.type,
  formula: dep.formula
})[0..100] as dependencies

// Return metric with all properties (updated_ values take precedence)
RETURN {
  metric: {
    type: 'Metric',
    guid: n.guid,
    name: n.name,
    status: effectiveStatus,
    priority: n.inherited_priority_level,
    formula: n.formula,
    notes: COALESCE(n.updated_parity_notes, n.parity_notes),
    raw: COALESCE(n.updated_db_raw, n.db_raw),
    serve: COALESCE(n.updated_db_serve, n.db_serve),
    semantic: n.pb_semantic,
    edwTable: COALESCE(n.updated_edw_table, n.edw_table),
    edwColumn: n.edw_column,
    adeTable: COALESCE(n.updated_ade_db_table, n.ade_db_table),
    adeColumn: n.ade_db_column,
    semanticName: n.pb_semantic_name,
    semanticModel: n.pb_semantic_model,
    dbEssential: n.db_essential,
    pbEssential: n.pb_essential,
    ado_link: COALESCE(n.updated_ado_link, n.ado_link)
  },
  direction: 'upstream',
  tables: fetchedTables[0..100],
  moreResults: size(fetchedTables) > 100,
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

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `guid` | string | Yes | — | Full GUID of the Attribute to trace (32-char hex) |
| `direction` | string | Yes | — | `"downstream"` (toward reports) or `"upstream"` (toward tables) |
| `offset` | int | No | 0 | Skip first N results for pagination |

### LLM-Facing Description

```
Trace lineage of an Attribute in a specific direction using live graph traversal.

DIRECTION:
- 'downstream': Find reports that USE this attribute (live BFS traversal)
- 'upstream': Find source tables and dependencies (live BFS traversal)

WHY SPLIT? High-connectivity attributes would return too much data in a single call.
Choose the direction relevant to your query.

CORRECT USAGE:
- First search: search-attributes(query="Product Category")
- Then trace downstream: trace-attribute(guid="BC105EDE...", direction="downstream")
- Or trace upstream: trace-attribute(guid="BC105EDE...", direction="upstream")

PAGINATION: Returns 100 results. If moreResults=true, call again with offset+100.
```

### Cypher Query (downstream)

```cypher
// Trace DOWNSTREAM lineage for an Attribute (toward reports)
// $guid: Full GUID of the Attribute
// $offset: Pagination offset
//
// LIVE TRAVERSAL: Follows incoming DEPENDS_ON relationships to find consumers.
// Traverses up to 10 hops to find Reports/GridReports/Documents that use this attribute.

MATCH (n:Attribute {guid: $guid})

// Get effective status (updated takes precedence)
WITH n,
     COALESCE(n.updated_parity_status, n.parity_status, 'No Status') as effectiveStatus

// Find prioritized reports that depend on this attribute (live traversal)
// Filter: Only prioritized reports (priority_level IS NOT NULL)
OPTIONAL MATCH (report)-[:DEPENDS_ON*1..10]->(n)
WHERE report.type IN ['Report', 'GridReport', 'Document']
  AND report.priority_level IS NOT NULL

WITH n, effectiveStatus, report
ORDER BY report.name ASC
SKIP $offset
LIMIT 101

// Collect paginated reports
WITH n, effectiveStatus, collect(DISTINCT {
  name: report.name,
  guid: report.guid,
  type: report.type,
  priority: report.priority_level,
  area: report.usage_area
}) as fetched

// Return attribute with all properties (updated_ values take precedence)
RETURN {
  attribute: {
    type: 'Attribute',
    guid: n.guid,
    name: n.name,
    status: effectiveStatus,
    priority: n.inherited_priority_level,
    forms_json: n.forms_json,
    notes: COALESCE(n.updated_parity_notes, n.parity_notes),
    raw: COALESCE(n.updated_db_raw, n.db_raw),
    serve: COALESCE(n.updated_db_serve, n.db_serve),
    semantic: n.pb_semantic,
    edwTable: COALESCE(n.updated_edw_table, n.edw_table),
    edwColumn: n.edw_column,
    adeTable: COALESCE(n.updated_ade_db_table, n.ade_db_table),
    adeColumn: n.ade_db_column,
    semanticName: n.pb_semantic_name,
    semanticModel: n.pb_semantic_model,
    dbEssential: n.db_essential,
    pbEssential: n.pb_essential,
    ado_link: COALESCE(n.updated_ado_link, n.ado_link)
  },
  direction: 'downstream',
  reports: fetched[0..100],
  moreResults: size(fetched) > 100
} as result
```

### Cypher Query (upstream)

```cypher
// Trace UPSTREAM lineage for an Attribute (toward source tables)
// $guid: Full GUID of the Attribute
// $offset: Pagination offset
//
// LIVE TRAVERSAL: Follows outgoing DEPENDS_ON relationships to find data sources.
// Tables are reached via direct relationships or through intermediate objects.

MATCH (n:Attribute {guid: $guid})

// Get effective status (updated takes precedence)
WITH n,
     COALESCE(n.updated_parity_status, n.parity_status, 'No Status') as effectiveStatus

// Find source tables via live traversal (Attribute -> ... -> LogicalTable)
OPTIONAL MATCH (n)-[:DEPENDS_ON*1..10]->(t)
WHERE t.type IN ['LogicalTable', 'Table']

WITH n, effectiveStatus, t
ORDER BY t.name ASC
SKIP $offset
LIMIT 101

// Collect paginated tables
WITH n, effectiveStatus, collect(DISTINCT {
  name: t.name,
  guid: t.guid,
  type: t.type,
  physicalTable: t.physical_table_name,
  database: t.database_instance
}) as fetchedTables

// Get direct dependencies (depth 1-2 for immediate dependencies)
OPTIONAL MATCH (n)-[:DEPENDS_ON*1..2]->(dep)
WHERE dep.type IN ['Fact', 'Column', 'Attribute', 'Transformation']

WITH n, effectiveStatus, fetchedTables, collect(DISTINCT {
  name: dep.name,
  guid: dep.guid,
  type: dep.type
})[0..100] as dependencies

// Return attribute with all properties (updated_ values take precedence)
RETURN {
  attribute: {
    type: 'Attribute',
    guid: n.guid,
    name: n.name,
    status: effectiveStatus,
    priority: n.inherited_priority_level,
    forms_json: n.forms_json,
    notes: COALESCE(n.updated_parity_notes, n.parity_notes),
    raw: COALESCE(n.updated_db_raw, n.db_raw),
    serve: COALESCE(n.updated_db_serve, n.db_serve),
    semantic: n.pb_semantic,
    edwTable: COALESCE(n.updated_edw_table, n.edw_table),
    edwColumn: n.edw_column,
    adeTable: COALESCE(n.updated_ade_db_table, n.ade_db_table),
    adeColumn: n.ade_db_column,
    semanticName: n.pb_semantic_name,
    semanticModel: n.pb_semantic_model,
    dbEssential: n.db_essential,
    pbEssential: n.pb_essential,
    ado_link: COALESCE(n.updated_ado_link, n.ado_link)
  },
  direction: 'upstream',
  tables: fetchedTables[0..100],
  moreResults: size(fetchedTables) > 100,
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

The trace tools use **live graph traversal** following `DEPENDS_ON` relationships up to 10 hops.

### Traversal Directions

| Direction | Purpose | Query Pattern | Target Types |
|-----------|---------|---------------|--------------|
| **Downstream** (M/A → Reports) | Find prioritized reports using an object | `(report)-[:DEPENDS_ON*1..10]->(n)` | `[Report, GridReport, Document]` |
| **Upstream** (M/A → Tables) | Find source tables | `(n)-[:DEPENDS_ON*1..10]->(table)` | `[LogicalTable, Table]` |

### Prioritized Filter

Downstream queries only return **prioritized reports** — those with `priority_level IS NOT NULL`. This aligns MCP tool behavior with the dashboard queries and ensures consistent results.

### Live Traversal Design

The trace tools perform **live BFS traversal** at query time:
- **Depth:** Fixed at 10 hops to balance completeness and performance
- **Pagination:** Results are paginated (100 per page) with `moreResults` flag
- **Ordering:** Results sorted by name for consistent pagination

**Why Live Traversal:** Ensures results reflect the current state of the graph without relying on potentially stale pre-computed data.

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

#### On Attribute

| Property | Description |
|----------|-------------|
| `guid` | Unique identifier |
| `name` | Object name |
| `forms_json` | Attribute forms in JSON |
| `parity_status` | Original parity status |
| `updated_parity_status` | Updated parity status (takes precedence) |
| `ado_link` | ADO work item URL |

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

## Stress Testing

This section documents stress testing methodology and baseline results for performance validation.

### Test Environment

| Component | Version/Details |
|-----------|-----------------|
| **MCP Server** | flow-microstrategy-mcp (Docker: `flow-mcp-test:latest`) |
| **Neo4j** | 5.22 (Docker: `msts-neo4j`) |
| **Transport** | HTTP mode on port 8888 |
| **Host Binding** | `0.0.0.0:80` (container) → `8888` (host) |

### Running the MCP Server for Testing

```bash
# Start MCP server in HTTP mode via Docker
docker run -d --name flow-mcp-stress \
  -p 8888:80 \
  -e FLOW_URI="bolt://host.docker.internal:7687" \
  -e FLOW_MCP_TRANSPORT="http" \
  -e FLOW_MCP_HTTP_HOST="0.0.0.0" \
  flow-mcp-test:latest

# Initialize the MCP client
curl -s -X POST http://localhost:8888/mcp \
  -u "neo4j:password" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"stress-test","version":"1.0"}}}'
```

### Performance Baseline Results

| Tool | Latency | Results | Threshold | Status |
|------|---------|---------|-----------|--------|
| `search-metrics` | 50-57ms | 100 results | < 500ms | ✅ PASS |
| `search-attributes` | 37ms | 51 results | < 500ms | ✅ PASS |
| `trace-metric upstream` | 72ms | 1 table, 3 deps | < 500ms | ✅ PASS |
| `trace-attribute upstream` | 81ms | 7 tables, 3 deps | < 1s | ✅ PASS |
| `trace-metric downstream` | 5442ms | 2 reports | < 10s | ✅ PASS |
| `trace-attribute downstream` | 6020ms | 53 reports | < 10s | ✅ PASS |

**Note:** Downstream queries are slower due to BFS traversal (up to 10 hops) through the graph.

### Stress Testing Commands

#### 1. High-Connectivity Objects

Find metrics/attributes with many report connections:

```bash
# Find high-connectivity attributes
curl -s -X POST http://localhost:8888/mcp \
  -u "neo4j:password" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"search-attributes","arguments":{"query":"Product"}}}' | \
  jq -r '.result.content[0].text' | \
  jq '.[0].results | sort_by(-.reportCount) | .[0:5] | .[] | {name, guid, reportCount}'
```

#### 2. Pagination Test

Iterate through all pages:

```bash
# Test pagination (offset 0, 100, 200...)
for offset in 0 100 200; do
  echo "=== Offset: $offset ==="
  curl -s -X POST http://localhost:8888/mcp \
    -u "neo4j:password" \
    -H "Content-Type: application/json" \
    -d "{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"tools/call\",\"params\":{\"name\":\"search-metrics\",\"arguments\":{\"query\":\"Value\",\"offset\":$offset}}}" | \
    jq -r '.result.content[0].text' | \
    jq '.[0] | {results: (.results | length), moreResults}'
done
```

#### 3. Concurrent Requests

```bash
# 10 concurrent requests
seq 1 10 | xargs -P10 -I{} curl -s -X POST http://localhost:8888/mcp \
  -u "neo4j:password" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":{},"method":"tools/call","params":{"name":"search-metrics","arguments":{"query":"Value"}}}'
```

#### 4. Trace Tool Testing

```bash
# Trace attribute downstream (find reports using it)
curl -s -X POST http://localhost:8888/mcp \
  -u "neo4j:password" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"trace-attribute","arguments":{"guid":"YOUR_GUID_HERE","direction":"downstream"}}}'

# Trace metric upstream (find source tables)
curl -s -X POST http://localhost:8888/mcp \
  -u "neo4j:password" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"trace-metric","arguments":{"guid":"YOUR_GUID_HERE","direction":"upstream"}}}'
```

### Concurrent Load Test Results

| Test | Total Time | Avg per Request | Success Rate |
|------|------------|-----------------|--------------|
| 10 parallel requests | 146ms | 14ms | 90-100% |
| 20 parallel requests | 187ms | 9ms | 100% |
| 50 parallel requests | 449ms | 9ms | 100% |

### Expected Response Time Thresholds

| Tool | Normal | Stress Threshold |
|------|--------|------------------|
| `search-metrics` | < 100ms | < 500ms |
| `search-attributes` | < 100ms | < 500ms |
| `trace-metric downstream` | < 6s | < 10s |
| `trace-metric upstream` | < 100ms | < 500ms |
| `trace-attribute downstream` | < 6s | < 10s |
| `trace-attribute upstream` | < 100ms | < 500ms |

### Key Metrics to Monitor

1. **Query time:** Should stay under thresholds above
2. **Memory:** Watch for spikes during traversal queries (MCP: ~11 MiB, Neo4j: ~3 GiB typical)
3. **Connection pool:** Monitor under concurrent load
4. **`moreResults` accuracy:** Verify pagination terminates correctly

### Sample Report Data

Reports returned from downstream traces include:

```json
{
  "name": "Report Name",
  "guid": "ABC123...",
  "type": "Report",
  "priority": 3,
  "area": "Usage Area"
}
```

The `priority` field corresponds to `report.priority_level` in the database (not `inherited_priority_level`, which is used for metrics/attributes).

---

## Update History

| Date | Version | Change |
|------|---------|--------|
| 2026-02-04 | 3.1.0 | Fixed downstream filter: `report.inherited_priority_level` → `report.priority_level`; Added stress testing documentation |
| 2026-02-04 | 3.0.0 | Replaced pre-computed lineage with live graph traversal (DEPENDS_ON*1..10); Added pagination to trace tools |
| 2026-02-03 | 2.0.0 | Major refactor: Consolidated 15 tools into 4 unified tools; Added full Cypher queries and implementation details |
| 2026-01-30 | 1.0.0 | Initial document with 12 MSTR tools |
