// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package constants defines HTTP-related constants.
package constants

import "time"

// HTTP constants
const (
	// RequestIDHeader is the HTTP header name for request ID
	RequestIDHeader = "X-Request-ID"

	// DefaultShutdownTimeout is the default timeout for graceful server shutdown
	DefaultShutdownTimeout = 25 * time.Second

	// HTTP Server timeout constants
	DefaultReadHeaderTimeout = 60 * time.Second
	DefaultWriteTimeout      = 60 * time.Second
	DefaultIdleTimeout       = 90 * time.Second
)
