package cli

import (
	"flag"
	"fmt"
	"os"
	"slices"
	"strings"
)

// osExit is a variable that can be mocked in tests
var osExit = os.Exit

const helpText = `flow-microstrategy-mcp - Flow Microstrategy Model Context Protocol Server

Powered by CI&T Flow - Built on top of Neo4j MCP.

Usage:
  flow-microstrategy-mcp [OPTIONS]

Options:
  -h, --help                          Show this help message
  -v, --version                       Show version information
  --flow-uri <URI>                    Neo4j connection URI (overrides environment variable FLOW_URI)
  --flow-username <USERNAME>          Database username (overrides environment variable FLOW_USERNAME)
  --flow-password <PASSWORD>          Database password (overrides environment variable FLOW_PASSWORD)
  --flow-database <DATABASE>          Database name (overrides environment variable FLOW_DATABASE)
  --flow-read-only <BOOLEAN>          Enable read-only mode: true or false (overrides environment variable FLOW_READ_ONLY)
  --flow-telemetry <BOOLEAN>          Enable telemetry: true or false (overrides environment variable FLOW_TELEMETRY)
  --flow-schema-sample-size <INT>     Number of nodes to sample for schema inference (overrides environment variable FLOW_SCHEMA_SAMPLE_SIZE)
  --flow-transport-mode <MODE>        MCP Transport mode (e.g., 'stdio', 'http') (overrides environment variable FLOW_MCP_TRANSPORT)
  --flow-http-port <PORT>             HTTP server port (overrides environment variable FLOW_MCP_HTTP_PORT)
  --flow-http-host <HOST>             HTTP server host (overrides environment variable FLOW_MCP_HTTP_HOST)
  --flow-http-allowed-origins <ORIGINS> Comma-separated list of allowed CORS origins (overrides environment variable FLOW_MCP_HTTP_ALLOWED_ORIGINS)
  --flow-http-tls-enabled <BOOLEAN>   Enable TLS/HTTPS for HTTP server: true or false (overrides environment variable FLOW_MCP_HTTP_TLS_ENABLED)
  --flow-http-tls-cert-file <PATH>    Path to TLS certificate file (overrides environment variable FLOW_MCP_HTTP_TLS_CERT_FILE)
  --flow-http-tls-key-file <PATH>     Path to TLS private key file (overrides environment variable FLOW_MCP_HTTP_TLS_KEY_FILE)
  --flow-enable-cypher-tools <BOOLEAN> Enable generic Cypher tools (get-schema, read-cypher, write-cypher): true or false (overrides environment variable FLOW_ENABLE_CYPHER_TOOLS)

Required Environment Variables:
  FLOW_URI        Neo4j database URI
  FLOW_USERNAME   Database username
  FLOW_PASSWORD   Database password

Optional Environment Variables:
  FLOW_DATABASE   Database name (default: neo4j)
  FLOW_TELEMETRY  Enable/disable telemetry (default: true)
  FLOW_READ_ONLY  Enable read-only mode (default: false)
  FLOW_SCHEMA_SAMPLE_SIZE Number of nodes to sample for schema inference (default: 100)
  FLOW_MCP_TRANSPORT MCP Transport mode (e.g., 'stdio', 'http') (default: stdio)
  FLOW_MCP_HTTP_PORT HTTP server port (default: 443 with TLS, 80 without TLS)
  FLOW_MCP_HTTP_HOST HTTP server host (default: 127.0.0.1)
  FLOW_MCP_HTTP_ALLOWED_ORIGINS Comma-separated list of allowed CORS origins (optional)
  FLOW_MCP_HTTP_TLS_ENABLED Enable TLS/HTTPS for HTTP server (default: false)
  FLOW_MCP_HTTP_TLS_CERT_FILE Path to TLS certificate file (required when TLS is enabled)
  FLOW_MCP_HTTP_TLS_KEY_FILE Path to TLS private key file (required when TLS is enabled)
  FLOW_ENABLE_CYPHER_TOOLS Enable generic Cypher tools (default: false)

Examples:
  # Using environment variables
  FLOW_URI=bolt://localhost:7687 FLOW_USERNAME=neo4j FLOW_PASSWORD=password flow-microstrategy-mcp

  # Using CLI flags (takes precedence over environment variables)
  flow-microstrategy-mcp --flow-uri bolt://localhost:7687 --flow-username neo4j --flow-password password

For more information, visit: https://github.com/brunogc-cit/flow-microstrategy-mcp
`

// Args holds configuration values parsed from command-line flags
type Args struct {
	URI                string
	Username           string
	Password           string
	Database           string
	ReadOnly           string
	Telemetry          string
	SchemaSampleSize   string
	TransportMode      string
	HTTPPort           string
	HTTPHost           string
	HTTPAllowedOrigins string
	HTTPTLSEnabled     string
	HTTPTLSCertFile    string
	HTTPTLSKeyFile     string
	EnableCypherTools  string
}

// this is a list of known configuration flags to be skipped in HandleArgs
// add new config flags here as needed
var argsSlice = []string{
	"--flow-uri",
	"--flow-username",
	"--flow-password",
	"--flow-database",
	"--flow-read-only",
	"--flow-telemetry",
	"--flow-schema-sample-size",
	"--flow-transport-mode",
	"--flow-http-port",
	"--flow-http-host",
	"--flow-http-allowed-origins",
	"--flow-http-tls-enabled",
	"--flow-http-tls-cert-file",
	"--flow-http-tls-key-file",
	"--flow-enable-cypher-tools",
}

