package mstr

import (
	"context"
	"log/slog"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/brunogc-cit/flow-microstrategy-mcp/internal/tools"
)

// GetAttributeDependenciesInput defines the input schema for the get-attribute-dependencies tool.
type GetAttributeDependenciesInput struct {
	Guid string `json:"guid" jsonschema:"required,description=The GUID of the Attribute to analyze"`
}

// GetAttributeDependenciesSpec returns the MCP tool specification.
func GetAttributeDependenciesSpec() mcp.Tool {
	return mcp.NewTool("get-attribute-dependencies",
		mcp.WithDescription(
			"Find what a MicroStrategy Attribute depends on (downstream dependencies). "+
				"Traverses the dependency chain up to 10 levels deep showing other Attributes, Columns, and related objects. "+
				"Use this to understand the complete definition chain of an attribute.",
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

	if args.Guid == "" {
		errMessage := "guid parameter is required"
		slog.Error(errMessage)
		return mcp.NewToolResultError(errMessage), nil
	}

	params := map[string]any{
		"neodash_selected_guid": []string{args.Guid},
	}

	slog.Info("executing get-attribute-dependencies query", "guid", args.Guid)

	records, err := deps.DBService.ExecuteReadQuery(ctx, DownstreamDependenciesQuery, params)
	if err != nil {
		slog.Error("error executing query", "error", err)
		return mcp.NewToolResultError(err.Error()), nil
	}

	if len(records) == 0 {
		return mcp.NewToolResultText("No dependencies found for this Attribute."), nil
	}

	response, err := deps.DBService.Neo4jRecordsToJSON(records)
	if err != nil {
		slog.Error("error formatting query results", "error", err)
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(response), nil
}
