package mstr

import (
	"context"
	"log/slog"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/brunogc-cit/flow-microstrategy-mcp/internal/tools"
)

// SearchMetricsInput defines the input schema for the search-metrics tool.
type SearchMetricsInput struct {
	SearchTerm    string   `json:"searchTerm,omitempty" jsonschema:"description=Comma-separated search terms to filter by name or GUID (e.g. 'revenue,cost')"`
	PriorityLevel []string `json:"priorityLevel,omitempty" jsonschema:"description=Filter by priority levels (e.g. ['P1 (Highest)','P2']). Use 'All Prioritized' for all."`
	BusinessArea  []string `json:"businessArea,omitempty" jsonschema:"description=Filter by business areas. Use 'All Areas' for all."`
	Status        []string `json:"status,omitempty" jsonschema:"description=Filter by parity status values. Use 'All Status' for all."`
	DataDomain    []string `json:"dataDomain,omitempty" jsonschema:"description=Filter by data domains. Use 'All Domains' for all."`
}

// SearchMetricsSpec returns the MCP tool specification.
func SearchMetricsSpec() mcp.Tool {
	return mcp.NewTool("search-metrics",
		mcp.WithDescription(
			"Search for MicroStrategy Metrics that are used by prioritized reports. "+
				"Returns metrics with type, priority, name, status, team, report count, and source table count. "+
				"Use this to find which metrics are most impactful for Power BI migration planning. "+
				"All filter parameters are optional.",
		),
		mcp.WithInputSchema[SearchMetricsInput](),
		mcp.WithTitleAnnotation("Search Metrics"),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(true),
	)
}

// SearchMetricsHandler returns a handler function for the search-metrics tool.
func SearchMetricsHandler(deps *tools.ToolDependencies) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleSearchMetrics(ctx, request, deps)
	}
}

func handleSearchMetrics(ctx context.Context, request mcp.CallToolRequest, deps *tools.ToolDependencies) (*mcp.CallToolResult, error) {
	if deps.DBService == nil {
		errMessage := "Database service is not initialized"
		slog.Error(errMessage)
		return mcp.NewToolResultError(errMessage), nil
	}

	var args SearchMetricsInput
	if err := request.BindArguments(&args); err != nil {
		slog.Error("error binding arguments", "error", err)
		return mcp.NewToolResultError(err.Error()), nil
	}

	params := map[string]any{
		"neodash_searchterm":     args.SearchTerm,
		"neodash_objecttype":     "Metric", // Fixed to Metric
		"neodash_priority_level": args.PriorityLevel,
		"neodash_business_area":  args.BusinessArea,
		"neodash_status":         args.Status,
		"neodash_data_domain":    args.DataDomain,
	}

	slog.Info("executing search-metrics query", "searchTerm", args.SearchTerm)

	records, err := deps.DBService.ExecuteReadQuery(ctx, SearchObjectsQuery, params)
	if err != nil {
		slog.Error("error executing search-metrics query", "error", err)
		return mcp.NewToolResultError(err.Error()), nil
	}

	if len(records) == 0 {
		return mcp.NewToolResultText("No Metrics found matching the specified criteria."), nil
	}

	response, err := deps.DBService.Neo4jRecordsToJSON(records)
	if err != nil {
		slog.Error("error formatting query results", "error", err)
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(response), nil
}
