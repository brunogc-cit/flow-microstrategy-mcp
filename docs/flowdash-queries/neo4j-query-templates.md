# Neo4j Query Templates: Metrics and Attributes

This guide provides parameterized questions and Cypher templates for querying
MicroStrategy metadata stored in Neo4j. It focuses on metrics and attributes,
including formulas, dependencies, and related tables.

## Schema snapshot (relevant to this guide)

- **Node labels (common)**: `Metric`, `Attribute`, `Fact`, `LogicalTable`, `Report`,
  `Prompt`, `Filter`, `DerivedMetric`, `MSTRObject`
- **Relationships (common)**: `DEPENDS_ON`, `BELONGS_TO`
- **Key properties**
  - `Metric`: `name`, `guid`, `formula`, `expressions_json`, `location`, `type`
  - `Attribute`: `name`, `guid`, `forms_json`, `edw_table`, `edw_column`,
    `ade_db_table`, `ade_db_column`, `location`, `type`
  - `Fact`: `name`, `guid`, `expressions_json`, `type`
  - `LogicalTable`: `name`, `guid`, `physical_table_name`, `database_instance`

## How to use placeholders

- Replace placeholders with values, for example `{metric_name}` or `{metric_guid}`.
- Prefer `{metric_guid}` or `{attribute_guid}` for precise matches.
- Use `{depth}` as a small integer (e.g., `1`, `2`, `3`) for dependency traversal.
- If using names, the templates use `toLower(...)` to make matching case-insensitive.

## Metric questions and templates

### How is the formula for `{metric_name}` calculated?

**By name**
```
MATCH (m:Metric)
WHERE toLower(m.name) = toLower('{metric_name}')
RETURN m.name, m.guid, m.formula, m.expressions_json, m.raw_json
```

**Example question (real data)**: How is the formula for `Retail Sales Units - Ignore Discount` calculated?
```
MATCH (m:Metric)
WHERE toLower(m.name) = toLower('Retail Sales Units - Ignore Discount')
RETURN m.name, m.guid, m.formula, m.expressions_json, m.raw_json
```

**By guid**
```
MATCH (m:Metric {guid: '{metric_guid}'})
RETURN m.name, m.guid, m.formula, m.expressions_json, m.raw_json
```

**Example question (real data)**: How is the formula for metric `E282FA144B8889DF9D098F872548FD55` calculated?
```
MATCH (m:Metric {guid: 'E282FA144B8889DF9D098F872548FD55'})
RETURN m.name, m.guid, m.formula, m.expressions_json, m.raw_json
```

### What does metric `{metric_name}` depend on (depth `{depth}`)?

```
MATCH (m:Metric)
WHERE m.guid = '{metric_guid}'
   OR toLower(m.name) = toLower('{metric_name}')
MATCH (m)-[:DEPENDS_ON*1..{depth}]->(dep)
RETURN DISTINCT
  labels(dep) AS dep_labels,
  dep.type AS dep_type,
  dep.name AS name,
  dep.guid AS guid
ORDER BY dep_labels, name
```

**Example question (real data)**: What does metric `Retail Sales Units - Ignore Discount` depend on (depth `2`)?
```
MATCH (m:Metric)
WHERE m.guid = 'E282FA144B8889DF9D098F872548FD55'
   OR toLower(m.name) = toLower('Retail Sales Units - Ignore Discount')
MATCH (m)-[:DEPENDS_ON*1..2]->(dep)
RETURN DISTINCT
  labels(dep) AS dep_labels,
  dep.type AS dep_type,
  dep.name AS name,
  dep.guid AS guid
ORDER BY dep_labels, name
```

### Show the dependency tree for `{metric_name}`

```
MATCH (m:Metric)
WHERE m.guid = '{metric_guid}'
   OR toLower(m.name) = toLower('{metric_name}')
MATCH path = (m)-[:DEPENDS_ON*1..{depth}]->(dep)
RETURN m, path
```

**Example question (real data)**: Show the dependency tree for metric `Retail Sales Units - Ignore Discount` (depth `2`).
```
MATCH (m:Metric)
WHERE m.guid = 'E282FA144B8889DF9D098F872548FD55'
   OR toLower(m.name) = toLower('Retail Sales Units - Ignore Discount')
MATCH path = (m)-[:DEPENDS_ON*1..2]->(dep)
RETURN m, path
```

### Which logical tables are used by metric `{metric_name}`?

```
MATCH (m:Metric)
WHERE m.guid = '{metric_guid}'
   OR toLower(m.name) = toLower('{metric_name}')
MATCH (m)-[:DEPENDS_ON*1..5]->(f:Fact)
MATCH (f)-[:DEPENDS_ON]->(t:LogicalTable)
RETURN DISTINCT t.name, t.guid, t.physical_table_name, t.database_instance
ORDER BY t.name
```

