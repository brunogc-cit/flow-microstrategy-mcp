# MicroStrategy MCP Tools — Code Review Reference

This document reflects the **current code** and is intended to help review and test the MicroStrategy (MSTR) tools. It includes tool inputs, outputs, LLM-facing descriptions, queries, and practical notes.

Scope:
- MSTR tools registered in `internal/server/tools_register.go`
- Query definitions in `internal/tools/mstr/queries.go`
- GDS tool included as an appendix (not part of MSTR but shipped in the same server)

Conventions used across tools:
- **Pagination**: offset-based, page size 100. Many tools return a `moreResults` boolean.
- **JSON output**: the handlers return JSON generated from Neo4j records.
- **Read-only**: all tools in this document are read-only.
- **GUIDs**: input parameter is `guid` (exact match); output fields use full GUIDs.

---

## 1) `get-metric-by-guid`

**Inputs**
- `guid` (string, required): Full GUID of the Metric to retrieve. Exact match required.

**Outputs**
- Records with fields: `Type`, `GUID`, `Name`, `Status`, `Group`, `SubGroup`, `Team`, `Priority`, `Formula`, `RAW`, `SERVE`, `SEMANTIC`, `EDWTable`, `EDWColumn`, `ADETable`, `ADEColumn`, `SemanticName`, `SemanticModel`, `DBEssential`, `PBEssential`, `Notes`, `ReportCount`, `TableCount`

**LLM Description (as in code)**
> "Get comprehensive details about a MicroStrategy Metric by GUID. Returns 22 fields including: name, status, team, priority, formula, EDW/ADE mappings (edwTable, edwColumn, adeTable, adeColumn), Power BI mappings (semanticName, semanticModel), Databricks mappings (raw, serve), and pre-computed counts (reportCount, tableCount). Use for detailed object inspection and gap analysis. Limited to 100 results per call."

**Description review**
- **Pros**: clear field list; ties to gap analysis; highlights pagination limit.
- **Cons**: does not clarify multi-record behaviour if multiple GUIDs provided internally.
- **Possible improvement**: add a sentence explaining the returned payload is a JSON array of records, even for a single GUID.

**Query used (as in code)** — `GetObjectDetailsQuery`
```cypher
MATCH (n:Metric)
WHERE n.guid IN $guids
RETURN 
  'Metric' as Type,
  n.guid as GUID,
  n.name as Name,
  COALESCE(n.updated_parity_status, n.parity_status, 'No Status') as Status,
  n.parity_group as `Group`,
  n.parity_subgroup as SubGroup,
  n.parity_team as Team,
  n.inherited_priority_level as Priority,
  n.formula as Formula,
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
  n.parity_notes as Notes,
  COALESCE(n.lineage_used_by_reports_count, 0) as ReportCount,
  COALESCE(n.lineage_source_tables_count, 0) as TableCount
LIMIT 100
UNION ALL
MATCH (n:Attribute)
WHERE n.guid IN $guids
RETURN 
  'Attribute' as Type,
  n.guid as GUID,
  n.name as Name,
  COALESCE(n.updated_parity_status, n.parity_status, 'No Status') as Status,
  n.parity_group as `Group`,
  n.parity_subgroup as SubGroup,
  n.parity_team as Team,
  n.inherited_priority_level as Priority,
  n.formula as Formula,
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
  n.parity_notes as Notes,
  COALESCE(n.lineage_used_by_reports_count, 0) as ReportCount,
  COALESCE(n.lineage_source_tables_count, 0) as TableCount
LIMIT 100
```

**Query review**
- **Pros**: full 22-field detail; uses updated parity status fallback; includes pre-computed counts.
- **Cons**: returns Metrics and Attributes in the same query; callers must filter by `Type`.
- **Possible improvement**: consider a `Type` filter parameter for clarity.

**Other info**
- If no records are found, the handler returns a text message: `"No Metric found with the specified GUID."`

---

## 2) `get-attribute-by-guid`

**Inputs**
- `guid` (string, required): Full GUID of the Attribute to retrieve. Exact match required.

**Outputs**
- Same fields as `get-metric-by-guid` (see above).

**LLM Description (as in code)**
> "Get comprehensive details about a MicroStrategy Attribute by GUID. Returns 22 fields including: name, status, team, priority, formula, EDW/ADE mappings (edwTable, edwColumn, adeTable, adeColumn), Power BI mappings (semanticName, semanticModel), Databricks mappings (raw, serve), and pre-computed counts (reportCount, tableCount). Use for detailed object inspection and gap analysis. Limited to 100 results per call."

**Description review**
- **Pros**: aligns with the Metric version; precise field list.
- **Cons**: does not explicitly mention the `Type` field in the response.
- **Possible improvement**: mention that `Type` is returned and will be `Attribute`.

**Query used (as in code)** — `GetObjectDetailsQuery`
```cypher
MATCH (n:Metric)
WHERE n.guid IN $guids
RETURN 
  'Metric' as Type,
  n.guid as GUID,
  n.name as Name,
  COALESCE(n.updated_parity_status, n.parity_status, 'No Status') as Status,
  n.parity_group as `Group`,
  n.parity_subgroup as SubGroup,
  n.parity_team as Team,
  n.inherited_priority_level as Priority,
  n.formula as Formula,
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
  n.parity_notes as Notes,
  COALESCE(n.lineage_used_by_reports_count, 0) as ReportCount,
  COALESCE(n.lineage_source_tables_count, 0) as TableCount
LIMIT 100
UNION ALL
MATCH (n:Attribute)
WHERE n.guid IN $guids
RETURN 
  'Attribute' as Type,
  n.guid as GUID,
  n.name as Name,
  COALESCE(n.updated_parity_status, n.parity_status, 'No Status') as Status,
  n.parity_group as `Group`,
  n.parity_subgroup as SubGroup,
  n.parity_team as Team,
  n.inherited_priority_level as Priority,
  n.formula as Formula,
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
  n.parity_notes as Notes,
  COALESCE(n.lineage_used_by_reports_count, 0) as ReportCount,
  COALESCE(n.lineage_source_tables_count, 0) as TableCount
LIMIT 100
```

**Query review**
- **Pros**: complete mapping fields; shared with the Metric detail tool.
- **Cons**: query returns both Metrics and Attributes, which is redundant for attribute-only calls.
- **Possible improvement**: split query or add label filter to reduce scanning.

**Other info**
- If no records are found, the handler returns: `"No Attribute found with the specified GUID."`

