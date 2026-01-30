package server

import (
	"github.com/mark3labs/mcp-go/server"

	"github.com/brunogc-cit/flow-microstrategy-mcp/internal/tools"
	"github.com/brunogc-cit/flow-microstrategy-mcp/internal/tools/gds"
	"github.com/brunogc-cit/flow-microstrategy-mcp/internal/tools/mstr"
)

// registerTools registers all enabled MCP tools and adds them to the provided MCP server.
// Tools are filtered according to the server configuration. For example, when the read-only
// mode is enabled (e.g. via the FLOW_READ_ONLY environment variable or the Config.ReadOnly flag),
// any tool that performs state mutation will be excluded; only tools annotated as read-only will be registered.
// Note: this read-only filtering relies on the tool annotation "readonly" (ReadOnlyHint). If the annotation
// is not defined or is set to false, the tool will be added (i.e., only tools with readonly=true are filtered in read-only mode).
func (s *Neo4jMCPServer) registerTools() error {
	filteredTools := s.getEnabledTools()
	s.MCPServer.AddTools(filteredTools...)
	return nil
}

type toolFilter func(tools []ToolDefinition) []ToolDefinition

type toolCategory int

const (
	cypherCategory toolCategory = 0 // Hidden - generic Neo4j tools (code kept but not exposed)
	gdsCategory    toolCategory = 1
	mstrCategory   toolCategory = 2 // MicroStrategy migration tools
)

type ToolDefinition struct {
	category   toolCategory
	definition server.ServerTool
	readonly   bool
}

func (s *Neo4jMCPServer) addGDSTools() {
	deps := &tools.ToolDependencies{
		DBService:        s.dbService,
		AnalyticsService: s.anService,
	}
	toolDefs := s.getAllToolsDefs(deps)
	toolDefinition := make([]server.ServerTool, 0)
	GDSTools := make([]ToolDefinition, 0, len(toolDefs))
	for _, t := range toolDefs {
		if t.category == gdsCategory {
			GDSTools = append(GDSTools, t)
		}
	}
	for _, toolDef := range GDSTools {
		toolDefinition = append(toolDefinition, toolDef.definition)
	}
	s.MCPServer.AddTools(toolDefinition...)
}

func (s *Neo4jMCPServer) getEnabledTools() []server.ServerTool {
	filters := make([]toolFilter, 0)

	// If read-only mode is enabled, expose only tools annotated as read-only.
	if s.config != nil && s.config.ReadOnly {
		filters = append(filters, filterWriteTools)
	}
	// If GDS is not installed, disable GDS tools.
	if !s.gdsInstalled {
		filters = append(filters, filterGDSTools)
	}
	// Always filter out generic cypher tools (hidden but code kept)
	filters = append(filters, filterCypherTools)

	deps := &tools.ToolDependencies{
		DBService:        s.dbService,
		AnalyticsService: s.anService,
	}
	toolDefs := s.getAllToolsDefs(deps)

	for _, filter := range filters {
		toolDefs = filter(toolDefs)
	}
	enabledTools := make([]server.ServerTool, 0)
	for _, toolDef := range toolDefs {
		enabledTools = append(enabledTools, toolDef.definition)
	}
	return enabledTools
}

func filterWriteTools(tools []ToolDefinition) []ToolDefinition {
	readOnlyTools := make([]ToolDefinition, 0, len(tools))
	for _, t := range tools {
		if t.readonly {
			readOnlyTools = append(readOnlyTools, t)
		}
	}
	return readOnlyTools
}

func filterGDSTools(tools []ToolDefinition) []ToolDefinition {
	nonGDSTools := make([]ToolDefinition, 0, len(tools))
	for _, t := range tools {
		if t.category != gdsCategory {
			nonGDSTools = append(nonGDSTools, t)
		}
	}
	return nonGDSTools
}

// filterCypherTools removes generic cypher tools (get-schema, read-cypher, write-cypher)
// These are hidden from users but the code is kept in internal/tools/cypher/
func filterCypherTools(tools []ToolDefinition) []ToolDefinition {
	nonCypherTools := make([]ToolDefinition, 0, len(tools))
	for _, t := range tools {
		if t.category != cypherCategory {
			nonCypherTools = append(nonCypherTools, t)
		}
	}
	return nonCypherTools
}

