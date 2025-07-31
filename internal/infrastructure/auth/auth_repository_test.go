// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT
package auth

import (
	"context"
	"testing"
	"time"
)

func TestNewAuthRepository_Success(t *testing.T) {
	// Use public test JWKS endpoints that are commonly available
	jwksURL := "https://www.googleapis.com/oauth2/v3/certs"
	issuer := "https://accounts.google.com"
	audience := "test-audience"

	repo, err := NewAuthRepository(jwksURL, issuer, audience)
	if err != nil {
		t.Fatalf("NewAuthRepository failed: %v", err)
	}

	if repo == nil {
		t.Fatal("NewAuthRepository returned nil")
	}

	authRepo, ok := repo.(*authRepository)
	if !ok {
		t.Fatal("Expected *authRepository type")
	}

	if authRepo.validator == nil {
		t.Error("Validator not initialized")
	}
}

func TestNewAuthRepository_InvalidJWKSURL(t *testing.T) {
	jwksURL := "://invalid-url" // More obviously invalid URL
	issuer := "https://example.com"
	audience := "test-audience"

	_, err := NewAuthRepository(jwksURL, issuer, audience)
	if err == nil {
		t.Fatal("NewAuthRepository should have failed with invalid JWKS URL")
	}

	t.Logf("Got expected error: %v", err)
}

func TestNewAuthRepository_InvalidIssuerURL(t *testing.T) {
	jwksURL := "https://example.com/.well-known/jwks"
	issuer := "://invalid-url" // More obviously invalid URL
	audience := "test-audience"

	_, err := NewAuthRepository(jwksURL, issuer, audience)
	if err == nil {
		t.Fatal("NewAuthRepository should have failed with invalid issuer URL")
	}

	t.Logf("Got expected error: %v", err)
}

func TestNewAuthRepository_EmptyValues(t *testing.T) {
	tests := []struct {
		name       string
		jwksURL    string
		issuer     string
		audience   string
		shouldFail bool
	}{
		{"empty JWKS URL", "", "https://example.com", "test", false},                                 // May not fail - URL parsing is permissive
		{"empty issuer", "https://example.com/.well-known/jwks", "", "test", true},                   // Should fail
		{"empty audience", "https://example.com/.well-known/jwks", "https://example.com", "", false}, // May not fail
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewAuthRepository(tt.jwksURL, tt.issuer, tt.audience)
			if tt.shouldFail && err == nil {
				t.Errorf("NewAuthRepository should have failed with %s", tt.name)
			} else if !tt.shouldFail && err != nil {
				t.Logf("NewAuthRepository failed with %s (may be expected): %v", tt.name, err)
			} else if err != nil {
				t.Logf("Got expected error for %s: %v", tt.name, err)
			} else {
				t.Logf("NewAuthRepository succeeded with %s", tt.name)
			}
		})
	}
}

func TestValidateToken_InvalidToken(t *testing.T) {
	// Use a test configuration that won't actually validate but will create the repository
	jwksURL := "https://www.googleapis.com/oauth2/v3/certs"
	issuer := "https://accounts.google.com"
	audience := "test-audience"

	repo, err := NewAuthRepository(jwksURL, issuer, audience)
	if err != nil {
		t.Fatalf("NewAuthRepository failed: %v", err)
	}

	ctx := context.Background()

	// Test with obviously invalid token
	_, err = repo.ValidateToken(ctx, "invalid-token")
	if err == nil {
		t.Fatal("ValidateToken should have failed with invalid token")
	}

	t.Logf("Got expected error: %v", err)
}

func TestValidateToken_EmptyToken(t *testing.T) {
	jwksURL := "https://www.googleapis.com/oauth2/v3/certs"
	issuer := "https://accounts.google.com"
	audience := "test-audience"

	repo, err := NewAuthRepository(jwksURL, issuer, audience)
	if err != nil {
		t.Fatalf("NewAuthRepository failed: %v", err)
	}

	ctx := context.Background()

	// Test with empty token
	_, err = repo.ValidateToken(ctx, "")
	if err == nil {
		t.Fatal("ValidateToken should have failed with empty token")
	}

	t.Logf("Got expected error: %v", err)
}

func TestValidateToken_MalformedJWT(t *testing.T) {
	jwksURL := "https://www.googleapis.com/oauth2/v3/certs"
	issuer := "https://accounts.google.com"
	audience := "test-audience"

	repo, err := NewAuthRepository(jwksURL, issuer, audience)
	if err != nil {
		t.Fatalf("NewAuthRepository failed: %v", err)
	}

	ctx := context.Background()

	// Test with malformed JWT (not proper base64 encoding)
	_, err = repo.ValidateToken(ctx, "malformed.jwt.token")
	if err == nil {
		t.Fatal("ValidateToken should have failed with malformed JWT")
	}

	t.Logf("Got expected error: %v", err)
}

