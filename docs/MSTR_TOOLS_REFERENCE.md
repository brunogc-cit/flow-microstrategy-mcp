# MicroStrategy MCP Tools - Referência para o Time de Cypher

Este documento descreve as tools MCP existentes para consulta de metadados MicroStrategy, suas queries Cypher correspondentes, e informações relevantes para criação e otimização de novas queries.

## Índice

1. [Visão Geral das Tools](#visão-geral-das-tools)
2. [Tools e Queries Detalhadas](#tools-e-queries-detalhadas)
3. [Mapeamento de Perguntas dos Usuários](#mapeamento-de-perguntas-dos-usuários)
4. [Perguntas Ainda Não Respondidas](#perguntas-ainda-não-respondidas)
5. [Schema do Banco de Dados](#schema-do-banco-de-dados)
6. [Considerações para Otimização](#considerações-para-otimização)
7. [Diretrizes para Novas Queries](#diretrizes-para-novas-queries)

---

## Visão Geral das Tools

O sistema possui **12 tools MicroStrategy** organizadas em categorias:

### Consultas por GUID
| Tool | Descrição | Query Utilizada |
|------|-----------|-----------------|
| `get-metric-by-guid` | Detalhes de uma Métrica pelo GUID | `GetObjectDetailsQuery` |
| `get-attribute-by-guid` | Detalhes de um Atributo pelo GUID | `GetObjectDetailsQuery` |

### Busca com Filtros
| Tool | Descrição | Query Utilizada |
|------|-----------|-----------------|
| `search-metrics` | Busca Métricas com filtros | `SearchObjectsQuery` |
| `search-attributes` | Busca Atributos com filtros | `SearchObjectsQuery` |

### Reports/Dependentes
| Tool | Descrição | Query Utilizada |
|------|-----------|-----------------|
| `get-reports-using-metric` | Reports que usam uma Métrica | `ReportsUsingObjectsQuery` |
| `get-reports-using-attribute` | Reports que usam um Atributo | `ReportsUsingObjectsQuery` |

### Tabelas Fonte (Lineage)
| Tool | Descrição | Query Utilizada |
|------|-----------|-----------------|
| `get-metric-source-tables` | Tabelas fonte de uma Métrica | `SourceTablesQuery` |
| `get-attribute-source-tables` | Tabelas fonte de um Atributo | `SourceTablesQuery` |

### Dependências Downstream (do que depende)
| Tool | Descrição | Query Utilizada |
|------|-----------|-----------------|
| `get-metric-dependencies` | Do que a Métrica depende | `DownstreamDependenciesQuery` |
| `get-attribute-dependencies` | Do que o Atributo depende | `DownstreamDependenciesQuery` |

### Dependentes Upstream (o que depende)
| Tool | Descrição | Query Utilizada |
|------|-----------|-----------------|
| `get-metric-dependents` | O que depende da Métrica | `UpstreamDependenciesQuery` |
| `get-attribute-dependents` | O que depende do Atributo | `UpstreamDependenciesQuery` |

---

## Tools e Queries Detalhadas

### Query 1: `GetObjectDetailsQuery`

**Tools que usam:** `get-metric-by-guid`, `get-attribute-by-guid`

**Parâmetros:**
- `neodash_selected_guid` (array de strings): GUIDs dos objetos (suporta prefix matching com STARTS WITH)

**Retorno:** Tipo, GUID, Nome, Status, Group, SubGroup, Team, RAW, SERVE, SEMANTIC, EDWTable, EDWColumn, ADETable, ADEColumn, SemanticName, SemanticModel, DBEssential, PBEssential, Notes

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

**Tools que usam:** `search-metrics`, `search-attributes`

**Parâmetros:**
- `neodash_searchterm` (string): Termos de busca separados por vírgula
- `neodash_objecttype` (string): "Metric", "Attribute" ou "All Types"
- `neodash_priority_level` (array): Níveis de prioridade como "P1 (Highest)", "P2", etc.
- `neodash_business_area` (array): Áreas de negócio
- `neodash_status` (array): Valores de status de paridade
- `neodash_data_domain` (array): Domínios de dados

**Retorno:** Type, Priority, Name, Status, Team, Reports (contagem), Tables (contagem), GUID

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

**Tools que usam:** `get-reports-using-metric`, `get-reports-using-attribute`

**Parâmetros:**
- `neodash_selected_guid` (array de strings): GUIDs dos objetos
- `neodash_priority_level` (array): Níveis de prioridade
- `neodash_business_area` (array): Áreas de negócio

**Retorno:** Selected Item, Report Name, Priority, Area, Department, Users, Usage

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

**Tools que usam:** `get-metric-source-tables`, `get-attribute-source-tables`

**Parâmetros:**
- `neodash_selected_guid` (array de strings): GUIDs dos objetos

**Retorno:** Selected Item, Table Name, Table GUID

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

**Tools que usam:** `get-metric-dependencies`, `get-attribute-dependencies`

**Parâmetros:**
- `neodash_selected_guid` (array de strings): GUIDs dos objetos

**Retorno:** Nó original (n) e caminhos de dependência (downstream)

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

**Tools que usam:** `get-metric-dependents`, `get-attribute-dependents`

**Parâmetros:**
- `neodash_selected_guid` (array de strings): GUIDs dos objetos

**Retorno:** Nó original (n) e caminhos upstream (upstream) - limitado a 1000 paths

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

## Mapeamento de Perguntas dos Usuários

### Perguntas JÁ RESPONDIDAS pelas Tools Existentes

| Pergunta do Usuário | Tool Recomendada |
|---------------------|------------------|
| "Quais são os detalhes da métrica X?" | `get-metric-by-guid` |
| "Qual o status de paridade do atributo Y?" | `get-attribute-by-guid` |
| "Quais métricas estão relacionadas a 'revenue'?" | `search-metrics` |
| "Encontre atributos com status 'Not Started'" | `search-attributes` |
| "Quais reports usam a métrica Z?" | `get-reports-using-metric` |
| "Quais reports usam o atributo W?" | `get-reports-using-attribute` |
| "De quais tabelas a métrica X se alimenta?" | `get-metric-source-tables` |
| "Quais são as tabelas fonte do atributo Y?" | `get-attribute-source-tables` |
| "Do que a métrica X depende?" | `get-metric-dependencies` |
| "Qual a cadeia de cálculo do atributo Y?" | `get-attribute-dependencies` |
| "O que será afetado se eu mudar a métrica X?" | `get-metric-dependents` |
| "Quais objetos dependem do atributo Y?" | `get-attribute-dependents` |
| "Quais métricas P1 existem na área de Finance?" | `search-metrics` (com filtros) |
| "Liste atributos do domínio 'Sales' com status 'In Progress'" | `search-attributes` (com filtros) |

---

## Perguntas Ainda NÃO Respondidas

### Alta Prioridade (Frequentemente Requisitadas)

| Pergunta | Sugestão de Implementação |
|----------|--------------------------|
| "Qual é a fórmula/definição completa da métrica X?" | Nova query retornando `formula`, `expressions_json`, `raw_json` |
| "Quais métricas usam o atributo Y na sua fórmula?" | Traversal reverso de dependência específico |
| "Qual o mapeamento Power BI equivalente para métrica X?" | Enriquecer `GetObjectDetailsQuery` com mais campos PB |
| "Mostre o grafo completo de dependências da métrica X" | Visualização de grafo com profundidade configurável |
| "Quais Facts são usados pela métrica X?" | Traversal específico para Facts |
| "Compare duas métricas (X e Y) - diferenças" | Nova tool de comparação |

### Média Prioridade

| Pergunta | Sugestão de Implementação |
|----------|--------------------------|
| "Quais métricas não estão mapeadas para Power BI?" | Query com filtro `pb_semantic IS NULL` |
| "Liste todas as métricas de um determinado Team" | Adicionar filtro por Team no `search-metrics` |
| "Quais reports são mais críticos (mais usuários)?" | Nova query ordenando por `usage_users_count` |
| "Quais tabelas EDW são mais utilizadas?" | Agregação por tabela EDW |
| "Mostre métricas órfãs (sem dependentes)" | Query identificando objetos sem upstream |
| "Qual a cobertura de migração por área?" | Agregação de status por `usage_area` |

### Baixa Prioridade (Nice to Have)

| Pergunta | Sugestão de Implementação |
|----------|--------------------------|
| "Histórico de mudanças de status de uma métrica" | Requer campos de auditoria no grafo |
| "Quais métricas foram atualizadas esta semana?" | Requer campos de timestamp |
| "Sugira ordem de migração baseado em dependências" | Algoritmo topológico sobre o grafo |

---

## Schema do Banco de Dados

### Labels de Nós Comuns

| Label | Descrição |
|-------|-----------|
| `MSTRObject` | Label genérico para todos objetos MicroStrategy |
| `Metric` | Métricas (também tem label MSTRObject) |
| `Attribute` | Atributos (também tem label MSTRObject) |
| `Fact` | Facts |
| `LogicalTable` | Tabelas lógicas |
| `Report` | Reports (type em MSTRObject) |
| `GridReport` | Grid Reports (type em MSTRObject) |
| `Document` | Documentos (type em MSTRObject) |
| `Filter` | Filtros (type em MSTRObject) |
| `Prompt` | Prompts (type em MSTRObject) |
| `Column` | Colunas |
| `DataProduct` | Domínios/Produtos de dados |

### Relacionamentos

| Relacionamento | Descrição |
|----------------|-----------|
| `DEPENDS_ON` | Relação de dependência (A)-[:DEPENDS_ON]->(B) significa A depende de B |
| `BELONGS_TO` | Pertencimento a domínio/produto de dados |

### Propriedades Importantes

#### Em MSTRObject/Metric/Attribute:
```
guid                      - Identificador único
name                      - Nome do objeto
type                      - Tipo ('Metric', 'Attribute', 'Report', etc.)
parity_status            - Status de paridade original
updated_parity_status    - Status de paridade atualizado (tem precedência)
parity_group             - Grupo de paridade
parity_subgroup          - Subgrupo de paridade
parity_team              - Time responsável
parity_notes             - Notas de paridade
inherited_priority_level - Nível de prioridade herdado

-- Mapeamentos de dados --
db_raw                   - Databricks RAW
db_serve                 - Databricks SERVE
pb_semantic              - Power BI Semantic
edw_table                - Tabela EDW
edw_column               - Coluna EDW
ade_db_table             - Tabela ADE
ade_db_column            - Coluna ADE
pb_semantic_name         - Nome no modelo semântico PB
pb_semantic_model        - Modelo semântico PB
db_essential             - Databricks Essential
pb_essential             - Power BI Essential

-- Lineage (arrays de GUIDs) --
lineage_source_tables       - GUIDs das tabelas fonte
lineage_source_tables_count - Contagem de tabelas fonte
lineage_used_by_reports     - GUIDs dos reports que usam o objeto
```

#### Em Metric:
```
formula          - Fórmula da métrica (texto)
expressions_json - Expressões em JSON
raw_json         - JSON original completo
location         - Localização no projeto
```

#### Em Attribute:
```
forms_json       - Forms do atributo em JSON
location         - Localização no projeto
```

#### Em Report/GridReport/Document:
```
priority_level      - Nível de prioridade (1, 2, 3, etc.)
usage_area          - Área de uso/negócio
usage_department    - Departamento
usage_users_count   - Contagem de usuários
usage_consistency   - Consistência de uso
usage_volume        - Volume de uso
```

#### Em LogicalTable:
```
physical_table_name - Nome da tabela física
database_instance   - Instância do banco
```

---

## Considerações para Otimização

### Performance Atual

1. **`SearchObjectsQuery`** - Query mais complexa
   - Usa CALL subqueries para agregação
   - Múltiplos filtros opcionais com CASE WHEN
   - EXISTS subqueries para filtros de relacionamento
   - **Potencial otimização:** Índices em `type`, `guid`, `priority_level`, `usage_area`

2. **`DownstreamDependenciesQuery` / `UpstreamDependenciesQuery`**
   - Traversal variável de 1..10 níveis
   - ALL() predicate nos nós intermediários
   - **Potencial otimização:** Limitar profundidade, usar apoc.path se disponível

3. **`ReportsUsingObjectsQuery` / `SourceTablesQuery`**
   - Dependem de arrays pré-computados (`lineage_used_by_reports`, `lineage_source_tables`)
   - **Vantagem:** Arrays pré-computados aceleram lookups
   - **Desvantagem:** Requer manutenção da integridade dos arrays

### Índices Recomendados

```cypher
-- Índices existentes (verificar):
CREATE INDEX IF NOT EXISTS FOR (n:MSTRObject) ON (n.guid);
CREATE INDEX IF NOT EXISTS FOR (n:MSTRObject) ON (n.type);
CREATE INDEX IF NOT EXISTS FOR (n:MSTRObject) ON (n.priority_level);
CREATE INDEX IF NOT EXISTS FOR (n:MSTRObject) ON (n.usage_area);
CREATE INDEX IF NOT EXISTS FOR (n:Metric) ON (n.guid);
CREATE INDEX IF NOT EXISTS FOR (n:Attribute) ON (n.guid);
CREATE INDEX IF NOT EXISTS FOR (n:DataProduct) ON (n.name);

-- Índice composto para busca:
CREATE INDEX IF NOT EXISTS FOR (n:MSTRObject) ON (n.type, n.guid);
```

### Padrões de Uso dos Parâmetros

Todas as queries usam parâmetros com prefixo `neodash_` (compatibilidade com NeoDash):
- `neodash_selected_guid` - Array de GUIDs selecionados
- `neodash_searchterm` - Termo de busca
- `neodash_objecttype` - Tipo do objeto
- `neodash_priority_level` - Array de níveis de prioridade
- `neodash_business_area` - Array de áreas de negócio
- `neodash_status` - Array de status
- `neodash_data_domain` - Array de domínios

---

## Diretrizes para Novas Queries

### Padrão de Estrutura

```cypher
-- 1. Processamento de parâmetros com CASE WHEN
WITH CASE WHEN $param IS NULL THEN default ELSE processed_value END as paramName

-- 2. MATCH inicial com filtros básicos
MATCH (n:Label)
WHERE n.property = value

-- 3. Filtros condicionais
WHERE (filterVar IS NULL OR n.property IN filterVar)

-- 4. Agregações em CALL subqueries
CALL {
  WITH n
  MATCH pattern
  RETURN aggregated_result
}

-- 5. RETURN com campos padronizados
RETURN 
  n.type as Type,
  n.name as Name,
  n.guid as GUID
ORDER BY relevantField DESC
```

### Checklist para Novas Queries

- [ ] Usar parâmetros com prefixo `neodash_` para compatibilidade
- [ ] Tratar NULL/empty para todos os parâmetros opcionais
- [ ] Usar `effectiveStatus` pattern para status de paridade
- [ ] Limitar resultados de traversal (ex: `[0..1000]`)
- [ ] Incluir GUID nos resultados para permitir drill-down
- [ ] Ordenar resultados de forma significativa
- [ ] Testar com GUIDs reais antes de implementar

### Template para Nova Tool

1. Criar arquivo em `internal/tools/mstr/nome_da_tool.go`
2. Adicionar query em `internal/tools/mstr/queries.go`
3. Registrar em `internal/server/tools_register.go`
4. Adicionar em `manifest.json`

---

## Arquivos de Referência

| Arquivo | Descrição |
|---------|-----------|
| `internal/tools/mstr/queries.go` | Todas as queries Cypher |
| `internal/tools/mstr/*.go` | Implementação das tools |
| `internal/server/tools_register.go` | Registro das tools no servidor |
| `manifest.json` | Manifesto MCP com lista de tools |
| `queries/01.cypher` | Queries originais do NeoDash |
| `queries/neo4j-query-templates.md` | Templates adicionais de query |

---

## Histórico de Atualizações

| Data | Versão | Mudança |
|------|--------|---------|
| 2026-01-30 | 1.0.0 | Documento inicial com 12 tools MSTR |
| 2026-01-24 | - | Arrays de lineage agora contêm GUIDs puros (sem formatação) |
