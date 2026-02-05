# MCP Tools Reference

**Date Created:** January 30, 2026  
**Last Updated:** February 5, 2026  
**API Server:** Remote MCP Server  
**Total Tools:** 4 MicroStrategy tools  
**Status:** ✅ PRODUCTION

---

## Overview

This document describes the MCP (Model Context Protocol) tools deployed on the remote API server for querying MicroStrategy metadata stored in Neo4j. These tools enable LLM agents to search for objects and trace their lineage.

### Tool Categories

| Category | Tools | Purpose |
|----------|-------|---------|
| **Search** | `search-metrics`, `search-attributes` | Find objects by GUID or name |
| **Trace** | `trace-metric`, `trace-attribute` | Explore lineage (reports, tables, dependencies) |

### Design Principles

1. **Unified Search** — Each search tool accepts GUIDs or names interchangeably
2. **Type-Specific** — Each tool operates only on its specific type (no Metric/Attribute unions)
3. **Updated Properties Take Precedence** — Properties with `updated_` prefix override originals (e.g., `updated_parity_status` > `parity_status`)
4. **ADO Integration** — All results include ADO work item links when available

---

## Tools Summary

| Tool | Description | Input |
|------|-------------|-------|
| `search-metrics` | Find Metrics by GUID or name | GUID(s) or search term |
| `search-attributes` | Find Attributes by GUID or name | GUID(s) or search term |
| `trace-metric` | Trace Metric lineage (reports, tables, dependencies) | GUID |
| `trace-attribute` | Trace Attribute lineage (reports, tables, dependencies) | GUID |

---

## Tool 1: search-metrics

### Purpose

Find Metrics by GUID or name. Accepts full GUIDs, partial GUIDs (8+ chars), or name search terms interchangeably.

### Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `query` | string | Yes | — | GUID (full or partial 8+ chars) or name search term |
| `status` | array | No | null | Filter by parity status: `["Complete", "Planned", "Not Planned"]` |
| `offset` | number | No | 0 | Skip first N results for pagination |

### Query

```cypher
-- Search for Metrics by GUID or name
-- $query: GUID (full/partial) or name search term
-- $status: optional parity status filter
-- $offset: pagination offset (0, 100, 200, ...)

-- Determine if query looks like a GUID (hex chars, 8+ length)
WITH $query as query,
     $query =~ '^[A-Fa-f0-9]{8,}$' as isGuidLike

MATCH (n:Metric)
WHERE n.guid IS NOT NULL
  AND (
    -- GUID match: exact or partial (starts with)
    (isGuidLike AND (n.guid = query OR n.guid STARTS WITH toUpper(query)))
    OR
    -- Name match: case-insensitive contains
    (NOT isGuidLike AND toLower(n.name) CONTAINS toLower(query))
  )
  AND ($status IS NULL OR COALESCE(n.updated_parity_status, n.parity_status) IN $status)

-- Compute effective status (updated takes precedence)
WITH n,
     COALESCE(n.updated_parity_status, n.parity_status, 'No Status') as effectiveStatus

ORDER BY n.name ASC
SKIP $offset
LIMIT 101  -- Fetch 101 to determine if more results exist

-- Collect results
WITH collect({
  type: 'Metric',
  guid: n.guid,
  name: n.name,
  status: effectiveStatus,
  formula: n.formula,
  ado_link: n.ado_link
}) as fetched

-- Return first 100; moreResults=true if 101st exists
RETURN 
  fetched[0..100] as results,
  size(fetched) > 100 as moreResults
```

### Response Structure

```typescript
interface SearchMetricsResponse {
  results: MetricResult[];
  moreResults: boolean;  // true = more pages available
}

interface MetricResult {
  type: "Metric";
  guid: string;           // Full 32-char GUID
  name: string;           // Display name
  status: string;         // "Complete" | "Planned" | "Not Planned" | "No Status"
  formula: string | null; // Metric formula
  ado_link: string | null; // ADO work item URL
}
```

### Usage Examples

#### ✅ Correct Usage

