// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT
package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	accesssvc "github.com/linuxfoundation/lfx-v2-access-check/gen/access_svc"
	"github.com/linuxfoundation/lfx-v2-access-check/internal/domain/contracts"
	"github.com/linuxfoundation/lfx-v2-access-check/pkg/constants"
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
	claims, ok := resultCtx.Value(constants.ClaimsContextKey).(*contracts.HeimdallClaims)
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
			if subject != "lfx.access_check.request" {
				t.Errorf("Expected subject 'lfx.access_check.request', got '%s'", subject)
			}
			return []byte("project:a27394a3-7a6c-4d0f-9e0f-692d8753924f#auditor@user:auth0|alice\ttrue"), nil
		},
	}
	service := NewAccessService(authRepo, messagingRepo)

	// Create context with claims
	claims := &contracts.HeimdallClaims{Principal: "test-user", Email: "test@example.com"}
	ctx := context.WithValue(context.Background(), constants.ClaimsContextKey, claims)

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

	expectedPrefix := "project:a27394a3-7a6c-4d0f-9e0f-692d8753924f#auditor@user:auth0|alice"
	found := false
	for _, r := range result.Results {
		if strings.HasPrefix(r, expectedPrefix) && strings.HasSuffix(r, "\ttrue") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected result with prefix %q and suffix \\ttrue, got %v", expectedPrefix, result.Results)
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
	ctx := context.WithValue(context.Background(), constants.ClaimsContextKey, claims)

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
	ctx := context.WithValue(context.Background(), constants.ClaimsContextKey, claims)

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
	ctx := context.WithValue(context.Background(), constants.ClaimsContextKey, claims)

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

	if !errors.Is(err, constants.ErrUnexpectedResponse) {
		t.Errorf("Expected ErrUnexpectedResponse, got '%s'", err.Error())
	}
}

