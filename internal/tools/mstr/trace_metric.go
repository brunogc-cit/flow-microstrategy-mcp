package mstr

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/brunogc-cit/flow-microstrategy-mcp/internal/tools"
	"github.com/mark3labs/mcp-go/mcp"
)

// TraceMetricInput defines the input parameters for the trace-metric tool
type TraceMetricInput struct {
	GUID      string `json:"guid" jsonschema:"required,description=Full GUID of the Metric to trace"`
	Direction string `json:"direction" jsonschema:"required,enum=downstream,enum=upstream,description=Trace direction: 'downstream' (toward reports - who uses this?) or 'upstream' (toward tables - where does data come from?)"`
	Offset    int    `json:"offset,omitempty" jsonschema:"default=0,description=Skip first N results for pagination"`
}

// traceMetricDownstreamQuery traces downstream lineage (toward reports - who uses this metric?)
// Uses LIVE graph traversal - finds objects that depend on this metric via DEPENDS_ON relationships
const traceMetricDownstreamQuery = `
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
// Reports connect to metrics through various paths (direct, via Prompts, Filters, etc.)
// Filter: Only prioritized reports (priority_level IS NOT NULL) - aligns with dashboard
OPTIONAL MATCH (report)-[:DEPENDS_ON*1..10]->(n)
WHERE report.type IN ['Report', 'GridReport', 'Document']
  AND report.priority_level IS NOT NULL

WITH n, effectiveStatus, report
ORDER BY report.name ASC
SKIP $offset
LIMIT 101

// Collect paginated reports (filter out null results from OPTIONAL MATCH)
WITH n, effectiveStatus, [r IN collect(DISTINCT {
  name: report.name,
  guid: report.guid,
  type: report.type,
  priority: report.priority_level,
  area: report.usage_area
}) WHERE r.guid IS NOT NULL] as fetched

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
`

// traceMetricUpstreamQuery traces upstream lineage (toward tables - where does data come from?)
// Uses LIVE graph traversal - follows DEPENDS_ON relationships toward data sources
const traceMetricUpstreamQuery = `
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

// Find source tables via live traversal (Metric -> ... -> Fact -> LogicalTable)
OPTIONAL MATCH (n)-[:DEPENDS_ON*1..10]->(t)
WHERE t.type IN ['LogicalTable', 'Table']

WITH n, effectiveStatus, t
ORDER BY t.name ASC
SKIP $offset
LIMIT 101

// Collect paginated tables (filter out null results from OPTIONAL MATCH)
WITH n, effectiveStatus, [tbl IN collect(DISTINCT {
  name: t.name,
  guid: t.guid,
  type: t.type,
  physicalTable: t.physical_table_name,
  database: t.database_instance
}) WHERE tbl.guid IS NOT NULL] as fetchedTables

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
`

// TraceMetricSpec returns the MCP tool definition for trace-metric
func TraceMetricSpec() mcp.Tool {
	return mcp.NewTool("trace-metric",
		mcp.WithDescription(
			"Trace lineage of a Metric in a specific direction using live graph traversal.\n\n"+
				"DIRECTION:\n"+
				"- 'downstream': Find PRIORITIZED reports that USE this metric (live BFS traversal)\n"+
				"- 'upstream': Find source tables and dependencies (live BFS traversal)\n\n"+
				"NOTE: Downstream only returns reports with priority_level (prioritized reports).\n\n"+
				"CORRECT USAGE:\n"+
				"- First search: search-metrics(query=\"Retail Sales\")\n"+
				"- Then trace downstream: trace-metric(guid=\"2F00974D...\", direction=\"downstream\")\n"+
				"- Or trace upstream: trace-metric(guid=\"2F00974D...\", direction=\"upstream\")\n\n"+
				"PAGINATION: Returns 100 results. If moreResults=true, call again with offset+100.",
		),
		mcp.WithInputSchema[TraceMetricInput](),
		mcp.WithTitleAnnotation("Trace Metric lineage by direction"),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(true),
	)
}

// TraceMetricHandler returns the handler function for the trace-metric tool
func TraceMetricHandler(deps *tools.ToolDependencies) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleTraceMetric(ctx, deps, request)
	}
}

func handleTraceMetric(ctx context.Context, deps *tools.ToolDependencies, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if deps.DBService == nil {
		errMessage := "Database service is not initialized"
		slog.Error(errMessage)
		return mcp.NewToolResultError(errMessage), nil
	}

	// Parse input
	var input TraceMetricInput
	if err := request.BindArguments(&input); err != nil {
		slog.Error("error binding arguments", "error", err)
		return mcp.NewToolResultError(fmt.Sprintf("Invalid input: %v", err)), nil
	}

	// Validate required fields
	if input.GUID == "" {
		return mcp.NewToolResultError("guid parameter is required"), nil
	}
	if input.Direction == "" {
		return mcp.NewToolResultError("direction parameter is required (must be 'downstream' or 'upstream')"), nil
	}

	// Select query based on direction
	var query string
	switch input.Direction {
	case "downstream":
		query = traceMetricDownstreamQuery
	case "upstream":
		query = traceMetricUpstreamQuery
	default:
		return mcp.NewToolResultError(fmt.Sprintf("Invalid direction '%s': must be 'downstream' or 'upstream'", input.Direction)), nil
	}

	params := map[string]any{
		"guid":   input.GUID,
		"offset": input.Offset,
	}

	slog.Info("executing trace-metric query", "guid", input.GUID, "direction", input.Direction, "offset", input.Offset)

	records, err := deps.DBService.ExecuteReadQuery(ctx, query, params)
	if err != nil {
		slog.Error("failed to execute trace-metric query", "error", err)
		return mcp.NewToolResultError(fmt.Sprintf("Query execution failed: %v", err)), nil
	}

	// Check if metric was found
	if len(records) == 0 {
		return mcp.NewToolResultError(fmt.Sprintf("Metric with GUID %s not found", input.GUID)), nil
	}

	response, err := deps.DBService.Neo4jRecordsToJSON(records)
	if err != nil {
		slog.Error("failed to format trace-metric results to JSON", "error", err)
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(response), nil
}
