package mstr

import (
	"context"
	"log/slog"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/neo4j/mcp/internal/tools"
)

// GetMetricSourceTablesInput defines the input schema for the get-metric-source-tables tool.
type GetMetricSourceTablesInput struct {
	Guid string `json:"guid" jsonschema:"required,description=The GUID of the Metric to look up"`
}

// GetMetricSourceTablesSpec returns the MCP tool specification.
func GetMetricSourceTablesSpec() mcp.Tool {
	return mcp.NewTool("get-metric-source-tables",
		mcp.WithDescription(
			"Find the source database tables that feed a specific MicroStrategy Metric. "+
				"Returns table names and GUIDs showing the data lineage. "+
				"Use this to understand which tables need to be mapped in Power BI.",
		),
		mcp.WithInputSchema[GetMetricSourceTablesInput](),
		mcp.WithTitleAnnotation("Get Metric Source Tables"),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(true),
	)
}

// GetMetricSourceTablesHandler returns a handler function.
func GetMetricSourceTablesHandler(deps *tools.ToolDependencies) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleGetMetricSourceTables(ctx, request, deps)
	}
}

func handleGetMetricSourceTables(ctx context.Context, request mcp.CallToolRequest, deps *tools.ToolDependencies) (*mcp.CallToolResult, error) {
	if deps.DBService == nil {
		errMessage := "Database service is not initialized"
		slog.Error(errMessage)
		return mcp.NewToolResultError(errMessage), nil
	}

	var args GetMetricSourceTablesInput
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

	slog.Info("executing get-metric-source-tables query", "guid", args.Guid)

	records, err := deps.DBService.ExecuteReadQuery(ctx, SourceTablesQuery, params)
	if err != nil {
		slog.Error("error executing query", "error", err)
		return mcp.NewToolResultError(err.Error()), nil
	}

	if len(records) == 0 {
		return mcp.NewToolResultText("No source tables found for this Metric."), nil
	}

	response, err := deps.DBService.Neo4jRecordsToJSON(records)
	if err != nil {
		slog.Error("error formatting query results", "error", err)
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(response), nil
}