```
# Search by full GUID
search-metrics(query="2F00974D44E1D0D24CA344ABD872806A")

# Search by partial GUID (8+ chars)
search-metrics(query="2F00974D")

# Search by name
search-metrics(query="Retail Sales")

# Search with status filter
search-metrics(query="sales", status=["Complete", "Planned"])

# Paginate results
search-metrics(query="revenue", offset=100)
```

#### ❌ Incorrect Usage

```
# DON'T use type parameter - this tool is Metric-only
search-metrics(query="sales", type="Attribute")  ❌

# DON'T use partial GUID shorter than 8 chars (too ambiguous)
search-metrics(query="2F00")  ❌

# DON'T use for lineage queries - use trace-metric instead
search-metrics(query="2F00974D", include_reports=true)  ❌
```

### Example Response

```json
{
  "results": [
    {
      "type": "Metric",
      "guid": "2F00974D44E1D0D24CA344ABD872806A",
      "name": "Retail Sales Value",
      "status": "Complete",
      "formula": "Sum(Sales Fact.sale_value)",
      "ado_link": "https://dev.azure.com/org/project/_workitems/edit/12345"
    },
    {
      "type": "Metric",
      "guid": "416AEF98418213C610991A810D9E0C05",
      "name": "Retail Sales Units",
      "status": "Planned",
      "formula": "Sum(Sales Fact.units_sold)",
      "ado_link": null
    }
  ],
  "moreResults": true
}
```

---

## Tool 2: search-attributes

### Purpose

Find Attributes by GUID or name. Accepts full GUIDs, partial GUIDs (8+ chars), or name search terms interchangeably.

### Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `query` | string | Yes | — | GUID (full or partial 8+ chars) or name search term |
| `status` | array | No | null | Filter by parity status: `["Complete", "Planned", "Not Planned"]` |
| `offset` | number | No | 0 | Skip first N results for pagination |

### Query

```cypher
-- Search for Attributes by GUID or name
-- $query: GUID (full/partial) or name search term
-- $status: optional parity status filter
-- $offset: pagination offset (0, 100, 200, ...)

-- Determine if query looks like a GUID (hex chars, 8+ length)
WITH $query as query,
     $query =~ '^[A-Fa-f0-9]{8,}$' as isGuidLike

MATCH (n:Attribute)
WHERE n.guid IS NOT NULL
  AND (
    -- GUID match: exact or partial (starts with)
    (isGuidLike AND (n.guid = query OR n.guid STARTS WITH toUpper(query)))
    OR
    -- Name match: case-insensitive contains
    (NOT isGuidLike AND toLower(n.name) CONTAINS toLower(query))
  )
  AND ($status IS NULL OR COALESCE(n.updated_parity_status, n.parity_status) IN $status)

-- Compute effective status (updated takes precedence)
WITH n,
     COALESCE(n.updated_parity_status, n.parity_status, 'No Status') as effectiveStatus

ORDER BY n.name ASC
SKIP $offset
LIMIT 101  -- Fetch 101 to determine if more results exist

-- Collect results
WITH collect({
  type: 'Attribute',
  guid: n.guid,
  name: n.name,
  status: effectiveStatus,
  forms_json: n.forms_json,
  ado_link: n.ado_link
}) as fetched

-- Return first 100; moreResults=true if 101st exists
RETURN 
  fetched[0..100] as results,
  size(fetched) > 100 as moreResults
```

### Response Structure

```typescript
interface SearchAttributesResponse {
  results: AttributeResult[];
  moreResults: boolean;  // true = more pages available
}

interface AttributeResult {
  type: "Attribute";
  guid: string;             // Full 32-char GUID
  name: string;             // Display name
  status: string;           // "Complete" | "Planned" | "Not Planned" | "No Status"
  forms_json: string | null; // Attribute forms definition
  ado_link: string | null;  // ADO work item URL
}
```

### Usage Examples

#### ✅ Correct Usage

```
# Search by full GUID
search-attributes(query="BC105EDE477D7CEF3296FFA6E4D26797")

# Search by partial GUID (8+ chars)
search-attributes(query="BC105EDE")

# Search by name
search-attributes(query="Product Category")

# Search with status filter
search-attributes(query="product", status=["Complete"])

# Paginate results
search-attributes(query="date", offset=100)
```

#### ❌ Incorrect Usage

