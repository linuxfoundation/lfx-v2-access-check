// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package config provides application configuration management.
package config

import (
	"flag"
	"log/slog"
	"os"

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
		Host:  *hostF,
		Port:  *httpPortF,
		Debug: *dbgF,

		// Default values for other fields
		JWKSUrl:  constants.DefaultJWKSURL,
		Audience: constants.DefaultAudience,
		Issuer:   constants.DefaultIssuer,
		NATSUrl:  constants.DefaultNATSURL,
	}

	// Override with environment variables if set
	if port := os.Getenv("PORT"); port != "" {
		config.Port = port
	}

	if host := os.Getenv("HOST"); host != "" {
		config.Host = host
	}

	if debug := os.Getenv("DEBUG"); debug != "" {
		config.Debug = true
	}

	if jwksURL := os.Getenv("JWKS_URL"); jwksURL != "" {
		config.JWKSUrl = jwksURL
	}

	if audience := os.Getenv("AUDIENCE"); audience != "" {
		config.Audience = audience
	}

	if issuer := os.Getenv("ISSUER"); issuer != "" {
		config.Issuer = issuer
	}

	if natsURL := os.Getenv("NATS_URL"); natsURL != "" {
		config.NATSUrl = natsURL
	}

	return config
}