---

## 3) `search-metrics`

**Inputs**
- `searchTerm` (string, optional): search by name or GUID (case-insensitive), comma-separated values allowed
- `priorityLevel` (string[], optional): P1–P5 or `All Prioritised`
- `businessArea` (string[], optional): business area or `All Areas`
- `status` (string[], optional): `Complete`, `Planned`, `Not Planned`, `No Status`, or `All Status`
- `dataDomain` (string[], optional): data domains or `All Domains`
- `offset` (number, optional): pagination offset, increments by 100

**Outputs**
- JSON object with:
  - `results`: array of `{ type, name, guid, status, priority, team, reports, tables }`
  - `moreResults`: boolean

**LLM Description (as in code)**
> "Search for MicroStrategy Metrics used by prioritized reports. Returns: type, name, guid, status, priority, team, reportCount, tableCount. Results are ordered by report count (most impactful first). PAGINATION: Returns 100 results per page. Use 'offset' parameter to paginate. Response includes 'moreResults' boolean - if true, call again with offset+100. Use for finding high-impact metrics for migration planning."

**Description review**
- **Pros**: explicitly describes output fields and pagination.
- **Cons**: does not mention that `results` is wrapped in an object with `moreResults`.
- **Possible improvement**: add a short example response structure.

**Query used (as in code)** — `SearchObjectsQuery`
```cypher
WITH CASE WHEN coalesce($neodash_searchterm, '') = '' THEN null ELSE [term IN split($neodash_searchterm, ',') | toLower(trim(term))] END as searchTerms,
     CASE WHEN coalesce($neodash_objecttype, '') = '' OR $neodash_objecttype = 'All Types' THEN ['Metric', 'Attribute'] ELSE [$neodash_objecttype] END as typeFilter,
     CASE WHEN $neodash_priority_level IS NULL OR size($neodash_priority_level) = 0 OR 'All Prioritized' IN $neodash_priority_level THEN null ELSE [p IN $neodash_priority_level | toInteger(replace(replace(replace(p, 'P', ''), ' (Highest)', ''), ' (Lowest)', ''))] END as priorityLevelFilter,
     CASE WHEN $neodash_business_area IS NULL OR size($neodash_business_area) = 0 OR 'All Areas' IN $neodash_business_area THEN null ELSE $neodash_business_area END as businessAreaFilter,
     CASE WHEN $neodash_status IS NULL OR size($neodash_status) = 0 OR 'All Status' IN $neodash_status THEN null ELSE $neodash_status END as filterStatusList,
     CASE WHEN $neodash_data_domain IS NULL OR size($neodash_data_domain) = 0 OR 'All Domains' IN $neodash_data_domain THEN null ELSE $neodash_data_domain END as dataDomainFilter,
     COALESCE($offset, 0) as offsetVal
MATCH (n:MSTRObject)
WHERE n.type IN typeFilter
  AND n.guid IS NOT NULL
  AND n.inherited_priority_level IS NOT NULL
  AND (searchTerms IS NULL OR any(term IN searchTerms WHERE toLower(n.name) CONTAINS term OR toLower(n.guid) CONTAINS term))
  AND (dataDomainFilter IS NULL OR ALL(domain IN dataDomainFilter WHERE EXISTS { MATCH (dp:DataProduct {name: domain})-[:BELONGS_TO]->(n) }))
WITH n, priorityLevelFilter, businessAreaFilter, filterStatusList, offsetVal,
     COALESCE(n.updated_parity_status, n.parity_status, 'No Status') as effectiveStatus
WHERE (filterStatusList IS NULL OR effectiveStatus IN filterStatusList)
  AND (businessAreaFilter IS NULL OR ALL(ba IN businessAreaFilter WHERE EXISTS { MATCH (r2:MSTRObject)-[:DEPENDS_ON]->(n) WHERE r2.type IN ['Report', 'GridReport', 'Document'] AND r2.priority_level IS NOT NULL AND r2.usage_area = ba } OR EXISTS { MATCH (r2:MSTRObject)-[:DEPENDS_ON]->(fp:MSTRObject)-[:DEPENDS_ON]->(n) WHERE r2.type IN ['Report', 'GridReport', 'Document'] AND r2.priority_level IS NOT NULL AND fp.type IN ['Filter', 'Prompt'] AND r2.usage_area = ba }))
CALL {
  WITH n, priorityLevelFilter, businessAreaFilter
  MATCH (r:MSTRObject)-[:DEPENDS_ON]->(n)
  WHERE r.type IN ['Report', 'GridReport', 'Document']
    AND r.priority_level IS NOT NULL
    AND (priorityLevelFilter IS NULL OR r.priority_level IN priorityLevelFilter)
    AND (businessAreaFilter IS NULL OR r.usage_area IN businessAreaFilter)
  RETURN collect(DISTINCT r.guid) as directGuids
}
CALL {
  WITH n, priorityLevelFilter, businessAreaFilter
  MATCH (r:MSTRObject)-[:DEPENDS_ON]->(fp:MSTRObject)-[:DEPENDS_ON]->(n)
  WHERE r.type IN ['Report', 'GridReport', 'Document']
    AND r.priority_level IS NOT NULL
    AND fp.type IN ['Filter', 'Prompt']
    AND (priorityLevelFilter IS NULL OR r.priority_level IN priorityLevelFilter)
    AND (businessAreaFilter IS NULL OR r.usage_area IN businessAreaFilter)
  RETURN collect(DISTINCT r.guid) as indirectGuids
}
WITH n, effectiveStatus, offsetVal, directGuids + [g IN indirectGuids WHERE NOT g IN directGuids] as allReportGuids
WHERE size(allReportGuids) > 0
WITH n, effectiveStatus, size(allReportGuids) as reportCount, COALESCE(n.lineage_source_tables_count, 0) as tableCount, offsetVal
ORDER BY reportCount DESC, n.name ASC
WITH collect({
  type: n.type,
  name: n.name,
  guid: n.guid,
  status: effectiveStatus,
  priority: n.inherited_priority_level,
  team: n.parity_team,
  reports: reportCount,
  tables: tableCount
}) as allResults, offsetVal
WITH allResults[offsetVal..offsetVal+101] as slicedResults
RETURN 
  slicedResults[0..100] as results,
  size(slicedResults) > 100 as moreResults
```