```
# DON'T use type parameter - this tool is Attribute-only
search-attributes(query="category", type="Metric")  ❌

# DON'T use partial GUID shorter than 8 chars
search-attributes(query="BC10")  ❌

# DON'T use for lineage queries - use trace-attribute instead
search-attributes(query="BC105EDE", get_tables=true)  ❌
```

### Example Response

```json
{
  "results": [
    {
      "type": "Attribute",
      "guid": "BC105EDE477D7CEF3296FFA6E4D26797",
      "name": "Product Category",
      "status": "Complete",
      "forms_json": "{\"ID\": \"product_category_id\", \"DESC\": \"product_category_desc\"}",
      "ado_link": "https://dev.azure.com/org/project/_workitems/edit/23456"
    },
    {
      "type": "Attribute",
      "guid": "D0365E2C4E48FCB35CC0FFA8E4999121",
      "name": "Product Brand",
      "status": "Planned",
      "forms_json": null,
      "ado_link": null
    }
  ],
  "moreResults": false
}
```

---

## Tool 3: trace-metric

### Purpose

Trace the lineage of a Metric in a specific direction using live graph traversal. Replaces the previous `get-metric-dependencies`, `get-metric-source-tables`, and `get-reports-using-metric` tools.

### Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `guid` | string | Yes | — | Full GUID of the Metric to trace |
| `direction` | string | Yes | — | `"downstream"` (toward reports) or `"upstream"` (toward tables) |
| `offset` | number | No | 0 | Skip first N results for pagination |

### Query (downstream)

```cypher
-- Trace downstream lineage for a Metric (live traversal)
-- $guid: Full GUID of the Metric
-- $offset: Pagination offset

MATCH (n:Metric {guid: $guid})

-- Get effective status (updated takes precedence)
WITH n,
     COALESCE(n.updated_parity_status, n.parity_status, 'No Status') as effectiveStatus

-- Find prioritized reports that depend on this metric (live BFS traversal, 10 hops)
-- Path filter: Only traverse through [Prompt, Filter] intermediate nodes (canonical dashboard pattern)
-- Filter: Only prioritized reports (priority_level IS NOT NULL)
OPTIONAL MATCH path = (report)-[:DEPENDS_ON*1..10]->(n)
WHERE report.type IN ['Report', 'GridReport', 'Document']
  AND report.priority_level IS NOT NULL
  AND ALL(mid IN nodes(path)[1..-1] WHERE mid.type IN ['Prompt', 'Filter'])

WITH n, effectiveStatus, report
ORDER BY report.name ASC
SKIP $offset
LIMIT 101

WITH n, effectiveStatus, collect(DISTINCT {
  name: report.name,
  guid: report.guid,
  type: report.type
}) as fetched

RETURN {
  metric: {
    guid: n.guid,
    name: n.name,
    status: effectiveStatus,
    formula: n.formula,
    ado_link: n.ado_link
  },
  direction: 'downstream',
  reports: fetched[0..100],
  moreResults: size(fetched) > 100
} as result
```

### Response Structure

```typescript
interface TraceMetricResponse {
  metric: {
    guid: string;
    name: string;
    status: string;
    formula: string | null;
    ado_link: string | null;
  };
  direction: "downstream" | "upstream";
  reports?: ReportRef[];      // For downstream (100 per page)
  tables?: TableRef[];        // For upstream (100 per page)
  dependencies?: DepRef[];    // For upstream
  moreResults: boolean;       // true if more pages available
}

interface ReportRef {
  name: string;
  guid: string;
  type: "Report" | "GridReport" | "Document";
}

interface TableRef {
  name: string;
  guid: string;
  type: "LogicalTable" | "Table";
  physicalTable: string | null;
  database: string | null;
}

interface DepRef {
  name: string;
  guid: string;
  type: string;  // "Fact", "Metric", "Attribute", "Column", etc.
  formula: string | null;
}
```

### Usage Examples

#### ✅ Correct Usage

```
# Trace a metric by its full GUID
trace-metric(guid="2F00974D44E1D0D24CA344ABD872806A")

# Use after finding a metric with search-metrics
1. search-metrics(query="Retail Sales Value")
2. trace-metric(guid="2F00974D44E1D0D24CA344ABD872806A")  # from search result
```