**Example question (real data)**: Which logical tables are used by metric `Retail Returns Sales Value Exc Tax (Return FC) LY`?
```
MATCH (m:Metric)
WHERE m.guid = '3502F30C474B9BE40C6D479B5F9F31BE'
   OR toLower(m.name) = toLower('Retail Returns Sales Value Exc Tax (Return FC) LY')
MATCH (m)-[:DEPENDS_ON*1..5]->(f:Fact)
MATCH (f)-[:DEPENDS_ON]->(t:LogicalTable)
RETURN DISTINCT t.name, t.guid, t.physical_table_name, t.database_instance
ORDER BY t.name
```

### Which attributes are used by metric `{metric_name}`?

```
MATCH (m:Metric)
WHERE m.guid = '{metric_guid}'
   OR toLower(m.name) = toLower('{metric_name}')
MATCH (m)-[:DEPENDS_ON*1..{depth}]->(a:Attribute)
RETURN DISTINCT a.name, a.guid, a.location
ORDER BY a.name
```

**Example question (real data)**: Which attributes are used by metric `Retail Sales Units - Ignore Discount` (depth `2`)?
```
MATCH (m:Metric)
WHERE m.guid = 'E282FA144B8889DF9D098F872548FD55'
   OR toLower(m.name) = toLower('Retail Sales Units - Ignore Discount')
MATCH (m)-[:DEPENDS_ON*1..2]->(a:Attribute)
RETURN DISTINCT a.name, a.guid, a.location
ORDER BY a.name
```

## Attribute questions and templates

### What is the definition of attribute `{attribute_name}`?

**By name**
```
MATCH (a:Attribute)
WHERE toLower(a.name) = toLower('{attribute_name}')
RETURN a.name, a.guid, a.forms_json, a.edw_table, a.edw_column,
       a.ade_db_table, a.ade_db_column, a.location
```

**Example question (real data)**: What is the definition of attribute `Weight`?
```
MATCH (a:Attribute)
WHERE toLower(a.name) = toLower('Weight')
RETURN a.name, a.guid, a.forms_json, a.edw_table, a.edw_column,
       a.ade_db_table, a.ade_db_column, a.location
```

**By guid**
```
MATCH (a:Attribute {guid: '{attribute_guid}'})
RETURN a.name, a.guid, a.forms_json, a.edw_table, a.edw_column,
       a.ade_db_table, a.ade_db_column, a.location
```

**Example question (real data)**: What is the definition of attribute `C2C379B547797C4DC6CC7996B7B9A21D`?
```
MATCH (a:Attribute {guid: 'C2C379B547797C4DC6CC7996B7B9A21D'})
RETURN a.name, a.guid, a.forms_json, a.edw_table, a.edw_column,
       a.ade_db_table, a.ade_db_column, a.location
```

### Which metrics or reports use attribute `{attribute_name}`?

```
MATCH (dep)-[:DEPENDS_ON]->(a:Attribute)
WHERE a.guid = '{attribute_guid}'
   OR toLower(a.name) = toLower('{attribute_name}')
RETURN DISTINCT labels(dep) AS dep_labels, dep.name, dep.guid, dep.type
ORDER BY dep_labels, dep.name
```

**Example question (real data)**: Which metrics or reports use attribute `Discount Code`?
```
MATCH (dep)-[:DEPENDS_ON]->(a:Attribute)
WHERE a.guid = '3D9380A841D05DDE69C2FC858E99739C'
   OR toLower(a.name) = toLower('Discount Code')
RETURN DISTINCT labels(dep) AS dep_labels, dep.name, dep.guid, dep.type
ORDER BY dep_labels, dep.name
```

### Which logical tables are associated with attribute `{attribute_name}` (via metrics and facts)?

```
MATCH (a:Attribute)
WHERE a.guid = '{attribute_guid}'
   OR toLower(a.name) = toLower('{attribute_name}')
MATCH (a)<-[:DEPENDS_ON]-(m:Metric)-[:DEPENDS_ON]->(f:Fact)-[:DEPENDS_ON]->(t:LogicalTable)
RETURN DISTINCT t.name, t.guid, t.physical_table_name, t.database_instance
ORDER BY t.name
```

**Example question (real data)**: Which logical tables are associated with attribute `Date` (via metrics and facts)?
```
MATCH (a:Attribute)
WHERE a.guid = '29B18CBE4323BD3D4D33AD9E718D4E79'
   OR toLower(a.name) = toLower('Date')
MATCH (a)<-[:DEPENDS_ON]-(m:Metric)-[:DEPENDS_ON]->(f:Fact)-[:DEPENDS_ON]->(t:LogicalTable)
RETURN DISTINCT t.name, t.guid, t.physical_table_name, t.database_instance
ORDER BY t.name
```
## Quick validation checklist

- Run 2–3 metric queries using real `guid` values and confirm `formula` returns.
- Run one dependency query with `depth = 2` and confirm mixed types appear.
- Run one “logical tables” query and confirm `LogicalTable` matches known tables.

## Notes and tips

- Some nodes carry both a specific label (e.g., `Metric`) and `MSTRObject`.
  The templates use the specific label for clarity.
- If a metric lacks `formula`, check `expressions_json` for the calculated logic.
- Use smaller `{depth}` values first to keep results manageable.
