// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT
package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	accesssvc "github.com/linuxfoundation/lfx-v2-access-check/gen/access_svc"
	"github.com/linuxfoundation/lfx-v2-access-check/internal/domain/contracts"
	"github.com/linuxfoundation/lfx-v2-access-check/pkg/constants"
	goa "goa.design/goa/v3/pkg"
	"goa.design/goa/v3/security"
)

// Mock implementations for testing — shared by service and client tests in this package.

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

type mockMessagingRepository struct {
	requestFunc func(ctx context.Context, subject string, data []byte, timeout time.Duration) ([]byte, error)
	closeFunc   func() error
}

func (m *mockMessagingRepository) Request(ctx context.Context, subject string, data []byte, timeout time.Duration) ([]byte, error) {
	if m.requestFunc != nil {
		return m.requestFunc(ctx, subject, data, timeout)
	}
	return []byte("project:a27394a3-7a6c-4d0f-9e0f-692d8753924f#auditor@user:auth0|alice\ttrue"), nil
}

func (m *mockMessagingRepository) Close() error {
	if m.closeFunc != nil {
		return m.closeFunc()
	}
	return nil
}

func (m *mockMessagingRepository) HealthCheck(_ context.Context) error {
	return nil
}

// contextWithClaims returns a context with HeimdallClaims pre-loaded.
func contextWithClaims(principal string) context.Context {
	claims := &contracts.HeimdallClaims{Principal: principal, Email: "test@example.com"}
	return context.WithValue(context.Background(), constants.ClaimsContextKey, claims)
}

// ===== AccessService unit tests =====

func TestNewAccessService(t *testing.T) {
	service := NewAccessService(&mockAuthRepository{}, &mockMessagingRepository{})
	if service == nil {
		t.Fatal("NewAccessService returned nil")
	}
}

func TestJWTAuth_Success(t *testing.T) {
	authRepo := &mockAuthRepository{
		validateTokenFunc: func(_ context.Context, _ string) (*contracts.HeimdallClaims, error) {
			return &contracts.HeimdallClaims{Principal: "test-user", Email: "test@example.com"}, nil
		},
	}
	service := NewAccessService(authRepo, &mockMessagingRepository{})

	resultCtx, err := service.JWTAuth(context.Background(), "Bearer valid-token", &security.JWTScheme{})
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
	service := NewAccessService(authRepo, &mockMessagingRepository{})

	_, err := service.JWTAuth(context.Background(), "valid-token", &security.JWTScheme{})
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
	service := NewAccessService(authRepo, &mockMessagingRepository{})

	_, err := service.JWTAuth(context.Background(), "invalid-token", &security.JWTScheme{})
	if err == nil {
		t.Fatal("JWTAuth should have failed with invalid token")
	}
	t.Logf("Got expected error: %v", err)
}

func TestCheckAccess_Success(t *testing.T) {
	messagingRepo := &mockMessagingRepository{
		requestFunc: func(_ context.Context, subject string, _ []byte, _ time.Duration) ([]byte, error) {
			if subject != constants.AccessCheckSubject {
				t.Errorf("expected subject %q, got %q", constants.AccessCheckSubject, subject)
			}
			return []byte("project:a27394a3-7a6c-4d0f-9e0f-692d8753924f#auditor@user:auth0|alice\ttrue"), nil
		},
	}
	service := NewAccessService(&mockAuthRepository{}, messagingRepo)

	ctx := context.WithValue(context.Background(), constants.ClaimsContextKey,
		&contracts.HeimdallClaims{Principal: "test-user", Email: "test@example.com"})

	result, err := service.CheckAccess(ctx, &accesssvc.CheckAccessPayload{
		Version:  "1",
		Requests: []string{"resource1", "resource2"},
	})
	if err != nil {
		t.Fatalf("CheckAccess failed: %v", err)
	}
	if len(result.Results) != 1 {
		t.Errorf("expected 1 result, got %d", len(result.Results))
	}

	expectedPrefix := "project:a27394a3-7a6c-4d0f-9e0f-692d8753924f#auditor@user:auth0|alice"
	found := false
	for _, r := range result.Results {
		if strings.HasPrefix(r, expectedPrefix) && strings.HasSuffix(r, "\ttrue") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected result with prefix %q and suffix \\ttrue, got %v", expectedPrefix, result.Results)
	}
}

