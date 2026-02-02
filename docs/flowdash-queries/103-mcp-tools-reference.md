# MCP Tools Reference

**Date Created:** January 30, 2026  
**Last Updated:** January 30, 2026  
**API Server:** Remote MCP Server  
**Total Tools:** 12 MicroStrategy tools  
**Status:** 🔧 OPTIMIZATION IN PROGRESS

---

## Overview

This document describes the MCP (Model Context Protocol) tools deployed on the remote API server for querying MicroStrategy metadata stored in Neo4j. These tools enable LLM agents to retrieve object details, search metadata, explore lineage, and analyze dependencies.

### Pre-computed Lineage: Why It Exists

The graph contains **pre-computed lineage arrays** stored as node properties:
- `lineage_used_by_reports` — Array of report GUIDs that use this M/A
- `lineage_source_tables` — Array of table GUIDs this M/A depends on
- `lineage_used_by_reports_count` — Count of reports
- `lineage_source_tables_count` — Count of tables

**History:**
1. **Testing Phase:** Pre-computed arrays were created to accelerate lineage queries during development and testing. Runtime BFS traversal was too slow for iterative testing.
2. **Production Adoption:** When the Neo4j dashboard was deployed to production, runtime lineage queries were **timing out** (>30 seconds) for high-connectivity objects (e.g., "Retail Sales Value" with 6,000+ reports).
3. **Solution:** Pre-computed arrays reduced query time from 30+ seconds to <100ms, enabling responsive UI.

**Trade-off:** Pre-computed data may become stale if the graph is modified. Runtime alternatives (documented below) are available for validation or post-modification queries.

### Primary Goal: LLM Context Optimization

**The main objective is to optimize query results to fit within LLM context windows** without losing essential information. Current queries return verbose data that can easily overflow context limits (typically 100K-200K tokens).

| Challenge | Current State | Target |
|-----------|---------------|--------|
| Object details | 18+ fields per object | 5-8 essential fields |
| Search results | Unlimited rows | Limited to 100 per call |
| Dependency paths | Full graph paths (up to 1000) | Summary counts + samples |
| Lineage arrays | Thousands of GUIDs | Counts + representative samples |

### Tool Categories

| Category | Tools | Purpose |
|----------|-------|---------|
| **GUID Lookup** | 2 | Retrieve object details by GUID |
| **Search** | 2 | Search objects with filters |
| **Reports** | 2 | Find reports using objects |
| **Lineage** | 2 | Get source tables for objects |
| **Downstream** | 2 | Get what objects depend on |
| **Upstream** | 2 | Get objects that depend on target |

---

## Tools Summary

### GUID Lookup Tools

| Tool | Description | Query Used |
|------|-------------|------------|
| `get-metric-by-guid` | Get Metric details by GUID | `GetObjectDetailsQuery` |
| `get-attribute-by-guid` | Get Attribute details by GUID | `GetObjectDetailsQuery` |

### Search Tools

| Tool | Description | Query Used |
|------|-------------|------------|
| `search-metrics` | Search Metrics with filters | `SearchObjectsQuery` |
| `search-attributes` | Search Attributes with filters | `SearchObjectsQuery` |

### Report Dependency Tools

| Tool | Description | Query Used |
|------|-------------|------------|
| `get-reports-using-metric` | Reports that use a Metric | `ReportsUsingObjectsQuery` |
| `get-reports-using-attribute` | Reports that use an Attribute | `ReportsUsingObjectsQuery` |

### Lineage (Source Tables) Tools

| Tool | Description | Query Used |
|------|-------------|------------|
| `get-metric-source-tables` | Source tables for a Metric | `SourceTablesQuery` |
| `get-attribute-source-tables` | Source tables for an Attribute | `SourceTablesQuery` |

### Downstream Dependency Tools

| Tool | Description | Query Used |
|------|-------------|------------|
| `get-metric-dependencies` | What a Metric depends on | `DownstreamDependenciesQuery` |
| `get-attribute-dependencies` | What an Attribute depends on | `DownstreamDependenciesQuery` |

### Upstream Dependency Tools

| Tool | Description | Query Used |
|------|-------------|------------|
| `get-metric-dependents` | What depends on a Metric | `UpstreamDependenciesQuery` |
| `get-attribute-dependents` | What depends on an Attribute | `UpstreamDependenciesQuery` |

### Statistics Tools (NEW)

| Tool | Description | Query Used |
|------|-------------|------------|
| `get-metrics-stats` | Count metrics by status/priority/domain | `MetricsStatsQuery` |
| `get-attributes-stats` | Count attributes by status/priority/domain | `AttributesStatsQuery` |
| `get-object-stats` | Summary stats for a specific object | `ObjectStatsQuery` |

**Purpose:** Statistics tools return aggregates only (counts, distributions). Detail tools return up to 100 results per call without totals. This separation keeps queries fast and responses small.

---

## Graph Traversal Rules

**Reference:** [90-symmetric-bfs-traversal.md](./90-symmetric-bfs-traversal.md)

All MCP tools MUST respect these traversal rules to ensure consistency with pre-computed lineage properties.

### Core Concept: Capture vs Traverse

The key distinction is between **capturing** an object (including it in results) and **traversing** through it (following its edges).

| Behavior | Types | What Happens |
|----------|-------|--------------|
| **TRAVERSE** | `Prompt`, `Filter` | BFS follows their edges to find more objects |
| **CAPTURE** | `Metric`, `Attribute`, `DerivedMetric`, `Transformation` | Added to results, but BFS stops here |

### Configuration Values

```json
{
  "graph": {
    "traversalTypes": ["Prompt", "Filter"],
    "countedTypes": ["Attribute", "Metric", "DerivedMetric", "Transformation"],
    "reverseLineage": {
      "reportTypes": ["Report", "GridReport", "Document"],
      "tableTypes": ["LogicalTable", "Table"]
    }
  }
}
```

### Traversal Directions

| Direction | Purpose | Intermediate Types | Target Types |
|-----------|---------|-------------------|--------------|
| **Inbound** (Reports → M/A) | Find reports using an object | `[Prompt, Filter]` | `[Report, GridReport, Document]` |
| **Outbound** (M/A → Tables) | Find source tables | `[Fact, Metric, Attribute, Column]` (hardcoded) | `[LogicalTable, Table]` |

### Required Path Filters

**ALL queries with variable-length paths MUST include intermediate node filtering:**

```cypher
-- INBOUND (Reports → M/A): Only traverse through Prompt and Filter
MATCH path = (r)-[:DEPENDS_ON*1..10]->(n)
WHERE r.type IN ['Report', 'GridReport', 'Document']
  AND ALL(mid IN nodes(path)[1..-1] WHERE mid.type IN ['Prompt', 'Filter'])

-- OUTBOUND (M/A → Tables): Only traverse through Fact, Metric, Attribute, Column
MATCH path = (n)-[:DEPENDS_ON*1..10]->(t)
WHERE t.type IN ['LogicalTable', 'Table']
  AND ALL(mid IN nodes(path)[1..-1] WHERE mid.type IN ['Fact', 'Metric', 'Attribute', 'Column'])
```

### Why DerivedMetric/Transformation Are NOT Traversed

| Reason | Explanation |
|--------|-------------|
| **Count inflation** | DerivedMetrics reference base Metrics, creating redundant paths |
| **Runtime objects** | Not schema-defined; transformations happen at query time |
| **Incorrect attribution** | Traversing through Transformation changes data semantics |
| **Consistency** | Pre-computed properties use same rules; mismatches cause confusion |

### Valid vs Invalid Paths

| Path | Captured | Why |
|------|----------|-----|
| Report → Metric | Metric ✓ | Direct dependency |
| Report → Prompt → Metric | Metric ✓ | Prompt is traversed |
| Report → Filter → Attribute | Attribute ✓ | Filter is traversed |
| Report → DerivedMetric | DerivedMetric ✓ | Direct, captured |
| Report → DerivedMetric → BaseMetric | Only DerivedMetric ✓ | DM not traversed, BaseMetric NOT reached |
| Report → Transformation → Fact | Only Transformation ✓ | T not traversed, Fact NOT reached |

---

## Optimized Query Specifications

**This is the primary focus of this document.** The queries must be optimized to return all necessary information while fitting within LLM context window limits.

### Context Window Constraints

| Model | Context Window | Safe Output Limit |
|-------|----------------|-------------------|
| Claude 3.5 Sonnet | 200K tokens | ~50K tokens for responses |
| GPT-4 | 128K tokens | ~30K tokens for responses |
| Smaller models | 8K-32K tokens | ~5K-10K tokens |

**Rule of thumb:** Query results should stay under **10,000 characters** (~2,500 tokens) per tool call.

### Optimization Strategies

| Strategy | Description | Example |
|----------|-------------|---------|
| **Field Selection** | Return only essential fields | 18 fields → 5 essential |
| **Result Limits** | Cap results per call | Fixed `LIMIT 100` |
| **Pagination** | Fixed 100 per page | `SKIP $offset LIMIT 100` |
| **Separation of Concerns** | Details vs Statistics split | Separate tools for counts |
| **Condensed Format** | Compact output strings | `"Name (GUID8): Status"` |

### Design Decision: Detail vs Statistics Tools

