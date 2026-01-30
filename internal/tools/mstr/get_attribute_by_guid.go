package mstr

import (
	"context"
	"log/slog"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/neo4j/mcp/internal/tools"
)

// GetAttributeByGuidInput defines the input schema for the get-attribute-by-guid tool.
type GetAttributeByGuidInput struct {
	Guid string `json:"guid" jsonschema:"required,description=The GUID of the Attribute to retrieve (supports prefix matching)"`
}

// GetAttributeByGuidSpec returns the MCP tool specification.
func GetAttributeByGuidSpec() mcp.Tool {
	return mcp.NewTool("get-attribute-by-guid",
		mcp.WithDescription(
			"Get detailed information about a MicroStrategy Attribute by its GUID. "+
				"Returns parity status, team, EDW/ADE table mappings, Power BI semantic model info, and migration notes. "+
				"Supports prefix matching - you can provide the first characters of the GUID.",
		),
		mcp.WithInputSchema[GetAttributeByGuidInput](),
		mcp.WithTitleAnnotation("Get Attribute by GUID"),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(true),
	)
}

// GetAttributeByGuidHandler returns a handler function for the get-attribute-by-guid tool.
func GetAttributeByGuidHandler(deps *tools.ToolDependencies) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleGetAttributeByGuid(ctx, request, deps)
	}
}

func handleGetAttributeByGuid(ctx context.Context, request mcp.CallToolRequest, deps *tools.ToolDependencies) (*mcp.CallToolResult, error) {
	if deps.DBService == nil {
		errMessage := "Database service is not initialized"
		slog.Error(errMessage)
		return mcp.NewToolResultError(errMessage), nil
	}

	var args GetAttributeByGuidInput
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

	slog.Info("executing get-attribute-by-guid query", "guid", args.Guid)

	records, err := deps.DBService.ExecuteReadQuery(ctx, GetObjectDetailsQuery, params)
	if err != nil {
		slog.Error("error executing get-attribute-by-guid query", "error", err)
		return mcp.NewToolResultError(err.Error()), nil
	}

	if len(records) == 0 {
		return mcp.NewToolResultText("No Attribute found with the specified GUID."), nil
	}

	response, err := deps.DBService.Neo4jRecordsToJSON(records)
	if err != nil {
		slog.Error("error formatting query results", "error", err)
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(response), nil
}