**Query review**
- **Pros**: robust filters; counts based on report usage; supports pagination.
- **Cons**: uses list collection and slicing, which may be memory heavy on very large datasets.
- **Possible improvement**: consider using `SKIP/LIMIT` earlier to reduce memory footprint.

**Other info**
- If no records are found, returns `{"results": [], "moreResults": false}`.

---

## 4) `search-attributes`

**Inputs**
- Same as `search-metrics` (filters + `offset`)

**Outputs**
- Same shape as `search-metrics` (`results` array + `moreResults` boolean)

**LLM Description (as in code)**
> "Search for MicroStrategy Attributes used by prioritized reports. Returns: type, name, guid, status, priority, team, reportCount, tableCount. Results are ordered by report count (most impactful first). PAGINATION: Returns 100 results per page. Use 'offset' parameter to paginate. Response includes 'moreResults' boolean - if true, call again with offset+100. Use for finding high-impact attributes for migration planning."

**Description review**
- **Pros**: clear and consistent with Metrics search.
- **Cons**: could mention that only Attributes are returned (tool fixes `objecttype`).
- **Possible improvement**: add a one-line note: “This tool only searches Attributes.”

**Query used (as in code)** — `SearchObjectsQuery` (same as above)
```cypher
WITH CASE WHEN coalesce($neodash_searchterm, '') = '' THEN null ELSE [term IN split($neodash_searchterm, ',') | toLower(trim(term))] END as searchTerms,
     CASE WHEN coalesce($neodash_objecttype, '') = '' OR $neodash_objecttype = 'All Types' THEN ['Metric', 'Attribute'] ELSE [$neodash_objecttype] END as typeFilter,
     CASE WHEN $neodash_priority_level IS NULL OR size($neodash_priority_level) = 0 OR 'All Prioritized' IN $neodash_priority_level THEN null ELSE [p IN $neodash_priority_level | toInteger(replace(replace(replace(p, 'P', ''), ' (Highest)', ''), ' (Lowest)', ''))] END as priorityLevelFilter,
     CASE WHEN $neodash_business_area IS NULL OR size($neodash_business_area) = 0 OR 'All Areas' IN $neodash_business_area THEN null ELSE $neodash_business_area END as businessAreaFilter,
     CASE WHEN $neodash_status IS NULL OR size($neodash_status) = 0 OR 'All Status' IN $neodash_status THEN null ELSE $neodash_status END as filterStatusList,
     CASE WHEN $neodash_data_domain IS NULL OR size($neodash_data_domain) = 0 OR 'All Domains' IN $neodash_data_domain THEN null ELSE $neodash_data_domain END as dataDomainFilter,
     COALESCE($offset, 0) as offsetVal
MATCH (n:MSTRObject)
WHERE n.type IN typeFilter
  AND n.guid IS NOT NULL
  AND n.inherited_priority_level IS NOT NULL
  AND (searchTerms IS NULL OR any(term IN searchTerms WHERE toLower(n.name) CONTAINS term OR toLower(n.guid) CONTAINS term))
  AND (dataDomainFilter IS NULL OR ALL(domain IN dataDomainFilter WHERE EXISTS { MATCH (dp:DataProduct {name: domain})-[:BELONGS_TO]->(n) }))
WITH n, priorityLevelFilter, businessAreaFilter, filterStatusList, offsetVal,
     COALESCE(n.updated_parity_status, n.parity_status, 'No Status') as effectiveStatus
WHERE (filterStatusList IS NULL OR effectiveStatus IN filterStatusList)
  AND (businessAreaFilter IS NULL OR ALL(ba IN businessAreaFilter WHERE EXISTS { MATCH (r2:MSTRObject)-[:DEPENDS_ON]->(n) WHERE r2.type IN ['Report', 'GridReport', 'Document'] AND r2.priority_level IS NOT NULL AND r2.usage_area = ba } OR EXISTS { MATCH (r2:MSTRObject)-[:DEPENDS_ON]->(fp:MSTRObject)-[:DEPENDS_ON]->(n) WHERE r2.type IN ['Report', 'GridReport', 'Document'] AND r2.priority_level IS NOT NULL AND fp.type IN ['Filter', 'Prompt'] AND r2.usage_area = ba }))
CALL {
  WITH n, priorityLevelFilter, businessAreaFilter
  MATCH (r:MSTRObject)-[:DEPENDS_ON]->(n)
  WHERE r.type IN ['Report', 'GridReport', 'Document']
    AND r.priority_level IS NOT NULL
    AND (priorityLevelFilter IS NULL OR r.priority_level IN priorityLevelFilter)
    AND (businessAreaFilter IS NULL OR r.usage_area IN businessAreaFilter)
  RETURN collect(DISTINCT r.guid) as directGuids
}
CALL {
  WITH n, priorityLevelFilter, businessAreaFilter
  MATCH (r:MSTRObject)-[:DEPENDS_ON]->(fp:MSTRObject)-[:DEPENDS_ON]->(n)
  WHERE r.type IN ['Report', 'GridReport', 'Document']
    AND r.priority_level IS NOT NULL
    AND fp.type IN ['Filter', 'Prompt']
    AND (priorityLevelFilter IS NULL OR r.priority_level IN priorityLevelFilter)
    AND (businessAreaFilter IS NULL OR r.usage_area IN businessAreaFilter)
  RETURN collect(DISTINCT r.guid) as indirectGuids
}
WITH n, effectiveStatus, offsetVal, directGuids + [g IN indirectGuids WHERE NOT g IN directGuids] as allReportGuids
WHERE size(allReportGuids) > 0
WITH n, effectiveStatus, size(allReportGuids) as reportCount, COALESCE(n.lineage_source_tables_count, 0) as tableCount, offsetVal
ORDER BY reportCount DESC, n.name ASC
WITH collect({
  type: n.type,
  name: n.name,
  guid: n.guid,
  status: effectiveStatus,
  priority: n.inherited_priority_level,
  team: n.parity_team,
  reports: reportCount,
  tables: tableCount
}) as allResults, offsetVal
WITH allResults[offsetVal..offsetVal+101] as slicedResults
RETURN 
  slicedResults[0..100] as results,
  size(slicedResults) > 100 as moreResults
```

**Query review**
- **Pros**: same as `search-metrics`.
- **Cons**: same as `search-metrics`.
- **Possible improvement**: same as `search-metrics`.

**Other info**
- If no records are found, returns `{"results": [], "moreResults": false}`.

---

## 5) `get-reports-using-metric`

