package tools

import (
	"github.com/brunogc-cit/flow-microstrategy-mcp/internal/analytics"
	"github.com/brunogc-cit/flow-microstrategy-mcp/internal/database"
)

// ToolDependencies contains all dependencies needed by tools
type ToolDependencies struct {
	DBService        database.Service
	AnalyticsService analytics.Service
	SchemaSampleSize int
}
