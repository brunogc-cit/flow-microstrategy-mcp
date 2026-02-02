package mstr

import (
	"context"
	"log/slog"

	"github.com/brunogc-cit/flow-microstrategy-mcp/internal/tools"
	"github.com/mark3labs/mcp-go/mcp"
)

// GetAttributeSourceTablesInput defines the input schema for the get-attribute-source-tables tool.
type GetAttributeSourceTablesInput struct {
	Guid   string `json:"guid" jsonschema:"required,description=GUID of the Attribute to analyze"`
	Offset int    `json:"offset,omitempty" jsonschema:"description=Pagination offset (0, 100, 200...). Default 0."`
}

// GetAttributeSourceTablesSpec returns the MCP tool specification.
func GetAttributeSourceTablesSpec() mcp.Tool {
	return mcp.NewTool("get-attribute-source-tables",
		mcp.WithDescription(
			"Find source database tables (LogicalTable/Table) that an Attribute depends on. "+
				"Returns for each table: name, guid, type, physicalTableName, databaseInstance. "+
				"Traverses the full dependency graph (up to 10 levels) through Facts, Metrics, Attributes, Columns. "+
				"PAGINATION: Returns 100 tables per page. Use 'offset' to paginate. "+
				"Response includes 'moreResults' boolean and 'totalTables' count. "+
				"Use for data lineage analysis and DBT model generation.",
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
		"guids":  []string{args.Guid},
		"offset": args.Offset,
	}

	slog.Info("executing get-attribute-source-tables query", "guid", args.Guid, "offset", args.Offset)

	records, err := deps.DBService.ExecuteReadQuery(ctx, SourceTablesQuery, params)
	if err != nil {
		slog.Error("error executing query", "error", err)
		return mcp.NewToolResultError(err.Error()), nil
	}

	if len(records) == 0 {
		return mcp.NewToolResultText(`{"objectName": "", "objectGUID": "", "objectType": "", "totalTables": 0, "tables": [], "moreResults": false}`), nil
	}

	response, err := deps.DBService.Neo4jRecordsToJSON(records)
	if err != nil {
		slog.Error("error formatting query results", "error", err)
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(response), nil
}