#### ❌ Incorrect Usage

```
# DON'T use partial GUID - trace requires full GUID
trace-metric(guid="2F00974D")  ❌

# DON'T use for Attributes - use trace-attribute instead
trace-metric(guid="BC105EDE477D7CEF3296FFA6E4D26797")  ❌  # This is an Attribute GUID

# DON'T use name - trace requires GUID
trace-metric(guid="Retail Sales Value")  ❌

# DON'T use for searching - use search-metrics first
trace-metric(query="sales")  ❌
```

### Example Response

```json
{
  "metric": {
    "guid": "2F00974D44E1D0D24CA344ABD872806A",
    "name": "Retail Sales Value",
    "status": "Complete",
    "formula": "Sum(Sales Fact.sale_value)",
    "ado_link": "https://dev.azure.com/org/project/_workitems/edit/12345"
  },
  "reports": [
    {
      "name": "Daily Sales Dashboard",
      "guid": "ABC123DE456789F0ABC123DE456789F0",
      "type": "Report"
    },
    {
      "name": "Weekly Trade Summary",
      "guid": "DEF456AB789012C3DEF456AB789012C3",
      "type": "GridReport"
    }
  ],
  "tables": [
    {
      "name": "FACT_SALES",
      "guid": "GHI789JK012345L6GHI789JK012345L6",
      "type": "LogicalTable",
      "physicalTable": "dbo.FACT_SALES_DAILY",
      "database": "EDW_PROD"
    },
    {
      "name": "LU_PRODUCT",
      "guid": "MNO345PQ678901R2MNO345PQ678901R2",
      "type": "LogicalTable",
      "physicalTable": "dbo.DIM_PRODUCT",
      "database": "EDW_PROD"
    }
  ],
  "dependencies": [
    {
      "name": "Sales Fact",
      "guid": "FACT123456789ABCFACT123456789ABC",
      "type": "Fact",
      "formula": null
    },
    {
      "name": "Base Sales",
      "guid": "MET456789ABCDEF0MET456789ABCDEF0",
      "type": "Metric",
      "formula": "Sum(sale_value)"
    }
  ]
}
```

---

## Tool 4: trace-attribute

### Purpose

Trace the lineage of an Attribute in a specific direction using live graph traversal. Replaces the previous `get-attribute-dependencies`, `get-attribute-source-tables`, and `get-reports-using-attribute` tools.

### Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `guid` | string | Yes | — | Full GUID of the Attribute to trace |
| `direction` | string | Yes | — | `"downstream"` (toward reports) or `"upstream"` (toward tables) |
| `offset` | number | No | 0 | Skip first N results for pagination |

### Query (downstream)

```cypher
-- Trace downstream lineage for an Attribute (live traversal)
-- $guid: Full GUID of the Attribute
-- $offset: Pagination offset

MATCH (n:Attribute {guid: $guid})

-- Get effective status (updated takes precedence)
WITH n,
     COALESCE(n.updated_parity_status, n.parity_status, 'No Status') as effectiveStatus

-- Find prioritized reports that depend on this attribute (live BFS traversal, 10 hops)
-- Path filter: Only traverse through [Prompt, Filter] intermediate nodes (canonical dashboard pattern)
-- Filter: Only prioritized reports (priority_level IS NOT NULL)
OPTIONAL MATCH path = (report)-[:DEPENDS_ON*1..10]->(n)
WHERE report.type IN ['Report', 'GridReport', 'Document']
  AND report.priority_level IS NOT NULL
  AND ALL(mid IN nodes(path)[1..-1] WHERE mid.type IN ['Prompt', 'Filter'])

WITH n, effectiveStatus, report
ORDER BY report.name ASC
SKIP $offset
LIMIT 101

WITH n, effectiveStatus, collect(DISTINCT {
  name: report.name,
  guid: report.guid,
  type: report.type
}) as fetched

RETURN {
  attribute: {
    guid: n.guid,
    name: n.name,
    status: effectiveStatus,
    forms_json: n.forms_json,
    ado_link: n.ado_link
  },
  direction: 'downstream',
  reports: fetched[0..100],
  moreResults: size(fetched) > 100
} as result
```

### Response Structure

