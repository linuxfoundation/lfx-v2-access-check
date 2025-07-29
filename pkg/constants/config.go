// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package constants defines configuration-related constants.
package constants

// Configuration constants
const (
	// Server environment variables
	EnvPort  = "PORT"
	EnvHost  = "HOST"
	EnvDebug = "DEBUG"

	// Authentication environment variables
	EnvJWKSURL  = "JWKS_URL"
	EnvAudience = "AUDIENCE"
	EnvIssuer   = "ISSUER"

	// Messaging environment variables
	EnvNATSURL = "NATS_URL"

	// Server defaults
	DefaultHost     = "0.0.0.0"
	DefaultHTTPPort = "8080"

	// Authentication defaults
	DefaultJWKSURL  = "http://heimdall:4457/.well-known/jwks"
	DefaultAudience = "access-check"
	DefaultIssuer   = "heimdall"

	// Messaging defaults
	DefaultNATSURL = "nats://nats:4222"
)
