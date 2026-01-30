package mstr

import (
	"context"
	"log/slog"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/brunogc-cit/flow-microstrategy-mcp/internal/tools"
)

// GetReportsUsingMetricInput defines the input schema for the get-reports-using-metric tool.
type GetReportsUsingMetricInput struct {
	Guid          string   `json:"guid" jsonschema:"required,description=The GUID of the Metric to look up"`
	PriorityLevel []string `json:"priorityLevel,omitempty" jsonschema:"description=Filter reports by priority levels (e.g. ['P1 (Highest)','P2']). Use 'All Prioritized' for all."`
	BusinessArea  []string `json:"businessArea,omitempty" jsonschema:"description=Filter reports by business areas. Use 'All Areas' for all."`
}

// GetReportsUsingMetricSpec returns the MCP tool specification.
func GetReportsUsingMetricSpec() mcp.Tool {
	return mcp.NewTool("get-reports-using-metric",
		mcp.WithDescription(
			"Find all MicroStrategy Reports and Documents that use a specific Metric. "+
				"Returns report details including name, priority, business area, department, user count, and usage pattern. "+
				"Use this for impact analysis before migration.",
		),
		mcp.WithInputSchema[GetReportsUsingMetricInput](),
		mcp.WithTitleAnnotation("Get Reports Using Metric"),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(true),
	)
}

// GetReportsUsingMetricHandler returns a handler function.
func GetReportsUsingMetricHandler(deps *tools.ToolDependencies) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleGetReportsUsingMetric(ctx, request, deps)
	}
}

func handleGetReportsUsingMetric(ctx context.Context, request mcp.CallToolRequest, deps *tools.ToolDependencies) (*mcp.CallToolResult, error) {
	if deps.DBService == nil {
		errMessage := "Database service is not initialized"
		slog.Error(errMessage)
		return mcp.NewToolResultError(errMessage), nil
	}

	var args GetReportsUsingMetricInput
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
		"neodash_selected_guid":  []string{args.Guid},
		"neodash_priority_level": args.PriorityLevel,
		"neodash_business_area":  args.BusinessArea,
	}

	slog.Info("executing get-reports-using-metric query", "guid", args.Guid)

	records, err := deps.DBService.ExecuteReadQuery(ctx, ReportsUsingObjectsQuery, params)
	if err != nil {
		slog.Error("error executing query", "error", err)
		return mcp.NewToolResultError(err.Error()), nil
	}

	if len(records) == 0 {
		return mcp.NewToolResultText("No reports found using this Metric."), nil
	}

	response, err := deps.DBService.Neo4jRecordsToJSON(records)
	if err != nil {
		slog.Error("error formatting query results", "error", err)
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(response), nil
}
