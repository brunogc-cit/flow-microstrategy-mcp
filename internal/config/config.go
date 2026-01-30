package config

import (
	"crypto/tls"
	"fmt"
	"log"
	"os"
	"slices"
	"strconv"

	"github.com/brunogc-cit/flow-microstrategy-mcp/internal/logger"
)

type TransportMode string

const (
	// DefaultSchemaSampleSize is the default number of nodes to sample per label when inferring schema
	DefaultSchemaSampleSize int32         = 100
	TransportModeStdio      TransportMode = "stdio"
	TransportModeHTTP       TransportMode = "http"
)

// ValidTransportModes defines the allowed transport mode values
var ValidTransportModes = []TransportMode{TransportModeStdio, TransportModeHTTP}

// Config holds the application configuration
type Config struct {
	URI                string
	Username           string
	Password           string
	Database           string
	ReadOnly           bool // If true, disables write tools
	Telemetry          bool // If false, disables telemetry
	LogLevel           string
	LogFormat          string
	SchemaSampleSize   int32
	TransportMode      TransportMode // MCP Transport mode (e.g., "stdio", "http")
	HTTPPort           string        // HTTP server port (default: "443" with TLS, "80" without TLS)
	HTTPHost           string        // HTTP server host (default: "127.0.0.1")
	HTTPAllowedOrigins string        // Comma-separated list of allowed CORS origins (optional, "*" for all)
	HTTPTLSEnabled     bool          // If true, enables TLS/HTTPS for HTTP server (default: false)
	HTTPTLSCertFile    string        // Path to TLS certificate file (required if HTTPTLSEnabled is true)
	HTTPTLSKeyFile     string        // Path to TLS private key file (required if HTTPTLSEnabled is true)
	APIToken           string        // Fixed API token for HTTP mode authentication (optional, enables server-side Neo4j credentials)
	MCPVersion         string        // MCP version string
}

// Validate validates the configuration and returns an error if invalid
func (c *Config) Validate() error {
	if c == nil {
		return fmt.Errorf("configuration is required but was nil")
	}

	// URI is always required
	if c.URI == "" {
		return fmt.Errorf("Neo4j URI is required but was empty")
	}

	// Default to stdio if not provided (maintains backward compatibility with tests constructing Config directly)
	if c.TransportMode == "" {
		c.TransportMode = TransportModeStdio
	}

	// Validate transport mode
	if !slices.Contains(ValidTransportModes, c.TransportMode) {
		return fmt.Errorf("invalid transport mode '%s', must be one of %v", c.TransportMode, ValidTransportModes)
	}

	// For STDIO mode, require username and password from environment
	// For HTTP mode with API token, require username and password (server-side credentials)
	// For HTTP mode without API token, credentials come from per-request Basic Auth headers
	switch c.TransportMode {
	case TransportModeStdio:
		if c.Username == "" {
			return fmt.Errorf("Neo4j username is required for STDIO mode")
		}
		if c.Password == "" {
			return fmt.Errorf("Neo4j password is required for STDIO mode")
		}
	case TransportModeHTTP:
		if c.APIToken != "" {
			// API token mode: server authenticates clients with token, uses configured Neo4j credentials
			if c.Username == "" {
				return fmt.Errorf("Neo4j username is required when using API token authentication (FLOW_API_TOKEN)")
			}
			if c.Password == "" {
				return fmt.Errorf("Neo4j password is required when using API token authentication (FLOW_API_TOKEN)")
			}
		} else if c.Username != "" || c.Password != "" {
			// Per-request auth mode: credentials should not be configured
			return fmt.Errorf("Neo4j username and password should not be set for HTTP transport mode without API token; credentials are provided per-request via Basic Auth headers, or set FLOW_API_TOKEN for server-side authentication")
		}
	}

	// For HTTP mode with TLS enabled, require certificate and key files
	if c.TransportMode == TransportModeHTTP && c.HTTPTLSEnabled {
		if c.HTTPTLSCertFile == "" {
			return fmt.Errorf("TLS certificate file is required when TLS is enabled (set FLOW_MCP_HTTP_TLS_CERT_FILE)")
		}
		if c.HTTPTLSKeyFile == "" {
			return fmt.Errorf("TLS key file is required when TLS is enabled (set FLOW_MCP_HTTP_TLS_KEY_FILE)")
		}

		// Validate that certificate and key files exist and are valid
		// This provides early, clear error messages before attempting to start the server
		if _, err := tls.LoadX509KeyPair(c.HTTPTLSCertFile, c.HTTPTLSKeyFile); err != nil {
			return fmt.Errorf("failed to load TLS certificate and key: %w", err)
		}
	}

	return nil
}

// CLIOverrides holds optional configuration values from CLI flags
type CLIOverrides struct {
	URI            string
	Username       string
	Password       string
	Database       string
	ReadOnly       string
	Telemetry      string
	TransportMode  string
	Port           string
	Host           string
	AllowedOrigins string
	TLSEnabled     string
	TLSCertFile    string
	TLSKeyFile     string
}