**Principle:** Detail queries should NOT include total counts. This enables efficient `SKIP`/`LIMIT` pagination without collecting all results into memory.

| Tool Type | Purpose | Returns | Pagination |
|-----------|---------|---------|------------|
| **Detail tools** | Return object data | 100 per page | `SKIP $offset LIMIT 100` |
| **Statistics tools** | Return counts/summaries | Aggregates only | N/A |

**Benefits:**
1. **Performance** — Detail queries don't need to scan entire dataset for count
2. **SRP** — Each tool does one thing well
3. **Flexibility** — Agent can get stats once, then fetch details as needed

**Example workflow:**
```
1. get-metrics-stats(domain="Finance")     → {total: 450, byStatus: {...}}
2. search-metrics(domain="Finance", limit=100, offset=0)   → [100 metrics]
3. search-metrics(domain="Finance", limit=100, offset=100) → [100 metrics]
...
```

---

## Optimized Query Specifications

Each query below includes:
1. **Component breakdown** — Line-by-line explanation
2. **Traversal rule compliance** — Verification against graph rules
3. **Pagination parameters** — Default 100, max 200

---

### Query 1: GetObjectDetailsQuery (Optimized)

**Tools:** `get-metric-by-guid`, `get-attribute-by-guid`

**Original Problem:** Returns 18 fields, many often NULL or irrelevant.

**Optimized:** Essential fields only, limited results.

#### Usage

Retrieve detailed information for specific Metrics or Attributes by their GUIDs. Use when you have one or more GUIDs and need object metadata.

#### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `guids` | array | Yes | Array of full or partial GUIDs to look up |

#### Query

```cypher
-- Find Metric or Attribute objects by GUID
MATCH (n:MSTRObject)
WHERE n.type IN ['Metric', 'Attribute'] 
  AND n.guid IN $guids

-- Return essential fields only (optimized from original 18 fields)
-- status: prefer updated_parity_status, fallback to parity_status
-- reportCount/tableCount: pre-computed during graph build
RETURN 
  n.type as type,
  n.guid as guid,
  n.name as name,
  COALESCE(n.updated_parity_status, n.parity_status, 'No Status') as status,
  n.parity_team as team,
  n.inherited_priority_level as priority,
  n.formula as formula,
  COALESCE(n.lineage_used_by_reports_count, 0) as reportCount,
  COALESCE(n.lineage_source_tables_count, 0) as tableCount
LIMIT 100  -- Safety cap; typical use is 1-10 GUIDs
```

**Traversal Rule Compliance:**

| Rule | Status | Notes |
|------|--------|-------|
| No variable-length paths | ✅ N/A | Direct property access only |
| Uses pre-computed lineage | ✅ Yes | `lineage_used_by_reports_count`, `lineage_source_tables_count` |
| Respects countedTypes | ✅ Yes | Only returns Metric/Attribute |

**Output size:** ~250 chars per object × 100 max = ~25KB

**Response Structure:**

```typescript
interface GetObjectDetailsResponse {
  type: "Metric" | "Attribute";
  guid: string;              // Full 32-char GUID
  name: string;              // Display name
  status: string;            // "Complete" | "Planned" | "Not Planned" | "No Status"
  team: string | null;       // Responsible team
  priority: number | null;   // Inherited priority (1-5)
  formula: string | null;    // Metric formula (null for Attributes)
  reportCount: number;       // Pre-computed count of reports using this
  tableCount: number;        // Pre-computed count of source tables
}
```

**Example Response:**

```json
[
  {
    "type": "Metric",
    "guid": "2F00974D44E1D0D24CA344ABD872806A",
    "name": "Retail Sales Value",
    "status": "Complete",
    "team": "Finance",
    "priority": 1,
    "formula": "Sum(Sales Fact.sale_value)",
    "reportCount": 6549,
    "tableCount": 12
  },
  {
    "type": "Attribute",
    "guid": "BC105EDE477D7CEF3296FFA6E4D26797",
    "name": "Product Category",
    "status": "Planned",
    "team": "Trade",
    "priority": 2,
    "formula": null,
    "reportCount": 81,
    "tableCount": 3
  }
]
```

---

### Query 2: SearchObjectsQuery (Optimized)

**Tools:** `search-metrics`, `search-attributes`

**Original Problem:** No row limit, returns all matches.

**Optimized:** Paginated results (100 per page) with hasMore indicator.

#### Usage

Search for Metrics or Attributes by name, filtered by status, team, or other properties. Results are paginated — **call again with incremented offset to retrieve more results**.

#### Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `type` | array | Yes | — | Object types: `["Metric"]`, `["Attribute"]`, or `["Metric", "Attribute"]` |
| `search` | string | No | null | Search term (case-insensitive, matches name contains) |
| `status` | array | No | null | Filter by parity status: `["Complete", "Planned"]` |
| `offset` | number | No | 0 | Skip first N results for pagination |

#### Pagination

Results are limited to **100 per page**. To retrieve all matching objects:

1. First call: `offset=0` → returns up to 100 results
2. If `moreResults=true`: call with `offset=100` → returns next page
3. Continue incrementing offset by 100 until `moreResults=false`

#### Query

```cypher
-- Search for Metrics/Attributes by type, name, and status
-- $type: required array, e.g. ["Metric"] or ["Metric", "Attribute"]
-- $search: optional, case-insensitive name contains
-- $status: optional, filter by parity status
-- $offset: pagination offset (0, 100, 200, ...)
MATCH (n:MSTRObject)
WHERE n.type IN $type
  AND n.guid IS NOT NULL
  AND ($search IS NULL OR toLower(n.name) CONTAINS toLower($search))
  AND ($status IS NULL OR n.parity_status IN $status)

-- Compute derived fields
-- effectiveStatus: prefer updated_parity_status over original
-- reportCount: pre-computed count of reports using this M/A
WITH n,
     COALESCE(n.updated_parity_status, n.parity_status, 'No Status') as effectiveStatus,
     COALESCE(n.lineage_used_by_reports_count, 0) as reportCount

-- Order by usage (most-used first), then alphabetically
ORDER BY reportCount DESC, n.name ASC
SKIP $offset
LIMIT 101  -- Fetch 101 to determine if more results exist

-- Collect results into array for pagination check
WITH collect({
  type: n.type,
  name: n.name,
  guid: n.guid,
  status: effectiveStatus,
  priority: n.inherited_priority_level,
  team: n.parity_team,
  reports: reportCount,
  tables: COALESCE(n.lineage_source_tables_count, 0)
}) as fetched

-- Return first 100; moreResults=true if 101st exists
RETURN 
  fetched[0..100] as results,
  size(fetched) > 100 as moreResults
```

**Traversal Rule Compliance:**

| Rule | Status | Notes |
|------|--------|-------|
| No variable-length paths | ✅ N/A | Direct property access only |
| Uses pre-computed lineage | ✅ Yes | `lineage_used_by_reports_count`, `lineage_source_tables_count` |
| Respects countedTypes | ✅ Yes | Only returns Metric/Attribute |

**Output:** 100 results per page (~15KB); `moreResults` indicates if additional pages exist

**Response Structure:**

```typescript
interface SearchObjectsResult {
  results: SearchObject[];   // Array of 0-100 objects
  moreResults: boolean;      // true = call again with offset+100; false = last page
}

interface SearchObject {
  type: "Metric" | "Attribute";
  name: string;
  guid: string;
  status: string;
  priority: number | null;
  team: string | null;
  reports: number;           // Count of reports using this
  tables: number;            // Count of source tables
}
```

**Example Response:**

```json
{
  "results": [
    {
      "type": "Metric",
      "name": "Retail Sales Value",
      "guid": "2F00974D44E1D0D24CA344ABD872806A",
      "status": "Complete",
      "priority": 1,
      "team": "Finance",
      "reports": 6549,
      "tables": 12
    },
    {
      "type": "Metric",
      "name": "Units Sold",
      "guid": "416AEF98418213C610991A810D9E0C05",
      "status": "Planned",
      "priority": 2,
      "team": "Trade",
      "reports": 234,
      "tables": 5
    }
  ],
  "moreResults": true
}
```

**Pagination Example:**
```
// Page 1: offset=0
→ returns 100 results, moreResults=true

// Page 2: offset=100
→ returns 100 results, moreResults=true

// Page 3: offset=200
→ returns 47 results, moreResults=false (last page)
```

---

### Query 3: ReportsUsingObjectsQuery (Optimized)

**Tools:** `get-reports-using-metric`, `get-reports-using-attribute`

**Original Problem:** Returns all reports, can be thousands (e.g., "Retail Sales Value" has 6,000+ reports).

**Optimized:** Paginated (100 per page) using pre-computed lineage arrays.

#### Usage

Find all reports that use a given Metric or Attribute. Uses pre-computed `lineage_used_by_reports` array for fast lookup. **Call multiple times with incremented offset to retrieve all reports.**

#### Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `guids` | array | Yes | — | Array of M/A GUIDs to query |
| `offset` | number | No | 0 | Skip first N reports for pagination |

#### Pagination

Reports are limited to **100 per page**. High-usage objects may have thousands of reports:

1. First call: `offset=0` → returns up to 100 reports
2. If `moreResults=true`: call with `offset=100` → returns next page
3. Continue incrementing offset by 100 until `moreResults=false`

#### Query

