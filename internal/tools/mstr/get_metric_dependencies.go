package mstr

import (
	"context"
	"log/slog"

	"github.com/brunogc-cit/flow-microstrategy-mcp/internal/tools"
	"github.com/mark3labs/mcp-go/mcp"
)

// GetMetricDependenciesInput defines the input schema for the get-metric-dependencies tool.
type GetMetricDependenciesInput struct {
	Guid string `json:"guid" jsonschema:"required,description=The GUID of the Metric to analyze"`
}

// GetMetricDependenciesSpec returns the MCP tool specification.
func GetMetricDependenciesSpec() mcp.Tool {
	return mcp.NewTool("get-metric-dependencies",
		mcp.WithDescription(
			"Find what a MicroStrategy Metric depends on (downstream dependencies). "+
				"Traverses the dependency chain up to 10 levels deep showing Facts, other Metrics, Attributes, and Columns. "+
				"Use this to understand the complete calculation chain of a metric.",
		),
		mcp.WithInputSchema[GetMetricDependenciesInput](),
		mcp.WithTitleAnnotation("Get Metric Dependencies"),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(true),
	)
}

// GetMetricDependenciesHandler returns a handler function.
func GetMetricDependenciesHandler(deps *tools.ToolDependencies) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleGetMetricDependencies(ctx, request, deps)
	}
}

func handleGetMetricDependencies(ctx context.Context, request mcp.CallToolRequest, deps *tools.ToolDependencies) (*mcp.CallToolResult, error) {
	if deps.DBService == nil {
		errMessage := "Database service is not initialized"
		slog.Error(errMessage)
		return mcp.NewToolResultError(errMessage), nil
	}

	var args GetMetricDependenciesInput
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
		"neodash_selected_guid": []string{args.Guid},
	}

	slog.Info("executing get-metric-dependencies query", "guid", args.Guid)

	records, err := deps.DBService.ExecuteReadQuery(ctx, DownstreamDependenciesQuery, params)
	if err != nil {
		slog.Error("error executing query", "error", err)
		return mcp.NewToolResultError(err.Error()), nil
	}

	if len(records) == 0 {
		return mcp.NewToolResultText("No dependencies found for this Metric."), nil
	}

	response, err := deps.DBService.Neo4jRecordsToJSON(records)
	if err != nil {
		slog.Error("error formatting query results", "error", err)
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(response), nil
}
