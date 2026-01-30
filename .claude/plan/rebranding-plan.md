# Rebranding Plan: Neo4j MCP → Flow Microstrategy MCP

## Summary

Full rebranding of the project with the following changes:
- **Project name:** Neo4j MCP → Flow Microstrategy MCP
- **Binary name:** neo4j-mcp → flow-microstrategy-mcp
- **Module path:** github.com/neo4j/mcp → github.com/brunogc-cit/flow-microstrategy-mcp
- **Environment variables:** NEO4J_* → FLOW_NEO4J_*
- **Attribution:** Add CI&T Flow attribution message

---

## Phase 1: Module Path Change (56 files)

Update `go.mod` and all import statements:

**Files to modify:**
- `go.mod` - Change module declaration
- All 55 Go files with imports (use `replace_all` for efficiency)

**Change:**
```
github.com/neo4j/mcp → github.com/brunogc-cit/flow-microstrategy-mcp
```

---

## Phase 2: Binary/Executable Name (14 files)

Rename binary from `neo4j-mcp` to `flow-microstrategy-mcp`:

| File | Changes |
|------|---------|
| `cmd/neo4j-mcp/main.go` | Rename directory to `cmd/flow-microstrategy-mcp/` |
| `.goreleaser.yaml` | Project name, main path |
| `Dockerfile` | Build path, entrypoint |
| `Taskfile.yml` | Binary references |
| `manifest.json` | Name field, command paths |
| `CONTRIBUTING.md` | Command examples |
| `README.md` | Installation, usage examples |
| `docs/*.md` | Command examples |
| `test/e2e/helpers/build_server.go` | Build path |

---

## Phase 3: Environment Variables (21 files)

Change prefix from `NEO4J_*` to `FLOW_NEO4J_*`:

| Old Variable | New Variable |
|--------------|--------------|
| `NEO4J_URI` | `FLOW_NEO4J_URI` |
| `NEO4J_USERNAME` | `FLOW_NEO4J_USERNAME` |
| `NEO4J_PASSWORD` | `FLOW_NEO4J_PASSWORD` |
| `NEO4J_DATABASE` | `FLOW_NEO4J_DATABASE` |
| `NEO4J_READ_ONLY` | `FLOW_NEO4J_READ_ONLY` |
| `NEO4J_TELEMETRY` | `FLOW_NEO4J_TELEMETRY` |
| `NEO4J_LOG_LEVEL` | `FLOW_NEO4J_LOG_LEVEL` |
| `NEO4J_LOG_FORMAT` | `FLOW_NEO4J_LOG_FORMAT` |
| `NEO4J_SCHEMA_SAMPLE_SIZE` | `FLOW_NEO4J_SCHEMA_SAMPLE_SIZE` |
| `NEO4J_MCP_TRANSPORT` | `FLOW_NEO4J_MCP_TRANSPORT` |
| `NEO4J_MCP_HTTP_HOST` | `FLOW_NEO4J_MCP_HTTP_HOST` |
| `NEO4J_MCP_HTTP_PORT` | `FLOW_NEO4J_MCP_HTTP_PORT` |
| `NEO4J_MCP_HTTP_ALLOWED_ORIGINS` | `FLOW_NEO4J_MCP_HTTP_ALLOWED_ORIGINS` |
| `NEO4J_MCP_HTTP_TLS_ENABLED` | `FLOW_NEO4J_MCP_HTTP_TLS_ENABLED` |
| `NEO4J_MCP_HTTP_TLS_CERT_FILE` | `FLOW_NEO4J_MCP_HTTP_TLS_CERT_FILE` |
| `NEO4J_MCP_HTTP_TLS_KEY_FILE` | `FLOW_NEO4J_MCP_HTTP_TLS_KEY_FILE` |

**Key files:**
- `internal/config/config.go` - Environment variable parsing
- `internal/cli/args.go` - CLI flag descriptions
- `README.md`, `CONTRIBUTING.md`, `docs/*.md` - Documentation

---

## Phase 4: Server/Branding Strings (5 files)

| File | Change |
|------|--------|
| `internal/server/server.go:68` | Server name: `"neo4j-mcp"` → `"flow-microstrategy-mcp"` |
| `internal/server/server.go:72` | Instructions text |
| `internal/server/middleware.go:55,70,71,78` | Auth realm: `"Neo4j MCP Server"` → `"Flow Microstrategy MCP"` |
| `Dockerfile:4` | Container label |
| `manifest.json` | Keywords and metadata |

---

## Phase 5: Documentation Updates (8 files)

| File | Changes |
|------|---------|
| `README.md` | Title, description, add attribution |
| `CONTRIBUTING.md` | Title, references |
| `CLAUDE.md` | Project description |
| `docs/CLIENT_SETUP.md` | Examples, references |
| `docs/TLS_SETUP.md` | Title, examples |
| `docs/BUILD_MCPB.md` | References |
| `test/integration/README.md` | Description |
| `test/e2e/README.md` | Description |

---

## Phase 6: Attribution Message

Add to `README.md` near the top:

```markdown
> **Powered by CI&T Flow**
>
> Flow Microstrategy MCP is built on top of the Neo4j MCP open-source project and customized for CI&T Flow. Visit [flow.ciandt.com](https://flow.ciandt.com) to learn more about CI&T Flow's AI productivity platform.
```

Also add abbreviated attribution in:
- Server instructions string
- CLI help/version output

---

## Files NOT Changed (Historical)

The following files will remain unchanged to preserve history:
- `.changes/*.md` - Version release notes
- `CHANGELOG.md` - Historical changelog

---

## Verification

After changes:

1. **Build test:**
   ```bash
   go build -C cmd/flow-microstrategy-mcp -o ../../bin/
   ```

2. **Unit tests:**
   ```bash
   go test ./... -cover
   ```

3. **Binary runs:**
   ```bash
   ./bin/flow-microstrategy-mcp --help
   ./bin/flow-microstrategy-mcp -v
   ```

4. **Environment variables work:**
   ```bash
   export FLOW_NEO4J_URI="bolt://localhost:7687"
   export FLOW_NEO4J_USERNAME="neo4j"
   export FLOW_NEO4J_PASSWORD="password"
   go run ./cmd/flow-microstrategy-mcp
   ```

---

## Impact Summary

| Category | Files Affected |
|----------|----------------|
| Module imports | 56 |
| Environment variables | 21 |
| Binary name | 14 |
| Documentation | 8 |
| Server strings | 5 |
| **Total unique files** | ~70 |