```cypher
-- Find reports using a Metric or Attribute
-- Uses PRE-COMPUTED lineage_used_by_reports array (not runtime BFS)
-- Pre-computation traversed only through [Prompt, Filter] per doc-90
MATCH (n:MSTRObject)
WHERE n.guid IN $guids

-- Get pre-computed report GUIDs array
WITH n, COALESCE(n.lineage_used_by_reports, []) as reportGuids

-- Slice 101 items to check if more exist (pagination trick)
-- $offset: 0, 100, 200, ... for pagination
WITH n, reportGuids[$offset..($offset + 101)] as slicedGuids

-- Resolve first 100 GUIDs to full report objects
UNWIND slicedGuids[0..100] as reportGuid
MATCH (r:MSTRObject {guid: reportGuid})

-- Collect report details
WITH n, size(slicedGuids) as fetchedCount, collect({
       name: r.name,
       guid: r.guid,
       type: r.type,        -- Report, GridReport, or Document
       priority: r.priority_level,
       area: r.usage_area,
       department: r.usage_department,
       users: r.usage_users_count
     }) as reports

-- Return results; moreResults=true if 101st exists
RETURN 
  n.name as objectName,
  n.guid as objectGUID,
  n.type as objectType,
  reports,
  fetchedCount > 100 as moreResults
```

**Traversal Rule Compliance:**

| Rule | Status | Notes |
|------|--------|-------|
| Uses pre-computed lineage | ✅ Yes | `lineage_used_by_reports` pre-computed with correct BFS |
| Respects traversalTypes | ✅ Inherited | Pre-computation traversed only through [Prompt, Filter] |
| Respects reportTypes | ✅ Yes | Filters to [Report, GridReport, Document] |
| No runtime BFS | ✅ Yes | Uses array lookup, not variable-length path |

**Why Pre-computed:** The `lineage_used_by_reports` array was computed during graph building using symmetric BFS that only traverses through `[Prompt, Filter]` (per doc-90). Runtime queries were timing out (>30s) for high-connectivity objects.

**Output:** 100 reports per page (~15KB); `moreResults` indicates if more pages exist

**Response Structure:**

```typescript
interface ReportsUsingResponse {
  objectName: string;        // Name of the queried M/A
  objectGUID: string;        // GUID of the queried M/A
  objectType: string;        // "Metric" or "Attribute"
  reports: Report[];         // Array of 0-100 reports for current page
  moreResults: boolean;      // true = call again with offset+100; false = last page
}

interface Report {
  name: string;
  guid: string;
  type: "Report" | "GridReport" | "Document";
  priority: number | null;   // 1-5 or null
  area: string | null;       // "Finance", "Trade", etc.
  department: string | null;
  users: number | null;      // User count
}
```

**Example Response:**

```json
{
  "objectName": "Retail Sales Value",
  "objectGUID": "2F00974D44E1D0D24CA344ABD872806A",
  "objectType": "Metric",
  "reports": [
    {
      "name": "Daily Sales Dashboard",
      "guid": "ABC123DE456789F0ABC123DE456789F0",
      "type": "Report",
      "priority": 1,
      "area": "Finance",
      "department": "FP&A",
      "users": 150
    },
    {
      "name": "Weekly Trade Summary",
      "guid": "DEF456AB789012C3DEF456AB789012C3",
      "type": "GridReport",
      "priority": 2,
      "area": "Trade",
      "department": "Merchandising",
      "users": 45
    }
  ],
  "moreResults": true
}
```

---

### Query 4: SourceTablesQuery (Optimized)

**Tools:** `get-metric-source-tables`, `get-attribute-source-tables`

**Original Problem:** Returns all tables, verbose format.

**Optimized:** Paginated (100 per page) using pre-computed lineage arrays.

#### Usage

Find all source tables (LogicalTable/Table) that a Metric or Attribute ultimately depends on. Uses pre-computed `lineage_source_tables` array for fast lookup. **Call multiple times with incremented offset to retrieve all tables.**

#### Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `guids` | array | Yes | — | Array of M/A GUIDs to query |
| `offset` | number | No | 0 | Skip first N tables for pagination |

#### Pagination

Tables are limited to **100 per page**. Most objects have <100 tables, but some may have more:

1. First call: `offset=0` → returns up to 100 tables
2. If `moreResults=true`: call with `offset=100` → returns next page
3. Continue incrementing offset by 100 until `moreResults=false`

#### Query

```cypher
-- Find source tables for a Metric or Attribute
-- Uses PRE-COMPUTED lineage_source_tables array (not runtime BFS)
-- Pre-computation traversed through [Fact, Metric, Attribute, Column] per doc-90
MATCH (n:MSTRObject)
WHERE n.guid IN $guids

-- Get pre-computed table GUIDs array
WITH n, COALESCE(n.lineage_source_tables, []) as tableGuids

-- Slice 101 items to check if more exist (pagination trick)
-- $offset: 0, 100, 200, ... for pagination
WITH n, tableGuids[$offset..($offset + 101)] as slicedGuids

-- Resolve first 100 GUIDs to full table objects
UNWIND slicedGuids[0..100] as tableGuid
MATCH (t:MSTRObject {guid: tableGuid})

-- Collect table details including physical schema info
WITH n, size(slicedGuids) as fetchedCount, collect({
       name: t.name,
       guid: t.guid,
       type: t.type,        -- LogicalTable or Table
       physicalTable: t.physical_table_name,
       database: t.database_instance
     }) as tables

-- Return results; moreResults=true if 101st exists
RETURN 
  n.name as objectName,
  n.guid as objectGUID,
  n.type as objectType,
  tables,
  fetchedCount > 100 as moreResults
```

**Traversal Rule Compliance:**

| Rule | Status | Notes |
|------|--------|-------|
| Uses pre-computed lineage | ✅ Yes | `lineage_source_tables` pre-computed with correct BFS |
| Respects outbound types | ✅ Inherited | Pre-computation traversed through [Fact, Metric, Attribute, Column] |
| Respects tableTypes | ✅ Yes | Filters to [LogicalTable, Table] |
| No runtime BFS | ✅ Yes | Uses array lookup, not variable-length path |

**Why Pre-computed:** The `lineage_source_tables` array was computed during graph building using BFS that traverses through `[Fact, Metric, Attribute, Column]` to reach `[LogicalTable, Table]` (per doc-90 DD-5). Runtime traversal was slow for complex metrics.

**Output:** 100 tables per page (~10KB); `moreResults` indicates if more pages exist

**Response Structure:**

```typescript
interface SourceTablesResponse {
  objectName: string;        // Name of the queried M/A
  objectGUID: string;        // GUID of the queried M/A
  objectType: string;        // "Metric" or "Attribute"
  tables: Table[];           // Array of 0-100 tables for current page
  moreResults: boolean;      // true = call again with offset+100; false = last page
}

interface Table {
  name: string;              // Logical table name
  guid: string;
  type: "LogicalTable" | "Table";
  physicalTable: string | null;   // Physical table name in DB
  database: string | null;        // Database instance
}
```

**Example Response:**

```json
{
  "objectName": "Retail Sales Value",
  "objectGUID": "2F00974D44E1D0D24CA344ABD872806A",
  "objectType": "Metric",
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
  "moreResults": false
}
```

---

### Query 5: DownstreamDependenciesQuery (Optimized)

**Tools:** `get-metric-dependencies`, `get-attribute-dependencies`

**Original Problem:** Returns full graph paths — can be megabytes.

**Optimized:** Direct dependencies paginated (100 per page) + transitive table count.

#### Usage

Get what a Metric or Attribute directly depends on (Facts, other Metrics, Attributes, Columns), plus a count of transitively-reachable tables. **Call multiple times with incremented offset to retrieve all direct dependencies.**

#### Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `guids` | array | Yes | — | Array of M/A GUIDs to query |
| `offset` | number | No | 0 | Skip first N direct dependencies for pagination |

#### Pagination

Direct dependencies are limited to **100 per page**. Complex metrics may have many:

1. First call: `offset=0` → returns up to 100 dependencies
2. If `moreResults=true`: call with `offset=100` → returns next page
3. Continue incrementing offset by 100 until `moreResults=false`

**Note:** `transitiveTableCount` is always the full count (not paginated).

#### Query

```cypher
-- Find downstream dependencies for a Metric or Attribute
-- Returns: direct dependencies (1-hop) + transitive table count
MATCH (n:MSTRObject)
WHERE n.guid IN $guids

-- Get ALL direct dependencies (1-hop, no path filter needed)
-- Types: Fact, Metric, Attribute, Column, etc.
OPTIONAL MATCH (n)-[:DEPENDS_ON]->(direct:MSTRObject)
WITH n, collect(DISTINCT {
       type: direct.type, 
       name: direct.name, 
       guid: direct.guid,
       formula: direct.formula
     }) as allDirectDeps

-- Get transitive table count using RUNTIME BFS
-- TRAVERSAL RULE: Only traverse through [Fact, Metric, Attribute, Column]
-- TARGET RULE: Only count [LogicalTable, Table]
-- This filter prevents paths through DerivedMetric/Transformation
OPTIONAL MATCH path = (n)-[:DEPENDS_ON*2..10]->(t:MSTRObject)
WHERE t.type IN ['LogicalTable', 'Table']
  AND ALL(mid IN nodes(path)[1..-1] WHERE mid.type IN ['Fact', 'Metric', 'Attribute', 'Column'])
WITH n, allDirectDeps, count(DISTINCT t) as transitiveTableCount

-- Slice 101 to check if more exist (pagination trick)
-- $offset: 0, 100, 200, ... for pagination
WITH n, allDirectDeps, transitiveTableCount,
     allDirectDeps[$offset..($offset + 101)] as slicedDeps

-- Return results; moreResults=true if 101st exists
-- transitiveTableCount should match precomputedTableCount (validation)
RETURN 
  n.name as objectName,
  n.guid as objectGUID,
  n.type as objectType,
  transitiveTableCount,
  slicedDeps[0..100] as directDependencies,
  n.lineage_source_tables_count as precomputedTableCount,
  size(slicedDeps) > 100 as moreResults
```

