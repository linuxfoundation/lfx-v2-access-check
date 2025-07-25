// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package constants defines configuration-related constants.
package constants

// Default configuration values
const (
	// Server defaults
	DefaultHost     = "localhost"
	DefaultHTTPPort = "8080"

	// Authentication defaults
	DefaultJWKSURL  = "http://heimdall:4457/.well-known/jwks"
	DefaultAudience = "access-check"
	DefaultIssuer   = "heimdall"

	// Messaging defaults
	DefaultNATSURL = "nats://nats:4222"
)
