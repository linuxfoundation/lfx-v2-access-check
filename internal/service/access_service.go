// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package service provides the core business logic services for access control.
package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	accesssvc "github.com/linuxfoundation/lfx-v2-access-check/gen/access_svc"
	"github.com/linuxfoundation/lfx-v2-access-check/internal/domain/contracts"
	"goa.design/goa/v3/security"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const (
	// claimsKey is the context key for storing JWT claims
	claimsKey contextKey = "claims"

	accessCheckSubject = "dev.lfx.access_check.request"
	natsTimeout        = 15 * time.Second
)

// AccessService implements both domain logic and GOA service interfaces
// This unified approach simplifies the architecture while maintaining clean separation
// of infrastructure concerns through dependency injection
type AccessService struct {
	authRepo      contracts.AuthRepository
	messagingRepo contracts.MessagingRepository
}

// NewAccessService creates a new unified access service
func NewAccessService(authRepo contracts.AuthRepository, messagingRepo contracts.MessagingRepository) *AccessService {
	return &AccessService{
		authRepo:      authRepo,
		messagingRepo: messagingRepo,
	}
}

// Verify interface compliance at compile time
var (
	_ accesssvc.Service = (*AccessService)(nil)
	_ accesssvc.Auther  = (*AccessService)(nil)
)

// ===== GOA Authentication Interface =====

// JWTAuth implements the authorization logic for the JWT security scheme
func (s *AccessService) JWTAuth(ctx context.Context, token string, _ *security.JWTScheme) (context.Context, error) {
	// Remove "Bearer " prefix if present
	if after, ok := strings.CutPrefix(token, "Bearer "); ok {
		token = after
	}

	// Validate token using auth repository
	claims, err := s.authRepo.ValidateToken(ctx, token)
	if err != nil {
		slog.ErrorContext(ctx, "JWT validation failed", "error", err)
		return nil, accesssvc.MakeUnauthorized(err)
	}

	// Add claims to context for use in endpoints
	ctx = context.WithValue(ctx, claimsKey, claims)

	slog.DebugContext(ctx, "JWT validation successful", "principal", claims.Principal)
	return ctx, nil
}

// ===== GOA Service Interface Implementation =====

// CheckAccess implements the access check endpoint with embedded business logic
func (s *AccessService) CheckAccess(ctx context.Context, p *accesssvc.CheckAccessPayload) (*accesssvc.CheckAccessResult, error) {
	// Extract claims from context
	claims, ok := ctx.Value(claimsKey).(*contracts.HeimdallClaims)
	if !ok {
		slog.ErrorContext(ctx, "Failed to get claims from context")
		return nil, accesssvc.MakeUnauthorized(fmt.Errorf("invalid authentication context"))
	}

	// Validate payload
	if p.Version != "1" {
		slog.WarnContext(ctx, "Unsupported API version", "version", p.Version)
		return nil, accesssvc.MakeBadRequest(fmt.Errorf("unsupported API version: %s", p.Version))
	}

	if len(p.Requests) == 0 {
		slog.WarnContext(ctx, "Empty requests array")
		return &accesssvc.CheckAccessResult{Results: []string{}}, nil
	}

	// === BUSINESS LOGIC (embedded in service) ===
	results, err := s.performAccessCheck(ctx, claims.Principal, p.Requests)
	if err != nil {
		slog.ErrorContext(ctx, "Access check failed", "error", err, "principal", claims.Principal)
		return nil, accesssvc.MakeBadRequest(err)
	}

	slog.InfoContext(ctx, "Access check completed", "principal", claims.Principal, "requests_count", len(p.Requests))

	return &accesssvc.CheckAccessResult{Results: results}, nil
}

// Readyz implements the readiness check endpoint with comprehensive health checks
func (s *AccessService) Readyz(ctx context.Context) ([]byte, error) {
	var healthIssues []string

	// Check if messaging repository is available and healthy
	if s.messagingRepo == nil {
		healthIssues = append(healthIssues, "messaging repository not initialized")
	} else {
		if err := s.messagingRepo.HealthCheck(ctx); err != nil {
			healthIssues = append(healthIssues, fmt.Sprintf("NATS connection unhealthy: %v", err))
		}
	}

	// Check if auth repository is available and healthy
	if s.authRepo == nil {
		healthIssues = append(healthIssues, "auth repository not initialized")
	} else {
		if err := s.authRepo.HealthCheck(ctx); err != nil {
			healthIssues = append(healthIssues, fmt.Sprintf("auth service unhealthy: %v", err))
		}
	}

	// If any health checks failed, return not ready
	if len(healthIssues) > 0 {
		slog.ErrorContext(ctx, "Readiness check failed", "issues", healthIssues)
		return nil, accesssvc.MakeNotReady(fmt.Errorf("service dependencies unhealthy: %v", healthIssues))
	}

	slog.DebugContext(ctx, "Readiness check passed - all dependencies healthy")
	return []byte("OK"), nil
}

// Livez implements the liveness check endpoint
func (s *AccessService) Livez(ctx context.Context) ([]byte, error) {
	// Liveness check - as long as the service is running, it's alive
	slog.DebugContext(ctx, "Liveness check requested")
	return []byte("OK"), nil
}

// ===== PRIVATE BUSINESS LOGIC METHODS =====

// performAccessCheck contains the core business logic for access checking
func (s *AccessService) performAccessCheck(ctx context.Context, principal string, resources []string) ([]string, error) {
	if principal == "" {
		slog.ErrorContext(ctx, "Principal is required for access check")
		return nil, errors.New("principal is required")
	}

	if len(resources) == 0 {
		return []string{}, nil
	}

	// Build access check message in the format expected by the backend
	accessCheckMessage := make([]byte, 0, 80*len(resources))

	for _, resource := range resources {
		if len(resource) == 0 {
			continue
		}

		// Build relation: resource@user:principal
		relation := make([]byte, 0, len(resource)+len(principal)+7)
		relation = append(relation, []byte(resource)...)
		relation = append(relation, []byte("@user:")...)
		relation = append(relation, []byte(principal)...)

		accessCheckMessage = append(accessCheckMessage, relation...)
		accessCheckMessage = append(accessCheckMessage, '\n')
	}

	// If no valid resources, return empty results
	if len(accessCheckMessage) == 0 {
		return []string{}, nil
	}

	// Trim trailing newline
	accessCheckMessage = accessCheckMessage[:len(accessCheckMessage)-1]

	// Make NATS request
	responseData, err := s.messagingRepo.Request(ctx, accessCheckSubject, accessCheckMessage, natsTimeout)
	if err != nil {
		slog.ErrorContext(ctx, "NATS request failed", "error", err, "subject", accessCheckSubject)
		return nil, fmt.Errorf("NATS request failed: %w", err)
	}

	// Sanity check response - if there's a space in the first 20 bytes, assume it's an error
	topRange := 20
	if len(responseData) < topRange {
		topRange = len(responseData)
	}
	if bytes.Contains(responseData[:topRange], []byte(" ")) {
		slog.ErrorContext(ctx, "Unexpected response from access check service", "response_preview", string(responseData[:topRange]))
		return nil, errors.New("unexpected response from access check service")
	}

	// Parse response - for now, we return the raw response as a single result
	// This can be enhanced to parse multiple results if needed
	results := []string{string(responseData)}

	return results, nil
}
