package mstr

import (
	"context"
	"log/slog"

	"github.com/brunogc-cit/flow-microstrategy-mcp/internal/tools"
	"github.com/mark3labs/mcp-go/mcp"
)

// GetMetricByGuidInput defines the input schema for the get-metric-by-guid tool.
type GetMetricByGuidInput struct {
	Guid string `json:"guid" jsonschema:"required,description=Full GUID of the Metric to retrieve. Exact match required."`
}

// GetMetricByGuidSpec returns the MCP tool specification.
func GetMetricByGuidSpec() mcp.Tool {
	return mcp.NewTool("get-metric-by-guid",
		mcp.WithDescription(
			"Get comprehensive details about a MicroStrategy Metric by GUID. "+
				"Returns 22 fields including: name, status, team, priority, formula, "+
				"EDW/ADE mappings (edwTable, edwColumn, adeTable, adeColumn), "+
				"Power BI mappings (semanticName, semanticModel), "+
				"Databricks mappings (raw, serve), "+
				"and pre-computed counts (reportCount, tableCount). "+
				"Use for detailed object inspection and gap analysis. "+
				"Limited to 100 results per call.",
		),
		mcp.WithInputSchema[GetMetricByGuidInput](),
		mcp.WithTitleAnnotation("Get Metric by GUID"),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(true),
	)
}

// GetMetricByGuidHandler returns a handler function for the get-metric-by-guid tool.
func GetMetricByGuidHandler(deps *tools.ToolDependencies) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleGetMetricByGuid(ctx, request, deps)
	}
}

func handleGetMetricByGuid(ctx context.Context, request mcp.CallToolRequest, deps *tools.ToolDependencies) (*mcp.CallToolResult, error) {
	if deps.DBService == nil {
		errMessage := "Database service is not initialized"
		slog.Error(errMessage)
		return mcp.NewToolResultError(errMessage), nil
	}

	var args GetMetricByGuidInput
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
		"guids": []string{args.Guid},
	}

	slog.Info("executing get-metric-by-guid query", "guid", args.Guid)

	records, err := deps.DBService.ExecuteReadQuery(ctx, GetObjectDetailsQuery, params)
	if err != nil {
		slog.Error("error executing get-metric-by-guid query", "error", err)
		return mcp.NewToolResultError(err.Error()), nil
	}

	if len(records) == 0 {
		return mcp.NewToolResultText("No Metric found with the specified GUID."), nil
	}

	response, err := deps.DBService.Neo4jRecordsToJSON(records)
	if err != nil {
		slog.Error("error formatting query results", "error", err)
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(response), nil
}
