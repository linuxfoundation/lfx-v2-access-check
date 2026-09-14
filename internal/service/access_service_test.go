// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT
package service

import (
	"context"
	"errors"
	"fmt"
	"testing"

	accesssvc "github.com/linuxfoundation/lfx-v2-access-check/gen/access_svc"
	"github.com/linuxfoundation/lfx-v2-access-check/internal/domain/contracts"
	"github.com/linuxfoundation/lfx-v2-access-check/pkg/constants"
	goa "goa.design/goa/v3/pkg"
	"goa.design/goa/v3/security"
)

// mockAuthRepository satisfies contracts.AuthRepository.
type mockAuthRepository struct {
	validateTokenFunc func(ctx context.Context, token string) (*contracts.HeimdallClaims, error)
}

func (m *mockAuthRepository) ValidateToken(ctx context.Context, token string) (*contracts.HeimdallClaims, error) {
	if m.validateTokenFunc != nil {
		return m.validateTokenFunc(ctx, token)
	}
	return &contracts.HeimdallClaims{Principal: "test-user", Email: "test@example.com"}, nil
}

func (m *mockAuthRepository) HealthCheck(_ context.Context) error {
	return nil
}

// mockAccessChecker satisfies contracts.AccessChecker.
// It works at domain level: callers supply and receive string slices —
// no NATS byte encoding required.
type mockAccessChecker struct {
	checkAccessFunc func(ctx context.Context, principal string, resources []string) ([]string, error)
	readTuplesFunc  func(ctx context.Context, principal string, objectType string) ([]string, error)
	healthCheckFunc func(ctx context.Context) error
}

func (m *mockAccessChecker) CheckAccess(ctx context.Context, principal string, resources []string) ([]string, error) {
	if m.checkAccessFunc != nil {
		return m.checkAccessFunc(ctx, principal, resources)
	}
	return []string{"project:a27394a3-7a6c-4d0f-9e0f-692d8753924f#auditor@user:auth0|alice\ttrue"}, nil
}

func (m *mockAccessChecker) ReadTuples(ctx context.Context, principal string, objectType string) ([]string, error) {
	if m.readTuplesFunc != nil {
		return m.readTuplesFunc(ctx, principal, objectType)
	}
	return []string{}, nil
}

func (m *mockAccessChecker) HealthCheck(ctx context.Context) error {
	if m.healthCheckFunc != nil {
		return m.healthCheckFunc(ctx)
	}
	return nil
}

// contextWithClaims returns a context with HeimdallClaims pre-loaded.
func contextWithClaims(principal string) context.Context {
	claims := &contracts.HeimdallClaims{Principal: principal, Email: "test@example.com"}
	return context.WithValue(context.Background(), constants.ClaimsContextKey, claims)
}

// ===== AccessService unit tests =====

func TestNewAccessService(t *testing.T) {
	svc := NewAccessService(&mockAuthRepository{}, &mockAccessChecker{})
	if svc == nil {
		t.Fatal("NewAccessService returned nil")
	}
}

func TestJWTAuth_Success(t *testing.T) {
	authRepo := &mockAuthRepository{
		validateTokenFunc: func(_ context.Context, _ string) (*contracts.HeimdallClaims, error) {
			return &contracts.HeimdallClaims{Principal: "test-user", Email: "test@example.com"}, nil
		},
	}
	svc := NewAccessService(authRepo, &mockAccessChecker{})

	resultCtx, err := svc.JWTAuth(context.Background(), "Bearer valid-token", &security.JWTScheme{})
	if err != nil {
		t.Fatalf("JWTAuth failed: %v", err)
	}

	claims, ok := resultCtx.Value(constants.ClaimsContextKey).(*contracts.HeimdallClaims)
	if !ok {
		t.Fatal("Claims not found in context")
	}
	if claims.Principal != "test-user" {
		t.Errorf("expected principal 'test-user', got '%s'", claims.Principal)
	}
}

