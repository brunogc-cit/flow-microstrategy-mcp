package mstr

import (
	"context"
	"log/slog"

	"github.com/brunogc-cit/flow-microstrategy-mcp/internal/tools"
	"github.com/mark3labs/mcp-go/mcp"
)

// GetAttributeDependentsInput defines the input schema for the get-attribute-dependents tool.
type GetAttributeDependentsInput struct {
	Guid string `json:"guid" jsonschema:"required,description=The GUID of the Attribute to analyze"`
}

// GetAttributeDependentsSpec returns the MCP tool specification.
func GetAttributeDependentsSpec() mcp.Tool {
	return mcp.NewTool("get-attribute-dependents",
		mcp.WithDescription(
			"Find what depends on a MicroStrategy Attribute (upstream dependencies). "+
				"Shows Reports, Documents, Metrics, and other objects that use this Attribute. "+
				"Use this for impact analysis: understanding what will be affected if the attribute changes.",
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

	if args.Guid == "" {
		errMessage := "guid parameter is required"
		slog.Error(errMessage)
		return mcp.NewToolResultError(errMessage), nil
	}

	params := map[string]any{
		"neodash_selected_guid": []string{args.Guid},
	}

	slog.Info("executing get-attribute-dependents query", "guid", args.Guid)

	records, err := deps.DBService.ExecuteReadQuery(ctx, UpstreamDependenciesQuery, params)
	if err != nil {
		slog.Error("error executing query", "error", err)
		return mcp.NewToolResultError(err.Error()), nil
	}

	if len(records) == 0 {
		return mcp.NewToolResultText("No dependents found for this Attribute."), nil
	}

	response, err := deps.DBService.Neo4jRecordsToJSON(records)
	if err != nil {
		slog.Error("error formatting query results", "error", err)
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(response), nil
}