**Traversal Rule Compliance:**

| Rule | Status | Notes |
|------|--------|-------|
| Direct deps (1-hop) | ✅ N/A | No path filter needed for single hop |
| Transitive tables | ✅ Yes | Uses `ALL(mid... IN ['Fact', 'Metric', 'Attribute', 'Column'])` |
| Target tableTypes | ✅ Yes | Filters to `['LogicalTable', 'Table']` |
| Max depth | ✅ Yes | Limited to 10 hops (`*2..10`) |

**Why Transitive Has Path Filter:** The transitive table query needs the `ALL(mid...)` filter because it uses variable-length paths. Without it, paths through DerivedMetric or Transformation would inflate counts.

**Output:** 100 direct deps per page (~15KB); `moreResults` indicates if more pages exist

**Response Structure:**

```typescript
interface DownstreamDependenciesResponse {
  objectName: string;
  objectGUID: string;
  objectType: string;
  transitiveTableCount: number;      // Tables reachable via valid paths (full count)
  precomputedTableCount: number;     // Should match (validation)
  directDependencies: Dependency[];  // Array of 0-100 deps for current page
  moreResults: boolean;              // true = call again with offset+100; false = last page
}

interface Dependency {
  type: string;              // "Fact", "Metric", "Attribute", "Column", etc.
  name: string;
  guid: string;
  formula: string | null;    // For metrics
}
```

**Example Response:**

```json
{
  "objectName": "Retail Sales Value",
  "objectGUID": "2F00974D44E1D0D24CA344ABD872806A",
  "objectType": "Metric",
  "transitiveTableCount": 12,
  "precomputedTableCount": 12,
  "directDependencies": [
    {
      "type": "Fact",
      "name": "Sales Fact",
      "guid": "FACT123456789ABCFACT123456789ABC",
      "formula": null
    },
    {
      "type": "Metric",
      "name": "Base Sales",
      "guid": "MET456789ABCDEF0MET456789ABCDEF0",
      "formula": "Sum(sale_value)"
    },
    {
      "type": "Attribute",
      "name": "Product",
      "guid": "ATTR789ABCDEF012ATTR789ABCDEF012",
      "formula": null
    }
  ],
  "moreResults": false
}
```

---

### Query 6: UpstreamDependenciesQuery (Optimized)

**Tools:** `get-metric-dependents`, `get-attribute-dependents`

**Original Problem:** Returns up to 1000 paths — massive output.

**Optimized:** Paginated (100 per page) using pre-computed lineage arrays.

#### Usage

Find all reports (and other objects) that depend on a given Metric or Attribute. Uses pre-computed `lineage_used_by_reports` array for fast lookup. **Call multiple times with incremented offset to retrieve all dependents.**

**Note:** This query is functionally similar to Query 3 (ReportsUsingObjects) — both return reports that use an M/A. The difference is semantic: "upstream" emphasizes the direction of dependency flow.

#### Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `guids` | array | Yes | — | Array of M/A GUIDs to query |
| `offset` | number | No | 0 | Skip first N reports for pagination |

#### Pagination

Reports are limited to **100 per page**. High-usage objects may have thousands:

1. First call: `offset=0` → returns up to 100 reports
2. If `moreResults=true`: call with `offset=100` → returns next page
3. Continue incrementing offset by 100 until `moreResults=false`

#### Query

```cypher
-- Find reports that depend on a Metric or Attribute (upstream)
-- Uses PRE-COMPUTED lineage_used_by_reports array (not runtime BFS)
-- Pre-computation traversed only through [Prompt, Filter] per doc-90
-- Functionally same as Query 3 (ReportsUsing); different semantic framing
MATCH (n:MSTRObject)
WHERE n.guid IN $guids

-- Get pre-computed report GUIDs array
WITH n, COALESCE(n.lineage_used_by_reports, []) as reportGuids

-- Slice 101 items to check if more exist (pagination trick)
-- $offset: 0, 100, 200, ... for pagination
WITH n, reportGuids[$offset..($offset + 101)] as slicedGuids

-- Resolve first 100 GUIDs to full report objects
UNWIND slicedGuids[0..100] as reportGuid
MATCH (r:MSTRObject {guid: reportGuid})

-- Collect report details
WITH n, size(slicedGuids) as fetchedCount, collect({
       name: r.name,
       guid: r.guid,
       type: r.type,        -- Report, GridReport, or Document
       priority: r.priority_level,
       area: r.usage_area,
       department: r.usage_department,
       users: r.usage_users_count
     }) as reports

-- Return results; moreResults=true if 101st exists
RETURN 
  n.name as objectName,
  n.guid as objectGUID,
  n.type as objectType,
  reports,
  fetchedCount > 100 as moreResults
```

**Traversal Rule Compliance:**

| Rule | Status | Notes |
|------|--------|-------|
| Uses pre-computed lineage | ✅ Yes | `lineage_used_by_reports` pre-computed correctly |
| Respects traversalTypes | ✅ Inherited | Pre-computation traversed only through [Prompt, Filter] |
| Respects reportTypes | ✅ Yes | Filters to [Report, GridReport, Document] |
| No runtime BFS | ✅ Yes | Uses array lookup, not variable-length path |

**Why Pre-computed:** Using pre-computed `lineage_used_by_reports` instead of runtime BFS because runtime queries were timing out (>30s) for high-connectivity objects in production.

**Output:** 100 reports per page (~15KB); `moreResults` indicates if more pages exist

**Response Structure:**

```typescript
interface UpstreamDependenciesResponse {
  objectName: string;
  objectGUID: string;
  objectType: string;
  reports: Report[];         // Array of 0-100 reports for current page
  moreResults: boolean;      // true = call again with offset+100; false = last page
}

interface Report {
  name: string;
  guid: string;
  type: "Report" | "GridReport" | "Document";
  priority: number | null;
  area: string | null;
  department: string | null;
  users: number | null;
}
```

**Example Response:**

```json
{
  "objectName": "Product Category",
  "objectGUID": "BC105EDE477D7CEF3296FFA6E4D26797",
  "objectType": "Attribute",
  "reports": [
    {
      "name": "Category Performance",
      "guid": "RPT123456789ABCRPT123456789ABC",
      "type": "Report",
      "priority": 1,
      "area": "Trade",
      "department": "Category Management",
      "users": 89
    },
    {
      "name": "Product Mix Analysis",
      "guid": "RPT456789ABCDEFRPT456789ABCDEF",
      "type": "GridReport",
      "priority": 2,
      "area": "Finance",
      "department": "Planning",
      "users": 34
    }
  ],
  "moreResults": true
}
```

---

### Output Size Comparison

| Query | Original | Optimized | Per Page |
|-------|----------|-----------|----------|
| GetObjectDetails | 5KB/object | 250 chars/object | 100 objects (~25KB) |
| SearchObjects | Unbounded | Paginated | 100 results (~15KB) |
| ReportsUsing | 6,000+ rows | Paginated | 100 reports (~15KB) |
| SourceTables | Unbounded | Paginated | 100 tables (~10KB) |
| Downstream | 100KB+ paths | Paginated + summary | 100 deps (~15KB) |
| Upstream | 500KB+ paths | Paginated + summary | 100 reports (~15KB) |

---

### Traversal Rule Compliance Summary

| Query | Uses Pre-computed | Runtime BFS | Path Filter | Status |
|-------|-------------------|-------------|-------------|--------|
| GetObjectDetails | ✅ Yes (counts) | ❌ No | N/A | ✅ Compliant |
| SearchObjects | ✅ Yes (counts) | ❌ No | N/A | ✅ Compliant |
| ReportsUsing | ✅ Yes (array) | ❌ No | Inherited | ✅ Compliant |
| SourceTables | ✅ Yes (array) | ❌ No | Inherited | ✅ Compliant |
| Downstream (direct) | ❌ No | ✅ Yes (1-hop) | N/A (1-hop) | ✅ Compliant |
| Downstream (trans) | ❌ No | ✅ Yes | ✅ ALL(mid...) | ✅ Compliant |
| Upstream | ✅ Yes (array) | ❌ No | Inherited | ✅ Compliant |
| **Runtime: ReportsUsing** | ❌ No | ✅ Yes | ✅ `[Prompt, Filter]` | ✅ Compliant |
| **Runtime: SourceTables** | ❌ No | ✅ Yes | ✅ `[Fact, Metric, Attr, Col]` | ✅ Compliant |
| **Runtime: Upstream** | ❌ No | ✅ Yes | ✅ `[Prompt, Filter]` | ✅ Compliant |

