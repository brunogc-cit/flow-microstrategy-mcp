package mstr

import (
	"context"
	"log/slog"

	"github.com/brunogc-cit/flow-microstrategy-mcp/internal/tools"
	"github.com/mark3labs/mcp-go/mcp"
)

// GetMetricDependentsInput defines the input schema for the get-metric-dependents tool.
type GetMetricDependentsInput struct {
	Guid   string `json:"guid" jsonschema:"required,description=GUID of the Metric to analyze"`
	Offset int    `json:"offset,omitempty" jsonschema:"description=Pagination offset (0, 100, 200...). Default 0."`
}

// GetMetricDependentsSpec returns the MCP tool specification.
func GetMetricDependentsSpec() mcp.Tool {
	return mcp.NewTool("get-metric-dependents",
		mcp.WithDescription(
			"Find all Reports, GridReports, and Documents that depend on a Metric (upstream/inbound dependencies). "+
				"Traverses the dependency graph through Prompts and Filters (up to 10 levels). "+
				"Returns for each report: name, guid, type, priority, area, department, userCount. "+
				"Also returns 'totalReports' count for the full dataset. "+
				"PAGINATION: Returns 100 reports per page. Use 'offset' to paginate. "+
				"Response includes 'moreResults' boolean. "+
				"Use for impact analysis before modifying or deprecating a metric.",
		),
		mcp.WithInputSchema[GetMetricDependentsInput](),
		mcp.WithTitleAnnotation("Get Metric Dependents"),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(true),
	)
}

// GetMetricDependentsHandler returns a handler function.
func GetMetricDependentsHandler(deps *tools.ToolDependencies) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleGetMetricDependents(ctx, request, deps)
	}
}

func handleGetMetricDependents(ctx context.Context, request mcp.CallToolRequest, deps *tools.ToolDependencies) (*mcp.CallToolResult, error) {
	if deps.DBService == nil {
		errMessage := "Database service is not initialized"
		slog.Error(errMessage)
		return mcp.NewToolResultError(errMessage), nil
	}

	var args GetMetricDependentsInput
	if err := request.BindArguments(&args); err != nil {
		slog.Error("error binding arguments", "error", err)
		return mcp.NewToolResultError(err.Error()), nil
	}

	if args.Guid == "" {
		errMessage := "guid parameter is required"
		slog.Error(errMessage)
		return mcp.NewToolResultError(errMessage), nil
	}

	params := map[string]any{
		"guids":  []string{args.Guid},
		"offset": args.Offset,
	}

	slog.Info("executing get-metric-dependents query", "guid", args.Guid, "offset", args.Offset)

	records, err := deps.DBService.ExecuteReadQuery(ctx, UpstreamDependenciesQuery, params)
	if err != nil {
		slog.Error("error executing query", "error", err)
		return mcp.NewToolResultError(err.Error()), nil
	}

	if len(records) == 0 {
		return mcp.NewToolResultText(`{"objectName": "", "objectGUID": "", "objectType": "", "totalReports": 0, "reports": [], "moreResults": false}`), nil
	}

	response, err := deps.DBService.Neo4jRecordsToJSON(records)
	if err != nil {
		slog.Error("error formatting query results", "error", err)
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(response), nil
}
