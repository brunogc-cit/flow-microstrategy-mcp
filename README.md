# Flow Microstrategy MCP

> **Powered by CI&T Flow**
>
> Flow Microstrategy MCP is built on top of the Neo4j MCP open-source project and customized for CI&T Flow. Visit [flow.ciandt.com](https://flow.ciandt.com) to learn more about CI&T Flow's AI productivity platform.

Model Context Protocol (MCP) server for Neo4j databases.

## Links

- [GitHub Repository](https://github.com/brunogc-cit/flow-microstrategy-mcp)

## Prerequisites

- A running Neo4j database instance; options include [Aura](https://neo4j.com/product/auradb/), [neo4j–desktop](https://neo4j.com/download/) or [self-managed](https://neo4j.com/deployment-center/#gdb-tab).
- APOC plugin installed in the Neo4j instance.
- Any MCP-compatible client (e.g. [VSCode](https://code.visualstudio.com/) with [MCP support](https://code.visualstudio.com/docs/copilot/customization/mcp-servers))

## Startup Checks & Adaptive Operation

The server performs several pre-flight checks at startup to ensure your environment is correctly configured.

**STDIO Mode - Mandatory Requirements**
In STDIO mode, the server verifies the following core requirements. If any of these checks fail (e.g., due to an invalid configuration, incorrect credentials, or a missing APOC installation), the server will not start:

- A valid connection to your Neo4j instance.
- The ability to execute queries.
- The presence of the APOC plugin.

**HTTP Mode - Verification Skipped**
In HTTP mode, startup verification checks are skipped because credentials come from per-request Basic Auth headers. The server starts immediately without connecting to Neo4j at startup.

**Optional Requirements**
If an optional dependency is missing, the server will start in an adaptive mode. For instance, if the Graph Data Science (GDS) library is not detected in your Neo4j installation, the server will still launch but will automatically disable all GDS-related tools, such as `list-gds-procedures`. All other tools will remain available.

## Installation (Binary)

Releases: https://github.com/brunogc-cit/flow-microstrategy-mcp/releases

1. Download the archive for your OS/arch.
2. Extract and place `flow-microstrategy-mcp` in a directory present in your PATH variables (see examples below).

Mac / Linux:

```bash
chmod +x flow-microstrategy-mcp
sudo mv flow-microstrategy-mcp /usr/local/bin/
```

Windows (PowerShell / cmd):

```powershell
move flow-microstrategy-mcp.exe C:\Windows\System32
```

Verify the flow-microstrategy-mcp installation:

```bash
flow-microstrategy-mcp -v
```

Should print the installed version.

## Transport Modes

The Flow Microstrategy MCP server supports two transport modes:

- **STDIO** (default): Standard MCP communication via stdin/stdout for desktop clients (Claude Desktop, VSCode)
- **HTTP**: RESTful HTTP server with per-request Bearer token or Basic Authentication for web-based clients and multi-tenant scenarios

### Key Differences

| Aspect               | STDIO                                                      | HTTP                                                                       |
| -------------------- | ---------------------------------------------------------- | -------------------------------------------------------------------------- |
| Startup Verification | Required - server verifies APOC, connectivity, queries     | Skipped - server starts immediately                                        |
| Credentials          | Set via environment variables                              | Per-request via Bearer token or Basic Auth headers                         |
| Telemetry            | Collects Neo4j version, edition, Cypher version at startup | Reports "unknown-http-mode" - actual version info not available at startup |

See the [Client Setup Guide](docs/CLIENT_SETUP.md) for configuration instructions for both modes.

## TLS/HTTPS Configuration

When using HTTP transport mode, you can enable TLS/HTTPS for secure communication:

### Environment Variables

- `FLOW_MCP_HTTP_TLS_ENABLED` - Enable TLS/HTTPS: `true` or `false` (default: `false`)
- `FLOW_MCP_HTTP_TLS_CERT_FILE` - Path to TLS certificate file (required when TLS is enabled)
- `FLOW_MCP_HTTP_TLS_KEY_FILE` - Path to TLS private key file (required when TLS is enabled)
- `FLOW_MCP_HTTP_PORT` - HTTP server port (default: `443` when TLS enabled, `80` when TLS disabled)

### Security Configuration

- **Minimum TLS Version**: Hardcoded to TLS 1.2 (allows TLS 1.3 negotiation)
- **Cipher Suites**: Uses Go's secure default cipher suites
- **Default Port**: Automatically uses port 443 when TLS is enabled (standard HTTPS port)

### Example Configuration

```bash
export FLOW_URI="bolt://localhost:7687"
export FLOW_MCP_TRANSPORT="http"
export FLOW_MCP_HTTP_TLS_ENABLED="true"
export FLOW_MCP_HTTP_TLS_CERT_FILE="/path/to/cert.pem"
export FLOW_MCP_HTTP_TLS_KEY_FILE="/path/to/key.pem"

flow-microstrategy-mcp
# Server will listen on https://127.0.0.1:443 by default
```

**Production Usage**: Use certificates from a trusted Certificate Authority (e.g., Let's Encrypt, or your organisation) for production deployments.

For detailed instructions on certificate generation, testing TLS, and production deployment, see [CONTRIBUTING.md](CONTRIBUTING.md#tlshttps-configuration).

## Configuration Options

The `flow-microstrategy-mcp` server can be configured using environment variables or CLI flags. CLI flags take precedence over environment variables.

### Environment Variables

See the [Client Setup Guide](docs/CLIENT_SETUP.md) for configuration examples.

### CLI Flags

You can override any environment variable using CLI flags:

```bash
flow-microstrategy-mcp --flow-uri "bolt://localhost:7687" \
          --flow-username "neo4j" \
          --flow-password "password" \
          --flow-database "neo4j" \
          --flow-read-only false \
          --flow-telemetry true
```

Available flags:

- `--flow-uri` - Neo4j connection URI (overrides FLOW_URI)
- `--flow-username` - Database username (overrides FLOW_USERNAME)
- `--flow-password` - Database password (overrides FLOW_PASSWORD)
- `--flow-database` - Database name (overrides FLOW_DATABASE)
- `--flow-read-only` - Enable read-only mode: `true` or `false` (overrides FLOW_READ_ONLY)
- `--flow-telemetry` - Enable telemetry: `true` or `false` (overrides FLOW_TELEMETRY)
- `--flow-schema-sample-size` - Modify the sample size used to infer the Neo4j schema
- `--flow-transport-mode` - Transport mode: `stdio` or `http` (overrides FLOW_MCP_TRANSPORT)
- `--flow-http-host` - HTTP server host (overrides FLOW_MCP_HTTP_HOST)
- `--flow-http-port` - HTTP server port (overrides FLOW_MCP_HTTP_PORT)
- `--flow-http-tls-enabled` - Enable TLS/HTTPS: `true` or `false` (overrides FLOW_MCP_HTTP_TLS_ENABLED)
- `--flow-http-tls-cert-file` - Path to TLS certificate file (overrides FLOW_MCP_HTTP_TLS_CERT_FILE)
- `--flow-http-tls-key-file` - Path to TLS private key file (overrides FLOW_MCP_HTTP_TLS_KEY_FILE)
- `--flow-enable-cypher-tools` - Enable generic Cypher tools: `true` or `false` (overrides FLOW_ENABLE_CYPHER_TOOLS, default: `false`)

Use `flow-microstrategy-mcp --help` to see all available options.

## Client Configuration

To configure MCP clients (VSCode, Claude Desktop, etc.) to use the Flow Microstrategy MCP server, see:

**[Client Setup Guide](docs/CLIENT_SETUP.md)** – Complete configuration for STDIO and HTTP modes

## Tools & Usage

### MicroStrategy Migration Tools

These tools enable LLM agents to search for MicroStrategy objects and trace their lineage:

| Tool                | ReadOnly | Purpose                                           | Notes                                                                 |
| ------------------- | -------- | ------------------------------------------------- | --------------------------------------------------------------------- |
| `search-metrics`    | `true`   | Find Metrics by GUID or name                      | Accepts full GUIDs, partial GUIDs (8+ chars), or name search terms    |
| `search-attributes` | `true`   | Find Attributes by GUID or name                   | Accepts full GUIDs, partial GUIDs (8+ chars), or name search terms    |
| `trace-metric`      | `true`   | Trace Metric lineage (reports, tables, deps)      | Returns reports using it, source tables, and direct dependencies      |
| `trace-attribute`   | `true`   | Trace Attribute lineage (reports, tables, deps)   | Returns reports using it, source tables, and direct dependencies      |

### Generic Cypher Tools (opt-in)

These tools are disabled by default and can be enabled by setting `FLOW_ENABLE_CYPHER_TOOLS=true`. They allow users to run arbitrary Cypher queries against the Neo4j database:

| Tool                  | ReadOnly | Purpose                                              | Notes                                                                 |
| --------------------- | -------- | ---------------------------------------------------- | --------------------------------------------------------------------- |
| `get-schema`          | `true`   | Retrieve database schema (labels, types, properties) | Useful for understanding the database structure before querying        |
| `read-cypher`         | `true`   | Execute read-only Cypher queries                     | Validates queries are read-only before execution                      |
| `write-cypher`        | `false`  | Execute write Cypher queries                         | Hidden in read-only mode. Use with caution in production               |

### Optional Tools

| Tool                  | ReadOnly | Purpose                                              | Notes                                                                 |
| --------------------- | -------- | ---------------------------------------------------- | --------------------------------------------------------------------- |
| `list-gds-procedures` | `true`   | List GDS procedures available in the Neo4j instance  | Only available if GDS library is installed                            |

### Readonly mode flag

Enable readonly mode by setting the `FLOW_READ_ONLY` environment variable to `true` (for example, `"FLOW_READ_ONLY": "true"`). Accepted values are `true` or `false` (default: `false`).

You can also override this setting using the `--flow-read-only` CLI flag:

```bash
flow-microstrategy-mcp --flow-uri "bolt://localhost:7687" --flow-username "neo4j" --flow-password "password" --flow-read-only true
```

When enabled, write tools (for example, `write-cypher`) are not exposed to clients.

### Query Classification

The `read-cypher` tool performs an extra round-trip to the Neo4j database to guarantee read-only operations.

Important notes:

- **Write operations**: `CREATE`, `MERGE`, `DELETE`, `SET`, etc., are treated as non-read queries.
- **Admin queries**: Commands like `SHOW USERS`, `SHOW DATABASES`, etc., are treated as non-read queries and must use `write-cypher` instead.
- **Profile queries**: `EXPLAIN PROFILE` queries are treated as non-read queries, even if the underlying statement is read-only.
- **Schema operations**: `CREATE INDEX`, `DROP CONSTRAINT`, etc., are treated as non-read queries.

## Example Natural Language Prompts

Below are some example prompts you can try in Copilot or any other MCP client:

- "What does my Neo4j instance contain? List all node labels, relationship types, and property keys."
- "Find all Person nodes and their relationships in my Neo4j instance."
- "Create a new User node with a name 'John' in my Neo4j instance."

## Security tips:

- Use a restricted Neo4j user for exploration.
- Review generated Cypher before executing in production databases.

## Logging

The server uses structured logging with support for multiple log levels and output formats.

### Configuration

**Log Level** (`FLOW_LOG_LEVEL`, default: `info`)

Controls the verbosity of log output. Supports all [MCP log levels](https://modelcontextprotocol.io/specification/2025-03-26/server/utilities/logging#log-levels): `debug`, `info`, `notice`, `warning`, `error`, `critical`, `alert`, `emergency`.

**Log Format** (`FLOW_LOG_FORMAT`, default: `text`)

Controls the output format:

- `text` - Human-readable text format (default)
- `json` - Structured JSON format (useful for log aggregation)

## Telemetry

By default, `flow-microstrategy-mcp` collects anonymous usage data to help us improve the product.
This includes information like the tools being used, the operating system, and CPU architecture.
We do not collect any personal or sensitive information.

To disable telemetry, set the `FLOW_TELEMETRY` environment variable to `"false"`. Accepted values are `true` or `false` (default: `true`).

You can also use the `--flow-telemetry` CLI flag to override this setting.

## Documentation

**[Client Setup Guide](docs/CLIENT_SETUP.md)** – Configure VSCode, Claude Desktop, and other MCP clients (STDIO and HTTP modes)
**[Contributing Guide](CONTRIBUTING.md)** – Contribution workflow, development environment, mocks & testing

Issues / feedback: open a GitHub issue with reproduction details (omit sensitive data).