**Key Insight:** Most queries avoid runtime BFS by using pre-computed lineage arrays, which were computed with correct traversal rules at graph-build time. Only the Downstream transitive query uses runtime BFS, and it properly includes the `ALL(mid IN nodes(path)...)` filter.

---

## Runtime Alternatives (No Pre-computed Lineage)

The pre-computed lineage arrays (`lineage_used_by_reports`, `lineage_source_tables`) are fast but may become stale if the graph changes. These runtime alternatives traverse the graph directly, ensuring fresh results at the cost of performance.

**When to use runtime queries:**
- After graph modifications before re-computation
- When validating pre-computed results
- For ad-hoc analysis on modified subgraphs

### Runtime Query: ReportsUsingObject

**Replaces:** `ReportsUsingObjectsQuery` (pre-computed version)

**Traversal Rule:** Inbound traversal through `[Prompt, Filter]` only (per doc-90)

```cypher
-- ═══════════════════════════════════════════════════════════════════════════
-- COMPONENT 1: Find Target M/A Objects
-- ═══════════════════════════════════════════════════════════════════════════
MATCH (n:MSTRObject)
WHERE n.guid IN $guids

-- ═══════════════════════════════════════════════════════════════════════════
-- COMPONENT 2: Traverse Inbound to Reports (RESPECTS TRAVERSAL RULES)
-- ═══════════════════════════════════════════════════════════════════════════
OPTIONAL MATCH path = (r:MSTRObject)-[:DEPENDS_ON*1..10]->(n)
WHERE r.type IN ['Report', 'GridReport', 'Document']
  AND ALL(mid IN nodes(path)[1..-1] WHERE mid.type IN ['Prompt', 'Filter'])
-- TRAVERSAL RULE: Only traverse through [Prompt, Filter] per doc-90
-- TARGET RULE: Only capture [Report, GridReport, Document]

WITH n, collect(DISTINCT r) as allReports

-- ═══════════════════════════════════════════════════════════════════════════
-- COMPONENT 3: Paginate and Return
-- ═══════════════════════════════════════════════════════════════════════════
WITH n, allReports, size(allReports) as totalReports
UNWIND allReports[$offset..($offset + 100)] as r

RETURN 
  n.name as objectName,
  n.guid as objectGUID,
  n.type as objectType,
  totalReports,
  collect({
    name: r.name,
    guid: r.guid,
    type: r.type,
    priority: r.priority_level,
    area: r.usage_area,
    department: r.usage_department,
    users: r.usage_users_count
  }) as reports
```

**Path Filter Explanation:**
- `nodes(path)[1..-1]` = all intermediate nodes (excludes start `r` and end `n`)
- `mid.type IN ['Prompt', 'Filter']` = only these types can appear on path
- Reports connected directly to M/A are included (path length 1, no intermediates)
- Reports connected via Prompt/Filter chains are included
- Reports connected via other object types (e.g., DerivedMetric) are EXCLUDED

**Performance:** Slower than pre-computed (~100-500ms vs ~10ms)

---

### Runtime Query: SourceTables

**Replaces:** `SourceTablesQuery` (pre-computed version)

**Traversal Rule:** Outbound traversal through `[Fact, Metric, Attribute, Column]` only (per doc-90)

```cypher
-- ═══════════════════════════════════════════════════════════════════════════
-- COMPONENT 1: Find Target M/A Objects
-- ═══════════════════════════════════════════════════════════════════════════
MATCH (n:MSTRObject)
WHERE n.guid IN $guids

-- ═══════════════════════════════════════════════════════════════════════════
-- COMPONENT 2: Traverse Outbound to Tables (RESPECTS TRAVERSAL RULES)
-- ═══════════════════════════════════════════════════════════════════════════
OPTIONAL MATCH path = (n)-[:DEPENDS_ON*1..10]->(t:MSTRObject)
WHERE t.type IN ['LogicalTable', 'Table']
  AND ALL(mid IN nodes(path)[1..-1] WHERE mid.type IN ['Fact', 'Metric', 'Attribute', 'Column'])
-- TRAVERSAL RULE: Only traverse through [Fact, Metric, Attribute, Column] per doc-90
-- TARGET RULE: Only capture [LogicalTable, Table]

WITH n, collect(DISTINCT t) as allTables

-- ═══════════════════════════════════════════════════════════════════════════
-- COMPONENT 3: Paginate and Return
-- ═══════════════════════════════════════════════════════════════════════════
WITH n, allTables, size(allTables) as totalTables
UNWIND allTables[$offset..($offset + 100)] as t

RETURN 
  n.name as objectName,
  n.guid as objectGUID,
  n.type as objectType,
  totalTables,
  collect({
    name: t.name,
    guid: t.guid,
    type: t.type,
    physicalTable: t.physical_table_name,
    database: t.database_instance
  }) as tables
```

**Why DerivedMetric/Transformation Are Excluded:**
Per doc-90, these types are NOT in the outbound traversal set:
- `DerivedMetric` → Would double-count tables already reached via base metrics
- `Transformation` → Would attribute source tables to wrong metrics

**Performance:** Slower than pre-computed (~50-200ms vs ~10ms)

---

### Runtime Query: UpstreamReports

**Replaces:** `UpstreamDependenciesQuery` (pre-computed version)

**Note:** Identical to `ReportsUsingObject` — both find reports that depend on an M/A.

```cypher
-- Same as Runtime Query: ReportsUsingObject
MATCH (n:MSTRObject)
WHERE n.guid IN $guids

OPTIONAL MATCH path = (r:MSTRObject)-[:DEPENDS_ON*1..10]->(n)
WHERE r.type IN ['Report', 'GridReport', 'Document']
  AND ALL(mid IN nodes(path)[1..-1] WHERE mid.type IN ['Prompt', 'Filter'])

WITH n, collect(DISTINCT r) as allReports
WITH n, allReports, size(allReports) as totalReports
UNWIND allReports[$offset..($offset + 100)] as r

RETURN 
  n.name as objectName,
  n.guid as objectGUID,
  n.type as objectType,
  totalReports,
  collect({
    name: r.name,
    guid: r.guid,
    type: r.type,
    priority: r.priority_level,
    area: r.usage_area,
    department: r.usage_department,
    users: r.usage_users_count
  }) as reports
```

---

### Runtime vs Pre-computed Comparison

| Aspect | Pre-computed | Runtime |
|--------|--------------|---------|
| **Speed** | ~10ms | ~50-500ms |
| **Freshness** | Stale if graph changed | Always current |
| **Memory** | Stored on nodes | Computed on demand |
| **Use case** | Production queries | Validation, ad-hoc |

**Recommendation:** Use pre-computed for production; use runtime for validation or after graph modifications.

---

## Statistics Query Specifications

Statistics tools return aggregates only — no individual records. Call these first to understand dataset size, then fetch details as needed.

### Query: MetricsStatsQuery

**Tool:** `get-metrics-stats`

**Purpose:** Get counts of metrics by status, priority, team, domain.

```cypher
-- ═══════════════════════════════════════════════════════════════════════════
-- COMPONENT 1: Optional Filters
-- ═══════════════════════════════════════════════════════════════════════════
WITH CASE WHEN $status IS NULL OR size($status) = 0 
          THEN null ELSE $status END as filterStatus,
     CASE WHEN $team IS NULL OR $team = '' 
          THEN null ELSE $team END as filterTeam

-- ═══════════════════════════════════════════════════════════════════════════
-- COMPONENT 2: Base Match
-- ═══════════════════════════════════════════════════════════════════════════
MATCH (n:Metric)
WHERE n.guid IS NOT NULL

-- ═══════════════════════════════════════════════════════════════════════════
-- COMPONENT 3: Apply Filters
-- ═══════════════════════════════════════════════════════════════════════════
WITH n, filterStatus, filterTeam,
     COALESCE(n.updated_parity_status, n.parity_status, 'No Status') as status
WHERE (filterStatus IS NULL OR status IN filterStatus)
  AND (filterTeam IS NULL OR n.parity_team = filterTeam)

-- ═══════════════════════════════════════════════════════════════════════════
-- COMPONENT 4: Aggregate Statistics
-- ═══════════════════════════════════════════════════════════════════════════
RETURN 
  count(*) as total,
  count(CASE WHEN status = 'Complete' THEN 1 END) as complete,
  count(CASE WHEN status = 'Planned' THEN 1 END) as planned,
  count(CASE WHEN status = 'Not Planned' THEN 1 END) as notPlanned,
  count(CASE WHEN status = 'No Status' THEN 1 END) as noStatus,
  count(CASE WHEN n.inherited_priority_level IS NOT NULL THEN 1 END) as prioritized,
  collect(DISTINCT n.parity_team) as teams
```

**Output:** Single row with counts (~200 bytes)

**Response Structure:**

```typescript
interface MetricsStatsResponse {
  total: number;
  complete: number;
  planned: number;
  notPlanned: number;
  noStatus: number;
  prioritized: number;       // Count with priority assigned
  teams: string[];           // Distinct team names
}
```

**Example Response:**

```json
{
  "total": 2847,
  "complete": 892,
  "planned": 1205,
  "notPlanned": 450,
  "noStatus": 300,
  "prioritized": 1652,
  "teams": ["Finance", "Trade", "Supply Chain", "ExecApp"]
}
```