func TestJWTAuth_WithoutBearerPrefix(t *testing.T) {
	authRepo := &mockAuthRepository{
		validateTokenFunc: func(_ context.Context, token string) (*contracts.HeimdallClaims, error) {
			if token != "valid-token" {
				t.Errorf("expected token 'valid-token', got '%s'", token)
			}
			return &contracts.HeimdallClaims{Principal: "test-user"}, nil
		},
	}
	svc := NewAccessService(authRepo, &mockAccessChecker{})

	_, err := svc.JWTAuth(context.Background(), "valid-token", &security.JWTScheme{})
	if err != nil {
		t.Fatalf("JWTAuth failed: %v", err)
	}
}

func TestJWTAuth_InvalidToken(t *testing.T) {
	authRepo := &mockAuthRepository{
		validateTokenFunc: func(_ context.Context, _ string) (*contracts.HeimdallClaims, error) {
			return nil, errors.New("invalid token")
		},
	}
	svc := NewAccessService(authRepo, &mockAccessChecker{})

	_, err := svc.JWTAuth(context.Background(), "invalid-token", &security.JWTScheme{})
	if err == nil {
		t.Fatal("JWTAuth should have failed with invalid token")
	}
	t.Logf("Got expected error: %v", err)
}

func TestCheckAccess_Success(t *testing.T) {
	const wantResult = "project:a27394a3-7a6c-4d0f-9e0f-692d8753924f#auditor@user:auth0|alice\ttrue"
	checker := &mockAccessChecker{
		checkAccessFunc: func(_ context.Context, principal string, resources []string) ([]string, error) {
			if principal != "test-user" {
				t.Errorf("unexpected principal: %s", principal)
			}
			if len(resources) != 2 {
				t.Errorf("expected 2 resources, got %d", len(resources))
			}
			return []string{wantResult}, nil
		},
	}
	svc := NewAccessService(&mockAuthRepository{}, checker)
	ctx := contextWithClaims("test-user")

	result, err := svc.CheckAccess(ctx, &accesssvc.CheckAccessPayload{
		Version:  "1",
		Requests: []string{"resource1", "resource2"},
	})
	if err != nil {
		t.Fatalf("CheckAccess failed: %v", err)
	}
	if len(result.Results) != 1 {
		t.Errorf("expected 1 result, got %d", len(result.Results))
	}
	if result.Results[0] != wantResult {
		t.Errorf("expected %q, got %q", wantResult, result.Results[0])
	}
}

func TestCheckAccess_MissingClaims(t *testing.T) {
	svc := NewAccessService(&mockAuthRepository{}, &mockAccessChecker{})

	_, err := svc.CheckAccess(context.Background(), &accesssvc.CheckAccessPayload{
		Version:  "1",
		Requests: []string{"resource1"},
	})
	if err == nil {
		t.Fatal("CheckAccess should fail without claims in context")
	}
	t.Logf("Got expected error: %v", err)
}

func TestCheckAccess_UnsupportedVersion(t *testing.T) {
	svc := NewAccessService(&mockAuthRepository{}, &mockAccessChecker{})
	ctx := contextWithClaims("test-user")

	_, err := svc.CheckAccess(ctx, &accesssvc.CheckAccessPayload{
		Version:  "2",
		Requests: []string{"resource1"},
	})
	if err == nil {
		t.Fatal("CheckAccess should fail with unsupported version")
	}
	t.Logf("Got expected error: %v", err)
}

func TestCheckAccess_EmptyRequests(t *testing.T) {
	svc := NewAccessService(&mockAuthRepository{}, &mockAccessChecker{})
	ctx := contextWithClaims("test-user")

	result, err := svc.CheckAccess(ctx, &accesssvc.CheckAccessPayload{
		Version:  "1",
		Requests: []string{},
	})
	if err != nil {
		t.Fatalf("CheckAccess failed: %v", err)
	}
	if len(result.Results) != 0 {
		t.Errorf("expected 0 results for empty requests, got %d", len(result.Results))
	}
}

