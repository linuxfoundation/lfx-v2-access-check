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
	Host     string
	Port     string
	Debug    bool
	BindAddr string

	// JWT configuration
	JWKSUrl  string
	Audience string
	Issuer   string

	// NATS configuration
	NATSUrl string
}

// LoadConfig loads configuration from CLI flags, environment variables, and defaults
// Priority: CLI flags > Environment variables > Defaults
func LoadConfig() *Config {
	slog.Info("Loading application configuration")

	// Define CLI flags
	var (
		hostF     = flag.String("host", constants.DefaultHost, "Server host")
		httpPortF = flag.String("http-port", constants.DefaultHTTPPort, "HTTP port")
		dbgF      = flag.Bool("debug", false, "Enable debug logging")
	)
	flag.Parse()

	config := &Config{
		// Start with CLI flag values (which include defaults)
		Host:     *hostF,
		Port:     *httpPortF,
		Debug:    *dbgF,
		BindAddr: "*", // Default bind to all interfaces

		// Default values for other fields
		JWKSUrl:  constants.DefaultJWKSURL,
		Audience: constants.DefaultAudience,
		Issuer:   constants.DefaultIssuer,
		NATSUrl:  constants.DefaultNATSURL,
	}

	// Override with environment variables if set
	if port := os.Getenv(constants.EnvPort); port != "" {
		config.Port = port
		slog.Info("Using PORT environment variable", "port", port)
	}

	if host := os.Getenv(constants.EnvHost); host != "" {
		config.Host = host
	}

	if debug := os.Getenv(constants.EnvDebug); debug != "" {
		if parseBool(debug) {
			config.Debug = true
			slog.Info("Debug mode enabled via DEBUG environment variable")
		} else {
			config.Debug = false
		}
	}

	if bindAddr := os.Getenv(constants.EnvBindAddr); bindAddr != "" {
		config.BindAddr = bindAddr
		slog.Info("Using BIND_ADDR environment variable", "bind_addr", bindAddr)
	}

	if jwksURL := os.Getenv(constants.EnvJWKSURL); jwksURL != "" {
		config.JWKSUrl = jwksURL
	}

	if audience := os.Getenv(constants.EnvAudience); audience != "" {
		config.Audience = audience
	}

	if issuer := os.Getenv(constants.EnvIssuer); issuer != "" {
		config.Issuer = issuer
	}

	if natsURL := os.Getenv(constants.EnvNATSURL); natsURL != "" {
		config.NATSUrl = natsURL
	}

	return config
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
