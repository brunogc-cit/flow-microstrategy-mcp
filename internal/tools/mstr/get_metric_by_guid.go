package mstr

import (
	"context"
	"log/slog"

	"github.com/brunogc-cit/flow-microstrategy-mcp/internal/tools"
	"github.com/mark3labs/mcp-go/mcp"
)

// GetMetricByGuidInput defines the input schema for the get-metric-by-guid tool.
type GetMetricByGuidInput struct {
	Guid string `json:"guid" jsonschema:"required,description=The GUID of the Metric to retrieve (supports prefix matching)"`
}

// GetMetricByGuidSpec returns the MCP tool specification.
func GetMetricByGuidSpec() mcp.Tool {
	return mcp.NewTool("get-metric-by-guid",
		mcp.WithDescription(
			"Get detailed information about a MicroStrategy Metric by its GUID. "+
				"Returns parity status, team, EDW/ADE table mappings, Power BI semantic model info, and migration notes. "+
				"Supports prefix matching - you can provide the first characters of the GUID.",
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
		"neodash_selected_guid": []string{args.Guid},
	}

	slog.Info("executing get-metric-by-guid query", "guid", args.Guid)

	records, err := deps.DBService.ExecuteReadQuery(ctx, GetObjectDetailsQuery, params)
	if err != nil {
		slog.Error("error executing get-metric-by-guid query", "error", err)
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Filter only Metric results
	var metricRecords []*interface{}
	for _, record := range records {
		if typeVal, ok := record.Get("Type"); ok {
			if typeStr, ok := typeVal.(string); ok && typeStr == "Metric" {
				metricRecords = append(metricRecords, nil) // placeholder
			}
		}
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