func TestCheckAccess_NATSFailure(t *testing.T) {
	checker := &mockAccessChecker{
		checkAccessFunc: func(_ context.Context, _ string, _ []string) ([]string, error) {
			return nil, errors.New("NATS connection failed")
		},
	}
	svc := NewAccessService(&mockAuthRepository{}, checker)

	_, err := svc.CheckAccess(contextWithClaims("test-user"), &accesssvc.CheckAccessPayload{
		Version:  "1",
		Requests: []string{"resource1"},
	})
	if err == nil {
		t.Fatal("CheckAccess should fail on client error")
	}
	t.Logf("Got expected error: %v", err)
}

func TestReadyz_Success(t *testing.T) {
	svc := NewAccessService(&mockAuthRepository{}, &mockAccessChecker{})

	result, err := svc.Readyz(context.Background())
	if err != nil {
		t.Fatalf("Readyz failed: %v", err)
	}
	if string(result) != "OK" {
		t.Errorf("expected 'OK', got '%s'", string(result))
	}
}

func TestReadyz_NilClient(t *testing.T) {
	svc := NewAccessService(&mockAuthRepository{}, nil)

	_, err := svc.Readyz(context.Background())
	if err == nil {
		t.Fatal("Readyz should fail when client is nil")
	}
	t.Logf("Got expected error: %v", err)
}

func TestReadyz_TypedNilClient(t *testing.T) {
	// A typed-nil pointer stored in the interface must be normalised at
	// construction time so that Readyz reports not-ready instead of panicking.
	var typedNil *mockAccessChecker
	svc := NewAccessService(&mockAuthRepository{}, typedNil)

	_, err := svc.Readyz(context.Background())
	if err == nil {
		t.Fatal("Readyz should fail when client is a typed-nil pointer")
	}
	t.Logf("Got expected error: %v", err)
}

func TestReadyz_ClientHealthCheckFails(t *testing.T) {
	checker := &mockAccessChecker{
		healthCheckFunc: func(_ context.Context) error {
			return constants.ErrMessagingRepoNotInit
		},
	}
	svc := NewAccessService(&mockAuthRepository{}, checker)

	_, err := svc.Readyz(context.Background())
	if err == nil {
		t.Fatal("Readyz should fail when client health check fails")
	}
	t.Logf("Got expected error: %v", err)
}

func TestLivez(t *testing.T) {
	svc := NewAccessService(&mockAuthRepository{}, &mockAccessChecker{})

	result, err := svc.Livez(context.Background())
	if err != nil {
		t.Fatalf("Livez failed: %v", err)
	}
	if string(result) != "OK" {
		t.Errorf("expected 'OK', got '%s'", string(result))
	}
}

