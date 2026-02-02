package mstr

import (
	"context"
	"log/slog"

	"github.com/brunogc-cit/flow-microstrategy-mcp/internal/tools"
	"github.com/mark3labs/mcp-go/mcp"
)

// GetReportsUsingAttributeInput defines the input schema for the get-reports-using-attribute tool.
type GetReportsUsingAttributeInput struct {
	GUID          string   `json:"guid" jsonschema:"required,description=GUID of the Attribute to analyze"`
	PriorityLevel []string `json:"priorityLevel,omitempty" jsonschema:"description=Filter by report priority: P1-P5. Use 'All Prioritized' for any."`
	BusinessArea  []string `json:"businessArea,omitempty" jsonschema:"description=Filter by business area. Use 'All Areas' for all."`
	Offset        int      `json:"offset,omitempty" jsonschema:"description=Pagination offset (0, 100, 200...). Default 0."`
}

// GetReportsUsingAttributeSpec returns the MCP tool specification.
func GetReportsUsingAttributeSpec() mcp.Tool {
	return mcp.NewTool("get-reports-using-attribute",
		mcp.WithDescription(
			"Find all Reports, GridReports, and Documents that use a specific Attribute. "+
				"Returns for each report: name, guid, type, priority (1-5), area, department, userCount. "+
				"PAGINATION: Returns 100 reports per page. Use 'offset' to paginate. "+
				"Response includes 'moreResults' boolean and total count via 'totalReports'. "+
				"Note: High-usage attributes may have thousands of reports. "+
				"Use for impact analysis: understanding what will be affected by changes.",
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

	if args.GUID == "" {
		errMessage := "guid parameter is required"
		slog.Error(errMessage)
		return mcp.NewToolResultError(errMessage), nil
	}

	params := map[string]any{
		"guids":         []string{args.GUID},
		"priorityLevel": args.PriorityLevel,
		"businessArea":  args.BusinessArea,
		"offset":        args.Offset,
	}

	slog.Info("executing get-reports-using-attribute query", "guid", args.GUID, "offset", args.Offset)

	records, err := deps.DBService.ExecuteReadQuery(ctx, ReportsUsingObjectsQuery, params)
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
