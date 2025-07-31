// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package constants defines error-related constants and error formatting.
package constants

import "errors"

// Error messages for consistent error handling across the application
const (
	// Authentication and authorization errors
	ErrMsgInvalidAuthContext        = "invalid authentication context"
	ErrMsgJWTValidationFailed       = "JWT validation failed"
	ErrMsgJWTValidatorNotInit       = "JWT validator not initialized"
	ErrMsgPrincipalRequired         = "principal is required"
	ErrMsgJWKSEndpointNotAccessible = "JWKS endpoint not accessible"

	// API and validation errors
	ErrMsgUnsupportedAPIVersion = "unsupported API version"
	ErrMsgServiceDepsUnhealthy  = "service dependencies unhealthy"
	ErrMsgUnexpectedResponse    = "unexpected response from access check service"

	// NATS connection errors
	ErrMsgNATSConnNotInit       = "NATS connection not initialized"
	ErrMsgNATSConnNotActive     = "NATS connection is not active"
	ErrMsgNATSConnClosed        = "NATS connection is closed"
	ErrMsgNATSConnDraining      = "NATS connection is draining"
	ErrMsgNATSConnNotResponsive = "NATS connection not responsive"
	ErrMsgNATSRequestFailed     = "NATS request failed"
	ErrMsgNATSMaxReconnects     = "NATS max-reconnects exhausted; connection closed"
	ErrMsgNATSConnUnhealthy     = "NATS connection unhealthy"

	// Repository initialization errors
	ErrMsgMessagingRepoNotInit = "messaging repository not initialized"
	ErrMsgAuthRepoNotInit      = "auth repository not initialized"
)

// Pre-defined error variables for common errors
var (
	ErrInvalidAuthContext   = errors.New(ErrMsgInvalidAuthContext)
	ErrPrincipalRequired    = errors.New(ErrMsgPrincipalRequired)
	ErrJWTValidatorNotInit  = errors.New(ErrMsgJWTValidatorNotInit)
	ErrUnexpectedResponse   = errors.New(ErrMsgUnexpectedResponse)
	ErrNATSConnNotInit      = errors.New(ErrMsgNATSConnNotInit)
	ErrNATSConnNotActive    = errors.New(ErrMsgNATSConnNotActive)
	ErrNATSConnClosed       = errors.New(ErrMsgNATSConnClosed)
	ErrNATSConnDraining     = errors.New(ErrMsgNATSConnDraining)
	ErrMessagingRepoNotInit = errors.New(ErrMsgMessagingRepoNotInit)
	ErrAuthRepoNotInit      = errors.New(ErrMsgAuthRepoNotInit)
)
