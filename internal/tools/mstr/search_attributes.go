package mstr

import (
	"context"
	"log/slog"

	"github.com/brunogc-cit/flow-microstrategy-mcp/internal/tools"
	"github.com/mark3labs/mcp-go/mcp"
)

// SearchAttributesInput defines the input schema for the search-attributes tool.
type SearchAttributesInput struct {
	SearchTerm    string   `json:"searchTerm,omitempty" jsonschema:"description=Comma-separated search terms to filter by name or GUID (e.g. 'date,customer')"`
	PriorityLevel []string `json:"priorityLevel,omitempty" jsonschema:"description=Filter by priority levels (e.g. ['P1 (Highest)','P2']). Use 'All Prioritized' for all."`
	BusinessArea  []string `json:"businessArea,omitempty" jsonschema:"description=Filter by business areas. Use 'All Areas' for all."`
	Status        []string `json:"status,omitempty" jsonschema:"description=Filter by parity status values. Use 'All Status' for all."`
	DataDomain    []string `json:"dataDomain,omitempty" jsonschema:"description=Filter by data domains. Use 'All Domains' for all."`
}

// SearchAttributesSpec returns the MCP tool specification.
func SearchAttributesSpec() mcp.Tool {
	return mcp.NewTool("search-attributes",
		mcp.WithDescription(
			"Search for MicroStrategy Attributes that are used by prioritized reports. "+
				"Returns attributes with type, priority, name, status, team, report count, and source table count. "+
				"Use this to find which attributes are most impactful for Power BI migration planning. "+
				"All filter parameters are optional.",
		),
		mcp.WithInputSchema[SearchAttributesInput](),
		mcp.WithTitleAnnotation("Search Attributes"),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(true),
	)
}

// SearchAttributesHandler returns a handler function for the search-attributes tool.
func SearchAttributesHandler(deps *tools.ToolDependencies) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleSearchAttributes(ctx, request, deps)
	}
}

func handleSearchAttributes(ctx context.Context, request mcp.CallToolRequest, deps *tools.ToolDependencies) (*mcp.CallToolResult, error) {
	if deps.DBService == nil {
		errMessage := "Database service is not initialized"
		slog.Error(errMessage)
		return mcp.NewToolResultError(errMessage), nil
	}

	var args SearchAttributesInput
	if err := request.BindArguments(&args); err != nil {
		slog.Error("error binding arguments", "error", err)
		return mcp.NewToolResultError(err.Error()), nil
	}

	params := map[string]any{
		"neodash_searchterm":     args.SearchTerm,
		"neodash_objecttype":     "Attribute", // Fixed to Attribute
		"neodash_priority_level": args.PriorityLevel,
		"neodash_business_area":  args.BusinessArea,
		"neodash_status":         args.Status,
		"neodash_data_domain":    args.DataDomain,
	}

	slog.Info("executing search-attributes query", "searchTerm", args.SearchTerm)

	records, err := deps.DBService.ExecuteReadQuery(ctx, SearchObjectsQuery, params)
	if err != nil {
		slog.Error("error executing search-attributes query", "error", err)
		return mcp.NewToolResultError(err.Error()), nil
	}

	if len(records) == 0 {
		return mcp.NewToolResultText("No Attributes found matching the specified criteria."), nil
	}

	response, err := deps.DBService.Neo4jRecordsToJSON(records)
	if err != nil {
		slog.Error("error formatting query results", "error", err)
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(response), nil
}
