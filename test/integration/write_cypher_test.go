//go:build integration

package integration

import (
	"testing"

	"github.com/brunogc-cit/flow-microstrategy-mcp/internal/tools/cypher"
	"github.com/brunogc-cit/flow-microstrategy-mcp/test/integration/helpers"
)

func TestWriteCypher(t *testing.T) {
	t.Parallel()
	tc := helpers.NewTestContext(t, dbs.GetDriver())

	personLabel := tc.GetUniqueLabel("Person")

	write := cypher.WriteCypherHandler(tc.Deps)
	tc.CallTool(write, map[string]any{
		"query":  "CREATE (p:" + personLabel + " {name: $name}) RETURN p",
		"params": map[string]any{"name": "Alice"},
	})

	tc.VerifyNodeInDB(personLabel, map[string]any{"name": "Alice"})
}
