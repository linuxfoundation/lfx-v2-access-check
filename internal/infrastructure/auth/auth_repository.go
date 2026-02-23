// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package auth provides JWT authentication infrastructure for the access service.
package auth

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	jwtmiddleware "github.com/auth0/go-jwt-middleware/v2"
	"github.com/auth0/go-jwt-middleware/v2/jwks"
	"github.com/auth0/go-jwt-middleware/v2/validator"
	"github.com/linuxfoundation/lfx-v2-access-check/internal/domain/contracts"
	"github.com/linuxfoundation/lfx-v2-access-check/pkg/constants"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

const (
	signatureAlgorithm = validator.PS256
)

type authRepository struct {
	validator *validator.Validator
}

// NewAuthRepository creates a new JWT-based authentication repository
func NewAuthRepository(jwksURL, issuer, audience string) (contracts.AuthRepository, error) {
	slog.Info("Initializing auth repository", "jwks_url", jwksURL, "issuer", issuer)

	// Parse URLs
	jwksU, err := url.Parse(jwksURL)
	if err != nil {
		slog.Error("Failed to parse JWKS URL", "error", err, "jwks_url", jwksURL)
		return nil, err
	}

	issuerU, err := url.Parse(issuer)
	if err != nil {
		slog.Error("Failed to parse issuer URL", "error", err, "issuer", issuer)
		return nil, err
	}

	// Set up JWKS provider with OTel-instrumented HTTP client
	httpClient := &http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}
	provider := jwks.NewCachingProvider(
		issuerU,
		constants.DefaultJWKSCacheTimeout,
		jwks.WithCustomJWKSURI(jwksU),
		jwks.WithCustomClient(httpClient),
	)

	// Factory for custom JWT claims
	customClaims := func() validator.CustomClaims {
		return &contracts.HeimdallClaims{}
	}

	// Create JWT validator
	jwtValidator, err := validator.New(
		provider.KeyFunc,
		signatureAlgorithm,
		issuerU.String(),
		[]string{audience},
		validator.WithCustomClaims(customClaims),
		validator.WithAllowedClockSkew(constants.JWTClockSkew),
	)
	if err != nil {
		slog.Error("Failed to create JWT validator", "error", err, "issuer", issuer, "audience", audience)
		return nil, err
	}

	return &authRepository{
		validator: jwtValidator,
	}, nil
}

// ValidateToken validates the provided token and returns the associated claims
func (r *authRepository) ValidateToken(ctx context.Context, token string) (*contracts.HeimdallClaims, error) {
	// Validate the token
	claims, err := r.validator.ValidateToken(ctx, token)
	if err != nil {
		slog.ErrorContext(ctx, "JWT token validation failed", "error", err)
		return nil, err
	}

	// Extract custom claims
	validatedClaims, ok := claims.(*validator.ValidatedClaims)
	if !ok {
		slog.ErrorContext(ctx, "Failed to cast to ValidatedClaims")
		return nil, jwtmiddleware.ErrJWTInvalid
	}

	customClaims, ok := validatedClaims.CustomClaims.(*contracts.HeimdallClaims)
	if !ok {
		slog.ErrorContext(ctx, "Failed to cast to HeimdallClaims")
		return nil, jwtmiddleware.ErrJWTInvalid
	}

	return customClaims, nil
}

// HealthCheck verifies the auth service is accessible and JWKS can be fetched
func (r *authRepository) HealthCheck(ctx context.Context) error {
	if r.validator == nil {
		return constants.ErrJWTValidatorNotInit
	}

	// Try to validate a minimal token structure to test JWKS connectivity
	// This won't succeed as a real validation, but will test the JWKS endpoint
	// If JWKS endpoint is down, this will fail appropriately
	testToken := "eyJ0eXAiOiJKV1QiLCJhbGciOiJQUzI1NiJ9.eyJpc3MiOiJ0ZXN0IiwiYXVkIjoidGVzdCIsImV4cCI6OTk5OTk5OTk5OX0.test"

	// The validation will fail (expected) but if JWKS is unreachable,
	// we'll get a network error instead of a validation error
	_, err := r.validator.ValidateToken(ctx, testToken)

	// Check if error indicates JWKS connectivity issues
	if err != nil {
		errStr := err.Error()
		// Network/connectivity related errors indicate health issues
		if strings.Contains(errStr, "connection refused") ||
			strings.Contains(errStr, "timeout") ||
			strings.Contains(errStr, "no such host") ||
			strings.Contains(errStr, "network is unreachable") {
			return fmt.Errorf("%s: %w", constants.ErrMsgJWKSEndpointNotAccessible, err)
		}
		// Other validation errors are expected for test token - JWKS is healthy
	}

	return nil
}