// ParseConfigFlags parses CLI flags and returns configuration values.
// It should be called after HandleArgs to ensure help/version flags are processed first.
func ParseConfigFlags() *Args {
	flowURI := flag.String("flow-uri", "", "Neo4j connection URI (overrides FLOW_URI env var)")
	flowUsername := flag.String("flow-username", "", "Neo4j username (overrides FLOW_USERNAME env var)")
	flowPassword := flag.String("flow-password", "", "Neo4j password (overrides FLOW_PASSWORD env var)")
	flowDatabase := flag.String("flow-database", "", "Neo4j database name (overrides FLOW_DATABASE env var)")
	flowReadOnly := flag.String("flow-read-only", "", "Enable read-only mode: true or false (overrides FLOW_READ_ONLY env var)")
	flowTelemetry := flag.String("flow-telemetry", "", "Enable telemetry: true or false (overrides FLOW_TELEMETRY env var)")
	flowSchemaSampleSize := flag.String("flow-schema-sample-size", "", "Number of nodes to sample for schema inference (overrides FLOW_SCHEMA_SAMPLE_SIZE env var)")
	flowTransportMode := flag.String("flow-transport-mode", "", "MCP Transport mode (e.g., 'stdio', 'http') (overrides FLOW_MCP_TRANSPORT env var)")
	flowHTTPPort := flag.String("flow-http-port", "", "HTTP server port (overrides FLOW_MCP_HTTP_PORT env var)")
	flowHTTPHost := flag.String("flow-http-host", "", "HTTP server host (overrides FLOW_MCP_HTTP_HOST env var)")
	flowHTTPAllowedOrigins := flag.String("flow-http-allowed-origins", "", "Comma-separated list of allowed CORS origins (overrides FLOW_MCP_HTTP_ALLOWED_ORIGINS env var)")
	flowHTTPTLSEnabled := flag.String("flow-http-tls-enabled", "", "Enable TLS/HTTPS for HTTP server: true or false (overrides FLOW_MCP_HTTP_TLS_ENABLED env var)")
	flowHTTPTLSCertFile := flag.String("flow-http-tls-cert-file", "", "Path to TLS certificate file (overrides FLOW_MCP_HTTP_TLS_CERT_FILE env var)")
	flowHTTPTLSKeyFile := flag.String("flow-http-tls-key-file", "", "Path to TLS private key file (overrides FLOW_MCP_HTTP_TLS_KEY_FILE env var)")
	flowEnableCypherTools := flag.String("flow-enable-cypher-tools", "", "Enable generic Cypher tools: true or false (overrides FLOW_ENABLE_CYPHER_TOOLS env var)")

	flag.Parse()

	return &Args{
		URI:                *flowURI,
		Username:           *flowUsername,
		Password:           *flowPassword,
		Database:           *flowDatabase,
		ReadOnly:           *flowReadOnly,
		Telemetry:          *flowTelemetry,
		SchemaSampleSize:   *flowSchemaSampleSize,
		TransportMode:      *flowTransportMode,
		HTTPPort:           *flowHTTPPort,
		HTTPHost:           *flowHTTPHost,
		HTTPAllowedOrigins: *flowHTTPAllowedOrigins,
		HTTPTLSEnabled:     *flowHTTPTLSEnabled,
		HTTPTLSCertFile:    *flowHTTPTLSCertFile,
		HTTPTLSKeyFile:     *flowHTTPTLSKeyFile,
		EnableCypherTools:  *flowEnableCypherTools,
	}
}

// HandleArgs processes command-line arguments for version and help flags.
// It exits the program after displaying the requested information.
// If unknown flags are encountered, it prints an error message and exits.
// Known configuration flags are skipped here so that the flag package in main.go can handle them properly.
func HandleArgs(version string) {
	if len(os.Args) <= 1 {
		return
	}

	flags := make(map[string]bool)
	var err error
	i := 1 // we start from 1 because os.Args[0] is the program name ("flow-microstrategy-mcp") - not a flag

	for i < len(os.Args) {
		arg := os.Args[i]

		// Allow configuration flags to be parsed by the flag package
		if slices.Contains(argsSlice, arg) {
			// Check if there's a value following the flag
			if i+1 >= len(os.Args) {
				err = fmt.Errorf("%s requires a value", arg)
				break
			}
			// Check if next argument is another flag (starts with -)
			nextArg := os.Args[i+1]
			if strings.HasPrefix(nextArg, "-") {
				err = fmt.Errorf("%s requires a value (got flag %s instead)", arg, nextArg)
				break
			}
			// Safe to skip flag and value - let flag package handle them
			i += 2
			continue
		}

		switch arg {
		case "-h", "--help":
			flags["help"] = true
			i++
		case "-v", "--version":
			flags["version"] = true
			i++
		default:
			if arg == "--" {
				// Stop processing our flags, let flag package handle the rest
				i = len(os.Args)
			} else {
				err = fmt.Errorf("unknown flag or argument: %s", arg)
				i++
			}
		}
		// Exit loop if an error occurred
		if err != nil {
			break
		}
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		osExit(1)
	}

	if flags["help"] {
		fmt.Print(helpText)
		osExit(0)
	}

	if flags["version"] {
		fmt.Printf("flow-microstrategy-mcp version: %s\n", version)
		osExit(0)
	}
}