---

### Query: ObjectStatsQuery

**Tool:** `get-object-stats`

**Purpose:** Get summary statistics for a specific object (report counts, table counts, priority distribution of dependents).

```cypher
-- ═══════════════════════════════════════════════════════════════════════════
-- COMPONENT 1: Find Object
-- ═══════════════════════════════════════════════════════════════════════════
MATCH (n:MSTRObject {guid: $guid})

-- ═══════════════════════════════════════════════════════════════════════════
-- COMPONENT 2: Return Pre-computed Stats
-- ═══════════════════════════════════════════════════════════════════════════
WITH n,
     COALESCE(n.lineage_used_by_reports_count, 0) as reportCount,
     COALESCE(n.lineage_source_tables_count, 0) as tableCount,
     COALESCE(n.lineage_used_by_reports, []) as reportGuids

-- ═══════════════════════════════════════════════════════════════════════════
-- COMPONENT 3: Aggregate Report Priorities
-- ═══════════════════════════════════════════════════════════════════════════
CALL {
  WITH reportGuids
  UNWIND reportGuids as rg
  MATCH (r:MSTRObject {guid: rg})
  WHERE r.priority_level IS NOT NULL
  RETURN r.priority_level as priority, count(*) as cnt
  ORDER BY priority
}

-- ═══════════════════════════════════════════════════════════════════════════
-- COMPONENT 4: Return Summary
-- ═══════════════════════════════════════════════════════════════════════════
RETURN 
  n.name as name,
  n.type as type,
  n.guid as guid,
  COALESCE(n.updated_parity_status, n.parity_status, 'No Status') as status,
  n.parity_team as team,
  reportCount,
  tableCount,
  collect({priority: priority, count: cnt}) as reportsByPriority
```

**Output:** Single object summary with aggregates (~500 bytes)

**Response Structure:**

```typescript
interface ObjectStatsResponse {
  name: string;
  type: "Metric" | "Attribute";
  guid: string;
  status: string;
  team: string | null;
  reportCount: number;
  tableCount: number;
  reportsByPriority: PriorityCount[];
}

interface PriorityCount {
  priority: number;
  count: number;
}
```

**Example Response:**

```json
{
  "name": "Retail Sales Value",
  "type": "Metric",
  "guid": "2F00974D44E1D0D24CA344ABD872806A",
  "status": "Complete",
  "team": "Finance",
  "reportCount": 6549,
  "tableCount": 12,
  "reportsByPriority": [
    { "priority": 1, "count": 23 },
    { "priority": 2, "count": 156 },
    { "priority": 3, "count": 892 },
    { "priority": 4, "count": 2341 },
    { "priority": 5, "count": 3137 }
  ]
}
```

---

### Statistics + Detail Workflow

**Recommended pattern for agents:**

```
Step 1: Get statistics (understand scope)
────────────────────────────────────────
Agent calls: get-metrics-stats(status=["Not Planned"])
Returns: {total: 450, teams: ["Finance", "Supply Chain", ...]}

Step 2: Get details (as needed)
────────────────────────────────────────
Agent calls: search-metrics(status=["Not Planned"], limit=100, offset=0)
Returns: [100 metrics]

Agent calls: search-metrics(status=["Not Planned"], limit=100, offset=100)
Returns: [100 metrics]

... continues until agent has enough data or reaches total
```

**Benefits:**
- Statistics query is fast (single aggregation)
- Detail queries don't need to count total
- Agent controls how much data to fetch

---

## Full Dependency Tree Strategies

While summary queries are optimal for most LLM interactions, some use cases require the **complete dependency tree** with full object details. This section documents strategies for returning full trees without overflowing context.

### Use Cases Requiring Full Trees

| Use Case | Why Full Tree Needed |
|----------|----------------------|
| **Gap Analysis** | Compare every MSTR object against ADE schema |
| **DBT Model Generation** | Need all source tables, columns, joins |
| **Impact Analysis** | Must trace all affected objects before changes |
| **Formula Translation** | Convert complete MSTR formula dependencies to SQL |
| **Parity Validation** | Validate every metric/attribute mapping |

### Strategy 1: Hierarchical Pagination

Return the tree level-by-level, allowing the agent to drill down as needed.

```cypher
-- Level 0: Root object summary
MATCH (n:MSTRObject {guid: $guid})
RETURN n.name, n.type, n.guid,
       size((n)-[:DEPENDS_ON]->()) as directDependencies,
       size((n)<-[:DEPENDS_ON]-()) as directDependents

-- Level 1: Direct dependencies (on-demand)
MATCH (n:MSTRObject {guid: $guid})-[:DEPENDS_ON]->(d)
RETURN d.name, d.type, left(d.guid, 8) as guid,
       size((d)-[:DEPENDS_ON]->()) as childCount
ORDER BY childCount DESC
LIMIT 50

-- Level 2+: Expand specific node (on-demand)
MATCH (n:MSTRObject {guid: $childGuid})-[:DEPENDS_ON]->(d)
RETURN d.name, d.type, left(d.guid, 8) as guid,
       size((d)-[:DEPENDS_ON]->()) as childCount
LIMIT 50
```

**Pros:** Agent controls depth, never overflows  
**Cons:** Multiple round-trips

### Strategy 2: Compressed Tree Format

Return the full tree in a condensed single-line-per-node format.

```cypher
MATCH (n:MSTRObject {guid: $guid})
OPTIONAL MATCH path = (n)-[:DEPENDS_ON*1..5]->(d)
WHERE ALL(mid IN nodes(path)[1..-1] WHERE mid.type IN ['Fact', 'Metric', 'Attribute', 'Column'])
WITH n, collect(DISTINCT {
  depth: length(path),
  type: d.type,
  name: d.name,
  guid: left(d.guid, 8)
}) as deps

// Format as compact tree
UNWIND deps as dep
WITH dep ORDER BY dep.depth, dep.type, dep.name
RETURN collect(
  repeat('  ', dep.depth) + dep.type[0] + ':' + dep.name + '(' + dep.guid + ')'
) as tree
```

**Output format (one line per node):**
```
M:Retail Sales Value(ABC123)
  F:Sales Fact(DEF456)
    C:sale_value(GHI789)
      T:FACT_SALES(JKL012)
  A:Product(MNO345)
    C:product_id(PQR678)
```

**Estimate:** ~50 chars per node × 100 nodes = 5KB

### Strategy 3: Chunked Streaming

For very large trees, return in chunks with continuation tokens.

```cypher
// First call - returns chunk 1 and continuation info
MATCH (n:MSTRObject {guid: $guid})-[:DEPENDS_ON*1..10]->(d)
WITH DISTINCT d
ORDER BY d.guid
WITH collect(d)[0..100] as chunk, count(*) as total
RETURN {
  chunk: [x IN chunk | {name: x.name, type: x.type, guid: x.guid}],
  total: total,
  hasMore: total > 100,
  nextCursor: CASE WHEN total > 100 THEN chunk[99].guid ELSE null END
}

// Subsequent calls - use cursor
MATCH (n:MSTRObject {guid: $guid})-[:DEPENDS_ON*1..10]->(d)
WHERE d.guid > $cursor
WITH DISTINCT d
ORDER BY d.guid
WITH collect(d)[0..100] as chunk
RETURN {
  chunk: [x IN chunk | {name: x.name, type: x.type, guid: x.guid}],
  hasMore: size(chunk) = 100,
  nextCursor: CASE WHEN size(chunk) = 100 THEN chunk[99].guid ELSE null END
}
```

### Strategy 4: Type-Filtered Trees

Return only specific node types relevant to the task.

```cypher
-- For DBT generation: only need tables and columns
MATCH (n:MSTRObject {guid: $guid})-[:DEPENDS_ON*1..10]->(d)
WHERE d.type IN ['LogicalTable', 'Table', 'Column']
RETURN DISTINCT d.type, d.name, d.guid,
       d.physical_table_name, d.database_instance
ORDER BY d.type, d.name

-- For formula analysis: only metrics and facts
MATCH (n:MSTRObject {guid: $guid})-[:DEPENDS_ON*1..10]->(d)
WHERE d.type IN ['Metric', 'Fact', 'DerivedMetric']
RETURN DISTINCT d.type, d.name, d.guid, d.formula
ORDER BY d.type, d.name
```

### Recommended Approach by Use Case

| Use Case | Strategy | Max Output |
|----------|----------|------------|
| Quick lookup | Summary (default) | 2KB |
| Interactive exploration | Hierarchical pagination | 5KB/level |
| Gap analysis report | Compressed tree | 10KB |
| DBT generation | Type-filtered | 8KB |
| Full audit | Chunked streaming | 10KB/chunk |

---

## DB Engineer Pain Points Analysis

**Source:** Data Transformation Team Work Items Analysis (Jan 2026, 59 work items, 143 comments)

This section documents frequently reported problems from DB engineers and how MCP tools should address them.

### Top Pain Point Categories