func TestMyGrants_Success(t *testing.T) {
	const principal = "auth0|testuser"
	wantGrants := []string{
		"project:a27394a3-7a6c-4d0f-9e0f-692d8753924f#auditor@user:auth0|testuser",
		"project:b3c72e18-1a2b-4c3d-8e9f-123456789abc#writer@user:auth0|testuser",
	}
	checker := &mockAccessChecker{
		readTuplesFunc: func(_ context.Context, p string, objectType string) ([]string, error) {
			if p != principal {
				t.Errorf("unexpected principal: %s", p)
			}
			if objectType != "project" {
				t.Errorf("unexpected object type: %s", objectType)
			}
			return wantGrants, nil
		},
	}
	svc := NewAccessService(&mockAuthRepository{}, checker)

	result, err := svc.MyGrants(contextWithClaims(principal), &accesssvc.MyGrantsPayload{
		BearerToken: "tok",
		Version:     "1",
		ObjectType:  "project",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Grants) != 2 {
		t.Errorf("expected 2 grants, got %d", len(result.Grants))
	}
}

func TestMyGrants_EmptyResults(t *testing.T) {
	checker := &mockAccessChecker{
		readTuplesFunc: func(_ context.Context, _ string, _ string) ([]string, error) {
			return []string{}, nil
		},
	}
	svc := NewAccessService(&mockAuthRepository{}, checker)

	result, err := svc.MyGrants(contextWithClaims("auth0|user"), &accesssvc.MyGrantsPayload{
		BearerToken: "tok",
		Version:     "1",
		ObjectType:  "committee",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Grants == nil {
		t.Error("grants should not be nil")
	}
	if len(result.Grants) != 0 {
		t.Errorf("expected 0 grants, got %d", len(result.Grants))
	}
}

func TestMyGrants_UnsupportedVersion(t *testing.T) {
	svc := NewAccessService(&mockAuthRepository{}, &mockAccessChecker{})

	_, err := svc.MyGrants(contextWithClaims("auth0|user"), &accesssvc.MyGrantsPayload{
		BearerToken: "tok",
		Version:     "2",
		ObjectType:  "project",
	})
	if err == nil {
		t.Fatal("expected error for unsupported version, got nil")
	}
}

func TestMyGrants_MissingClaims(t *testing.T) {
	svc := NewAccessService(&mockAuthRepository{}, &mockAccessChecker{})

	_, err := svc.MyGrants(context.Background(), &accesssvc.MyGrantsPayload{
		BearerToken: "tok",
		Version:     "1",
		ObjectType:  "project",
	})
	if err == nil {
		t.Fatal("expected unauthorized error, got nil")
	}
}

// ===== Goa error-mapping tests =====
// These tests assert the *goa.ServiceError.Name field so that regressions in
// the InternalServerError / ServiceUnavailable mapping cannot pass silently.

func goaErrorName(t *testing.T, err error) string {
	t.Helper()
	var svcErr *goa.ServiceError
	if !errors.As(err, &svcErr) {
		t.Fatalf("expected *goa.ServiceError, got %T: %v", err, err)
	}
	return svcErr.Name
}

func TestCheckAccess_ErrorMapping(t *testing.T) {
	natsErr := errors.New("NATS connection failed")

	tests := []struct {
		name        string
		clientErr   error
		wantGoaName string
	}{
		{
			name:        "ErrUnexpectedResponse → 500 InternalServerError",
			clientErr:   fmt.Errorf("wrap: %w", constants.ErrUnexpectedResponse),
			wantGoaName: "InternalServerError",
		},
		{
			name:        "NATS transport error → 503 ServiceUnavailable",
			clientErr:   natsErr,
			wantGoaName: "ServiceUnavailable",
		},
		{
			name:        "ErrPrincipalRequired → 401 Unauthorized",
			clientErr:   constants.ErrPrincipalRequired,
			wantGoaName: "Unauthorized",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			checker := &mockAccessChecker{
				checkAccessFunc: func(_ context.Context, _ string, _ []string) ([]string, error) {
					return nil, tc.clientErr
				},
			}
			svc := NewAccessService(&mockAuthRepository{}, checker)

			_, err := svc.CheckAccess(contextWithClaims("alice"), &accesssvc.CheckAccessPayload{
				Version:  "1",
				Requests: []string{"project:abc#viewer"},
			})
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if got := goaErrorName(t, err); got != tc.wantGoaName {
				t.Errorf("expected Goa error name %q, got %q", tc.wantGoaName, got)
			}
		})
	}
}

func TestMyGrants_ErrorMapping(t *testing.T) {
	natsErr := errors.New("NATS connection failed")

	tests := []struct {
		name        string
		clientErr   error
		wantGoaName string
	}{
		{
			name:        "ErrUnexpectedResponse → 500 InternalServerError",
			clientErr:   fmt.Errorf("wrap: %w", constants.ErrUnexpectedResponse),
			wantGoaName: "InternalServerError",
		},
		{
			name:        "NATS transport error → 503 ServiceUnavailable",
			clientErr:   natsErr,
			wantGoaName: "ServiceUnavailable",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			checker := &mockAccessChecker{
				readTuplesFunc: func(_ context.Context, _ string, _ string) ([]string, error) {
					return nil, tc.clientErr
				},
			}
			svc := NewAccessService(&mockAuthRepository{}, checker)

			_, err := svc.MyGrants(contextWithClaims("alice"), &accesssvc.MyGrantsPayload{
				BearerToken: "tok",
				Version:     "1",
				ObjectType:  "project",
			})
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if got := goaErrorName(t, err); got != tc.wantGoaName {
				t.Errorf("expected Goa error name %q, got %q", tc.wantGoaName, got)
			}
		})
	}
}