**Inputs**
- `guid` (string, required): GUID of the Metric to analyse
- `priorityLevel` (string[], optional): P1–P5 or `All Prioritised`
- `businessArea` (string[], optional): business area or `All Areas`
- `offset` (number, optional): pagination offset

**Outputs**
- JSON object with:
  - `objectName`, `objectGUID`, `objectType`
  - `totalReports`
  - `reports`: array of `{ name, guid, type, priority, area, department, users }`
  - `moreResults`

**LLM Description (as in code)**
> "Find all Reports, GridReports, and Documents that use a specific Metric. Returns for each report: name, guid, type, priority (1-5), area, department, userCount. PAGINATION: Returns 100 reports per page. Use 'offset' to paginate. Response includes 'moreResults' boolean and total count via 'totalReports'. Note: High-usage metrics (e.g., 'Retail Sales Value') may have 6000+ reports. Use for impact analysis: understanding what will be affected by changes."

**Description review**
- **Pros**: includes scale warning; clear outputs.
- **Cons**: mixes “userCount” but output field is `users`.
- **Possible improvement**: align wording to `users`.

**Query used (as in code)** — `ReportsUsingObjectsQuery`
```cypher
WITH $guids as selectedGuids,
     CASE WHEN $priorityLevel IS NULL OR size($priorityLevel) = 0 OR 'All Prioritized' IN $priorityLevel THEN null ELSE [p IN $priorityLevel | toInteger(replace(replace(replace(p, 'P', ''), ' (Highest)', ''), ' (Lowest)', ''))] END as priorityLevelFilter,
     CASE WHEN $businessArea IS NULL OR size($businessArea) = 0 OR 'All Areas' IN $businessArea THEN null ELSE $businessArea END as businessAreaFilter,
     COALESCE($offset, 0) as offsetVal
MATCH (n:MSTRObject)
WHERE n.guid IN selectedGuids

// Runtime BFS traversal through [Prompt, Filter] to find reports
OPTIONAL MATCH path = (r:MSTRObject)-[:DEPENDS_ON*1..10]->(n)
WHERE r.type IN ['Report', 'GridReport', 'Document']
  AND r.priority_level IS NOT NULL
  AND ALL(mid IN nodes(path)[1..-1] WHERE mid.type IN ['Prompt', 'Filter'])
  AND (priorityLevelFilter IS NULL OR r.priority_level IN priorityLevelFilter)
  AND (businessAreaFilter IS NULL OR r.usage_area IN businessAreaFilter)

WITH n, collect(DISTINCT r) as allReports, offsetVal
WITH n, allReports, size(allReports) as totalReports, offsetVal,
     allReports[offsetVal..offsetVal+101] as slicedReports

UNWIND CASE WHEN size(slicedReports) > 0 THEN slicedReports[0..100] ELSE [null] END as r
WITH n, totalReports, offsetVal, size(slicedReports) as slicedSize,
     CASE WHEN r IS NOT NULL THEN collect({
       name: r.name,
       guid: r.guid,
       type: r.type,
       priority: r.priority_level,
       area: r.usage_area,
       department: r.usage_department,
       users: r.usage_users_count
     }) ELSE [] END as reports

RETURN 
  n.name as objectName,
  n.guid as objectGUID,
  n.type as objectType,
  totalReports,
  reports,
  slicedSize > 100 as moreResults
```

**Query review**
- **Pros**: runtime traversal ensures fresh lineage; pagination handled.
- **Cons**: BFS path limit 10 could miss deeper dependencies.
- **Possible improvement**: expose max depth as a parameter (with safe defaults).

**Other info**
- If no records are found, returns an object with empty fields and `totalReports: 0`.

---

## 6) `get-reports-using-attribute`

**Inputs**
- Same as `get-reports-using-metric`

**Outputs**
- Same shape as `get-reports-using-metric`

**LLM Description (as in code)**
> "Find all Reports, GridReports, and Documents that use a specific Attribute. Returns for each report: name, guid, type, priority (1-5), area, department, userCount. PAGINATION: Returns 100 reports per page. Use 'offset' to paginate. Response includes 'moreResults' boolean and total count via 'totalReports'. Note: High-usage attributes may have thousands of reports. Use for impact analysis: understanding what will be affected by changes."

**Description review**
- **Pros**: includes scale note; well aligned with search.
- **Cons**: same “userCount” naming mismatch.
- **Possible improvement**: change “userCount” to “users” for consistency.

**Query used (as in code)** — `ReportsUsingObjectsQuery` (same as above)
```cypher
WITH $guids as selectedGuids,
     CASE WHEN $priorityLevel IS NULL OR size($priorityLevel) = 0 OR 'All Prioritized' IN $priorityLevel THEN null ELSE [p IN $priorityLevel | toInteger(replace(replace(replace(p, 'P', ''), ' (Highest)', ''), ' (Lowest)', ''))] END as priorityLevelFilter,
     CASE WHEN $businessArea IS NULL OR size($businessArea) = 0 OR 'All Areas' IN $businessArea THEN null ELSE $businessArea END as businessAreaFilter,
     COALESCE($offset, 0) as offsetVal
MATCH (n:MSTRObject)
WHERE n.guid IN selectedGuids

// Runtime BFS traversal through [Prompt, Filter] to find reports
OPTIONAL MATCH path = (r:MSTRObject)-[:DEPENDS_ON*1..10]->(n)
WHERE r.type IN ['Report', 'GridReport', 'Document']
  AND r.priority_level IS NOT NULL
  AND ALL(mid IN nodes(path)[1..-1] WHERE mid.type IN ['Prompt', 'Filter'])
  AND (priorityLevelFilter IS NULL OR r.priority_level IN priorityLevelFilter)
  AND (businessAreaFilter IS NULL OR r.usage_area IN businessAreaFilter)

WITH n, collect(DISTINCT r) as allReports, offsetVal
WITH n, allReports, size(allReports) as totalReports, offsetVal,
     allReports[offsetVal..offsetVal+101] as slicedReports

UNWIND CASE WHEN size(slicedReports) > 0 THEN slicedReports[0..100] ELSE [null] END as r
WITH n, totalReports, offsetVal, size(slicedReports) as slicedSize,
     CASE WHEN r IS NOT NULL THEN collect({
       name: r.name,
       guid: r.guid,
       type: r.type,
       priority: r.priority_level,
       area: r.usage_area,
       department: r.usage_department,
       users: r.usage_users_count
     }) ELSE [] END as reports

RETURN 
  n.name as objectName,
  n.guid as objectGUID,
  n.type as objectType,
  totalReports,
  reports,
  slicedSize > 100 as moreResults
```