// LoadConfig loads configuration from environment variables, applies CLI overrides, and validates.
// CLI flag values take precedence over environment variables.
// Returns an error if required configuration is missing or invalid.
func LoadConfig(cliOverrides *CLIOverrides) (*Config, error) {
	logLevel := GetEnvWithDefault("FLOW_LOG_LEVEL", "info")
	logFormat := GetEnvWithDefault("FLOW_LOG_FORMAT", "text")

	// Validate log level and use default if invalid
	if !slices.Contains(logger.ValidLogLevels, logLevel) {
		fmt.Fprintf(os.Stderr, "Warning: invalid FLOW_LOG_LEVEL '%s', using default 'info'. Valid values: %v\n", logLevel, logger.ValidLogLevels)
		logLevel = "info"
	}

	// Validate log format and use default if invalid
	if !slices.Contains(logger.ValidLogFormats, logFormat) {
		fmt.Fprintf(os.Stderr, "Warning: invalid FLOW_LOG_FORMAT '%s', using default 'text'. Valid values: %v\n", logFormat, logger.ValidLogFormats)
		logFormat = "text"
	}

	cfg := &Config{
		URI:                GetEnv("FLOW_URI"),
		Username:           GetEnv("FLOW_USERNAME"),
		Password:           GetEnv("FLOW_PASSWORD"),
		Database:           GetEnvWithDefault("FLOW_DATABASE", "neo4j"),
		ReadOnly:           ParseBool(GetEnv("FLOW_READ_ONLY"), false),
		Telemetry:          ParseBool(GetEnv("FLOW_TELEMETRY"), true),
		LogLevel:           logLevel,
		LogFormat:          logFormat,
		SchemaSampleSize:   ParseInt32(GetEnv("FLOW_SCHEMA_SAMPLE_SIZE"), DefaultSchemaSampleSize),
		TransportMode:      GetTransportModeWithDefault("FLOW_MCP_TRANSPORT", TransportModeStdio),
		HTTPPort:           GetEnv("FLOW_MCP_HTTP_PORT"), // Default set after TLS determination
		HTTPHost:           GetEnvWithDefault("FLOW_MCP_HTTP_HOST", "127.0.0.1"),
		HTTPAllowedOrigins: GetEnv("FLOW_MCP_HTTP_ALLOWED_ORIGINS"),
		HTTPTLSEnabled:     ParseBool(GetEnv("FLOW_MCP_HTTP_TLS_ENABLED"), false),
		HTTPTLSCertFile:    GetEnv("FLOW_MCP_HTTP_TLS_CERT_FILE"),
		HTTPTLSKeyFile:     GetEnv("FLOW_MCP_HTTP_TLS_KEY_FILE"),
		APIToken:           GetEnv("FLOW_API_TOKEN"),
	}

	// Apply CLI overrides if provided
	if cliOverrides != nil {
		if cliOverrides.URI != "" {
			cfg.URI = cliOverrides.URI
		}
		if cliOverrides.Username != "" {
			cfg.Username = cliOverrides.Username
		}
		if cliOverrides.Password != "" {
			cfg.Password = cliOverrides.Password
		}
		if cliOverrides.Database != "" {
			cfg.Database = cliOverrides.Database
		}
		if cliOverrides.ReadOnly != "" {
			cfg.ReadOnly = ParseBool(cliOverrides.ReadOnly, false)
		}
		if cliOverrides.Telemetry != "" {
			cfg.Telemetry = ParseBool(cliOverrides.Telemetry, true)
		}
		if cliOverrides.TransportMode != "" {
			cfg.TransportMode = TransportMode(cliOverrides.TransportMode)
		}
		if cliOverrides.Port != "" {
			cfg.HTTPPort = cliOverrides.Port
		}
		if cliOverrides.Host != "" {
			cfg.HTTPHost = cliOverrides.Host
		}
		if cliOverrides.AllowedOrigins != "" {
			cfg.HTTPAllowedOrigins = cliOverrides.AllowedOrigins
		}
		if cliOverrides.TLSEnabled != "" {
			cfg.HTTPTLSEnabled = ParseBool(cliOverrides.TLSEnabled, false)
		}
		if cliOverrides.TLSCertFile != "" {
			cfg.HTTPTLSCertFile = cliOverrides.TLSCertFile
		}
		if cliOverrides.TLSKeyFile != "" {
			cfg.HTTPTLSKeyFile = cliOverrides.TLSKeyFile
		}
	}

	// Set default HTTP port based on TLS configuration if not explicitly provided
	// Default to 443 for HTTPS, 80 for HTTP
	if cfg.HTTPPort == "" {
		if cfg.HTTPTLSEnabled {
			cfg.HTTPPort = "443"
		} else {
			cfg.HTTPPort = "80"
		}
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// GetEnv returns the value of an environment variable or empty string if not set
func GetEnv(key string) string {
	return os.Getenv(key)
}

// GetEnvWithDefault returns the value of an environment variable or a default value
func GetEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// GetTransportModeWithDefault returns the value of an environment variable or a default value
func GetTransportModeWithDefault(key, defaultValue TransportMode) TransportMode {
	if value := os.Getenv(string(key)); value != "" {
		return TransportMode(value)
	}
	return defaultValue
}

// ParseBool parses a string to bool using strconv.ParseBool.
// Returns the default value if the string is empty or invalid.
// Logs a warning if the value is non-empty but invalid.
// Accepts: "1", "t", "T", "true", "True", "TRUE" for true
//
//	"0", "f", "F", "false", "False", "FALSE" for false
func ParseBool(value string, defaultValue bool) bool {
	if value == "" {
		return defaultValue
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		log.Printf("Warning: Invalid boolean value %q, using default: %v", value, defaultValue)
		return defaultValue
	}
	return parsed
}

// ParseInt32 parses a string to int32.
// Returns the default value if the string is empty or invalid.
func ParseInt32(value string, defaultValue int32) int32 {
	if value == "" {
		return defaultValue
	}
	parsed, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		log.Printf("Warning: Invalid integer value %q, using default: %v", value, defaultValue)
		return defaultValue
	}
	return int32(parsed)
}
