package mstr

import (
	"context"
	"log/slog"

	"github.com/brunogc-cit/flow-microstrategy-mcp/internal/tools"
	"github.com/mark3labs/mcp-go/mcp"
)

// GetMetricDependenciesInput defines the input schema for the get-metric-dependencies tool.
type GetMetricDependenciesInput struct {
	Guid   string `json:"guid" jsonschema:"required,description=GUID of the Metric to analyze"`
	Offset int    `json:"offset,omitempty" jsonschema:"description=Pagination offset for direct dependencies (0, 100, 200...). Default 0."`
}

// GetMetricDependenciesSpec returns the MCP tool specification.
func GetMetricDependenciesSpec() mcp.Tool {
	return mcp.NewTool("get-metric-dependencies",
		mcp.WithDescription(
			"Find what a Metric directly depends on (downstream/outbound dependencies). "+
				"Returns two parts: "+
				"(1) directDependencies: Objects at 1-hop distance (Facts, Metrics, Attributes, Columns) with type, name, guid, formula. "+
				"(2) transitiveTableCount: Total count of tables reachable through the full dependency chain (2-10 hops). "+
				"PAGINATION: Direct dependencies are paginated (100 per page). Use 'offset' to paginate. "+
				"Response includes 'moreResults' boolean and 'totalDirectDeps' count. "+
				"Use for understanding metric calculation logic and formula translation to SQL/DBT.",
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
		"guids":  []string{args.Guid},
		"offset": args.Offset,
	}

	slog.Info("executing get-metric-dependencies query", "guid", args.Guid, "offset", args.Offset)

	records, err := deps.DBService.ExecuteReadQuery(ctx, DownstreamDependenciesQuery, params)
	if err != nil {
		slog.Error("error executing query", "error", err)
		return mcp.NewToolResultError(err.Error()), nil
	}

	if len(records) == 0 {
		return mcp.NewToolResultText(`{"objectName": "", "objectGUID": "", "objectType": "", "totalDirectDeps": 0, "transitiveTableCount": 0, "directDependencies": [], "moreResults": false}`), nil
	}

	response, err := deps.DBService.Neo4jRecordsToJSON(records)
	if err != nil {
		slog.Error("error formatting query results", "error", err)
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(response), nil
}
