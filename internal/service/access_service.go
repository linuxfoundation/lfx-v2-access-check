// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package service provides the core business logic services for access control.
package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	accesssvc "github.com/linuxfoundation/lfx-v2-access-check/gen/access_svc"
	"github.com/linuxfoundation/lfx-v2-access-check/internal/domain/contracts"
	"github.com/linuxfoundation/lfx-v2-access-check/pkg/constants"
	"goa.design/goa/v3/security"
)

// AccessService is a thin Goa adapter: it validates JWT tokens, delegates all
// NATS protocol work to AccessCheckClient, and maps domain errors to Goa HTTP
// error types. It owns no message-encoding logic.
type AccessService struct {
	authRepo contracts.AuthRepository
	client   *AccessCheckClient
}

// NewAccessService creates a new AccessService wired to the given repositories.
func NewAccessService(authRepo contracts.AuthRepository, messagingRepo contracts.MessagingRepository) *AccessService {
	return &AccessService{
		authRepo: authRepo,
		client:   NewAccessCheckClient(messagingRepo),
	}
}

// Verify interface compliance at compile time.
var (
	_ accesssvc.Service = (*AccessService)(nil)
	_ accesssvc.Auther  = (*AccessService)(nil)
)

// ===== Package-level helpers =====

// claimsFromContext extracts HeimdallClaims injected by JWTAuth.
func claimsFromContext(ctx context.Context) (*contracts.HeimdallClaims, bool) {
	claims, ok := ctx.Value(constants.ClaimsContextKey).(*contracts.HeimdallClaims)
	return claims, ok
}

// requireAPIVersion returns a non-nil error when version does not match the
// single supported API version. Callers wrap the result in accesssvc.MakeBadRequest.
func requireAPIVersion(version string) error {
	if version != constants.SupportedAPIVersion {
		return fmt.Errorf("%s: %s", constants.ErrMsgUnsupportedAPIVersion, version)
	}
	return nil
}

// ===== GOA Authentication Interface =====

// JWTAuth implements the authorization logic for the JWT security scheme.
func (s *AccessService) JWTAuth(ctx context.Context, token string, _ *security.JWTScheme) (context.Context, error) {
	if after, ok := strings.CutPrefix(token, constants.BearerTokenPrefix); ok {
		token = after
	}

	claims, err := s.authRepo.ValidateToken(ctx, token)
	if err != nil {
		slog.ErrorContext(ctx, "JWT validation failed", "error", err)
		if errors.Is(err, constants.ErrUnexpectedResponse) {
			return nil, accesssvc.MakeInternalServerError(constants.ErrUnexpectedResponse)
		}
		return nil, accesssvc.MakeUnauthorized(constants.ErrInvalidToken)
	}

	ctx = context.WithValue(ctx, constants.ClaimsContextKey, claims)
	slog.DebugContext(ctx, "JWT validation successful", "principal", claims.Principal)
	return ctx, nil
}

// ===== GOA Service Interface =====

// CheckAccess validates the request and delegates to AccessCheckClient.
func (s *AccessService) CheckAccess(ctx context.Context, p *accesssvc.CheckAccessPayload) (*accesssvc.CheckAccessResult, error) {
	claims, ok := claimsFromContext(ctx)
	if !ok {
		slog.ErrorContext(ctx, "Failed to get claims from context")
		return nil, accesssvc.MakeUnauthorized(constants.ErrInvalidAuthContext)
	}

	if err := requireAPIVersion(p.Version); err != nil {
		slog.WarnContext(ctx, "Unsupported API version", "version", p.Version)
		return nil, accesssvc.MakeBadRequest(err)
	}

	if len(p.Requests) == 0 {
		slog.WarnContext(ctx, "Empty requests array")
		return &accesssvc.CheckAccessResult{Results: []string{}}, nil
	}

	results, err := s.client.CheckAccess(ctx, claims.Principal, p.Requests)
	if err != nil {
		slog.ErrorContext(ctx, "Access check failed", "error", err, "principal", claims.Principal)
		switch {
		case errors.Is(err, constants.ErrPrincipalRequired):
			return nil, accesssvc.MakeUnauthorized(err)
		case errors.Is(err, constants.ErrUnexpectedResponse):
			return nil, accesssvc.MakeInternalServerError(constants.ErrUnexpectedResponse)
		default:
			return nil, accesssvc.MakeServiceUnavailable(constants.ErrAccessCheckFailed)
		}
	}

	slog.InfoContext(ctx, "Access check completed", "principal", claims.Principal, "requests_count", len(p.Requests))
	return &accesssvc.CheckAccessResult{Results: results}, nil
}

