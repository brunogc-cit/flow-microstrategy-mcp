package mstr

import (
	"context"
	"log/slog"

	"github.com/brunogc-cit/flow-microstrategy-mcp/internal/tools"
	"github.com/mark3labs/mcp-go/mcp"
)

// GetAttributeByGUIDInput defines the input schema for the get-attribute-by-guid tool.
type GetAttributeByGUIDInput struct {
	GUID string `json:"guid" jsonschema:"required,description=Full GUID of the Attribute to retrieve. Exact match required."`
}

// GetAttributeByGUIDSpec returns the MCP tool specification.
func GetAttributeByGUIDSpec() mcp.Tool {
	return mcp.NewTool("get-attribute-by-guid",
		mcp.WithDescription(
			"Get comprehensive details about a MicroStrategy Attribute by GUID. "+
				"Returns 22 fields including: name, status, team, priority, formula, "+
				"EDW/ADE mappings (edwTable, edwColumn, adeTable, adeColumn), "+
				"Power BI mappings (semanticName, semanticModel), "+
				"Databricks mappings (raw, serve), "+
				"and pre-computed counts (reportCount, tableCount). "+
				"Use for detailed object inspection and gap analysis. "+
				"Limited to 100 results per call.",
		),
		mcp.WithInputSchema[GetAttributeByGUIDInput](),
		mcp.WithTitleAnnotation("Get Attribute by GUID"),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(true),
	)
}

// GetAttributeByGUIDHandler returns a handler function for the get-attribute-by-guid tool.
func GetAttributeByGUIDHandler(deps *tools.ToolDependencies) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleGetAttributeByGUID(ctx, request, deps)
	}
}

func handleGetAttributeByGUID(ctx context.Context, request mcp.CallToolRequest, deps *tools.ToolDependencies) (*mcp.CallToolResult, error) {
	if deps.DBService == nil {
		errMessage := "Database service is not initialized"
		slog.Error(errMessage)
		return mcp.NewToolResultError(errMessage), nil
	}

	var args GetAttributeByGUIDInput
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
		"guids": []string{args.GUID},
	}

	slog.Info("executing get-attribute-by-guid query", "guid", args.GUID)

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
