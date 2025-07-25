// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT
package service

import (
	"context"
	"errors"
	"testing"
	"time"

	accesssvc "github.com/linuxfoundation/lfx-v2-access-check/gen/access_svc"
	"github.com/linuxfoundation/lfx-v2-access-check/internal/domain/contracts"
	"goa.design/goa/v3/security"
)

// Mock implementations for testing
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
	return []byte("allow"), nil
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

func TestNewAccessService(t *testing.T) {
	authRepo := &mockAuthRepository{}
	messagingRepo := &mockMessagingRepository{}

	service := NewAccessService(authRepo, messagingRepo)

	if service == nil {
		t.Fatal("NewAccessService returned nil")
	}

	// Service is properly initialized if it's not nil
	// Internal field validation is not necessary for public API tests
}

func TestJWTAuth_Success(t *testing.T) {
	authRepo := &mockAuthRepository{
		validateTokenFunc: func(_ context.Context, _ string) (*contracts.HeimdallClaims, error) {
			return &contracts.HeimdallClaims{Principal: "test-user", Email: "test@example.com"}, nil
		},
	}
	messagingRepo := &mockMessagingRepository{}
	service := NewAccessService(authRepo, messagingRepo)

	ctx := context.Background()
	scheme := &security.JWTScheme{}

	// Test with Bearer prefix
	resultCtx, err := service.JWTAuth(ctx, "Bearer valid-token", scheme)
	if err != nil {
		t.Fatalf("JWTAuth failed: %v", err)
	}

	// Check that claims are in context
	claims, ok := resultCtx.Value(claimsKey).(*contracts.HeimdallClaims)
	if !ok {
		t.Fatal("Claims not found in context")
	}

	if claims.Principal != "test-user" {
		t.Errorf("Expected principal 'test-user', got '%s'", claims.Principal)
	}
}

func TestJWTAuth_WithoutBearerPrefix(t *testing.T) {
	authRepo := &mockAuthRepository{
		validateTokenFunc: func(_ context.Context, token string) (*contracts.HeimdallClaims, error) {
			if token != "valid-token" {
				t.Errorf("Expected token 'valid-token', got '%s'", token)
			}
			return &contracts.HeimdallClaims{Principal: "test-user"}, nil
		},
	}
	messagingRepo := &mockMessagingRepository{}
	service := NewAccessService(authRepo, messagingRepo)

	ctx := context.Background()
	scheme := &security.JWTScheme{}

	// Test without Bearer prefix
	_, err := service.JWTAuth(ctx, "valid-token", scheme)
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
	messagingRepo := &mockMessagingRepository{}
	service := NewAccessService(authRepo, messagingRepo)

	ctx := context.Background()
	scheme := &security.JWTScheme{}

	_, err := service.JWTAuth(ctx, "invalid-token", scheme)
	if err == nil {
		t.Fatal("JWTAuth should have failed with invalid token")
	}

	// Just verify it's an error - the exact type checking is less important for unit tests
	t.Logf("Got expected error: %v", err)
}

func TestCheckAccess_Success(t *testing.T) {
	authRepo := &mockAuthRepository{}
	messagingRepo := &mockMessagingRepository{
		requestFunc: func(_ context.Context, subject string, _ []byte, _ time.Duration) ([]byte, error) {
			if subject != "dev.lfx.access_check.request" {
				t.Errorf("Expected subject 'dev.lfx.access_check.request', got '%s'", subject)
			}
			return []byte("allow"), nil
		},
	}
	service := NewAccessService(authRepo, messagingRepo)

	// Create context with claims
	claims := &contracts.HeimdallClaims{Principal: "test-user", Email: "test@example.com"}
	ctx := context.WithValue(context.Background(), claimsKey, claims)

	payload := &accesssvc.CheckAccessPayload{
		Version:  "1",
		Requests: []string{"resource1", "resource2"},
	}

	result, err := service.CheckAccess(ctx, payload)
	if err != nil {
		t.Fatalf("CheckAccess failed: %v", err)
	}

	if result == nil {
		t.Fatal("CheckAccess returned nil result")
	}

	if len(result.Results) != 1 {
		t.Errorf("Expected 1 result, got %d", len(result.Results))
	}

	if result.Results[0] != "allow" {
		t.Errorf("Expected result 'allow', got '%s'", result.Results[0])
	}
}

func TestCheckAccess_MissingClaims(t *testing.T) {
	authRepo := &mockAuthRepository{}
	messagingRepo := &mockMessagingRepository{}
	service := NewAccessService(authRepo, messagingRepo)

	// Context without claims
	ctx := context.Background()

	payload := &accesssvc.CheckAccessPayload{
		Version:  "1",
		Requests: []string{"resource1"},
	}

	_, err := service.CheckAccess(ctx, payload)
	if err == nil {
		t.Fatal("CheckAccess should have failed without claims")
	}

	t.Logf("Got expected error: %v", err)
}

func TestCheckAccess_UnsupportedVersion(t *testing.T) {
	authRepo := &mockAuthRepository{}
	messagingRepo := &mockMessagingRepository{}
	service := NewAccessService(authRepo, messagingRepo)

	// Create context with claims
	claims := &contracts.HeimdallClaims{Principal: "test-user"}
	ctx := context.WithValue(context.Background(), claimsKey, claims)

	payload := &accesssvc.CheckAccessPayload{
		Version:  "2", // Unsupported version
		Requests: []string{"resource1"},
	}

	_, err := service.CheckAccess(ctx, payload)
	if err == nil {
		t.Fatal("CheckAccess should have failed with unsupported version")
	}

	t.Logf("Got expected error: %v", err)
}

