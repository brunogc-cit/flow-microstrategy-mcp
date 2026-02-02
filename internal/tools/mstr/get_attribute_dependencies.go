package mstr

import (
	"context"
	"log/slog"

	"github.com/brunogc-cit/flow-microstrategy-mcp/internal/tools"
	"github.com/mark3labs/mcp-go/mcp"
)

// GetAttributeDependenciesInput defines the input schema for the get-attribute-dependencies tool.
type GetAttributeDependenciesInput struct {
	GUID   string `json:"guid" jsonschema:"required,description=GUID of the Attribute to analyze"`
	Offset int    `json:"offset,omitempty" jsonschema:"description=Pagination offset for direct dependencies (0, 100, 200...). Default 0."`
}

// GetAttributeDependenciesSpec returns the MCP tool specification.
func GetAttributeDependenciesSpec() mcp.Tool {
	return mcp.NewTool("get-attribute-dependencies",
		mcp.WithDescription(
			"Find what an Attribute directly depends on (downstream/outbound dependencies). "+
				"Returns two parts: "+
				"(1) directDependencies: Objects at 1-hop distance (Facts, Metrics, Attributes, Columns) with type, name, guid, formula. "+
				"(2) transitiveTableCount: Total count of tables reachable through the full dependency chain (2-10 hops). "+
				"PAGINATION: Direct dependencies are paginated (100 per page). Use 'offset' to paginate. "+
				"Response includes 'moreResults' boolean and 'totalDirectDeps' count. "+
				"Use for understanding attribute structure and data lineage.",
		),
		mcp.WithInputSchema[GetAttributeDependenciesInput](),
		mcp.WithTitleAnnotation("Get Attribute Dependencies"),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(true),
	)
}

// GetAttributeDependenciesHandler returns a handler function.
func GetAttributeDependenciesHandler(deps *tools.ToolDependencies) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleGetAttributeDependencies(ctx, request, deps)
	}
}

func handleGetAttributeDependencies(ctx context.Context, request mcp.CallToolRequest, deps *tools.ToolDependencies) (*mcp.CallToolResult, error) {
	if deps.DBService == nil {
		errMessage := "Database service is not initialized"
		slog.Error(errMessage)
		return mcp.NewToolResultError(errMessage), nil
	}

	var args GetAttributeDependenciesInput
	if err := request.BindArguments(&args); err != nil {
		slog.Error("error binding arguments", "error", err)
		return mcp.NewToolResultError(err.Error()), nil
	}

	if args.GUID == "" {
		errMessage := "guid parameter is required"
		slog.Error(errMessage)
		return mcp.NewToolResultError(errMessage), nil
	}

	params := map[string]any{
		"guids":  []string{args.GUID},
		"offset": args.Offset,
	}

	slog.Info("executing get-attribute-dependencies query", "guid", args.GUID, "offset", args.Offset)

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
