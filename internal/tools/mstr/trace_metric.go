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
}

// traceMetricDownstreamQuery traces downstream lineage (toward reports - who uses this metric?)
const traceMetricDownstreamQuery = `
// Trace DOWNSTREAM lineage for a Metric (toward reports)
// $guid: Full GUID of the Metric
//
// Design Decision: Split upstream/downstream to avoid returning too many objects.
// Downstream = follow reverse dependencies toward consumers (Reports).
// Uses pre-computed lineage_used_by_reports for performance.

MATCH (n:Metric {guid: $guid})

// Get effective status (updated takes precedence)
WITH n,
     COALESCE(n.updated_parity_status, n.parity_status, 'No Status') as effectiveStatus

// Get reports using this metric (from pre-computed lineage)
OPTIONAL MATCH (r:MSTRObject)
WHERE r.guid IN COALESCE(n.lineage_used_by_reports, [])
WITH n, effectiveStatus, collect(DISTINCT {
  name: r.name,
  guid: r.guid,
  type: r.type,
  priority: r.priority_level,
  area: r.usage_area
})[0..50] as reports

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
    reportCount: COALESCE(n.lineage_used_by_reports_count, 0),
    tableCount: COALESCE(n.lineage_source_tables_count, 0),
    ado_link: COALESCE(n.updated_ado_link, n.ado_link)
  },
  direction: 'downstream',
  reports: reports
} as result
`

// traceMetricUpstreamQuery traces upstream lineage (toward tables - where does data come from?)
const traceMetricUpstreamQuery = `
// Trace UPSTREAM lineage for a Metric (toward source tables)
// $guid: Full GUID of the Metric
//
// Design Decision: Split upstream/downstream to avoid returning too many objects.
// Upstream = follow dependencies toward data sources (Tables, Facts).
// Uses pre-computed lineage_source_tables for performance.

MATCH (n:Metric {guid: $guid})

// Get effective status (updated takes precedence)
WITH n,
     COALESCE(n.updated_parity_status, n.parity_status, 'No Status') as effectiveStatus

// Get source tables (from pre-computed lineage)
OPTIONAL MATCH (t:MSTRObject)
WHERE t.guid IN COALESCE(n.lineage_source_tables, [])
WITH n, effectiveStatus, collect(DISTINCT {
  name: t.name,
  guid: t.guid,
  type: t.type,
  physicalTable: t.physical_table_name,
  database: t.database_instance
})[0..50] as tables

// Get direct dependencies (1-hop toward sources)
OPTIONAL MATCH (n)-[:DEPENDS_ON]->(dep:MSTRObject)
WITH n, effectiveStatus, tables, collect(DISTINCT {
  name: dep.name,
  guid: dep.guid,
  type: dep.type,
  formula: dep.formula
})[0..50] as dependencies

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
    reportCount: COALESCE(n.lineage_used_by_reports_count, 0),
    tableCount: COALESCE(n.lineage_source_tables_count, 0),
    ado_link: COALESCE(n.updated_ado_link, n.ado_link)
  },
  direction: 'upstream',
  tables: tables,
  dependencies: dependencies
} as result
`

// TraceMetricSpec returns the MCP tool definition for trace-metric
func TraceMetricSpec() mcp.Tool {
	return mcp.NewTool("trace-metric",
		mcp.WithDescription(
			"Trace lineage of a Metric in a specific direction.\n\n"+
				"DIRECTION:\n"+
				"- 'downstream': Find reports that USE this metric (M/A → Reports)\n"+
				"- 'upstream': Find source tables and dependencies (M/A → Tables)\n\n"+
				"WHY SPLIT? High-connectivity metrics (e.g., 'Retail Sales Value' with 6000+ reports) "+
				"would return too much data in a single call. Choose the direction relevant to your query.\n\n"+
				"CORRECT USAGE:\n"+
				"- First search: search-metrics(query=\"Retail Sales\")\n"+
				"- Then trace downstream: trace-metric(guid=\"2F00974D...\", direction=\"downstream\")\n"+
				"- Or trace upstream: trace-metric(guid=\"2F00974D...\", direction=\"upstream\")\n\n"+
				"RETURNS:\n"+
				"- downstream: metric details + reports[] (max 50)\n"+
				"- upstream: metric details + tables[] (max 50) + dependencies[] (max 50)",
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
		"guid": input.GUID,
	}

	slog.Info("executing trace-metric query", "guid", input.GUID, "direction", input.Direction)

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
