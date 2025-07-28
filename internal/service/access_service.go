// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package service provides the core business logic services for access control.
package service

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"strings"

	accesssvc "github.com/linuxfoundation/lfx-v2-access-check/gen/access_svc"
	"github.com/linuxfoundation/lfx-v2-access-check/internal/domain/contracts"
	"github.com/linuxfoundation/lfx-v2-access-check/pkg/constants"
	"goa.design/goa/v3/security"
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
	if after, ok := strings.CutPrefix(token, constants.BearerTokenPrefix); ok {
		token = after
	}

	// Validate token using auth repository
	claims, err := s.authRepo.ValidateToken(ctx, token)
	if err != nil {
		slog.ErrorContext(ctx, "JWT validation failed", "error", err)
		return nil, accesssvc.MakeUnauthorized(err)
	}

	// Add claims to context for use in endpoints
	ctx = context.WithValue(ctx, constants.ClaimsContextKey, claims)

	slog.DebugContext(ctx, "JWT validation successful", "principal", claims.Principal)
	return ctx, nil
}

// ===== GOA Service Interface Implementation =====

// CheckAccess implements the access check endpoint with embedded business logic
func (s *AccessService) CheckAccess(ctx context.Context, p *accesssvc.CheckAccessPayload) (*accesssvc.CheckAccessResult, error) {
	// Extract claims from context
	claims, ok := ctx.Value(constants.ClaimsContextKey).(*contracts.HeimdallClaims)
	if !ok {
		slog.ErrorContext(ctx, "Failed to get claims from context")
		return nil, accesssvc.MakeUnauthorized(constants.ErrInvalidAuthContext)
	}

	// Validate payload
	if p.Version != constants.SupportedAPIVersion {
		slog.WarnContext(ctx, "Unsupported API version", "version", p.Version)
		return nil, accesssvc.MakeBadRequest(fmt.Errorf("%s: %s", constants.ErrMsgUnsupportedAPIVersion, p.Version))
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
		healthIssues = append(healthIssues, constants.ErrMsgMessagingRepoNotInit)
	} else {
		if err := s.messagingRepo.HealthCheck(ctx); err != nil {
			healthIssues = append(healthIssues, fmt.Sprintf("%s: %v", constants.ErrMsgNATSConnUnhealthy, err))
		}
	}

	// Check if auth repository is available and healthy
	if s.authRepo == nil {
		healthIssues = append(healthIssues, constants.ErrMsgAuthRepoNotInit)
	} else {
		if err := s.authRepo.HealthCheck(ctx); err != nil {
			healthIssues = append(healthIssues, fmt.Sprintf("auth service unhealthy: %v", err))
		}
	}

	// If any health checks failed, return not ready
	if len(healthIssues) > 0 {
		slog.ErrorContext(ctx, "Readiness check failed", "issues", healthIssues)
		return nil, accesssvc.MakeNotReady(fmt.Errorf("%s: %v", constants.ErrMsgServiceDepsUnhealthy, healthIssues))
	}

	slog.DebugContext(ctx, "Readiness check passed - all dependencies healthy")
	return []byte(constants.HealthOKResponse), nil
}

// Livez implements the liveness check endpoint
func (s *AccessService) Livez(ctx context.Context) ([]byte, error) {
	// Liveness check - as long as the service is running, it's alive
	slog.DebugContext(ctx, "Liveness check requested")
	return []byte(constants.HealthOKResponse), nil
}

// ===== PRIVATE BUSINESS LOGIC METHODS =====

// performAccessCheck contains the core business logic for access checking
func (s *AccessService) performAccessCheck(ctx context.Context, principal string, resources []string) ([]string, error) {
	if principal == "" {
		slog.ErrorContext(ctx, "Principal is required for access check")
		return nil, constants.ErrPrincipalRequired
	}

	if len(resources) == 0 {
		return []string{}, nil
	}

	// Build access check message
	message := s.buildAccessCheckMessage(principal, resources)
	if message == "" {
		return []string{}, nil
	}

	// Make NATS request
	responseData, err := s.messagingRepo.Request(ctx, constants.AccessCheckSubject, []byte(message), constants.DefaultNATSTimeout)
	if err != nil {
		slog.ErrorContext(ctx, "NATS request failed", "error", err, "subject", constants.AccessCheckSubject)
		return nil, fmt.Errorf("%s: %w", constants.ErrMsgNATSRequestFailed, err)
	}

	// Parse and validate response
	return s.parseAccessCheckResponse(ctx, responseData)
}

// buildAccessCheckMessage creates the NATS message for access checking using efficient string building
// add newlines after each item, then remove trailing one
func (s *AccessService) buildAccessCheckMessage(principal string, resources []string) string {
	var builder strings.Builder

	// Calculate actual capacity based on real data instead of estimates
	totalCapacity := 0
	for _, resource := range resources {
		if resource != "" {
			// resource + "@user:" + principal + newline
			totalCapacity += len(resource) + len(constants.UserRelationPrefix) + len(principal) + 1
		}
	}

	if totalCapacity > 0 {
		builder.Grow(totalCapacity)
	}

	for _, resource := range resources {
		if resource == "" {
			continue
		}

		// Build relation: resource@user:principal
		builder.WriteString(resource)
		builder.WriteString(constants.UserRelationPrefix)
		builder.WriteString(principal)
		builder.WriteByte('\n')
	}

	message := builder.String()

	// Remove trailing newline if present
	if len(message) > 0 && message[len(message)-1] == '\n' {
		message = message[:len(message)-1]
	}

	return message
} // parseAccessCheckResponse parses and validates the NATS response
func (s *AccessService) parseAccessCheckResponse(ctx context.Context, responseData []byte) ([]string, error) {
	// Sanity check response - if there's a space in the first N bytes, assume it's an error
	topRange := constants.DefaultResponseSanityCheckBytes
	if len(responseData) < topRange {
		topRange = len(responseData)
	}
	if bytes.Contains(responseData[:topRange], []byte(" ")) {
		slog.ErrorContext(ctx, "Unexpected response from access check service", "response_preview", string(responseData[:topRange]))
		return nil, constants.ErrUnexpectedResponse
	}

	// Parse response - split by newlines to get individual results
	lines := bytes.Split(responseData, []byte("\n"))
	results := make([]string, 0, len(lines))

	for _, line := range lines {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		results = append(results, string(line))
	}

	return results, nil
}
