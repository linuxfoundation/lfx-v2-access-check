// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package constants defines service-related constants.
package constants

// ContextKey is a custom type for context keys to avoid collisions
type ContextKey string

// Service constants
const (
	// Context keys for storing values in request context
	ClaimsContextKey ContextKey = "claims"

	// RequestIDContextKey is already defined in http.go as RequestIDHeader
	// but we can add an alias here for clarity in service contexts
	ServiceRequestIDKey ContextKey = RequestIDHeader

	// NATS subjects for messaging
	AccessCheckSubject = "dev.lfx.access_check.request"

	// API version constants
	SupportedAPIVersion = "1"

	// Authentication constants
	BearerTokenPrefix = "Bearer "

	// Health check constants
	HealthOKResponse = "OK"

	// Relation building constants
	UserRelationPrefix = "@user:"
	RelationSeparator  = "@"
)