| Category | % of Work | Items | Description |
|----------|-----------|-------|-------------|
| **Data Gap Analysis** | 25% | 15 | Comparing MSTR vs ADE, semantic model review |
| **Dimension/Attribute Addition** | 20% | 12 | Adding new dimensions to serve layer |
| **Metric Implementation** | 17% | 10 | Tax calculations, parity metrics validation |
| **Migration Tasks** | 14% | 8 | Move tables to serve layer, epic tracking |
| **Technical Spikes** | 10% | 6 | Reprocessing strategies, schema drift |
| **Semantic Model Updates** | 8% | 5 | Add metrics to PBI models |

### Pain Point → MCP Tool Mapping

#### 1. Gap Analysis (25% of work)

**Problem:** Manual comparison of MSTR metrics/attributes against ADE schema is time-consuming and error-prone.

**Required MCP Capabilities:**
- Get all metrics/attributes in a domain
- Return formula definitions and source tables
- Compare against external schema (ADE)

**Recommended Tools:**
| Tool | Purpose |
|------|---------|
| `search-metrics` | Find all metrics in a domain |
| `get-metric-source-tables` | Get source table mappings |
| `get-metric-dependencies` | Full dependency tree for gap identification |

**Output Requirements:**
- Must return formula text for comparison
- Must include EDW/ADE table mappings
- Should flag unmapped objects

#### 2. Metric Formula Translation (17% of work)

**Problem:** Engineers need to understand MSTR formulas and translate them to DBT SQL.

**Required MCP Capabilities:**
- Return complete formula with all nested dependencies
- Show calculation logic hierarchy
- Include base Facts and their columns

**Recommended New Tool:** `get-metric-formula-tree`

```cypher
// Returns formula with full dependency context
MATCH (m:Metric {guid: $guid})
OPTIONAL MATCH (m)-[:DEPENDS_ON*1..5]->(dep)
WHERE dep.type IN ['Metric', 'Fact', 'Attribute', 'Column']
RETURN {
  metric: m.name,
  formula: m.formula,
  dependencies: collect(DISTINCT {
    type: dep.type,
    name: dep.name,
    formula: dep.formula,
    column: dep.physical_column_name,
    table: dep.physical_table_name
  })
}
```

#### 3. Dimension Addition (20% of work)

**Problem:** Adding new dimensions requires understanding existing schema, relationships, and DBT patterns.

**Required MCP Capabilities:**
- Get attribute definition with all forms
- Return physical table/column mappings
- Show related dimensions (same table)

**Recommended Tools:**
| Tool | Purpose |
|------|---------|
| `get-attribute-by-guid` | Full attribute definition |
| `get-attribute-source-tables` | Physical table mappings |
| `get-attribute-dependencies` | Related columns and joins |

**Output Requirements:**
- Must include `forms_json` for attribute forms
- Must return physical column names
- Should include join relationships

#### 4. Parity Validation (implicit in 17%)

**Problem:** Validating data parity between ADE and MSTR/PBI models requires comparing aggregations.

**Required MCP Capabilities:**
- Get parity status for objects
- Return mapping details (EDW → ADE → PBI)
- Identify discrepancies

**Recommended Enhancement:** Add parity comparison fields to all object queries:
- `parity_status` (Complete, Planned, Not Planned)
- `edw_table`, `edw_column`
- `ade_db_table`, `ade_db_column`
- `pb_semantic_name`, `pb_semantic_model`

### Priority Improvements for MCP Tools

| Priority | Improvement | Pain Points Addressed | Impact |
|----------|-------------|----------------------|--------|
| **P0** | Add `formula` field to metric queries | Metric Implementation | 17% |
| **P0** | Full dependency tree with pagination | Gap Analysis | 25% |
| **P1** | Add `forms_json` to attribute queries | Dimension Addition | 20% |
| **P1** | Include physical table/column names | All | 62% |
| **P2** | Domain filtering for bulk queries | Gap Analysis | 25% |
| **P2** | Parity comparison fields | Parity Validation | 17% |

### Example: Gap Analysis Workflow

How an agent would use MCP tools for a typical gap analysis task:

```
1. search-metrics(domain="Finance", status="Not Planned")
   → Returns 50 unmapped metrics with GUIDs

2. For each metric:
   get-metric-by-guid(guid)
   → Returns formula, EDW mapping, parity status

3. get-metric-dependencies(guid)
   → Returns full dependency tree with tables

4. Compare against ADE schema
   → Identify missing tables/columns

5. Generate gap report with mappings
```

---

## User Question Mapping

### Questions Answered by Existing Tools

| User Question | Recommended Tool |
|---------------|------------------|
| "What are the details of metric X?" | `get-metric-by-guid` |
| "What is the parity status of attribute Y?" | `get-attribute-by-guid` |
| "Which metrics are related to 'revenue'?" | `search-metrics` |
| "Find attributes with status 'Not Started'" | `search-attributes` |
| "Which reports use metric Z?" | `get-reports-using-metric` |
| "Which reports use attribute W?" | `get-reports-using-attribute` |
| "Which tables feed metric X?" | `get-metric-source-tables` |
| "What are the source tables for attribute Y?" | `get-attribute-source-tables` |
| "What does metric X depend on?" | `get-metric-dependencies` |
| "What is the calculation chain for attribute Y?" | `get-attribute-dependencies` |
| "What will be affected if I change metric X?" | `get-metric-dependents` |
| "Which objects depend on attribute Y?" | `get-attribute-dependents` |
| "Which P1 metrics exist in Finance area?" | `search-metrics` (with filters) |
| "List attributes in 'Sales' domain with 'In Progress' status" | `search-attributes` (with filters) |

### Questions NOT Yet Answered (Future Enhancements)

#### High Priority

| Question | Suggested Implementation |
|----------|--------------------------|
| "What is the full formula/definition of metric X?" | New query returning `formula`, `expressions_json`, `raw_json` |
| "Which metrics use attribute Y in their formula?" | Reverse dependency traversal specific to formulas |
| "What is the Power BI equivalent mapping for metric X?" | Enrich `GetObjectDetailsQuery` with additional PB fields |
| "Show the complete dependency graph for metric X" | Graph visualization with configurable depth |
| "Which Facts are used by metric X?" | Specific traversal for Facts |
| "Compare two metrics (X and Y) - differences" | New comparison tool |

#### Medium Priority

| Question | Suggested Implementation |
|----------|--------------------------|
| "Which metrics are not mapped to Power BI?" | Query with filter `pb_semantic IS NULL` |
| "List all metrics for a specific Team" | Add Team filter to `search-metrics` |
| "Which reports are most critical (most users)?" | New query ordering by `usage_users_count` |
| "Which EDW tables are most used?" | Aggregation by EDW table |
| "Show orphan metrics (no dependents)" | Query identifying objects without upstream |
| "What is the migration coverage by area?" | Status aggregation by `usage_area` |

#### Low Priority

| Question | Suggested Implementation |
|----------|--------------------------|
| "History of status changes for a metric" | Requires audit fields in graph |
| "Which metrics were updated this week?" | Requires timestamp fields |
| "Suggest migration order based on dependencies" | Topological algorithm on graph |

---

## Database Schema Reference

### Node Labels

| Label | Description |
|-------|-------------|
| `MSTRObject` | Generic label for all MicroStrategy objects |
| `Metric` | Metrics (also has MSTRObject label) |
| `Attribute` | Attributes (also has MSTRObject label) |
| `Fact` | Facts |
| `LogicalTable` | Logical tables |
| `Report` | Reports (type in MSTRObject) |
| `GridReport` | Grid Reports (type in MSTRObject) |
| `Document` | Documents (type in MSTRObject) |
| `Filter` | Filters (type in MSTRObject) |
| `Prompt` | Prompts (type in MSTRObject) |
| `Column` | Columns |
| `DataProduct` | Data domains/products |

### Relationships

| Relationship | Description |
|--------------|-------------|
| `DEPENDS_ON` | Dependency relation: `(A)-[:DEPENDS_ON]->(B)` means A depends on B |
| `BELONGS_TO` | Membership in data domain/product |

### Key Properties

#### On MSTRObject/Metric/Attribute

| Property | Description |
|----------|-------------|
| `guid` | Unique identifier |
| `name` | Object name |
| `type` | Type ('Metric', 'Attribute', 'Report', etc.) |
| `parity_status` | Original parity status |
| `updated_parity_status` | Updated parity status (takes precedence) |
| `parity_group` | Parity group |
| `parity_subgroup` | Parity subgroup |
| `parity_team` | Responsible team |
| `parity_notes` | Parity notes |
| `inherited_priority_level` | Inherited priority level |

**Data Mapping Properties:**

| Property | Description |
|----------|-------------|
| `db_raw` | Databricks RAW |
| `db_serve` | Databricks SERVE |
| `pb_semantic` | Power BI Semantic |
| `edw_table` | EDW Table |
| `edw_column` | EDW Column |
| `ade_db_table` | ADE Table |
| `ade_db_column` | ADE Column |
| `pb_semantic_name` | Name in PB semantic model |
| `pb_semantic_model` | PB semantic model |
| `db_essential` | Databricks Essential |
| `pb_essential` | Power BI Essential |

**Lineage Properties (arrays of GUIDs):**

| Property | Description |
|----------|-------------|
| `lineage_source_tables` | Source table GUIDs |
| `lineage_source_tables_count` | Count of source tables |
| `lineage_used_by_reports` | Report GUIDs that use this object |

#### On Metric