```typescript
interface TraceAttributeResponse {
  attribute: {
    guid: string;
    name: string;
    status: string;
    forms_json: string | null;
    ado_link: string | null;
  };
  direction: "downstream" | "upstream";
  reports?: ReportRef[];      // For downstream (100 per page)
  tables?: TableRef[];        // For upstream (100 per page)
  dependencies?: DepRef[];    // For upstream
  moreResults: boolean;       // true if more pages available
}

interface ReportRef {
  name: string;
  guid: string;
  type: "Report" | "GridReport" | "Document";
}

interface TableRef {
  name: string;
  guid: string;
  type: "LogicalTable" | "Table";
  physicalTable: string | null;
  database: string | null;
}

interface DepRef {
  name: string;
  guid: string;
  type: string;  // "Column", "Attribute", etc.
}
```

### Usage Examples

#### ✅ Correct Usage

```
# Trace an attribute by its full GUID
trace-attribute(guid="BC105EDE477D7CEF3296FFA6E4D26797")

# Use after finding an attribute with search-attributes
1. search-attributes(query="Product Category")
2. trace-attribute(guid="BC105EDE477D7CEF3296FFA6E4D26797")  # from search result
```

#### ❌ Incorrect Usage

```
# DON'T use partial GUID - trace requires full GUID
trace-attribute(guid="BC105EDE")  ❌

# DON'T use for Metrics - use trace-metric instead
trace-attribute(guid="2F00974D44E1D0D24CA344ABD872806A")  ❌  # This is a Metric GUID

# DON'T use name - trace requires GUID
trace-attribute(guid="Product Category")  ❌

# DON'T use for searching - use search-attributes first
trace-attribute(query="product")  ❌
```

### Example Response

```json
{
  "attribute": {
    "guid": "BC105EDE477D7CEF3296FFA6E4D26797",
    "name": "Product Category",
    "status": "Complete",
    "forms_json": "{\"ID\": \"product_category_id\", \"DESC\": \"product_category_desc\"}",
    "ado_link": "https://dev.azure.com/org/project/_workitems/edit/23456"
  },
  "reports": [
    {
      "name": "Category Performance",
      "guid": "RPT123456789ABCRPT123456789ABC",
      "type": "Report"
    },
    {
      "name": "Product Mix Analysis",
      "guid": "RPT456789ABCDEFRPT456789ABCDEF",
      "type": "GridReport"
    }
  ],
  "tables": [
    {
      "name": "LU_PRODUCT",
      "guid": "MNO345PQ678901R2MNO345PQ678901R2",
      "type": "LogicalTable",
      "physicalTable": "dbo.DIM_PRODUCT",
      "database": "EDW_PROD"
    }
  ],
  "dependencies": [
    {
      "name": "product_category_id",
      "guid": "COL123456789ABCCOL123456789ABC",
      "type": "Column"
    },
    {
      "name": "product_category_desc",
      "guid": "COL456789ABCDEFCOL456789ABCDEF",
      "type": "Column"
    }
  ]
}
```

---

## Common Workflows

### Workflow 1: Find and Trace a Metric

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

### Workflow 2: Find and Trace an Attribute

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

### Workflow 3: Lookup by Known GUID

```
# If you already have a GUID, search confirms the object exists
search-metrics(query="2F00974D44E1D0D24CA344ABD872806A")
→ Returns the metric details

# Then trace for full lineage
trace-metric(guid="2F00974D44E1D0D24CA344ABD872806A")
```

### Workflow 4: Browse by Status

```
# Find all "Not Planned" metrics
search-metrics(query="*", status=["Not Planned"])
→ Returns metrics needing attention

# Paginate through results
search-metrics(query="*", status=["Not Planned"], offset=100)
```

---

## Graph Traversal Rules

The trace tools use **live graph traversal** following `DEPENDS_ON` relationships up to 10 hops.

### Traversal Directions

| Direction | Purpose | Query Pattern | Target Types | Intermediate Filter |
|-----------|---------|---------------|--------------|---------------------|
| **Downstream** (M/A → Reports) | Find prioritized reports using an object | `path = (report)-[:DEPENDS_ON*1..10]->(n)` | `[Report, GridReport, Document]` | `['Prompt', 'Filter']` |
| **Upstream** (M/A → Tables) | Find source tables | `path = (n)-[:DEPENDS_ON*1..10]->(table)` | `[LogicalTable, Table]` | `['Fact', 'Metric', 'Attribute', 'Column']` |

