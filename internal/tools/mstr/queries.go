package mstr

// =============================================================================
// Query 1: Get Object Details by GUID
// =============================================================================

// GetObjectDetailsQuery returns detailed information about a Metric or Attribute by GUID.
// Returns 22 fields including parity status, mappings, formula, and pre-computed counts.
// Parameters:
//   - guids: array of GUIDs to look up (exact match)
const GetObjectDetailsQuery = `
MATCH (n:Metric)
WHERE n.guid IN $guids
RETURN 
  'Metric' as Type,
  n.guid as GUID,
  n.name as Name,
  COALESCE(n.updated_parity_status, n.parity_status, 'No Status') as Status,
  n.parity_group as ` + "`Group`" + `,
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
  n.parity_group as ` + "`Group`" + `,
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
LIMIT 100`

// =============================================================================
// Query 2: Search Objects with Filters
// =============================================================================

// SearchObjectsQuery finds Metrics/Attributes used by prioritized reports with various filters.
// Returns paginated results (100 per page) with moreResults indicator.
// Parameters:
//   - neodash_searchterm: comma-separated search terms (string)
//   - neodash_objecttype: "Metric", "Attribute", or "All Types" (string)
//   - neodash_priority_level: array of priority levels like "P1 (Highest)", "P2", etc.
//   - neodash_business_area: array of business areas
//   - neodash_status: array of status values
//   - neodash_data_domain: array of data domains
//   - offset: pagination offset (0, 100, 200, ...)
const SearchObjectsQuery = `
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
  size(slicedResults) > 100 as moreResults`

// =============================================================================
// Query 3: Reports Using Objects (Runtime BFS)
// =============================================================================

// ReportsUsingObjectsQuery finds reports that use selected Metrics/Attributes.
// Uses runtime BFS traversal through [Prompt, Filter] to find reports.
// Returns paginated results (100 per page) with structured JSON.
// Parameters:
//   - guids: array of GUIDs to look up
//   - priorityLevel: optional array of priority levels
//   - businessArea: optional array of business areas
//   - offset: pagination offset (0, 100, 200, ...)
const ReportsUsingObjectsQuery = `
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
  slicedSize > 100 as moreResults`

// =============================================================================
// Query 4: Source Tables for Objects (Runtime BFS)
// =============================================================================

// SourceTablesQuery finds source tables for selected Metrics/Attributes.
// Uses runtime BFS traversal through [Fact, Metric, Attribute, Column] to find tables.
// Returns paginated results (100 per page) with physical table info.
// Parameters:
//   - guids: array of GUIDs to look up
//   - offset: pagination offset (0, 100, 200, ...)
const SourceTablesQuery = `
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
  slicedSize > 100 as moreResults`

// =============================================================================
// Query 5: Downstream Dependencies (what object depends on)
// =============================================================================

// DownstreamDependenciesQuery finds what the object depends on (downstream in the dependency chain).
// Returns direct dependencies (1-hop) with pagination, plus transitive table count via runtime BFS.
// Parameters:
//   - guids: array of GUIDs to look up
//   - offset: pagination offset for direct dependencies (0, 100, 200, ...)
const DownstreamDependenciesQuery = `
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
  size(slicedDeps) > 100 as moreResults`

// =============================================================================
// Query 6: Upstream Dependencies (what depends on object)
// =============================================================================

// UpstreamDependenciesQuery finds what depends on the object (upstream - Reports, Documents, etc).
// Uses runtime BFS traversal through [Prompt, Filter] to find reports.
// Returns paginated results (100 per page) with total count.
// Parameters:
//   - guids: array of GUIDs to look up
//   - offset: pagination offset (0, 100, 200, ...)
const UpstreamDependenciesQuery = `
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
  slicedSize > 100 as moreResults`

// =============================================================================
// Statistics Queries
// =============================================================================

// MetricsStatsQuery returns aggregate statistics for Metrics.
// Parameters:
//   - status: optional array of status values to filter
//   - team: optional team name to filter
const MetricsStatsQuery = `
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
  collect(DISTINCT n.parity_team) as teams`

// AttributesStatsQuery returns aggregate statistics for Attributes.
// Parameters:
//   - status: optional array of status values to filter
//   - team: optional team name to filter
const AttributesStatsQuery = `
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
  collect(DISTINCT n.parity_team) as teams`

// ObjectStatsQuery returns summary statistics for a specific object.
// Uses runtime BFS for fresh counts.
// Parameters:
//   - guid: the GUID of the object to analyze
const ObjectStatsQuery = `
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
  reportsByPriority`