**Query review**
- **Pros/Cons**: same as `get-reports-using-metric`.

**Other info**
- Same empty-response behaviour as `get-reports-using-metric`.

---

## 7) `get-metric-source-tables`

**Inputs**
- `guid` (string, required): GUID of the Metric to analyse
- `offset` (number, optional): pagination offset

**Outputs**
- JSON object with:
  - `objectName`, `objectGUID`, `objectType`
  - `totalTables`
  - `tables`: array of `{ name, guid, type, physicalTable, database }`
  - `moreResults`

**LLM Description (as in code)**
> "Find source database tables (LogicalTable/Table) that a Metric depends on. Returns for each table: name, guid, type, physicalTableName, databaseInstance. Traverses the full dependency graph (up to 10 levels) through Facts, Metrics, Attributes, Columns. PAGINATION: Returns 100 tables per page. Use 'offset' to paginate. Response includes 'moreResults' boolean and 'totalTables' count. Use for data lineage analysis and DBT model generation."

**Description review**
- **Pros**: clear on traversal path and output fields.
- **Cons**: output field names in text (`physicalTableName`, `databaseInstance`) differ from actual fields (`physicalTable`, `database`).
- **Possible improvement**: align field names in the description.

**Query used (as in code)** — `SourceTablesQuery`
```cypher
WITH $guids as selectedGuids,
     COALESCE($offset, 0) as offsetVal
MATCH (n:MSTRObject)
WHERE n.guid IN selectedGuids

// Runtime BFS traversal through [Fact, Metric, Attribute, Column] to find tables
OPTIONAL MATCH path = (n)-[:DEPENDS_ON*1..10]->(t:MSTRObject)
WHERE t.type IN ['LogicalTable', 'Table']
  AND ALL(mid IN nodes(path)[1..-1] WHERE mid.type IN ['Fact', 'Metric', 'Attribute', 'Column'])

WITH n, collect(DISTINCT t) as allTables, offsetVal
WITH n, allTables, size(allTables) as totalTables, offsetVal,
     allTables[offsetVal..offsetVal+101] as slicedTables

UNWIND CASE WHEN size(slicedTables) > 0 THEN slicedTables[0..100] ELSE [null] END as t
WITH n, totalTables, offsetVal, size(slicedTables) as slicedSize,
     CASE WHEN t IS NOT NULL THEN collect({
       name: t.name,
       guid: t.guid,
       type: t.type,
       physicalTable: t.physical_table_name,
       database: t.database_instance
     }) ELSE [] END as tables

RETURN 
  n.name as objectName,
  n.guid as objectGUID,
  n.type as objectType,
  totalTables,
  tables,
  slicedSize > 100 as moreResults
```

**Query review**
- **Pros**: runtime traversal with strict node-type constraints; paginated output.
- **Cons**: may miss table lineage that requires deeper than 10 hops.
- **Possible improvement**: allow depth configuration, with safe upper bounds.

**Other info**
- Empty response returns `totalTables: 0` with empty arrays.

---

## 8) `get-attribute-source-tables`

**Inputs**
- `guid` (string, required): GUID of the Attribute to analyse
- `offset` (number, optional): pagination offset

**Outputs**
- Same shape as `get-metric-source-tables`

**LLM Description (as in code)**
> "Find source database tables (LogicalTable/Table) that an Attribute depends on. Returns for each table: name, guid, type, physicalTableName, databaseInstance. Traverses the full dependency graph (up to 10 levels) through Facts, Metrics, Attributes, Columns. PAGINATION: Returns 100 tables per page. Use 'offset' to paginate. Response includes 'moreResults' boolean and 'totalTables' count. Use for data lineage analysis and DBT model generation."

**Description review**
- **Pros**: consistent with Metric version.
- **Cons**: same field-name mismatch as above.
- **Possible improvement**: align field names with output.

**Query used (as in code)** — `SourceTablesQuery` (same as above)
```cypher
WITH $guids as selectedGuids,
     COALESCE($offset, 0) as offsetVal
MATCH (n:MSTRObject)
WHERE n.guid IN selectedGuids

// Runtime BFS traversal through [Fact, Metric, Attribute, Column] to find tables
OPTIONAL MATCH path = (n)-[:DEPENDS_ON*1..10]->(t:MSTRObject)
WHERE t.type IN ['LogicalTable', 'Table']
  AND ALL(mid IN nodes(path)[1..-1] WHERE mid.type IN ['Fact', 'Metric', 'Attribute', 'Column'])

WITH n, collect(DISTINCT t) as allTables, offsetVal
WITH n, allTables, size(allTables) as totalTables, offsetVal,
     allTables[offsetVal..offsetVal+101] as slicedTables

UNWIND CASE WHEN size(slicedTables) > 0 THEN slicedTables[0..100] ELSE [null] END as t
WITH n, totalTables, offsetVal, size(slicedTables) as slicedSize,
     CASE WHEN t IS NOT NULL THEN collect({
       name: t.name,
       guid: t.guid,
       type: t.type,
       physicalTable: t.physical_table_name,
       database: t.database_instance
     }) ELSE [] END as tables

RETURN 
  n.name as objectName,
  n.guid as objectGUID,
  n.type as objectType,
  totalTables,
  tables,
  slicedSize > 100 as moreResults
```

**Query review**
- **Pros/Cons**: same as `get-metric-source-tables`.

**Other info**
- Same empty-response behaviour.

---

## 9) `get-metric-dependencies`

**Inputs**
- `guid` (string, required): GUID of the Metric to analyse
- `offset` (number, optional): pagination offset for direct dependencies

**Outputs**
- JSON object with:
  - `objectName`, `objectGUID`, `objectType`
  - `totalDirectDeps`
  - `transitiveTableCount`
  - `directDependencies`: array of `{ type, name, guid, formula }`
  - `moreResults`

**LLM Description (as in code)**
> "Find what a Metric directly depends on (downstream/outbound dependencies). Returns two parts: (1) directDependencies: Objects at 1-hop distance (Facts, Metrics, Attributes, Columns) with type, name, guid, formula. (2) transitiveTableCount: Total count of tables reachable through the full dependency chain (2-10 hops). PAGINATION: Direct dependencies are paginated (100 per page). Use 'offset' to paginate. Response includes 'moreResults' boolean and 'totalDirectDeps' count. Use for understanding metric calculation logic and formula translation to SQL/DBT."

