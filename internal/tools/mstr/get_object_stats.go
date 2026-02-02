package mstr

import (
	"context"
	"log/slog"

	"github.com/brunogc-cit/flow-microstrategy-mcp/internal/tools"
	"github.com/mark3labs/mcp-go/mcp"
)

// GetObjectStatsInput defines the input schema for the get-object-stats tool.
type GetObjectStatsInput struct {
	Guid string `json:"guid" jsonschema:"required,description=GUID of the Metric or Attribute to analyze"`
}

// GetObjectStatsSpec returns the MCP tool specification.
func GetObjectStatsSpec() mcp.Tool {
	return mcp.NewTool("get-object-stats",
		mcp.WithDescription(
			"Get summary statistics for a specific MicroStrategy object (Metric or Attribute). "+
				"Returns: name, type, guid, status, team, reportCount, tableCount, reportsByPriority. "+
				"reportsByPriority shows distribution: [{priority: 1, count: 23}, {priority: 2, count: 156}, ...]. "+
				"NO PAGINATION - returns a single object summary (~500 bytes). "+
				"Use for quick impact assessment of a single object without fetching full report list.",
		),
		mcp.WithInputSchema[GetObjectStatsInput](),
		mcp.WithTitleAnnotation("Get Object Statistics"),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(true),
	)
}

// GetObjectStatsHandler returns a handler function.
func GetObjectStatsHandler(deps *tools.ToolDependencies) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleGetObjectStats(ctx, request, deps)
	}
}

func handleGetObjectStats(ctx context.Context, request mcp.CallToolRequest, deps *tools.ToolDependencies) (*mcp.CallToolResult, error) {
	if deps.DBService == nil {
		errMessage := "Database service is not initialized"
		slog.Error(errMessage)
		return mcp.NewToolResultError(errMessage), nil
	}

	var args GetObjectStatsInput
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
		"guid": args.Guid,
	}

	slog.Info("executing get-object-stats query", "guid", args.Guid)

	records, err := deps.DBService.ExecuteReadQuery(ctx, ObjectStatsQuery, params)
	if err != nil {
		slog.Error("error executing query", "error", err)
		return mcp.NewToolResultError(err.Error()), nil
	}

	if len(records) == 0 {
		return mcp.NewToolResultText("No object found with the specified GUID."), nil
	}

	response, err := deps.DBService.Neo4jRecordsToJSON(records)
	if err != nil {
		slog.Error("error formatting query results", "error", err)
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(response), nil
}