func TestCheckAccess_MissingClaims(t *testing.T) {
	service := NewAccessService(&mockAuthRepository{}, &mockMessagingRepository{})

	_, err := service.CheckAccess(context.Background(), &accesssvc.CheckAccessPayload{
		Version:  "1",
		Requests: []string{"resource1"},
	})
	if err == nil {
		t.Fatal("CheckAccess should fail without claims in context")
	}
	t.Logf("Got expected error: %v", err)
}

func TestCheckAccess_UnsupportedVersion(t *testing.T) {
	service := NewAccessService(&mockAuthRepository{}, &mockMessagingRepository{})
	ctx := contextWithClaims("test-user")

	_, err := service.CheckAccess(ctx, &accesssvc.CheckAccessPayload{
		Version:  "2",
		Requests: []string{"resource1"},
	})
	if err == nil {
		t.Fatal("CheckAccess should fail with unsupported version")
	}
	t.Logf("Got expected error: %v", err)
}

func TestCheckAccess_EmptyRequests(t *testing.T) {
	service := NewAccessService(&mockAuthRepository{}, &mockMessagingRepository{})
	ctx := contextWithClaims("test-user")

	result, err := service.CheckAccess(ctx, &accesssvc.CheckAccessPayload{
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
	messagingRepo := &mockMessagingRepository{
		requestFunc: func(_ context.Context, _ string, _ []byte, _ time.Duration) ([]byte, error) {
			return nil, errors.New("NATS connection failed")
		},
	}
	service := NewAccessService(&mockAuthRepository{}, messagingRepo)

	_, err := service.CheckAccess(contextWithClaims("test-user"), &accesssvc.CheckAccessPayload{
		Version:  "1",
		Requests: []string{"resource1"},
	})
	if err == nil {
		t.Fatal("CheckAccess should fail on NATS error")
	}
	t.Logf("Got expected error: %v", err)
}

func TestReadyz_Success(t *testing.T) {
	service := NewAccessService(&mockAuthRepository{}, &mockMessagingRepository{})

	result, err := service.Readyz(context.Background())
	if err != nil {
		t.Fatalf("Readyz failed: %v", err)
	}
	if string(result) != "OK" {
		t.Errorf("expected 'OK', got '%s'", string(result))
	}
}

func TestReadyz_MessagingRepoNil(t *testing.T) {
	service := NewAccessService(&mockAuthRepository{}, nil)

	_, err := service.Readyz(context.Background())
	if err == nil {
		t.Fatal("Readyz should fail with nil messaging repo")
	}
	t.Logf("Got expected error: %v", err)
}

func TestLivez(t *testing.T) {
	service := NewAccessService(&mockAuthRepository{}, &mockMessagingRepository{})

	result, err := service.Livez(context.Background())
	if err != nil {
		t.Fatalf("Livez failed: %v", err)
	}
	if string(result) != "OK" {
		t.Errorf("expected 'OK', got '%s'", string(result))
	}
}

func TestMyGrants_Success(t *testing.T) {
	const principal = "auth0|testuser"
	messagingRepo := &mockMessagingRepository{
		requestFunc: func(_ context.Context, subject string, _ []byte, _ time.Duration) ([]byte, error) {
			if subject != constants.ReadTuplesSubject {
				t.Errorf("unexpected NATS subject: %s", subject)
			}
			return []byte(`{"results":["project:a27394a3-7a6c-4d0f-9e0f-692d8753924f#auditor@user:auth0|testuser","project:b3c72e18-1a2b-4c3d-8e9f-123456789abc#writer@user:auth0|testuser"]}`), nil
		},
	}
	svc := NewAccessService(&mockAuthRepository{}, messagingRepo)

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
	messagingRepo := &mockMessagingRepository{
		requestFunc: func(_ context.Context, _ string, _ []byte, _ time.Duration) ([]byte, error) {
			return []byte(`{"results":[]}`), nil
		},
	}
	svc := NewAccessService(&mockAuthRepository{}, messagingRepo)

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
	svc := NewAccessService(&mockAuthRepository{}, &mockMessagingRepository{})

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
	svc := NewAccessService(&mockAuthRepository{}, &mockMessagingRepository{})

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
			svc := NewAccessService(&mockAuthRepository{}, &mockMessagingRepository{
				requestFunc: func(_ context.Context, _ string, _ []byte, _ time.Duration) ([]byte, error) {
					return nil, tc.clientErr
				},
			})
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
			svc := NewAccessService(&mockAuthRepository{}, &mockMessagingRepository{
				requestFunc: func(_ context.Context, _ string, _ []byte, _ time.Duration) ([]byte, error) {
					return nil, tc.clientErr
				},
			})
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
