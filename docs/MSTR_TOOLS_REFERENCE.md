# MicroStrategy MCP Tools - Reference for the Cypher Team

This document describes the existing MCP tools for querying MicroStrategy metadata, their corresponding Cypher queries, and relevant information for creating and optimising new queries.

## Table of Contents

1. [Tools Overview](#tools-overview)
2. [Tools and Detailed Queries](#tools-and-detailed-queries)
3. [User Questions Mapping](#user-questions-mapping)
4. [Questions Not Yet Answered](#questions-not-yet-answered)
5. [Database Schema](#database-schema)
6. [Optimisation Considerations](#optimisation-considerations)
7. [Guidelines for New Queries](#guidelines-for-new-queries)

---

## Tools Overview

The system has **12 MicroStrategy tools** organised into categories:

### GUID Queries
| Tool | Description | Query Used |
|------|-------------|------------|
| `get-metric-by-guid` | Details of a Metric by GUID | `GetObjectDetailsQuery` |
| `get-attribute-by-guid` | Details of an Attribute by GUID | `GetObjectDetailsQuery` |

### Search with Filters
| Tool | Description | Query Used |
|------|-------------|------------|
| `search-metrics` | Search Metrics with filters | `SearchObjectsQuery` |
| `search-attributes` | Search Attributes with filters | `SearchObjectsQuery` |

### Reports/Dependents
| Tool | Description | Query Used |
|------|-------------|------------|
| `get-reports-using-metric` | Reports that use a Metric | `ReportsUsingObjectsQuery` |
| `get-reports-using-attribute` | Reports that use an Attribute | `ReportsUsingObjectsQuery` |

### Source Tables (Lineage)
| Tool | Description | Query Used |
|------|-------------|------------|
| `get-metric-source-tables` | Source tables of a Metric | `SourceTablesQuery` |
| `get-attribute-source-tables` | Source tables of an Attribute | `SourceTablesQuery` |

### Downstream Dependencies (what it depends on)
| Tool | Description | Query Used |
|------|-------------|------------|
| `get-metric-dependencies` | What the Metric depends on | `DownstreamDependenciesQuery` |
| `get-attribute-dependencies` | What the Attribute depends on | `DownstreamDependenciesQuery` |

### Upstream Dependents (what depends on it)
| Tool | Description | Query Used |
|------|-------------|------------|
| `get-metric-dependents` | What depends on the Metric | `UpstreamDependenciesQuery` |
| `get-attribute-dependents` | What depends on the Attribute | `UpstreamDependenciesQuery` |

---

## Tools and Detailed Queries

### Query 1: `GetObjectDetailsQuery`

**Tools that use it:** `get-metric-by-guid`, `get-attribute-by-guid`

**Parameters:**
- `neodash_selected_guid` (array of strings): Object GUIDs (supports prefix matching with STARTS WITH)

**Returns:** Type, GUID, Name, Status, Group, SubGroup, Team, RAW, SERVE, SEMANTIC, EDWTable, EDWColumn, ADETable, ADEColumn, SemanticName, SemanticModel, DBEssential, PBEssential, Notes

```cypher
WITH $neodash_selected_guid as selectedGuids
WHERE selectedGuids IS NOT NULL AND size(selectedGuids) > 0
MATCH (n:Metric)
WHERE any(g IN selectedGuids WHERE n.guid STARTS WITH g)
RETURN 
  'Metric' as Type,
  n.guid as GUID,
  n.name as Name,
  CASE WHEN n.updated_parity_status IS NOT NULL AND n.updated_parity_status <> '' 
       THEN n.updated_parity_status ELSE n.parity_status END as Status,
  n.parity_group as Group,
  n.parity_subgroup as SubGroup,
  n.parity_team as Team,
  n.db_raw as RAW,
  n.db_serve as SERVE,
  n.pb_semantic as SEMANTIC,
  n.edw_table as EDWTable,
  n.edw_column as EDWColumn,
  n.ade_db_table as ADETable,
  n.ade_db_column as ADEColumn,
  n.pb_semantic_name as SemanticName,
  n.pb_semantic_model as SemanticModel,
  n.db_essential as DBEssential,
  n.pb_essential as PBEssential,
  n.parity_notes as Notes
UNION
WITH $neodash_selected_guid as selectedGuids
WHERE selectedGuids IS NOT NULL AND size(selectedGuids) > 0
MATCH (n:Attribute)
WHERE any(g IN selectedGuids WHERE n.guid STARTS WITH g)
RETURN 
  'Attribute' as Type,
  n.guid as GUID,
  n.name as Name,
  CASE WHEN n.updated_parity_status IS NOT NULL AND n.updated_parity_status <> '' 
       THEN n.updated_parity_status ELSE n.parity_status END as Status,
  n.parity_group as Group,
  n.parity_subgroup as SubGroup,
  n.parity_team as Team,
  n.db_raw as RAW,
  n.db_serve as SERVE,
  n.pb_semantic as SEMANTIC,
  n.edw_table as EDWTable,
  n.edw_column as EDWColumn,
  n.ade_db_table as ADETable,
  n.ade_db_column as ADEColumn,
  n.pb_semantic_name as SemanticName,
  n.pb_semantic_model as SemanticModel,
  n.db_essential as DBEssential,
  n.pb_essential as PBEssential,
  n.parity_notes as Notes
```

---

### Query 2: `SearchObjectsQuery`

**Tools that use it:** `search-metrics`, `search-attributes`

**Parameters:**
- `neodash_searchterm` (string): Search terms separated by comma
- `neodash_objecttype` (string): "Metric", "Attribute" or "All Types"
- `neodash_priority_level` (array): Priority levels such as "P1 (Highest)", "P2", etc.
- `neodash_business_area` (array): Business areas
- `neodash_status` (array): Parity status values
- `neodash_data_domain` (array): Data domains

**Returns:** Type, Priority, Name, Status, Team, Reports (count), Tables (count), GUID

```cypher
WITH CASE WHEN coalesce($neodash_searchterm, '') = '' THEN null ELSE [term IN split($neodash_searchterm, ',') | toLower(trim(term))] END as searchTerms,
     CASE WHEN coalesce($neodash_objecttype, '') = '' OR $neodash_objecttype = 'All Types' THEN ['Metric', 'Attribute'] ELSE [$neodash_objecttype] END as typeFilter,
     CASE WHEN $neodash_priority_level IS NULL OR size($neodash_priority_level) = 0 OR 'All Prioritized' IN $neodash_priority_level THEN null ELSE [p IN $neodash_priority_level | toInteger(replace(replace(replace(p, 'P', ''), ' (Highest)', ''), ' (Lowest)', ''))] END as priorityLevelFilter,
     CASE WHEN $neodash_business_area IS NULL OR size($neodash_business_area) = 0 OR 'All Areas' IN $neodash_business_area THEN null ELSE $neodash_business_area END as businessAreaFilter,
     CASE WHEN $neodash_status IS NULL OR size($neodash_status) = 0 OR 'All Status' IN $neodash_status THEN null ELSE $neodash_status END as filterStatusList,
     CASE WHEN $neodash_data_domain IS NULL OR size($neodash_data_domain) = 0 OR 'All Domains' IN $neodash_data_domain THEN null ELSE $neodash_data_domain END as dataDomainFilter
MATCH (n:MSTRObject)
WHERE n.type IN typeFilter
  AND n.guid IS NOT NULL
  AND n.inherited_priority_level IS NOT NULL
  AND (searchTerms IS NULL OR any(term IN searchTerms WHERE toLower(n.name) CONTAINS term OR toLower(n.guid) CONTAINS term))
  AND (dataDomainFilter IS NULL OR ALL(domain IN dataDomainFilter WHERE EXISTS { MATCH (dp:DataProduct {name: domain})-[:BELONGS_TO]->(n) }))
WITH n, priorityLevelFilter, businessAreaFilter, filterStatusList,
     CASE WHEN n.updated_parity_status IS NOT NULL AND n.updated_parity_status <> '' 
          THEN n.updated_parity_status ELSE n.parity_status END as effectiveStatus
WHERE (filterStatusList IS NULL OR effectiveStatus IN filterStatusList)
  AND (businessAreaFilter IS NULL OR ALL(ba IN businessAreaFilter WHERE EXISTS { MATCH (r2:MSTRObject)-[:DEPENDS_ON]->(n) WHERE r2.type IN ['Report', 'GridReport', 'Document'] AND r2.priority_level IS NOT NULL AND r2.usage_area = ba } OR EXISTS { MATCH (r2:MSTRObject)-[:DEPENDS_ON]->(fp:MSTRObject)-[:DEPENDS_ON]->(n) WHERE r2.type IN ['Report', 'GridReport', 'Document'] AND r2.priority_level IS NOT NULL AND fp.type IN ['Filter', 'Prompt'] AND r2.usage_area = ba }))
CALL {
  WITH n, priorityLevelFilter, businessAreaFilter, effectiveStatus
  MATCH (r:MSTRObject)-[:DEPENDS_ON]->(n)
  WHERE r.type IN ['Report', 'GridReport', 'Document']
    AND r.priority_level IS NOT NULL
    AND (priorityLevelFilter IS NULL OR r.priority_level IN priorityLevelFilter)
    AND (businessAreaFilter IS NULL OR r.usage_area IN businessAreaFilter)
  RETURN collect(DISTINCT r.guid) as directGuids
}
CALL {
  WITH n, priorityLevelFilter, businessAreaFilter, effectiveStatus
  MATCH (r:MSTRObject)-[:DEPENDS_ON]->(fp:MSTRObject)-[:DEPENDS_ON]->(n)
  WHERE r.type IN ['Report', 'GridReport', 'Document']
    AND r.priority_level IS NOT NULL
    AND fp.type IN ['Filter', 'Prompt']
    AND (priorityLevelFilter IS NULL OR r.priority_level IN priorityLevelFilter)
    AND (businessAreaFilter IS NULL OR r.usage_area IN businessAreaFilter)
  RETURN collect(DISTINCT r.guid) as indirectGuids
}
WITH n, effectiveStatus, directGuids + [g IN indirectGuids WHERE NOT g IN directGuids] as allReportGuids
WHERE size(allReportGuids) > 0
RETURN 
      n.type as Type,
      n.inherited_priority_level as Priority,
      n.name as Name,
      effectiveStatus as Status,
      n.parity_team as Team,
      size(allReportGuids) as Reports,
      COALESCE(n.lineage_source_tables_count, 0) as Tables,
      n.guid as GUID
ORDER BY Reports DESC
```

---

### Query 3: `ReportsUsingObjectsQuery`

**Tools that use it:** `get-reports-using-metric`, `get-reports-using-attribute`

**Parameters:**
- `neodash_selected_guid` (array of strings): Object GUIDs
- `neodash_priority_level` (array): Priority levels
- `neodash_business_area` (array): Business areas

**Returns:** Selected Item, Report Name, Priority, Area, Department, Users, Usage

```cypher
WITH $neodash_selected_guid as selectedGuids,
     CASE WHEN $neodash_priority_level IS NULL OR size($neodash_priority_level) = 0 OR 'All Prioritized' IN $neodash_priority_level THEN null ELSE [p IN $neodash_priority_level | toInteger(replace(replace(replace(p, 'P', ''), ' (Highest)', ''), ' (Lowest)', ''))] END as priorityLevelFilter,
     CASE WHEN $neodash_business_area IS NULL OR size($neodash_business_area) = 0 OR 'All Areas' IN $neodash_business_area THEN null ELSE $neodash_business_area END as businessAreaFilter
WHERE selectedGuids IS NOT NULL AND size(selectedGuids) > 0
MATCH (n:MSTRObject)
WHERE n.guid IN selectedGuids AND n.lineage_used_by_reports IS NOT NULL
WITH n, n.lineage_used_by_reports as reportGuids, priorityLevelFilter, businessAreaFilter
UNWIND reportGuids as reportGuid
MATCH (r:MSTRObject {guid: reportGuid})
WHERE r.type IN ['Report', 'GridReport', 'Document']
  AND r.priority_level IS NOT NULL
  AND (priorityLevelFilter IS NULL OR r.priority_level IN priorityLevelFilter)
  AND (businessAreaFilter IS NULL OR r.usage_area IN businessAreaFilter)
RETURN DISTINCT 
       n.name + ' (' + left(n.guid, 7) + ')' as `Selected Item`, 
       r.name + ' (' + left(r.guid, 7) + ')'  as `Report Name`,
       r.priority_level as `Priority`,
       r.usage_area as `Area`,
       r.usage_department as `Department`,
       r.usage_users_count as `Users`,
       r.usage_consistency + '|' + r.usage_volume as `Usage`
ORDER BY `Selected Item`, `Report Name`
```

---

### Query 4: `SourceTablesQuery`

**Tools that use it:** `get-metric-source-tables`, `get-attribute-source-tables`

**Parameters:**
- `neodash_selected_guid` (array of strings): Object GUIDs

**Returns:** Selected Item, Table Name, Table GUID

```cypher
WITH $neodash_selected_guid as selectedGuids
WHERE selectedGuids IS NOT NULL AND size(selectedGuids) > 0
MATCH (n:MSTRObject)
WHERE n.guid IN selectedGuids AND n.lineage_source_tables IS NOT NULL
WITH n, n.lineage_source_tables as tableGuids
UNWIND tableGuids as tableGuid
MATCH (t:MSTRObject {guid: tableGuid})
RETURN DISTINCT 
       n.name + ' (' + left(n.guid, 7) + ')' as `Selected Item`, 
       t.name as `Table Name`, 
       t.guid as `Table GUID`
ORDER BY `Selected Item`, `Table Name`
```

---

### Query 5: `DownstreamDependenciesQuery`

**Tools that use it:** `get-metric-dependencies`, `get-attribute-dependencies`

**Parameters:**
- `neodash_selected_guid` (array of strings): Object GUIDs

**Returns:** Original node (n) and dependency paths (downstream)

```cypher
WITH $neodash_selected_guid as selectedGuids
WHERE selectedGuids IS NOT NULL AND size(selectedGuids) > 0
MATCH (n:MSTRObject)
WHERE n.guid IN selectedGuids
OPTIONAL MATCH downstream = (n)-[:DEPENDS_ON*1..10]->(d:MSTRObject)
WHERE ALL(mid IN nodes(downstream)[1..-1] WHERE mid.type IN ['Fact', 'Metric', 'Attribute', 'Column'])
RETURN n, downstream
```

---

### Query 6: `UpstreamDependenciesQuery`

**Tools that use it:** `get-metric-dependents`, `get-attribute-dependents`

**Parameters:**
- `neodash_selected_guid` (array of strings): Object GUIDs

**Returns:** Original node (n) and upstream paths (upstream) - limited to 1000 paths

```cypher
WITH $neodash_selected_guid as selectedGuids
WHERE selectedGuids IS NOT NULL AND size(selectedGuids) > 0
MATCH (n:MSTRObject)
WHERE n.guid IN selectedGuids
OPTIONAL MATCH upstream = (r:MSTRObject)-[:DEPENDS_ON*1..10]->(n)
WHERE r.type IN ['Report', 'GridReport', 'Document']
WITH n, collect(upstream)[0..1000] as paths
UNWIND paths as upstream
RETURN n, upstream
```

---

## User Questions Mapping

### Questions ALREADY ANSWERED by Existing Tools

| User Question | Recommended Tool |
|---------------|------------------|
| "What are the details of metric X?" | `get-metric-by-guid` |
| "What is the parity status of attribute Y?" | `get-attribute-by-guid` |
| "Which metrics are related to 'revenue'?" | `search-metrics` |
| "Find attributes with status 'Not Started'" | `search-attributes` |
| "Which reports use metric Z?" | `get-reports-using-metric` |
| "Which reports use attribute W?" | `get-reports-using-attribute` |
| "Which tables does metric X feed from?" | `get-metric-source-tables` |
| "What are the source tables for attribute Y?" | `get-attribute-source-tables` |
| "What does metric X depend on?" | `get-metric-dependencies` |
| "What is the calculation chain for attribute Y?" | `get-attribute-dependencies` |
| "What will be affected if I change metric X?" | `get-metric-dependents` |
| "Which objects depend on attribute Y?" | `get-attribute-dependents` |
| "Which P1 metrics exist in the Finance area?" | `search-metrics` (with filters) |
| "List attributes from the 'Sales' domain with status 'In Progress'" | `search-attributes` (with filters) |

---

## Questions Not Yet Answered

### High Priority (Frequently Requested)

| Question | Implementation Suggestion |
|----------|---------------------------|
| "What is the complete formula/definition of metric X?" | New query returning `formula`, `expressions_json`, `raw_json` |
| "Which metrics use attribute Y in their formula?" | Specific reverse dependency traversal |
| "What is the Power BI equivalent mapping for metric X?" | Enrich `GetObjectDetailsQuery` with more PB fields |
| "Show the complete dependency graph for metric X" | Graph visualisation with configurable depth |
| "Which Facts are used by metric X?" | Specific traversal for Facts |
| "Compare two metrics (X and Y) - differences" | New comparison tool |

### Medium Priority

| Question | Implementation Suggestion |
|----------|---------------------------|
| "Which metrics are not mapped to Power BI?" | Query with filter `pb_semantic IS NULL` |
| "List all metrics for a specific Team" | Add Team filter to `search-metrics` |
| "Which reports are most critical (most users)?" | New query ordering by `usage_users_count` |
| "Which EDW tables are most used?" | Aggregation by EDW table |
| "Show orphan metrics (without dependents)" | Query identifying objects without upstream |
| "What is the migration coverage by area?" | Status aggregation by `usage_area` |

### Low Priority (Nice to Have)

| Question | Implementation Suggestion |
|----------|---------------------------|
| "Change history of a metric's status" | Requires audit fields in the graph |
| "Which metrics were updated this week?" | Requires timestamp fields |
| "Suggest migration order based on dependencies" | Topological algorithm over the graph |

---

## Database Schema

### Common Node Labels

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
| `DEPENDS_ON` | Dependency relationship (A)-[:DEPENDS_ON]->(B) means A depends on B |
| `BELONGS_TO` | Belonging to domain/data product |

### Important Properties

#### In MSTRObject/Metric/Attribute:
```
guid                      - Unique identifier
name                      - Object name
type                      - Type ('Metric', 'Attribute', 'Report', etc.)
parity_status            - Original parity status
updated_parity_status    - Updated parity status (takes precedence)
parity_group             - Parity group
parity_subgroup          - Parity subgroup
parity_team              - Responsible team
parity_notes             - Parity notes
inherited_priority_level - Inherited priority level

-- Data mappings --
db_raw                   - Databricks RAW
db_serve                 - Databricks SERVE
pb_semantic              - Power BI Semantic
edw_table                - EDW Table
edw_column               - EDW Column
ade_db_table             - ADE Table
ade_db_column            - ADE Column
pb_semantic_name         - Name in PB semantic model
pb_semantic_model        - PB semantic model
db_essential             - Databricks Essential
pb_essential             - Power BI Essential

-- Lineage (arrays of GUIDs) --
lineage_source_tables       - Source table GUIDs
lineage_source_tables_count - Source table count
lineage_used_by_reports     - GUIDs of reports that use the object
```

#### In Metric:
```
formula          - Metric formula (text)
expressions_json - Expressions in JSON
raw_json         - Complete original JSON
location         - Location in the project
```

#### In Attribute:
```
forms_json       - Attribute forms in JSON
location         - Location in the project
```

#### In Report/GridReport/Document:
```
priority_level      - Priority level (1, 2, 3, etc.)
usage_area          - Usage/business area
usage_department    - Department
usage_users_count   - User count
usage_consistency   - Usage consistency
usage_volume        - Usage volume
```

#### In LogicalTable:
```
physical_table_name - Physical table name
database_instance   - Database instance
```

---

## Optimisation Considerations

### Current Performance

1. **`SearchObjectsQuery`** - Most complex query
   - Uses CALL subqueries for aggregation
   - Multiple optional filters with CASE WHEN
   - EXISTS subqueries for relationship filters
   - **Potential optimisation:** Indices on `type`, `guid`, `priority_level`, `usage_area`

2. **`DownstreamDependenciesQuery` / `UpstreamDependenciesQuery`**
   - Variable traversal of 1..10 levels
   - ALL() predicate on intermediate nodes
   - **Potential optimisation:** Limit depth, use apoc.path if available

3. **`ReportsUsingObjectsQuery` / `SourceTablesQuery`**
   - Depend on pre-computed arrays (`lineage_used_by_reports`, `lineage_source_tables`)
   - **Advantage:** Pre-computed arrays speed up lookups
   - **Disadvantage:** Requires maintaining array integrity

### Recommended Indices

```cypher
-- Existing indices (verify):
CREATE INDEX IF NOT EXISTS FOR (n:MSTRObject) ON (n.guid);
CREATE INDEX IF NOT EXISTS FOR (n:MSTRObject) ON (n.type);
CREATE INDEX IF NOT EXISTS FOR (n:MSTRObject) ON (n.priority_level);
CREATE INDEX IF NOT EXISTS FOR (n:MSTRObject) ON (n.usage_area);
CREATE INDEX IF NOT EXISTS FOR (n:Metric) ON (n.guid);
CREATE INDEX IF NOT EXISTS FOR (n:Attribute) ON (n.guid);
CREATE INDEX IF NOT EXISTS FOR (n:DataProduct) ON (n.name);

-- Composite index for search:
CREATE INDEX IF NOT EXISTS FOR (n:MSTRObject) ON (n.type, n.guid);
```

### Parameter Usage Patterns

All queries use parameters with `neodash_` prefix (NeoDash compatibility):
- `neodash_selected_guid` - Array of selected GUIDs
- `neodash_searchterm` - Search term
- `neodash_objecttype` - Object type
- `neodash_priority_level` - Array of priority levels
- `neodash_business_area` - Array of business areas
- `neodash_status` - Array of statuses
- `neodash_data_domain` - Array of domains

---

## Guidelines for New Queries

### Structure Pattern

```cypher
-- 1. Parameter processing with CASE WHEN
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

-- 5. RETURN with standardised fields
RETURN 
  n.type as Type,
  n.name as Name,
  n.guid as GUID
ORDER BY relevantField DESC
```

### Checklist for New Queries

- [ ] Use parameters with `neodash_` prefix for compatibility
- [ ] Handle NULL/empty for all optional parameters
- [ ] Use `effectiveStatus` pattern for parity status
- [ ] Limit traversal results (e.g., `[0..1000]`)
- [ ] Include GUID in results to allow drill-down
- [ ] Order results meaningfully
- [ ] Test with real GUIDs before implementing

### Template for New Tool

1. Create file in `internal/tools/mstr/tool_name.go`
2. Add query in `internal/tools/mstr/queries.go`
3. Register in `internal/server/tools_register.go`
4. Add to `manifest.json`

---

## Reference Files

| File | Description |
|------|-------------|
| `internal/tools/mstr/queries.go` | All Cypher queries |
| `internal/tools/mstr/*.go` | Tool implementations |
| `internal/server/tools_register.go` | Tool registration in the server |
| `manifest.json` | MCP manifest with tool list |
| `queries/01.cypher` | Original NeoDash queries |
| `queries/neo4j-query-templates.md` | Additional query templates |

---

## Update History

| Date | Version | Change |
|------|---------|--------|
| 2026-01-30 | 1.0.0 | Initial document with 12 MSTR tools |
| 2026-01-24 | - | Lineage arrays now contain pure GUIDs (unformatted) |
