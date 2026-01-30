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

------------------------------------------------

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


------------------------------------------------

// NOTE: lineage arrays now contain pure GUIDs (2026-01-24)
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


------------------------------------------------

// NOTE: lineage arrays now contain pure GUIDs (2026-01-24)
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

------------------------------------------------

WITH $neodash_selected_guid as selectedGuids
WHERE selectedGuids IS NOT NULL AND size(selectedGuids) > 0
MATCH (n:MSTRObject)
WHERE n.guid IN selectedGuids
OPTIONAL MATCH downstream = (n)-[:DEPENDS_ON*1..10]->(d:MSTRObject)
WHERE ALL(mid IN nodes(downstream)[1..-1] WHERE mid.type IN ['Fact', 'Metric', 'Attribute', 'Column'])
RETURN n, downstream

------------------------------------------------

WITH $neodash_selected_guid as selectedGuids
WHERE selectedGuids IS NOT NULL AND size(selectedGuids) > 0
MATCH (n:MSTRObject)
WHERE n.guid IN selectedGuids
OPTIONAL MATCH upstream = (r:MSTRObject)-[:DEPENDS_ON*1..10]->(n)
WHERE r.type IN ['Report', 'GridReport', 'Document']
WITH n, collect(upstream)[0..1000] as paths
UNWIND paths as upstream
RETURN n, upstream