**Description review**
- **Pros**: detailed structure; explains two-part output.
- **Cons**: does not explicitly mention `objectName/objectGUID/objectType` fields.
- **Possible improvement**: add a short note about the wrapper object fields.

**Query used (as in code)** — `DownstreamDependenciesQuery`
```cypher
WITH $guids as selectedGuids,
     COALESCE($offset, 0) as offsetVal
MATCH (n:MSTRObject)
WHERE n.guid IN selectedGuids

// Get ALL direct dependencies (1-hop)
OPTIONAL MATCH (n)-[:DEPENDS_ON]->(direct:MSTRObject)
WITH n, offsetVal, collect(DISTINCT {
  type: direct.type,
  name: direct.name,
  guid: direct.guid,
  formula: direct.formula
}) as allDirectDeps

// Runtime BFS for transitive table count (2-10 hops)
// Traversal through [Fact, Metric, Attribute, Column], target [LogicalTable, Table]
OPTIONAL MATCH path = (n)-[:DEPENDS_ON*2..10]->(t:MSTRObject)
WHERE t.type IN ['LogicalTable', 'Table']
  AND ALL(mid IN nodes(path)[1..-1] WHERE mid.type IN ['Fact', 'Metric', 'Attribute', 'Column'])
WITH n, allDirectDeps, offsetVal, count(DISTINCT t) as transitiveTableCount

// Pagination for direct dependencies
WITH n, allDirectDeps, transitiveTableCount, offsetVal,
     allDirectDeps[offsetVal..offsetVal+101] as slicedDeps

RETURN 
  n.name as objectName,
  n.guid as objectGUID,
  n.type as objectType,
  size(allDirectDeps) as totalDirectDeps,
  transitiveTableCount,
  slicedDeps[0..100] as directDependencies,
  size(slicedDeps) > 100 as moreResults
```

**Query review**
- **Pros**: separates direct dependencies from transitive table counts; pagination on direct dependencies.
- **Cons**: `transitiveTableCount` excludes 1-hop tables by using `*2..10`.
- **Possible improvement**: consider a separate count that includes 1-hop tables, or clarify this in the description.

**Other info**
- Empty response returns zero counts with empty arrays.

---

## 10) `get-attribute-dependencies`

**Inputs**
- Same as `get-metric-dependencies`

**Outputs**
- Same shape as `get-metric-dependencies`

**LLM Description (as in code)**
> "Find what an Attribute directly depends on (downstream/outbound dependencies). Returns two parts: (1) directDependencies: Objects at 1-hop distance (Facts, Metrics, Attributes, Columns) with type, name, guid, formula. (2) transitiveTableCount: Total count of tables reachable through the full dependency chain (2-10 hops). PAGINATION: Direct dependencies are paginated (100 per page). Use 'offset' to paginate. Response includes 'moreResults' boolean and 'totalDirectDeps' count. Use for understanding attribute structure and data lineage."

**Description review**
- **Pros**: clear and consistent.
- **Cons**: same wrapper-field omission as above.
- **Possible improvement**: add note about `objectName/objectGUID/objectType`.

**Query used (as in code)** — `DownstreamDependenciesQuery` (same as above)
```cypher
WITH $guids as selectedGuids,
     COALESCE($offset, 0) as offsetVal
MATCH (n:MSTRObject)
WHERE n.guid IN selectedGuids

// Get ALL direct dependencies (1-hop)
OPTIONAL MATCH (n)-[:DEPENDS_ON]->(direct:MSTRObject)
WITH n, offsetVal, collect(DISTINCT {
  type: direct.type,
  name: direct.name,
  guid: direct.guid,
  formula: direct.formula
}) as allDirectDeps

// Runtime BFS for transitive table count (2-10 hops)
// Traversal through [Fact, Metric, Attribute, Column], target [LogicalTable, Table]
OPTIONAL MATCH path = (n)-[:DEPENDS_ON*2..10]->(t:MSTRObject)
WHERE t.type IN ['LogicalTable', 'Table']
  AND ALL(mid IN nodes(path)[1..-1] WHERE mid.type IN ['Fact', 'Metric', 'Attribute', 'Column'])
WITH n, allDirectDeps, offsetVal, count(DISTINCT t) as transitiveTableCount

// Pagination for direct dependencies
WITH n, allDirectDeps, transitiveTableCount, offsetVal,
     allDirectDeps[offsetVal..offsetVal+101] as slicedDeps

RETURN 
  n.name as objectName,
  n.guid as objectGUID,
  n.type as objectType,
  size(allDirectDeps) as totalDirectDeps,
  transitiveTableCount,
  slicedDeps[0..100] as directDependencies,
  size(slicedDeps) > 100 as moreResults
```

**Query review**
- **Pros/Cons**: same as `get-metric-dependencies`.

**Other info**
- Same empty-response behaviour.

---

## 11) `get-metric-dependents`

**Inputs**
- `guid` (string, required): GUID of the Metric to analyse
- `offset` (number, optional): pagination offset

**Outputs**
- JSON object with:
  - `objectName`, `objectGUID`, `objectType`
  - `totalReports`
  - `reports`: array of `{ name, guid, type, priority, area, department, users }`
  - `moreResults`

**LLM Description (as in code)**
> "Find all Reports, GridReports, and Documents that depend on a Metric (upstream/inbound dependencies). Traverses the dependency graph through Prompts and Filters (up to 10 levels). Returns for each report: name, guid, type, priority, area, department, userCount. Also returns 'totalReports' count for the full dataset. PAGINATION: Returns 100 reports per page. Use 'offset' to paginate. Response includes 'moreResults' boolean. Use for impact analysis before modifying or deprecating a metric."

**Description review**
- **Pros**: clearly states traversal and pagination.
- **Cons**: “userCount” naming mismatch with `users`.
- **Possible improvement**: align names to avoid confusion.

