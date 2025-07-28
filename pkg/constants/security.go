// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package constants defines security-related constants.
package constants

import "time"

// Security and timing constants
const (
	// JWTClockSkew is the allowed time difference for JWT validation
	JWTClockSkew = 5 * time.Second

	// DefaultJWKSCacheTimeout is the default timeout for JWKS caching
	DefaultJWKSCacheTimeout = 5 * time.Minute
)
