package mstr

import (
	"context"
	"log/slog"

	"github.com/brunogc-cit/flow-microstrategy-mcp/internal/tools"
	"github.com/mark3labs/mcp-go/mcp"
)

// GetMetricsStatsInput defines the input schema for the get-metrics-stats tool.
type GetMetricsStatsInput struct {
	Status []string `json:"status,omitempty" jsonschema:"description=Optional filter by status: Complete, Planned, Not Planned, No Status"`
	Team   string   `json:"team,omitempty" jsonschema:"description=Optional filter by team name"`
}

// GetMetricsStatsSpec returns the MCP tool specification.
func GetMetricsStatsSpec() mcp.Tool {
	return mcp.NewTool("get-metrics-stats",
		mcp.WithDescription(
			"Get aggregate statistics for all MicroStrategy Metrics. "+
				"Returns counts by parity status: total, complete, planned, notPlanned, noStatus. "+
				"Also returns: prioritized (count with priority assigned), teams (distinct team names). "+
				"NO PAGINATION - returns a single summary row (~200 bytes). "+
				"Use BEFORE search-metrics to understand dataset scope and plan pagination strategy. "+
				"Example workflow: (1) get-metrics-stats → see 450 total, (2) search-metrics with offset to fetch pages.",
		),
		mcp.WithInputSchema[GetMetricsStatsInput](),
		mcp.WithTitleAnnotation("Get Metrics Statistics"),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(true),
	)
}

// GetMetricsStatsHandler returns a handler function.
func GetMetricsStatsHandler(deps *tools.ToolDependencies) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleGetMetricsStats(ctx, request, deps)
	}
}

func handleGetMetricsStats(ctx context.Context, request mcp.CallToolRequest, deps *tools.ToolDependencies) (*mcp.CallToolResult, error) {
	if deps.DBService == nil {
		errMessage := "Database service is not initialized"
		slog.Error(errMessage)
		return mcp.NewToolResultError(errMessage), nil
	}

	var args GetMetricsStatsInput
	if err := request.BindArguments(&args); err != nil {
		slog.Error("error binding arguments", "error", err)
		return mcp.NewToolResultError(err.Error()), nil
	}

	params := map[string]any{
		"status": args.Status,
		"team":   args.Team,
	}

	slog.Info("executing get-metrics-stats query", "status", args.Status, "team", args.Team)

	records, err := deps.DBService.ExecuteReadQuery(ctx, MetricsStatsQuery, params)
	if err != nil {
		slog.Error("error executing query", "error", err)
		return mcp.NewToolResultError(err.Error()), nil
	}

	if len(records) == 0 {
		return mcp.NewToolResultText(`{"total": 0, "complete": 0, "planned": 0, "notPlanned": 0, "noStatus": 0, "prioritized": 0, "teams": []}`), nil
	}

	response, err := deps.DBService.Neo4jRecordsToJSON(records)
	if err != nil {
		slog.Error("error formatting query results", "error", err)
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(response), nil
}