### Intermediate Type Filters

Both directions use **path-constrained BFS** to ensure traversals follow canonical dashboard patterns:

**Downstream (to reports):**
```cypher
AND ALL(mid IN nodes(path)[1..-1] WHERE mid.type IN ['Prompt', 'Filter'])
```

**Upstream (to tables):**
```cypher
AND ALL(mid IN nodes(path)[1..-1] WHERE mid.type IN ['Fact', 'Metric', 'Attribute', 'Column'])
```

These filters ensure:
- Downstream paths only traverse through Prompts and Filters (report dependency chain)
- Upstream paths traverse through Facts, Metrics, Attributes, and Columns (data lineage chain)
- Results match the dashboard's canonical query behavior

### Prioritized Filter

Downstream queries only return **prioritized reports** — those with `priority_level IS NOT NULL`. This aligns MCP tool behavior with the dashboard queries and ensures consistent results.

### Live Traversal Design

The trace tools perform **live BFS traversal** at query time:
- **Depth:** Fixed at 10 hops to balance completeness and performance
- **Pagination:** Results are paginated (100 per page) with `moreResults` flag
- **Ordering:** Results sorted by name for consistent pagination

**Why Live Traversal:** Ensures results reflect the current state of the graph without relying on potentially stale pre-computed data.

---

## Database Schema Reference

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
| `Column` | Columns |

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

| Old Tool | New Tool | Notes |
|----------|----------|-------|
| `get-metric-by-guid` | `search-metrics` | Use GUID as query parameter |
| `get-attribute-by-guid` | `search-attributes` | Use GUID as query parameter |
| `search-metrics` (old) | `search-metrics` | Same, but unified with GUID lookup |
| `search-attributes` (old) | `search-attributes` | Same, but unified with GUID lookup |
| `get-reports-using-metric` | `trace-metric` | Returns reports in `reports` array |
| `get-reports-using-attribute` | `trace-attribute` | Returns reports in `reports` array |
| `get-metric-source-tables` | `trace-metric` | Returns tables in `tables` array |
| `get-attribute-source-tables` | `trace-attribute` | Returns tables in `tables` array |
| `get-metric-dependencies` | `trace-metric` | Returns deps in `dependencies` array |
| `get-attribute-dependencies` | `trace-attribute` | Returns deps in `dependencies` array |
| `get-metric-dependents` | `trace-metric` | Returns reports in `reports` array |
| `get-attribute-dependents` | `trace-attribute` | Returns reports in `reports` array |

### Removed Fields

The following fields have been removed from responses:
- `parity_group` / `Group`
- `parity_subgroup` / `SubGroup`
- `lineage_used_by_reports_count` / `reportCount`
- `lineage_source_tables_count` / `tableCount`
- `team` (NeoDash-specific)
- `priority` / `inherited_priority_level` (NeoDash-specific)

---

## Changelog

| Date | Version | Change |
|------|---------|--------|
| 2026-02-05 | 4.1.0 | Added intermediate type filters to trace queries: downstream uses `['Prompt', 'Filter']`, upstream uses `['Fact', 'Metric', 'Attribute', 'Column']`. Fixes 39 vs 44 table divergence (see doc 106). |
| 2026-02-04 | 4.0.0 | Replaced pre-computed lineage with live graph traversal; Added pagination (offset) to trace tools; Trace tools now use DEPENDS_ON*1..10 traversal |
| 2026-02-03 | 3.0.0 | Major refactor: Reduced to 4 tools (search-metrics, search-attributes, trace-metric, trace-attribute); Unified search by GUID or name; Added ADO link; Removed counts, groups, and NeoDash-specific parameters |
| 2026-01-30 | 2.3.0 | Added consistent verbose comments to all 6 optimized queries |
| 2026-01-30 | 2.0.0 | Added Usage/Parameters/Pagination sections |
| 2026-01-30 | 1.0.0 | Initial document with 12 MSTR tools |
