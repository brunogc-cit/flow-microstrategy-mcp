package mstr

import (
	"context"
	"log/slog"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/brunogc-cit/flow-microstrategy-mcp/internal/tools"
)

// GetAttributeSourceTablesInput defines the input schema for the get-attribute-source-tables tool.
type GetAttributeSourceTablesInput struct {
	Guid string `json:"guid" jsonschema:"required,description=The GUID of the Attribute to look up"`
}

// GetAttributeSourceTablesSpec returns the MCP tool specification.
func GetAttributeSourceTablesSpec() mcp.Tool {
	return mcp.NewTool("get-attribute-source-tables",
		mcp.WithDescription(
			"Find the source database tables that feed a specific MicroStrategy Attribute. "+
				"Returns table names and GUIDs showing the data lineage. "+
				"Use this to understand which tables need to be mapped in Power BI.",
		),
		mcp.WithInputSchema[GetAttributeSourceTablesInput](),
		mcp.WithTitleAnnotation("Get Attribute Source Tables"),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(true),
	)
}

// GetAttributeSourceTablesHandler returns a handler function.
func GetAttributeSourceTablesHandler(deps *tools.ToolDependencies) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleGetAttributeSourceTables(ctx, request, deps)
	}
}

func handleGetAttributeSourceTables(ctx context.Context, request mcp.CallToolRequest, deps *tools.ToolDependencies) (*mcp.CallToolResult, error) {
	if deps.DBService == nil {
		errMessage := "Database service is not initialized"
		slog.Error(errMessage)
		return mcp.NewToolResultError(errMessage), nil
	}

	var args GetAttributeSourceTablesInput
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

	slog.Info("executing get-attribute-source-tables query", "guid", args.Guid)

	records, err := deps.DBService.ExecuteReadQuery(ctx, SourceTablesQuery, params)
	if err != nil {
		slog.Error("error executing query", "error", err)
		return mcp.NewToolResultError(err.Error()), nil
	}

	if len(records) == 0 {
		return mcp.NewToolResultText("No source tables found for this Attribute."), nil
	}

	response, err := deps.DBService.Neo4jRecordsToJSON(records)
	if err != nil {
		slog.Error("error formatting query results", "error", err)
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(response), nil
}
