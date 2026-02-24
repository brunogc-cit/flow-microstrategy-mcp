# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Flow Microstrategy MCP is a Model Context Protocol (MCP) server for Neo4j, built on top of Neo4j MCP and customized for CI&T Flow. It enables AI/LLM clients to interact with Neo4j databases through standardized MCP tools.

## Build & Development Commands

```bash
# Install dependencies
go mod download

# Run tests with coverage
go test ./... -cover

# Run tests for a specific package
go test ./internal/tools/cypher -v

# Run a single test
go test ./internal/tools/cypher -v -run TestGetSchemaHandler

# Build binary
go build -C cmd/flow-microstrategy-mcp -o ../../bin/

# Run from source
go run ./cmd/flow-microstrategy-mcp

# Install globally
go install -C cmd/flow-microstrategy-mcp

# Run integration tests (requires Docker)
go test ./test/integration/... -tags=integration -v

# Run e2e tests (requires Docker)
go test ./test/e2e/... -tags=e2e -v

# Regenerate mocks after changing interfaces
go generate ./...

# Test with MCP Inspector
npx @modelcontextprotocol/inspector go run ./cmd/flow-microstrategy-mcp
```

## Required Environment Variables

**STDIO mode (default):**
```bash
export FLOW_URI="bolt://localhost:7687"
export FLOW_USERNAME="neo4j"
export FLOW_PASSWORD="password"
export FLOW_ENABLE_CYPHER_TOOLS="true"  # Optional: enables get-schema, read-cypher, write-cypher (default: false)
```

**HTTP mode:**
```bash
export FLOW_URI="bolt://localhost:7687"
export FLOW_MCP_TRANSPORT="http"
# Credentials come from per-request Basic Auth headers
```

## Architecture

### Entry Point
- `cmd/flow-microstrategy-mcp/main.go` - CLI entry, config loading, driver init, server startup

### Core Packages

