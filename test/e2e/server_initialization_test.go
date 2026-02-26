//go:build e2e

package e2e

import (
	"context"
	"testing"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/brunogc-cit/flow-microstrategy-mcp/test/e2e/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServerInitializationE2E(t *testing.T) {
	ctx := context.Background()
	cfg := dbs.GetDriverConf()

	t.Run("successful initialization with all required parameters", func(t *testing.T) {
		t.Parallel()

		args := []string{
			"--flow-uri", cfg.URI,
			"--flow-username", cfg.Username,
			"--flow-password", cfg.Password,
			"--flow-database", cfg.Database,
		}

		mcpClient, err := client.NewStdioMCPClient(server, []string{}, args...)
		require.NoError(t, err, "failed to create MCP client")

		defer mcpClient.Close()

		// Test initialization
		initRequest := helpers.BuildInitializeRequest()
		initResponse, err := mcpClient.Initialize(ctx, initRequest)
		require.NoError(t, err, "failed to initialize MCP server")

		// Verify server info
		assert.Equal(t, "flow-microstrategy-mcp", initResponse.ServerInfo.Name)
		assert.NotEmpty(t, initResponse.ServerInfo.Version)

		// Verify capabilities
		assert.NotNil(t, initResponse.Capabilities)
		assert.NotNil(t, initResponse.Capabilities.Tools)

		t.Log("Server initialized successfully with expected name and capabilities")
	})

	t.Run("initialization without a database name", func(t *testing.T) {
		t.Parallel()

		args := []string{
			"--flow-uri", cfg.URI,
			"--flow-username", cfg.Username,
			"--flow-password", cfg.Password,
		}

		mcpClient, err := client.NewStdioMCPClient(server, []string{}, args...)
		require.NoError(t, err, "failed to create MCP client")

		defer mcpClient.Close()

		// Test should pass as the default database is neo4j
		initRequest := helpers.BuildInitializeRequest()
		initResponse, err := mcpClient.Initialize(ctx, initRequest)
		assert.Equal(t, "flow-microstrategy-mcp", initResponse.ServerInfo.Name)

	})

	t.Run("initialization with read-only mode enabled", func(t *testing.T) {
		t.Parallel()

		args := []string{
			"--flow-uri", cfg.URI,
			"--flow-username", cfg.Username,
			"--flow-password", cfg.Password,
			"--flow-database", cfg.Database,
			"--flow-read-only", "true",
		}

		mcpClient, err := client.NewStdioMCPClient(server, []string{}, args...)
		require.NoError(t, err, "failed to create MCP client")

		defer mcpClient.Close()

		initRequest := helpers.BuildInitializeRequest()
		initResponse, err := mcpClient.Initialize(ctx, initRequest)
		require.NoError(t, err, "failed to initialize MCP server in read-only mode")

		assert.Equal(t, "flow-microstrategy-mcp", initResponse.ServerInfo.Name)

		listToolsResponse, err := mcpClient.ListTools(ctx, mcp.ListToolsRequest{})
		require.NoError(t, err, "failed to list tools in read-only mode")

		for _, tool := range listToolsResponse.Tools {
			if tool.Name == "write-cypher" {
				t.Fatal("write-cypher tool found using readonly mode")
			}
		}
		// Cypher tools disabled by default: 4 MSTR + 0-1 GDS = 4-5 tools
		assert.GreaterOrEqual(t, len(listToolsResponse.Tools), 4, "read-only mode should have at least 4 tools (MSTR tools)")
		assert.LessOrEqual(t, len(listToolsResponse.Tools), 5, "read-only mode should have at most 5 tools (MSTR + list-gds-procedures)")
	})

	t.Run("initialization with read-only mode and cypher tools enabled", func(t *testing.T) {
		t.Parallel()

		args := []string{
			"--flow-uri", cfg.URI,
			"--flow-username", cfg.Username,
			"--flow-password", cfg.Password,
			"--flow-database", cfg.Database,
			"--flow-read-only", "true",
			"--flow-enable-cypher-tools", "true",
		}

		mcpClient, err := client.NewStdioMCPClient(server, []string{}, args...)
		require.NoError(t, err, "failed to create MCP client")

		defer mcpClient.Close()

		initRequest := helpers.BuildInitializeRequest()
		initResponse, err := mcpClient.Initialize(ctx, initRequest)
		require.NoError(t, err, "failed to initialize MCP server in read-only mode with cypher tools")

		assert.Equal(t, "flow-microstrategy-mcp", initResponse.ServerInfo.Name)

		listToolsResponse, err := mcpClient.ListTools(ctx, mcp.ListToolsRequest{})
		require.NoError(t, err, "failed to list tools in read-only mode with cypher tools")

		for _, tool := range listToolsResponse.Tools {
			if tool.Name == "write-cypher" {
				t.Fatal("write-cypher tool found using readonly mode (even with cypher tools enabled)")
			}
		}
		// 4 MSTR + 2 readonly Cypher (get-schema, read-cypher) + 0-1 GDS = 6-7 tools
		assert.GreaterOrEqual(t, len(listToolsResponse.Tools), 6, "read-only + cypher should have at least 6 tools")
		assert.LessOrEqual(t, len(listToolsResponse.Tools), 7, "read-only + cypher should have at most 7 tools")
	})

	t.Run("initialization with read-only mode disabled", func(t *testing.T) {
		t.Parallel()

		args := []string{
			"--flow-uri", cfg.URI,
			"--flow-username", cfg.Username,
			"--flow-password", cfg.Password,
			"--flow-database", cfg.Database,
			"--flow-read-only", "false",
			"--flow-enable-cypher-tools", "true",
		}

		mcpClient, err := client.NewStdioMCPClient(server, []string{}, args...)
		require.NoError(t, err, "failed to create MCP client")

		defer mcpClient.Close()

		initRequest := helpers.BuildInitializeRequest()
		initResponse, err := mcpClient.Initialize(ctx, initRequest)
		require.NoError(t, err, "failed to initialize MCP server with cypher tools enabled")

		assert.Equal(t, "flow-microstrategy-mcp", initResponse.ServerInfo.Name)

		listToolsResponse, err := mcpClient.ListTools(ctx, mcp.ListToolsRequest{})
		require.NoError(t, err, "failed to list tools with cypher tools enabled")
		// 4 MSTR + 3 Cypher (get-schema, read-cypher, write-cypher) + 0-1 GDS = 7-8 tools
		assert.GreaterOrEqual(t, len(listToolsResponse.Tools), 7, "cypher enabled should have at least 7 tools")
		assert.LessOrEqual(t, len(listToolsResponse.Tools), 8, "cypher enabled should have at most 8 tools")
	})
	t.Run("initialization with telemetry disabled", func(t *testing.T) {
		t.Parallel()

		args := []string{
			"--flow-uri", cfg.URI,
			"--flow-username", cfg.Username,
			"--flow-password", cfg.Password,
			"--flow-database", cfg.Database,
			"--flow-telemetry", "false",
		}

		mcpClient, err := client.NewStdioMCPClient(server, []string{}, args...)
		require.NoError(t, err, "failed to create MCP client")

		defer mcpClient.Close()

		// Test initialization with telemetry disabled
		initRequest := helpers.BuildInitializeRequest()
		initResponse, err := mcpClient.Initialize(ctx, initRequest)
		require.NoError(t, err, "failed to initialize MCP server with telemetry disabled")

		assert.Equal(t, "flow-microstrategy-mcp", initResponse.ServerInfo.Name)

		t.Log("Server initialized successfully with telemetry disabled")
	})

	t.Run("initialization with schema sample size override", func(t *testing.T) {
		t.Parallel()

		args := []string{
			"--flow-uri", cfg.URI,
			"--flow-username", cfg.Username,
			"--flow-password", cfg.Password,
			"--flow-database", cfg.Database,
			"--flow-schema-sample-size", "50",
		}

		mcpClient, err := client.NewStdioMCPClient(server, []string{}, args...)
		require.NoError(t, err, "failed to create MCP client")

		defer mcpClient.Close()

		// Test initialization with custom schema sample size
		initRequest := helpers.BuildInitializeRequest()
		initResponse, err := mcpClient.Initialize(ctx, initRequest)
		require.NoError(t, err, "failed to initialize MCP server with custom schema sample size")

		assert.Equal(t, "flow-microstrategy-mcp", initResponse.ServerInfo.Name)

		t.Log("Server initialized successfully with custom schema sample size")
	})

	t.Run("client initialization with invalid schema sample size", func(t *testing.T) {
		t.Parallel()

		args := []string{
			"--flow-uri", cfg.URI,
			"--flow-username", cfg.Username,
			"--flow-password", cfg.Password,
			"--flow-database", cfg.Database,
			"--flow-schema-sample-size", "not-a-number",
		}

		mcpClient, err := client.NewStdioMCPClient(server, []string{}, args...)
		require.NoError(t, err, "failed to create MCP client")

		defer mcpClient.Close()

		// Server should handle invalid schema sample size gracefully (falling back to default)
		initRequest := helpers.BuildInitializeRequest()
		initResponse, err := mcpClient.Initialize(ctx, initRequest)
		require.NoError(t, err, "failed to initialize MCP server with invalid schema sample size")

		assert.Equal(t, "flow-microstrategy-mcp", initResponse.ServerInfo.Name)

		t.Log("Server initialized successfully with invalid schema sample size (using default value)")
	})
}
