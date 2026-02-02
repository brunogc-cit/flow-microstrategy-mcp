package mstr

import (
	"context"
	"log/slog"

	"github.com/brunogc-cit/flow-microstrategy-mcp/internal/tools"
	"github.com/mark3labs/mcp-go/mcp"
)

// GetAttributeDependentsInput defines the input schema for the get-attribute-dependents tool.
type GetAttributeDependentsInput struct {
	GUID   string `json:"guid" jsonschema:"required,description=GUID of the Attribute to analyze"`
	Offset int    `json:"offset,omitempty" jsonschema:"description=Pagination offset (0, 100, 200...). Default 0."`
}

// GetAttributeDependentsSpec returns the MCP tool specification.
func GetAttributeDependentsSpec() mcp.Tool {
	return mcp.NewTool("get-attribute-dependents",
		mcp.WithDescription(
			"Find all Reports, GridReports, and Documents that depend on an Attribute (upstream/inbound dependencies). "+
				"Traverses the dependency graph through Prompts and Filters (up to 10 levels). "+
				"Returns for each report: name, guid, type, priority, area, department, userCount. "+
				"Also returns 'totalReports' count for the full dataset. "+
				"PAGINATION: Returns 100 reports per page. Use 'offset' to paginate. "+
				"Response includes 'moreResults' boolean. "+
				"Use for impact analysis before modifying or deprecating an attribute.",
		),
		mcp.WithInputSchema[GetAttributeDependentsInput](),
		mcp.WithTitleAnnotation("Get Attribute Dependents"),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(true),
	)
}

// GetAttributeDependentsHandler returns a handler function.
func GetAttributeDependentsHandler(deps *tools.ToolDependencies) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleGetAttributeDependents(ctx, request, deps)
	}
}

func handleGetAttributeDependents(ctx context.Context, request mcp.CallToolRequest, deps *tools.ToolDependencies) (*mcp.CallToolResult, error) {
	if deps.DBService == nil {
		errMessage := "Database service is not initialized"
		slog.Error(errMessage)
		return mcp.NewToolResultError(errMessage), nil
	}

	var args GetAttributeDependentsInput
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

	slog.Info("executing get-attribute-dependents query", "guid", args.GUID, "offset", args.Offset)

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