**Query used (as in code)** — `UpstreamDependenciesQuery`
```cypher
WITH $guids as selectedGuids,
     COALESCE($offset, 0) as offsetVal
MATCH (n:MSTRObject)
WHERE n.guid IN selectedGuids

// Runtime BFS traversal through [Prompt, Filter] to find reports
OPTIONAL MATCH path = (r:MSTRObject)-[:DEPENDS_ON*1..10]->(n)
WHERE r.type IN ['Report', 'GridReport', 'Document']
  AND ALL(mid IN nodes(path)[1..-1] WHERE mid.type IN ['Prompt', 'Filter'])

WITH n, collect(DISTINCT r) as allReports, offsetVal
WITH n, allReports, size(allReports) as totalReports, offsetVal,
     allReports[offsetVal..offsetVal+101] as slicedReports

UNWIND CASE WHEN size(slicedReports) > 0 THEN slicedReports[0..100] ELSE [null] END as r
WITH n, totalReports, offsetVal, size(slicedReports) as slicedSize,
     CASE WHEN r IS NOT NULL THEN collect({
       name: r.name,
       guid: r.guid,
       type: r.type,
       priority: r.priority_level,
       area: r.usage_area,
       department: r.usage_department,
       users: r.usage_users_count
     }) ELSE [] END as reports

RETURN 
  n.name as objectName,
  n.guid as objectGUID,
  n.type as objectType,
  totalReports,
  reports,
  slicedSize > 100 as moreResults
```

**Query review**
- **Pros**: runtime BFS ensures fresh lineage; paginated output.
- **Cons**: fixed hop depth; collects all reports before slicing.
- **Possible improvement**: use `SKIP/LIMIT` earlier or introduce a max depth parameter.

**Other info**
- Empty response returns an object with `totalReports: 0`.

---

## 12) `get-attribute-dependents`

**Inputs**
- Same as `get-metric-dependents`

**Outputs**
- Same as `get-metric-dependents`

**LLM Description (as in code)**
> "Find all Reports, GridReports, and Documents that depend on an Attribute (upstream/inbound dependencies). Traverses the dependency graph through Prompts and Filters (up to 10 levels). Returns for each report: name, guid, type, priority, area, department, userCount. Also returns 'totalReports' count for the full dataset. PAGINATION: Returns 100 reports per page. Use 'offset' to paginate. Response includes 'moreResults' boolean. Use for impact analysis before modifying or deprecating an attribute."

**Description review**
- **Pros**: consistent with Metric dependents.
- **Cons**: same naming mismatch for `users`.
- **Possible improvement**: align to `users`.

**Query used (as in code)** — `UpstreamDependenciesQuery` (same as above)
```cypher
WITH $guids as selectedGuids,
     COALESCE($offset, 0) as offsetVal
MATCH (n:MSTRObject)
WHERE n.guid IN selectedGuids

// Runtime BFS traversal through [Prompt, Filter] to find reports
OPTIONAL MATCH path = (r:MSTRObject)-[:DEPENDS_ON*1..10]->(n)
WHERE r.type IN ['Report', 'GridReport', 'Document']
  AND ALL(mid IN nodes(path)[1..-1] WHERE mid.type IN ['Prompt', 'Filter'])

WITH n, collect(DISTINCT r) as allReports, offsetVal
WITH n, allReports, size(allReports) as totalReports, offsetVal,
     allReports[offsetVal..offsetVal+101] as slicedReports

UNWIND CASE WHEN size(slicedReports) > 0 THEN slicedReports[0..100] ELSE [null] END as r
WITH n, totalReports, offsetVal, size(slicedReports) as slicedSize,
     CASE WHEN r IS NOT NULL THEN collect({
       name: r.name,
       guid: r.guid,
       type: r.type,
       priority: r.priority_level,
       area: r.usage_area,
       department: r.usage_department,
       users: r.usage_users_count
     }) ELSE [] END as reports

RETURN 
  n.name as objectName,
  n.guid as objectGUID,
  n.type as objectType,
  totalReports,
  reports,
  slicedSize > 100 as moreResults
```

**Query review**
- **Pros/Cons**: same as `get-metric-dependents`.

**Other info**
- Same empty-response behaviour.

---

## 13) `get-metrics-stats`

**Inputs**
- `status` (string[], optional): `Complete`, `Planned`, `Not Planned`, `No Status`
- `team` (string, optional): team name

**Outputs**
- JSON object with: `total`, `complete`, `planned`, `notPlanned`, `noStatus`, `prioritized`, `teams`

**LLM Description (as in code)**
> "Get aggregate statistics for all MicroStrategy Metrics. Returns counts by parity status: total, complete, planned, notPlanned, noStatus. Also returns: prioritized (count with priority assigned), teams (distinct team names). NO PAGINATION - returns a single summary row (~200 bytes). Use BEFORE search-metrics to understand dataset scope and plan pagination strategy. Example workflow: (1) get-metrics-stats → see 450 total, (2) search-metrics with offset to fetch pages."

**Description review**
- **Pros**: strong guidance for workflow; minimal payload noted.
- **Cons**: example uses a hard-coded number which may mislead.
- **Possible improvement**: change to a generic example (“e.g.”).

**Query used (as in code)** — `MetricsStatsQuery`
```cypher
WITH CASE WHEN $status IS NULL OR size($status) = 0 THEN null ELSE $status END as filterStatus,
     CASE WHEN $team IS NULL OR $team = '' THEN null ELSE $team END as filterTeam
MATCH (n:Metric)
WHERE n.guid IS NOT NULL
WITH n, filterStatus, filterTeam,
     COALESCE(n.updated_parity_status, n.parity_status, 'No Status') as status
WHERE (filterStatus IS NULL OR status IN filterStatus)
  AND (filterTeam IS NULL OR n.parity_team = filterTeam)
RETURN 
  count(*) as total,
  count(CASE WHEN status = 'Complete' THEN 1 END) as complete,
  count(CASE WHEN status = 'Planned' THEN 1 END) as planned,
  count(CASE WHEN status = 'Not Planned' THEN 1 END) as notPlanned,
  count(CASE WHEN status = 'No Status' THEN 1 END) as noStatus,
  count(CASE WHEN n.inherited_priority_level IS NOT NULL THEN 1 END) as prioritized,
  collect(DISTINCT n.parity_team) as teams
```

**Query review**
- **Pros**: clear aggregation; no pagination needed.
- **Cons**: `teams` returns distinct values without counts.
- **Possible improvement**: include counts per team if needed for prioritisation.

**Other info**
- If no records, returns zeroed counts and empty `teams`.

---

## 14) `get-attributes-stats`

**Inputs**
- Same as `get-metrics-stats`

**Outputs**
- Same as `get-metrics-stats`

**LLM Description (as in code)**
> "Get aggregate statistics for all MicroStrategy Attributes. Returns counts by parity status: total, complete, planned, notPlanned, noStatus. Also returns: prioritized (count with priority assigned), teams (distinct team names). NO PAGINATION - returns a single summary row (~200 bytes). Use BEFORE search-attributes to understand dataset scope and plan pagination strategy."

