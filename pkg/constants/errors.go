// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package constants defines error-related constants and error formatting.
package constants

import "errors"

// ErrMsg* string constants are kept only when the string is used at runtime in
// more than one role — e.g. as a log message, as a Sprintf prefix, or as both
// a string and an errors.New argument. Constants whose only consumer is the
// paired errors.New call below have been inlined there.

// Error message strings used as log/format prefixes or passed to fmt.Errorf
// alongside a dynamic value.
const (
	// Authentication and authorization
	ErrMsgJWKSEndpointNotAccessible = "JWKS endpoint not accessible"

	// API validation (combined with the supplied version value at call site)
	ErrMsgUnsupportedAPIVersion = "unsupported API version"
	ErrMsgServiceDepsUnhealthy  = "service dependencies unhealthy"

	// NATS diagnostics — used in OTel span status, log lines, and fmt.Errorf
	ErrMsgNATSConnNotResponsive = "NATS connection not responsive"
	ErrMsgNATSRequestFailed     = "NATS request failed"
	ErrMsgNATSMaxReconnects     = "NATS max-reconnects exhausted; connection closed"
	ErrMsgNATSConnUnhealthy     = "NATS connection unhealthy"

	// Repository initialization — used both as plain strings in health-check
	// messages and (ErrMsgMessagingRepoNotInit) to create ErrMessagingRepoNotInit.
	ErrMsgMessagingRepoNotInit = "messaging repository not initialized"
	ErrMsgAuthRepoNotInit      = "auth repository not initialized"
)

// Pre-defined error variables for common errors.
// Strings are inlined directly so each error var is self-contained and the
// name alone is the authoritative description.
var (
	ErrInvalidAuthContext  = errors.New("invalid authentication context")
	ErrPrincipalRequired   = errors.New("principal is required")
	ErrJWTValidatorNotInit = errors.New("JWT validator not initialized")
	ErrUnexpectedResponse  = errors.New("unexpected response from access check service")
	ErrInvalidToken        = errors.New("invalid or expired token")
	ErrAccessCheckFailed   = errors.New("access check failed")
	ErrReadingTuplesFailed = errors.New("reading tuples failed")

	// NATS connection state errors — returned by the messaging repository health check.
	ErrNATSConnNotInit  = errors.New("NATS connection not initialized")
	ErrNATSConnNotActive = errors.New("NATS connection is not active")
	ErrNATSConnClosed   = errors.New("NATS connection is closed")
	ErrNATSConnDraining = errors.New("NATS connection is draining")

	// ErrMessagingRepoNotInit uses the ErrMsg constant so the health-check
	// plaintext and the sentinel error share a single source of truth.
	ErrMessagingRepoNotInit = errors.New(ErrMsgMessagingRepoNotInit)
)