**internal/server/**
- `server.go` - MCP server lifecycle, transport modes (STDIO/HTTP), requirement verification
- `tools_register.go` - Tool registration with category-based filtering (cypher/GDS)
- `middleware.go` - HTTP middleware (CORS, auth, logging)

**internal/tools/**
- `types.go` - `ToolDependencies` struct (DBService, AnalyticsService)
- `cypher/` - Generic Cypher tools (opt-in via FLOW_ENABLE_CYPHER_TOOLS): `get-schema`, `read-cypher`, `write-cypher`
- `gds/` - GDS tools: `list-gds-procedures`

**internal/database/**
- `interfaces.go` - `Service` interface (QueryExecutor + RecordFormatter + Helpers)
- `service.go` - Neo4j driver wrapper implementation
- `mocks/` - Generated mocks for testing

**internal/config/** - Environment and CLI config loading with validation

**internal/analytics/** - Telemetry event emission

### Tool Pattern

Each tool has three files:
1. `*_spec.go` - Tool definition using `mcp.NewTool()` with schema and annotations
2. `*_handler.go` - Handler implementation returning `mcp.ToolHandler`
3. `*_handler_test.go` - Tests using gomock

Example tool registration in `tools_register.go`:
```go
{
    category: cypherCategory,
    definition: server.ServerTool{
        Tool:    cypher.GetSchemaSpec(),
        Handler: cypher.GetSchemaHandler(deps, sampleSize),
    },
    readonly: true,  // Determines visibility in read-only mode
}
```

### Transport Modes
- **STDIO**: Verifies Neo4j requirements (APOC, connectivity) at startup
- **HTTP**: Skips startup verification; per-request Basic Auth; lazy requirement verification on first client initialize

### Test Structure
- `internal/*_test.go` - Unit tests with mocks (no Neo4j required)
- `test/integration/` - Integration tests with testcontainers (tag: `integration`)
- `test/e2e/` - End-to-end tests with full server binary (tag: `e2e`)

## MCP Error Handling

Return errors through tool results, not Go errors:
```go
// Business/operational errors - use MCP tool result
return mcp.NewToolResultError("Operation failed: " + err.Error()), nil

// Success
return mcp.NewToolResultText(result), nil
```

## Adding New Tools

1. Create spec in `internal/tools/<category>/<tool>_spec.go`
2. Create handler in `internal/tools/<category>/<tool>_handler.go`
3. Add tests in `internal/tools/<category>/<tool>_handler_test.go`
4. Register in `internal/server/tools_register.go` under appropriate category
5. Set `readonly: true/false` based on whether tool mutates state

## Release & Deployment Workflow

### Current Workflow

```
Developer → Push to branch → Create PR
                ↓
        build-and-test.yml runs (unit, integration, e2e)
                ↓
        PR merged to main
                ↓
        If .changes/unreleased/* modified:
            changie.yml → creates Release PR
                ↓
        Release PR merged:
            CHANGELOG.md updated
                ↓
            release.yml → GoReleaser → GitHub Release + Binaries
```

### Target Workflow (After Azure Deployment Implementation)

```
Developer → Push to branch → Create PR
                ↓
        ┌───────────────────────────────────────────────────────────┐
        │                    PR OPENED/UPDATED                       │
        ├───────────────────────────────────────────────────────────┤
        │  Parallel execution:                                       │
        │  ├─ build-and-test.yml (unit, integration, e2e)           │
        │  ├─ pr-check.yml (validates changie entry exists)         │
        │  └─ preview.yml (deploy ephemeral preview environment)    │
        │                                                            │
        │  → Comment on PR with preview URL:                         │
        │    https://ca-mcp-asos-pr-{N}.*.azurecontainerapps.io     │
        └───────────────────────────────────────────────────────────┘
                ↓
        ┌───────────────────────────────────────────────────────────┐
        │                    PR MERGED TO MAIN                       │
        ├───────────────────────────────────────────────────────────┤
        │  preview.yml (on: closed) → Delete ephemeral environment  │
        │                                                            │
        │  If .changes/unreleased/* modified:                        │
        │      changie.yml → creates Release PR                      │
        └───────────────────────────────────────────────────────────┘
                ↓
        ┌───────────────────────────────────────────────────────────┐
        │                    RELEASE PR MERGED                       │
        ├───────────────────────────────────────────────────────────┤
        │  CHANGELOG.md updated by changie merge                     │
        │          ↓                                                 │
        │  release.yml triggered:                                    │
        │  ├─ GoReleaser → GitHub Release + Multi-platform Binaries │
        │  └─ deploy-prod.yml → Deploy to Azure Production          │
        │                                                            │
        │  Production URL:                                           │
        │    https://ca-mcp-asos-prod.*.azurecontainerapps.io       │
        └───────────────────────────────────────────────────────────┘
```

### Changie Commands

```bash
# Check current version
changie latest

# Preview next version
changie next auto

# Create new change entry (required for code PRs)
changie new

# Batch changes (CI only - don't run locally)
changie batch auto

# Merge into CHANGELOG (CI only)
changie merge
```

### GitHub Secrets

| Secret | Purpose |
|--------|---------|
| `FLOW_URL` | Neo4j connection URI for tests |
| `FLOW_USERNAME` | Neo4j username for tests |
| `FLOW_PASSWORD` | Neo4j password for tests |
| `FLOW_URI_PREVIEW` | Neo4j URI for preview environments |
| `FLOW_USERNAME_PREVIEW` | Neo4j username for preview |
| `FLOW_PASSWORD_PREVIEW` | Neo4j password for preview |
| `FLOW_URI_PROD` | Neo4j URI for production |
| `FLOW_USERNAME_PROD` | Neo4j username for production |
| `FLOW_PASSWORD_PROD` | Neo4j password for production |
| `AZURE_CREDENTIALS` | Service Principal JSON for Azure deployment |
| `TEAM_GRAPHQL_PERSONAL_ACCESS_TOKEN` | GitHub API access for changie/release |
| `MACOS_SIGN_P12` | macOS code signing certificate |
| `MACOS_SIGN_PASSWORD` | macOS code signing password |
| `MACOS_NOTARY_*` | macOS notarization credentials |