// MyGrants validates the request and delegates to AccessCheckClient.
func (s *AccessService) MyGrants(ctx context.Context, p *accesssvc.MyGrantsPayload) (*accesssvc.MyGrantsResult, error) {
	claims, ok := claimsFromContext(ctx)
	if !ok {
		slog.ErrorContext(ctx, "Failed to get claims from context")
		return nil, accesssvc.MakeUnauthorized(constants.ErrInvalidAuthContext)
	}

	if err := requireAPIVersion(p.Version); err != nil {
		slog.WarnContext(ctx, "Unsupported API version", "version", p.Version)
		return nil, accesssvc.MakeBadRequest(err)
	}

	if claims.Principal == "" {
		slog.ErrorContext(ctx, "Principal is required for my-grants")
		return nil, accesssvc.MakeUnauthorized(constants.ErrPrincipalRequired)
	}

	grants, err := s.client.ReadTuples(ctx, claims.Principal, p.ObjectType)
	if err != nil {
		slog.ErrorContext(ctx, "Reading tuples failed", "error", err, "principal", claims.Principal, "subject", constants.ReadTuplesSubject, "object_type", p.ObjectType)
		if errors.Is(err, constants.ErrUnexpectedResponse) {
			return nil, accesssvc.MakeInternalServerError(constants.ErrUnexpectedResponse)
		}
		return nil, accesssvc.MakeServiceUnavailable(constants.ErrReadingTuplesFailed)
	}

	slog.InfoContext(ctx, "My grants completed", "principal", claims.Principal, "object_type", p.ObjectType, "grants_count", len(grants))
	return &accesssvc.MyGrantsResult{Grants: grants}, nil
}

// Readyz checks that both messaging and auth dependencies are healthy.
func (s *AccessService) Readyz(ctx context.Context) ([]byte, error) {
	var healthIssues []string

	if err := s.client.HealthCheck(ctx); err != nil {
		if errors.Is(err, constants.ErrMessagingRepoNotInit) {
			healthIssues = append(healthIssues, constants.ErrMsgMessagingRepoNotInit)
		} else {
			healthIssues = append(healthIssues, fmt.Sprintf("%s: %v", constants.ErrMsgNATSConnUnhealthy, err))
		}
	}

	if s.authRepo == nil {
		healthIssues = append(healthIssues, constants.ErrMsgAuthRepoNotInit)
	} else {
		if err := s.authRepo.HealthCheck(ctx); err != nil {
			healthIssues = append(healthIssues, fmt.Sprintf("auth service unhealthy: %v", err))
		}
	}

	if len(healthIssues) > 0 {
		slog.ErrorContext(ctx, "Readiness check failed", "issues", healthIssues)
		return nil, accesssvc.MakeNotReady(fmt.Errorf("%s: %v", constants.ErrMsgServiceDepsUnhealthy, healthIssues))
	}

	slog.DebugContext(ctx, "Readiness check passed - all dependencies healthy")
	return []byte(constants.HealthOKResponse), nil
}

// Livez always succeeds; as long as the process is running the service is live.
func (s *AccessService) Livez(ctx context.Context) ([]byte, error) {
	slog.DebugContext(ctx, "Liveness check requested")
	return []byte(constants.HealthOKResponse), nil
}
