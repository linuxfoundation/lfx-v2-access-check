// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package config provides application configuration management.
package config

import (
	"flag"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/linuxfoundation/lfx-v2-access-check/pkg/constants"
)

// Config holds application configuration
type Config struct {
	// Server configuration
	Host  string
	Port  string
	Debug bool

	// JWT configuration
	JWKSUrl  string
	Audience string
	Issuer   string

	// NATS configuration
	NATSUrl string
}

// LoadConfig loads configuration from CLI flags, environment variables, and defaults
// Priority: CLI flags > Environment variables > Defaults (following POC pattern)
func LoadConfig() *Config {
	slog.Info("Loading application configuration")

	// Check environment first, then use constants
	defaultPort := os.Getenv(constants.EnvPort)
	if defaultPort == "" {
		defaultPort = constants.DefaultHTTPPort
	}

	defaultHost := os.Getenv(constants.EnvHost)
	if defaultHost == "" {
		defaultHost = "*"
	}

	// Define CLI flags
	var (
		portF = flag.String("p", defaultPort, "listen port")             // Short flag "p"
		hostF = flag.String("bind", defaultHost, "interface to bind on") // Short flag "bind"
		dbgF  = flag.Bool("d", false, "enable debug logging")            // Short flag "d"
	)
	flag.Parse()

	config := &Config{
		// Start with CLI flag values (which include defaults)
		Port:  *portF,
		Debug: *dbgF,
		Host:  *hostF,

		// Load other configuration from environment with defaults
		JWKSUrl:  getEnvOrDefault(constants.EnvJWKSURL, constants.DefaultJWKSURL),
		Audience: getEnvOrDefault(constants.EnvAudience, constants.DefaultAudience),
		Issuer:   getEnvOrDefault(constants.EnvIssuer, constants.DefaultIssuer),
		NATSUrl:  getEnvOrDefault(constants.EnvNATSURL, constants.DefaultNATSURL),
	}

	// Handle debug flag from environment (POC checks both DEBUG env and -d flag)
	if debugEnv := os.Getenv(constants.EnvDebug); debugEnv != "" && parseBool(debugEnv) {
		config.Debug = true
		slog.Info("Debug mode enabled via DEBUG environment variable")
	}

	return config
}

// ServerAddress constructs the server address combining POC and traditional logic
// POC logic: If Host is "*", bind to all interfaces (":" + port)
// Traditional logic: Otherwise use Host:Port
// This handles both patterns in a single unified method
func (c *Config) ServerAddress() string {
	// Use POC binding logic when Host is "*" (wildcard)
	if c.Host == "*" {
		return ":" + c.Port // Bind to all interfaces (POC pattern)
	}
	// Use traditional Host:Port for specific interfaces
	return c.Host + ":" + c.Port
}

// HostPortAddress provides traditional Host:Port address (kept for backward compatibility)
// This always returns Host:Port regardless of wildcard logic
func (c *Config) HostPortAddress() string {
	return c.Host + ":" + c.Port
}

// getEnvOrDefault returns environment variable value or default if not set
func getEnvOrDefault(envKey, defaultValue string) string {
	if value := os.Getenv(envKey); value != "" {
		return value
	}
	return defaultValue
}

// parseBool parses a string value to boolean with support for common boolean representations
// Returns true for: "true", "1", "yes", "on", "y", "t" (case-insensitive)
// Returns false for: "false", "0", "no", "off", "n", "f" (case-insensitive)
// Returns false for any other value
func parseBool(s string) bool {
	// Normalize the input: trim whitespace and convert to lowercase
	normalized := strings.ToLower(strings.TrimSpace(s))

	// Use Go's built-in strconv.ParseBool first as it handles standard cases
	if val, err := strconv.ParseBool(normalized); err == nil {
		return val
	}

	// Handle additional common boolean representations
	switch normalized {
	case "yes", "y", "on":
		return true
	case "no", "n", "off":
		return false
	default:
		return false
	}
}