**Description review**
- **Pros**: concise and aligned with metrics stats.
- **Cons**: does not include example workflow like metrics.
- **Possible improvement**: add the same two-step workflow for consistency.

**Query used (as in code)** — `AttributesStatsQuery`
```cypher
WITH CASE WHEN $status IS NULL OR size($status) = 0 THEN null ELSE $status END as filterStatus,
     CASE WHEN $team IS NULL OR $team = '' THEN null ELSE $team END as filterTeam
MATCH (n:Attribute)
WHERE n.guid IS NOT NULL
WITH n, filterStatus, filterTeam,
     COALESCE(n.updated_parity_status, n.parity_status, 'No Status') as status
WHERE (filterStatus IS NULL OR status IN filterStatus)
  AND (filterTeam IS NULL OR n.parity_team = filterTeam)
RETURN 
  count(*) as total,
  count(CASE WHEN status = 'Complete' THEN 1 END) as complete,
  count(CASE WHEN status = 'Planned' THEN 1 END) as planned,
  count(CASE WHEN status = 'Not Planned' THEN 1 END) as notPlanned,
  count(CASE WHEN status = 'No Status' THEN 1 END) as noStatus,
  count(CASE WHEN n.inherited_priority_level IS NOT NULL THEN 1 END) as prioritized,
  collect(DISTINCT n.parity_team) as teams
```

**Query review**
- **Pros**: clean aggregation for stats; easy to cache.
- **Cons**: same as metrics stats (no counts per team).
- **Possible improvement**: add counts per team if needed.

**Other info**
- If no records, returns zeroed counts.

---

## 15) `get-object-stats`

**Inputs**
- `guid` (string, required): GUID of the Metric or Attribute to analyse

**Outputs**
- JSON object with: `name`, `type`, `guid`, `status`, `team`, `reportCount`, `tableCount`, `reportsByPriority`

**LLM Description (as in code)**
> "Get summary statistics for a specific MicroStrategy object (Metric or Attribute). Returns: name, type, guid, status, team, reportCount, tableCount, reportsByPriority. reportsByPriority shows distribution: [{priority: 1, count: 23}, {priority: 2, count: 156}, ...]. NO PAGINATION - returns a single object summary (~500 bytes). Use for quick impact assessment of a single object without fetching full report list."

**Description review**
- **Pros**: includes an example structure; clear on use-case.
- **Cons**: example counts may be misconstrued as actual values.
- **Possible improvement**: add “example only” wording.

**Query used (as in code)** — `ObjectStatsQuery`
```cypher
MATCH (n:MSTRObject {guid: $guid})

// Runtime BFS for report count
OPTIONAL MATCH pathR = (r:MSTRObject)-[:DEPENDS_ON*1..10]->(n)
WHERE r.type IN ['Report', 'GridReport', 'Document']
  AND ALL(mid IN nodes(pathR)[1..-1] WHERE mid.type IN ['Prompt', 'Filter'])
WITH n, collect(DISTINCT r) as allReports

// Runtime BFS for table count
OPTIONAL MATCH pathT = (n)-[:DEPENDS_ON*1..10]->(t:MSTRObject)
WHERE t.type IN ['LogicalTable', 'Table']
  AND ALL(mid IN nodes(pathT)[1..-1] WHERE mid.type IN ['Fact', 'Metric', 'Attribute', 'Column'])
WITH n, allReports, count(DISTINCT t) as tableCount

// Aggregate reports by priority
UNWIND CASE WHEN size(allReports) > 0 THEN allReports ELSE [null] END as r
WITH n, size(allReports) as reportCount, tableCount,
     CASE WHEN r IS NOT NULL AND r.priority_level IS NOT NULL THEN r.priority_level ELSE null END as priority
WITH n, reportCount, tableCount, priority, count(*) as cnt
WHERE priority IS NOT NULL OR reportCount = 0
WITH n, reportCount, tableCount, 
     CASE WHEN priority IS NOT NULL THEN collect({priority: priority, count: cnt}) ELSE [] END as reportsByPriority

RETURN 
  n.name as name,
  n.type as type,
  n.guid as guid,
  COALESCE(n.updated_parity_status, n.parity_status, 'No Status') as status,
  n.parity_team as team,
  reportCount,
  tableCount,
  reportsByPriority
```

**Query review**
- **Pros**: uses runtime traversal for freshness; includes priority distribution.
- **Cons**: counts reports across all priorities but the aggregation uses `collect` after `UNWIND`, which may be inefficient on large datasets.
- **Possible improvement**: consider `apoc.map.groupBy` or aggregated `CASE` counts per priority.

**Other info**
- If the GUID is not found, returns `"No object found with the specified GUID."`

---

## Appendix: GDS Tool (Non-MSTR)

### `list-gds-procedures`

**Inputs**
- None

**Outputs**
- Records with: `name`, `description`, `signature`, `type`

**LLM Description (as in code)**
> "Use this tool to discover what graph science and analytics functions are available in the current Neo4j environment. It returns a structured list describing each function — what it does, how to use it, the inputs it needs, and what kind of results it produces. Do this before any reasoning, query generation, or analysis so you know what capabilities exist. Graph science and analytics functions help you with centrality, community detection, similarity, path finding, and identifying dependencies between nodes. The tool helps you understand the analytical capabilities of the system so that you can plan or compose the right graph science operations automatically. An empty response indicates that GDS is not installed and the user should be told to install it. Remember to use unique names for graph data science projections to avoid collisions and to drop them afterwards to save memory. You must always tell the user the function you will use."

**Description review**
- **Pros**: comprehensive and instructive.
- **Cons**: may be too long for a tool description; includes behavioural instructions not enforced by code.
- **Possible improvement**: split guidance into shorter sentences and move best-practice notes to documentation.

**Query used (as in code)** — `listGdsProceduresQuery`
```cypher
CALL gds.list() YIELD name, description, signature, type
WHERE type = "procedure"
AND name CONTAINS "stream"
AND NOT (name CONTAINS "estimate")
RETURN name, description, signature, type
```

**Query review**
- **Pros**: focused on stream procedures; reduces noise.
- **Cons**: excludes non-stream procedures and estimates which may still be relevant.
- **Possible improvement**: allow a parameter to include non-stream procedures.

**Other info**
- If GDS is not installed, the tool returns an error message indicating installation is required.