| Property | Description |
|----------|-------------|
| `formula` | Metric formula (text) |
| `expressions_json` | Expressions in JSON |
| `raw_json` | Original complete JSON |
| `location` | Location in project |

#### On Attribute

| Property | Description |
|----------|-------------|
| `forms_json` | Attribute forms in JSON |
| `location` | Location in project |

#### On Report/GridReport/Document

| Property | Description |
|----------|-------------|
| `priority_level` | Priority level (1, 2, 3, etc.) |
| `usage_area` | Usage/business area |
| `usage_department` | Department |
| `usage_users_count` | User count |
| `usage_consistency` | Usage consistency |
| `usage_volume` | Usage volume |

#### On LogicalTable

| Property | Description |
|----------|-------------|
| `physical_table_name` | Physical table name |
| `database_instance` | Database instance |

---

## Optimization Guidelines

### Query Performance Notes

1. **`SearchObjectsQuery`** — Most complex query
   - Uses CALL subqueries for aggregation
   - Multiple optional filters with CASE WHEN
   - EXISTS subqueries for relationship filters
   - **Optimization:** Indexes on `type`, `guid`, `priority_level`, `usage_area`

2. **`DownstreamDependenciesQuery` / `UpstreamDependenciesQuery`**
   - Variable traversal 1..10 levels
   - ALL() predicate on intermediate nodes
   - **Optimization:** Limit depth, use apoc.path if available

3. **`ReportsUsingObjectsQuery` / `SourceTablesQuery`**
   - Depend on pre-computed arrays (`lineage_used_by_reports`, `lineage_source_tables`)
   - **Advantage:** Pre-computed arrays accelerate lookups
   - **Disadvantage:** Requires array integrity maintenance

### Recommended Indexes

```cypher
-- Primary indexes
CREATE INDEX IF NOT EXISTS FOR (n:MSTRObject) ON (n.guid);
CREATE INDEX IF NOT EXISTS FOR (n:MSTRObject) ON (n.type);
CREATE INDEX IF NOT EXISTS FOR (n:MSTRObject) ON (n.priority_level);
CREATE INDEX IF NOT EXISTS FOR (n:MSTRObject) ON (n.usage_area);
CREATE INDEX IF NOT EXISTS FOR (n:Metric) ON (n.guid);
CREATE INDEX IF NOT EXISTS FOR (n:Attribute) ON (n.guid);
CREATE INDEX IF NOT EXISTS FOR (n:DataProduct) ON (n.name);

-- Composite index for search
CREATE INDEX IF NOT EXISTS FOR (n:MSTRObject) ON (n.type, n.guid);
```

### Parameter Naming Convention

All queries use consistent, LLM-friendly parameter names:

| Parameter | Type | Description |
|-----------|------|-------------|
| `guids` | array | Target object GUIDs |
| `search` | string | Search term (name contains) |
| `type` | array | Object type filter (e.g., `["Metric"]` or `["Metric", "Attribute"]`) |
| `priority` | array | Priority level filter (e.g., `["P1", "P2"]`) |
| `area` | array | Business area filter |
| `status` | array | Parity status filter |
| `domain` | array | Data domain filter |
| `team` | string | Team name filter |
| `offset` | number | Skip first N results for pagination (default: 0) |

**Note:** All queries return 100 results per page. Use `offset` to paginate (0, 100, 200, ...). Use statistics tools for total counts.

---

## Query Testing Protocol

### Testing Optimized Queries

Before deploying optimized queries, verify they:
1. Return correct data (semantic equivalence)
2. Stay within size limits
3. Maintain acceptable performance

### Test Cases

| Test | Query | Validation |
|------|-------|------------|
| **Single object** | GetObjectDetails with 1 GUID | Output < 500 chars |
| **Multi object** | GetObjectDetails with 5 GUIDs | Output < 2KB |
| **Search empty** | SearchObjects with no matches | Returns `{totalCount: 0, results: []}` |
| **Search many** | SearchObjects for "sales" | Returns ≤50 results + totalCount |
| **High-report object** | ReportsUsing for object with 1000+ reports | Summary + 15 samples < 3KB |
| **High-table object** | SourceTables for object with 50+ tables | Summary + 25 tables < 2KB |
| **Complex lineage** | Downstream for deeply nested metric | Counts + 10 samples < 3KB |
| **Popular object** | Upstream for widely-used attribute | Priority summary + samples < 2KB |

### Sample Test GUIDs

From validation data (90-symmetric-bfs-traversal.md):

| Scenario | GUID | Reports | Tables |
|----------|------|---------|--------|
| Zero counts | `9B89CC834B7EF00714F9A893753A656C` | 0 | 0 |
| Low counts | `416AEF98418213C610991A810D9E0C05` | 5 | 2 |
| Medium counts | `D0365E2C4E48FCB35CC0FFA8E4999121` | 13 | 13 |
| High counts | `BC105EDE477D7CEF3296FFA6E4D26797` | 81 | 44 |
| Very high | `2F00974D44E1D0D24CA344ABD872806A` | 6549 | — |

### Size Measurement

Run queries and measure output:

```bash
# Measure query output size
echo "MATCH (n:Metric {guid: 'ABC123'}) RETURN n" | \
  cypher-shell -u neo4j -p password | wc -c
```

Target: All optimized queries should return **< 10KB** for typical use cases.

---

## Implementation Guidelines

### New Query Structure Pattern

```cypher
-- 1. Process parameters with CASE WHEN
WITH CASE WHEN $param IS NULL THEN default ELSE processed_value END as paramName

-- 2. Initial MATCH with basic filters
MATCH (n:Label)
WHERE n.property = value

-- 3. Conditional filters
WHERE (filterVar IS NULL OR n.property IN filterVar)

-- 4. Aggregations in CALL subqueries
CALL {
  WITH n
  MATCH pattern
  RETURN aggregated_result
}

-- 5. RETURN with standardized fields
RETURN 
  n.type as Type,
  n.name as Name,
  n.guid as GUID
ORDER BY relevantField DESC
```

### New Query Checklist

- [ ] Use consistent parameter names (guids, search, type, status, limit, offset)
- [ ] Handle NULL/empty for all optional parameters
- [ ] Use `effectiveStatus` pattern for parity status
- [ ] Limit traversal results (e.g., `[0..1000]`)
- [ ] Include GUID in results for drill-down capability
- [ ] Order results meaningfully
- [ ] Test with real GUIDs before implementation

### Adding a New Tool

1. Create file in `internal/tools/mstr/tool_name.go`
2. Add query to `internal/tools/mstr/queries.go`
3. Register in `internal/server/tools_register.go`
4. Add to `manifest.json`

---

## Server Reference Files

| File | Description |
|------|-------------|
| `internal/tools/mstr/queries.go` | All Cypher queries |
| `internal/tools/mstr/*.go` | Tool implementations |
| `internal/server/tools_register.go` | Tool server registration |
| `manifest.json` | MCP manifest with tool list |
| `queries/01.cypher` | Original NeoDash queries |
| `queries/neo4j-query-templates.md` | Additional query templates |

---

## Changelog

| Date | Version | Change |
|------|---------|--------|
| 2026-01-30 | 2.3.0 | Added consistent verbose comments to all 6 optimized queries for human validation |
| 2026-01-30 | 2.2.0 | Removed legacy query section (verbose, non-optimized queries); Document now contains only optimized queries |
| 2026-01-30 | 2.1.0 | Removed total counts from detail queries (use stats tools); Changed `hasMore` to `moreResults`; Uses "LIMIT 101" trick to determine pagination without counting |
| 2026-01-30 | 2.0.0 | Added Usage/Parameters/Pagination sections to all queries; Documented pre-computed lineage history (testing → production timeout fix) |
| 2026-01-30 | 1.9.0 | Added runtime alternatives for ReportsUsing, SourceTables, Upstream that don't rely on pre-computed lineage arrays |
| 2026-01-30 | 1.8.0 | Restored `$offset` pagination (fixed 100 per page); Removed input sanitization for LLM agent use |
| 2026-01-30 | 1.7.0 | Simplified pagination: fixed 100 results per page |
| 2026-01-30 | 1.6.0 | Added TypeScript response structures and JSON examples for all 9 queries |
| 2026-01-30 | 1.5.0 | Renamed all parameters to LLM-friendly names; Consistent naming: `guids`, `search`, `type`, `status`, `priority`, `area`, `domain`, `team` |
| 2026-01-30 | 1.4.0 | **Design decision:** Separated detail tools from statistics tools; Detail queries use efficient SKIP/LIMIT or array slicing without total counts; Added 3 statistics tools (MetricsStats, AttributesStats, ObjectStats) |
| 2026-01-30 | 1.3.0 | Increased pagination defaults (50→100, max 200); Added line-by-line component breakdowns for all 6 queries; Added traversal rule compliance verification table |
| 2026-01-30 | 1.2.0 | Added full dependency tree strategies (pagination, compression, streaming); DB engineer pain points analysis from Data Transformation team (59 work items); Priority improvements mapping |
| 2026-01-30 | 1.1.0 | Added LLM context optimization focus, optimized queries, traversal rules, testing protocol |
| 2026-01-30 | 1.0.0 | Initial document with 12 MSTR tools |
| 2026-01-24 | — | Lineage arrays now contain pure GUIDs (no formatting) |