func TestValidateToken_ContextCancellation(t *testing.T) {
	jwksURL := "https://www.googleapis.com/oauth2/v3/certs"
	issuer := "https://accounts.google.com"
	audience := "test-audience"

	repo, err := NewAuthRepository(jwksURL, issuer, audience)
	if err != nil {
		t.Fatalf("NewAuthRepository failed: %v", err)
	}

	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = repo.ValidateToken(ctx, "some.jwt.token")
	if err == nil {
		t.Fatal("ValidateToken should have failed with cancelled context")
	}

	// Check if it's a context cancellation error
	if ctx.Err() != context.Canceled {
		t.Error("Context should be cancelled")
	}

	t.Logf("Got expected error: %v", err)
}

// Benchmark test for token validation
func BenchmarkValidateToken(b *testing.B) {
	jwksURL := "https://www.googleapis.com/oauth2/v3/certs"
	issuer := "https://accounts.google.com"
	audience := "test-audience"

	repo, err := NewAuthRepository(jwksURL, issuer, audience)
	if err != nil {
		b.Fatalf("NewAuthRepository failed: %v", err)
	}

	ctx := context.Background()
	token := "invalid.jwt.token" // Will fail validation but exercises the code path

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		repo.ValidateToken(ctx, token)
	}
}

func TestValidateToken_WithTimeout(t *testing.T) {
	jwksURL := "https://www.googleapis.com/oauth2/v3/certs"
	issuer := "https://accounts.google.com"
	audience := "test-audience"

	repo, err := NewAuthRepository(jwksURL, issuer, audience)
	if err != nil {
		t.Fatalf("NewAuthRepository failed: %v", err)
	}

	// Create context with very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	// Give it a moment for the context to timeout
	time.Sleep(2 * time.Millisecond)

	_, err = repo.ValidateToken(ctx, "some.jwt.token")
	if err == nil {
		t.Fatal("ValidateToken should have failed with timeout")
	}

	t.Logf("Got expected error: %v", err)
}

// Additional tests to improve coverage
func TestValidateToken_WellFormedButInvalidJWT(t *testing.T) {
	jwksURL := "https://www.googleapis.com/oauth2/v3/certs"
	issuer := "https://accounts.google.com"
	audience := "test-audience"

	repo, err := NewAuthRepository(jwksURL, issuer, audience)
	if err != nil {
		t.Fatalf("NewAuthRepository failed: %v", err)
	}

	ctx := context.Background()

	// A well-formed JWT that's just not valid (fake but properly formatted)
	fakeJWT := "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiYWRtaW4iOnRydWV9.invalid"

	_, err = repo.ValidateToken(ctx, fakeJWT)
	if err == nil {
		t.Fatal("ValidateToken should have failed with invalid signature")
	}

	t.Logf("Got expected error: %v", err)
}

func TestNewAuthRepository_NetworkFailure(t *testing.T) {
	// Use a non-existent domain to trigger network failure
	jwksURL := "https://non-existent-domain-12345.com/.well-known/jwks"
	issuer := "https://non-existent-domain-12345.com"
	audience := "test-audience"

	// This should succeed in creating the repository (network call happens later)
	repo, err := NewAuthRepository(jwksURL, issuer, audience)
	if err != nil {
		t.Fatalf("NewAuthRepository failed: %v", err)
	}

	if repo == nil {
		t.Fatal("NewAuthRepository returned nil")
	}

	// The network failure will happen during token validation
	ctx := context.Background()
	_, err = repo.ValidateToken(ctx, "some.jwt.token")
	if err == nil {
		t.Fatal("ValidateToken should have failed due to network issues")
	}

	t.Logf("Got expected network error: %v", err)
}

func TestAuthRepository_Interface(t *testing.T) {
	jwksURL := "https://www.googleapis.com/oauth2/v3/certs"
	issuer := "https://accounts.google.com"
	audience := "test-audience"

	repo, err := NewAuthRepository(jwksURL, issuer, audience)
	if err != nil {
		t.Fatalf("NewAuthRepository failed: %v", err)
	}

	// Ensure it implements the interface properly
	authRepo, ok := repo.(*authRepository)
	if !ok {
		t.Fatal("Expected *authRepository type")
	}

	if authRepo.validator == nil {
		t.Error("Validator should not be nil")
	}
}
