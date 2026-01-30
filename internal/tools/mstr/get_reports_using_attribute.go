package mstr

import (
	"context"
	"log/slog"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/neo4j/mcp/internal/tools"
)

// GetReportsUsingAttributeInput defines the input schema for the get-reports-using-attribute tool.
type GetReportsUsingAttributeInput struct {
	Guid          string   `json:"guid" jsonschema:"required,description=The GUID of the Attribute to look up"`
	PriorityLevel []string `json:"priorityLevel,omitempty" jsonschema:"description=Filter reports by priority levels (e.g. ['P1 (Highest)','P2']). Use 'All Prioritized' for all."`
	BusinessArea  []string `json:"businessArea,omitempty" jsonschema:"description=Filter reports by business areas. Use 'All Areas' for all."`
}

// GetReportsUsingAttributeSpec returns the MCP tool specification.
func GetReportsUsingAttributeSpec() mcp.Tool {
	return mcp.NewTool("get-reports-using-attribute",
		mcp.WithDescription(
			"Find all MicroStrategy Reports and Documents that use a specific Attribute. "+
				"Returns report details including name, priority, business area, department, user count, and usage pattern. "+
				"Use this for impact analysis before migration.",
		),
		mcp.WithInputSchema[GetReportsUsingAttributeInput](),
		mcp.WithTitleAnnotation("Get Reports Using Attribute"),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(true),
	)
}

// GetReportsUsingAttributeHandler returns a handler function.
func GetReportsUsingAttributeHandler(deps *tools.ToolDependencies) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleGetReportsUsingAttribute(ctx, request, deps)
	}
}

func handleGetReportsUsingAttribute(ctx context.Context, request mcp.CallToolRequest, deps *tools.ToolDependencies) (*mcp.CallToolResult, error) {
	if deps.DBService == nil {
		errMessage := "Database service is not initialized"
		slog.Error(errMessage)
		return mcp.NewToolResultError(errMessage), nil
	}

	var args GetReportsUsingAttributeInput
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
		"neodash_selected_guid":  []string{args.Guid},
		"neodash_priority_level": args.PriorityLevel,
		"neodash_business_area":  args.BusinessArea,
	}

	slog.Info("executing get-reports-using-attribute query", "guid", args.Guid)

	records, err := deps.DBService.ExecuteReadQuery(ctx, ReportsUsingObjectsQuery, params)
	if err != nil {
		slog.Error("error executing query", "error", err)
		return mcp.NewToolResultError(err.Error()), nil
	}

	if len(records) == 0 {
		return mcp.NewToolResultText("No reports found using this Attribute."), nil
	}

	response, err := deps.DBService.Neo4jRecordsToJSON(records)
	if err != nil {
		slog.Error("error formatting query results", "error", err)
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(response), nil
}