// contextWithClaims returns a context with HeimdallClaims pre-loaded.
func contextWithClaims(principal string) context.Context {
	claims := &contracts.HeimdallClaims{Principal: principal, Email: "test@example.com"}
	return context.WithValue(context.Background(), constants.ClaimsContextKey, claims)
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

func TestPerformReadTuples_Success(t *testing.T) {
	messagingRepo := &mockMessagingRepository{
		requestFunc: func(_ context.Context, subject string, _ []byte, _ time.Duration) ([]byte, error) {
			if subject != constants.ReadTuplesSubject {
				t.Errorf("unexpected NATS subject: %s", subject)
			}
			return []byte(`{"results":["project:abc#auditor@user:alice","committee:xyz#writer@user:alice"]}`), nil
		},
	}
	svc := NewAccessService(&mockAuthRepository{}, messagingRepo)

	results, err := svc.performReadTuples(context.Background(), "alice", "project")
	if err != nil {
		t.Fatalf("performReadTuples failed: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
}

func TestPerformReadTuples_NilResultsNormalized(t *testing.T) {
	messagingRepo := &mockMessagingRepository{
		requestFunc: func(_ context.Context, _ string, _ []byte, _ time.Duration) ([]byte, error) {
			return []byte(`{}`), nil
		},
	}
	svc := NewAccessService(&mockAuthRepository{}, messagingRepo)

	results, err := svc.performReadTuples(context.Background(), "alice", "project")
	if err != nil {
		t.Fatalf("performReadTuples failed: %v", err)
	}
	if results == nil {
		t.Error("expected empty slice, got nil")
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestPerformReadTuples_NATSFailure(t *testing.T) {
	messagingRepo := &mockMessagingRepository{
		requestFunc: func(_ context.Context, _ string, _ []byte, _ time.Duration) ([]byte, error) {
			return nil, errors.New("nats timeout")
		},
	}
	svc := NewAccessService(&mockAuthRepository{}, messagingRepo)

	_, err := svc.performReadTuples(context.Background(), "alice", "project")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "NATS request to subject") {
		t.Errorf("expected NATS transport error message, got '%s'", err.Error())
	}
}

func TestPerformReadTuples_UnmarshalFailure(t *testing.T) {
	messagingRepo := &mockMessagingRepository{
		requestFunc: func(_ context.Context, _ string, _ []byte, _ time.Duration) ([]byte, error) {
			return []byte(`not valid json`), nil
		},
	}
	svc := NewAccessService(&mockAuthRepository{}, messagingRepo)

	_, err := svc.performReadTuples(context.Background(), "alice", "project")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, constants.ErrUnexpectedResponse) {
		t.Errorf("expected ErrUnexpectedResponse, got '%s'", err.Error())
	}
}

func TestPerformReadTuples_BackendError(t *testing.T) {
	messagingRepo := &mockMessagingRepository{
		requestFunc: func(_ context.Context, _ string, _ []byte, _ time.Duration) ([]byte, error) {
			return []byte(`{"error":"store not found"}`), nil
		},
	}
	svc := NewAccessService(&mockAuthRepository{}, messagingRepo)

	_, err := svc.performReadTuples(context.Background(), "alice", "project")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "FGA error") {
		t.Errorf("expected FGA error message, got '%s'", err.Error())
	}
	if strings.Contains(err.Error(), "NATS request") {
		t.Errorf("backend error should not look like a NATS transport error, got '%s'", err.Error())
	}
}

// Unit tests for refactored helper methods

func TestBuildAccessCheckMessage(t *testing.T) {
	authRepo := &mockAuthRepository{}
	messagingRepo := &mockMessagingRepository{}
	service := NewAccessService(authRepo, messagingRepo)

	tests := []struct {
		name      string
		principal string
		resources []string
		expected  string
	}{
		{
			name:      "empty resources",
			principal: "user1",
			resources: []string{},
			expected:  "",
		},
		{
			name:      "single resource",
			principal: "user1",
			resources: []string{"repo1"},
			expected:  "repo1@user:user1",
		},
		{
			name:      "multiple resources",
			principal: "user1",
			resources: []string{"repo1", "repo2"},
			expected:  "repo1@user:user1\nrepo2@user:user1",
		},
		{
			name:      "empty resource filtered out",
			principal: "user1",
			resources: []string{"repo1", "", "repo2"},
			expected:  "repo1@user:user1\nrepo2@user:user1",
		},
		{
			name:      "all empty resources",
			principal: "user1",
			resources: []string{"", "", ""},
			expected:  "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := service.buildAccessCheckMessage(test.principal, test.resources)
			if result != test.expected {
				t.Errorf("Expected '%s', got '%s'", test.expected, result)
			}
		})
	}
}

func TestParseAccessCheckResponse(t *testing.T) {
	authRepo := &mockAuthRepository{}
	messagingRepo := &mockMessagingRepository{}
	service := NewAccessService(authRepo, messagingRepo)

	ctx := context.Background()

	tests := []struct {
		name         string
		responseData []byte
		expected     []string
		expectError  bool
	}{
		{
			name:         "valid response",
			responseData: []byte("true\nfalse\ntrue"),
			expected:     []string{"true", "false", "true"},
			expectError:  false,
		},
		{
			name:         "empty response",
			responseData: []byte(""),
			expected:     []string{},
			expectError:  false,
		},
		{
			name:         "response with empty lines",
			responseData: []byte("true\n\nfalse\n"),
			expected:     []string{"true", "false"},
			expectError:  false,
		},
		{
			name:         "response with spaces (error)",
			responseData: []byte("error message here"),
			expected:     nil,
			expectError:  true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result, err := service.parseAccessCheckResponse(ctx, test.responseData)

			if test.expectError {
				if err == nil {
					t.Fatal("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if len(result) != len(test.expected) {
				t.Errorf("Expected %d results, got %d", len(test.expected), len(result))
				return
			}

			for i, expected := range test.expected {
				if result[i] != expected {
					t.Errorf("Expected result[%d] = '%s', got '%s'", i, expected, result[i])
				}
			}
		})
	}
}
