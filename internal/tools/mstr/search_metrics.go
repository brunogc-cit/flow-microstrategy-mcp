package mstr

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/brunogc-cit/flow-microstrategy-mcp/internal/tools"
	"github.com/mark3labs/mcp-go/mcp"
)

// SearchMetricsInput defines the input parameters for the search-metrics tool
type SearchMetricsInput struct {
	Query  string   `json:"query" jsonschema:"required,description=GUID (full or partial 8+ chars) or name search term"`
	Status []string `json:"status,omitempty" jsonschema:"description=Filter by parity status: Complete, Planned, Not Planned"`
	Offset int      `json:"offset,omitempty" jsonschema:"default=0,description=Skip first N results for pagination"`
}

const searchMetricsQuery = `
// Search for Metrics by GUID or name
// $query: GUID (full/partial) or name search term
// $status: optional parity status filter (applies to effective status)
// $offset: pagination offset (0, 100, 200, ...)
//
// Design Decision: Returns ALL Metrics matching the query, including those without
// parity mapping. Objects not in the parity matrix get status "No Status".
// The updated_parity_status property (from ADO backlog sync) takes precedence
// over the computed parity_status.

// Determine if query looks like a GUID (hex chars, 8+ length)
WITH $query as query,
     $query =~ '^[A-Fa-f0-9]{8,}$' as isGuidLike

MATCH (n:Metric)
WHERE n.guid IS NOT NULL
  AND (
    // GUID match: exact or partial (starts with)
    (isGuidLike AND (n.guid = query OR n.guid STARTS WITH toUpper(query)))
    OR
    // Name match: case-insensitive contains
    (NOT isGuidLike AND toLower(n.name) CONTAINS toLower(query))
  )
  AND ($status IS NULL OR COALESCE(n.updated_parity_status, n.parity_status) IN $status)

// Compute effective values (updated_ properties take precedence)
WITH n,
     COALESCE(n.updated_parity_status, n.parity_status, 'No Status') as effectiveStatus

ORDER BY n.name ASC
SKIP $offset
LIMIT 101  // Fetch 101 to determine if more results exist

// Collect results with all properties (updated_ values take precedence)
WITH collect({
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
}) as fetched

// Return first 100; moreResults=true if 101st exists
RETURN 
  fetched[0..100] as results,
  size(fetched) > 100 as moreResults
`

// SearchMetricsSpec returns the MCP tool definition for search-metrics
func SearchMetricsSpec() mcp.Tool {
	return mcp.NewTool("search-metrics",
		mcp.WithDescription(
			"Find Metrics by GUID or name. Accepts full GUIDs, partial GUIDs (8+ chars), or name search terms.\n\n"+
				"USE FOR:\n"+
				"- Finding a metric by its full GUID: search-metrics(query=\"2F00974D44E1D0D24CA344ABD872806A\")\n"+
				"- Finding metrics by partial GUID (8+ chars): search-metrics(query=\"2F00974D\")\n"+
				"- Searching metrics by name: search-metrics(query=\"Retail Sales\")\n"+
				"- Filtering by parity status: search-metrics(query=\"sales\", status=[\"Complete\"])\n"+
				"- Getting metric details (formula, mappings, counts) before tracing lineage\n\n"+
				"DO NOT USE FOR:\n"+
				"- Partial GUIDs with less than 8 characters (too ambiguous)\n"+
				"- Lineage tracing (use trace-metric with the GUID instead)\n"+
				"- Searching Attributes (use search-attributes instead)\n\n"+
				"PAGINATION: Returns 100 results. If moreResults=true, call again with offset+100.",
		),
		mcp.WithInputSchema[SearchMetricsInput](),
		mcp.WithTitleAnnotation("Search for Metrics by GUID or name"),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(true),
	)
}

// SearchMetricsHandler returns the handler function for the search-metrics tool
func SearchMetricsHandler(deps *tools.ToolDependencies) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleSearchMetrics(ctx, deps, request)
	}
}

func handleSearchMetrics(ctx context.Context, deps *tools.ToolDependencies, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if deps.DBService == nil {
		errMessage := "Database service is not initialized"
		slog.Error(errMessage)
		return mcp.NewToolResultError(errMessage), nil
	}

	// Parse input
	var input SearchMetricsInput
	if err := request.BindArguments(&input); err != nil {
		slog.Error("error binding arguments", "error", err)
		return mcp.NewToolResultError(fmt.Sprintf("Invalid input: %v", err)), nil
	}

	// Validate required field
	if input.Query == "" {
		return mcp.NewToolResultError("query parameter is required"), nil
	}

	// Build parameters
	params := map[string]any{
		"query":  input.Query,
		"offset": input.Offset,
	}

	// Handle status filter - nil if empty, otherwise the array
	if len(input.Status) > 0 {
		params["status"] = input.Status
	} else {
		params["status"] = nil
	}

	records, err := deps.DBService.ExecuteReadQuery(ctx, searchMetricsQuery, params)
	if err != nil {
		slog.Error("failed to execute search-metrics query", "error", err)
		return mcp.NewToolResultError(fmt.Sprintf("Query execution failed: %v", err)), nil
	}

	response, err := deps.DBService.Neo4jRecordsToJSON(records)
	if err != nil {
		slog.Error("failed to format search-metrics results to JSON", "error", err)
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(response), nil
}