func TestCheckAccess_EmptyRequests(t *testing.T) {
	authRepo := &mockAuthRepository{}
	messagingRepo := &mockMessagingRepository{}
	service := NewAccessService(authRepo, messagingRepo)

	// Create context with claims
	claims := &contracts.HeimdallClaims{Principal: "test-user"}
	ctx := context.WithValue(context.Background(), claimsKey, claims)

	payload := &accesssvc.CheckAccessPayload{
		Version:  "1",
		Requests: []string{}, // Empty requests
	}

	result, err := service.CheckAccess(ctx, payload)
	if err != nil {
		t.Fatalf("CheckAccess failed: %v", err)
	}

	if result == nil {
		t.Fatal("CheckAccess returned nil result")
	}

	if len(result.Results) != 0 {
		t.Errorf("Expected 0 results for empty requests, got %d", len(result.Results))
	}
}

func TestCheckAccess_NATSFailure(t *testing.T) {
	authRepo := &mockAuthRepository{}
	messagingRepo := &mockMessagingRepository{
		requestFunc: func(_ context.Context, _ string, _ []byte, _ time.Duration) ([]byte, error) {
			return nil, errors.New("NATS connection failed")
		},
	}
	service := NewAccessService(authRepo, messagingRepo)

	// Create context with claims
	claims := &contracts.HeimdallClaims{Principal: "test-user"}
	ctx := context.WithValue(context.Background(), claimsKey, claims)

	payload := &accesssvc.CheckAccessPayload{
		Version:  "1",
		Requests: []string{"resource1"},
	}

	_, err := service.CheckAccess(ctx, payload)
	if err == nil {
		t.Fatal("CheckAccess should have failed with NATS error")
	}

	t.Logf("Got expected error: %v", err)
}

func TestReadyz_Success(t *testing.T) {
	authRepo := &mockAuthRepository{}
	messagingRepo := &mockMessagingRepository{}
	service := NewAccessService(authRepo, messagingRepo)

	ctx := context.Background()

	result, err := service.Readyz(ctx)
	if err != nil {
		t.Fatalf("Readyz failed: %v", err)
	}

	if string(result) != "OK" {
		t.Errorf("Expected 'OK', got '%s'", string(result))
	}
}

func TestReadyz_MessagingRepoNil(t *testing.T) {
	authRepo := &mockAuthRepository{}
	service := NewAccessService(authRepo, nil) // nil messaging repo

	ctx := context.Background()

	_, err := service.Readyz(ctx)
	if err == nil {
		t.Fatal("Readyz should have failed with nil messaging repo")
	}

	t.Logf("Got expected error: %v", err)
}

func TestLivez(t *testing.T) {
	authRepo := &mockAuthRepository{}
	messagingRepo := &mockMessagingRepository{}
	service := NewAccessService(authRepo, messagingRepo)

	ctx := context.Background()

	result, err := service.Livez(ctx)
	if err != nil {
		t.Fatalf("Livez failed: %v", err)
	}

	if string(result) != "OK" {
		t.Errorf("Expected 'OK', got '%s'", string(result))
	}
}

func TestPerformAccessCheck_EmptyPrincipal(t *testing.T) {
	authRepo := &mockAuthRepository{}
	messagingRepo := &mockMessagingRepository{}
	service := NewAccessService(authRepo, messagingRepo)

	ctx := context.Background()

	_, err := service.performAccessCheck(ctx, "", []string{"resource1"})
	if err == nil {
		t.Fatal("performAccessCheck should have failed with empty principal")
	}

	if err.Error() != "principal is required" {
		t.Errorf("Expected 'principal is required', got '%s'", err.Error())
	}
}

func TestPerformAccessCheck_EmptyResources(t *testing.T) {
	authRepo := &mockAuthRepository{}
	messagingRepo := &mockMessagingRepository{}
	service := NewAccessService(authRepo, messagingRepo)

	ctx := context.Background()

	result, err := service.performAccessCheck(ctx, "test-user", []string{})
	if err != nil {
		t.Fatalf("performAccessCheck failed: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("Expected empty result for empty resources, got %d items", len(result))
	}
}

func TestPerformAccessCheck_UnexpectedResponse(t *testing.T) {
	authRepo := &mockAuthRepository{}
	messagingRepo := &mockMessagingRepository{
		requestFunc: func(_ context.Context, _ string, _ []byte, _ time.Duration) ([]byte, error) {
			// Return response with space in first 20 bytes (indicates error)
			return []byte("error message here"), nil
		},
	}
	service := NewAccessService(authRepo, messagingRepo)

	ctx := context.Background()

	_, err := service.performAccessCheck(ctx, "test-user", []string{"resource1"})
	if err == nil {
		t.Fatal("performAccessCheck should have failed with unexpected response")
	}

	if err.Error() != "unexpected response from access check service" {
		t.Errorf("Expected 'unexpected response from access check service', got '%s'", err.Error())
	}
}