// getAllToolsDefs returns all available tools with their specs and handlers
func (s *Neo4jMCPServer) getAllToolsDefs(deps *tools.ToolDependencies) []ToolDefinition {

	return []ToolDefinition{
		// =============================================================================
		// HIDDEN: Generic Cypher Tools (code kept but not exposed to users)
		// =============================================================================
		// These tools are filtered out by filterCypherTools() but kept for potential future use
		// {
		// 	category: cypherCategory,
		// 	definition: server.ServerTool{
		// 		Tool:    cypher.GetSchemaSpec(),
		// 		Handler: cypher.GetSchemaHandler(deps, s.config.SchemaSampleSize),
		// 	},
		// 	readonly: true,
		// },
		// {
		// 	category: cypherCategory,
		// 	definition: server.ServerTool{
		// 		Tool:    cypher.ReadCypherSpec(),
		// 		Handler: cypher.ReadCypherHandler(deps),
		// 	},
		// 	readonly: true,
		// },
		// {
		// 	category: cypherCategory,
		// 	definition: server.ServerTool{
		// 		Tool:    cypher.WriteCypherSpec(),
		// 		Handler: cypher.WriteCypherHandler(deps),
		// 	},
		// 	readonly: false,
		// },

		// =============================================================================
		// GDS Category/Section
		// =============================================================================
		{
			category: gdsCategory,
			definition: server.ServerTool{
				Tool:    gds.ListGDSProceduresSpec(),
				Handler: gds.ListGdsProceduresHandler(deps),
			},
			readonly: true,
		},

		// =============================================================================
		// MicroStrategy Migration Tools - Details by GUID
		// =============================================================================
		{
			category: mstrCategory,
			definition: server.ServerTool{
				Tool:    mstr.GetMetricByGuidSpec(),
				Handler: mstr.GetMetricByGuidHandler(deps),
			},
			readonly: true,
		},
		{
			category: mstrCategory,
			definition: server.ServerTool{
				Tool:    mstr.GetAttributeByGuidSpec(),
				Handler: mstr.GetAttributeByGuidHandler(deps),
			},
			readonly: true,
		},

		// =============================================================================
		// MicroStrategy Migration Tools - Search
		// =============================================================================
		{
			category: mstrCategory,
			definition: server.ServerTool{
				Tool:    mstr.SearchMetricsSpec(),
				Handler: mstr.SearchMetricsHandler(deps),
			},
			readonly: true,
		},
		{
			category: mstrCategory,
			definition: server.ServerTool{
				Tool:    mstr.SearchAttributesSpec(),
				Handler: mstr.SearchAttributesHandler(deps),
			},
			readonly: true,
		},

		// =============================================================================
		// MicroStrategy Migration Tools - Reports Using Objects
		// =============================================================================
		{
			category: mstrCategory,
			definition: server.ServerTool{
				Tool:    mstr.GetReportsUsingMetricSpec(),
				Handler: mstr.GetReportsUsingMetricHandler(deps),
			},
			readonly: true,
		},
		{
			category: mstrCategory,
			definition: server.ServerTool{
				Tool:    mstr.GetReportsUsingAttributeSpec(),
				Handler: mstr.GetReportsUsingAttributeHandler(deps),
			},
			readonly: true,
		},

		// =============================================================================
		// MicroStrategy Migration Tools - Source Tables
		// =============================================================================
		{
			category: mstrCategory,
			definition: server.ServerTool{
				Tool:    mstr.GetMetricSourceTablesSpec(),
				Handler: mstr.GetMetricSourceTablesHandler(deps),
			},
			readonly: true,
		},
		{
			category: mstrCategory,
			definition: server.ServerTool{
				Tool:    mstr.GetAttributeSourceTablesSpec(),
				Handler: mstr.GetAttributeSourceTablesHandler(deps),
			},
			readonly: true,
		},

		// =============================================================================
		// MicroStrategy Migration Tools - Dependencies (Downstream)
		// =============================================================================
		{
			category: mstrCategory,
			definition: server.ServerTool{
				Tool:    mstr.GetMetricDependenciesSpec(),
				Handler: mstr.GetMetricDependenciesHandler(deps),
			},
			readonly: true,
		},
		{
			category: mstrCategory,
			definition: server.ServerTool{
				Tool:    mstr.GetAttributeDependenciesSpec(),
				Handler: mstr.GetAttributeDependenciesHandler(deps),
			},
			readonly: true,
		},

		// =============================================================================
		// MicroStrategy Migration Tools - Dependents (Upstream)
		// =============================================================================
		{
			category: mstrCategory,
			definition: server.ServerTool{
				Tool:    mstr.GetMetricDependentsSpec(),
				Handler: mstr.GetMetricDependentsHandler(deps),
			},
			readonly: true,
		},
		{
			category: mstrCategory,
			definition: server.ServerTool{
				Tool:    mstr.GetAttributeDependentsSpec(),
				Handler: mstr.GetAttributeDependentsHandler(deps),
			},
			readonly: true,
		},
	}
}
