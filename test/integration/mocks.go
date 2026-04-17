// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package integration provides integration test utilities and mocks.
package integration

import (
	"context"
	"errors"
	"time"

	"github.com/linuxfoundation/lfx-v2-access-check/internal/domain/contracts"
)

// MockAuthRepository provides a test implementation of AuthRepository
type MockAuthRepository struct{}

// ValidateToken validates a JWT token and returns claims for testing
func (m *MockAuthRepository) ValidateToken(_ context.Context, token string) (*contracts.HeimdallClaims, error) {
	// Return a test claim for valid tokens
	if token == "valid-token" {
		return &contracts.HeimdallClaims{
			Principal: "test-user",
			Email:     "test@example.com",
		}, nil
	}
	return nil, errors.New("unauthorized")
}

// HealthCheck always returns healthy for mock
func (m *MockAuthRepository) HealthCheck(_ context.Context) error {
	return nil
}

// MockMessagingRepository provides a test implementation of MessagingRepository
type MockMessagingRepository struct{}

// Request sends a mock request message and returns a mock response for testing
func (m *MockMessagingRepository) Request(_ context.Context, _ string, _ []byte, _ time.Duration) ([]byte, error) {
	// Return mock response
	return []byte("project:a27394a3-7a6c-4d0f-9e0f-692d8753924f#auditor@user:auth0|alice\ttrue\ncommittee:b3c72e18-1a2b-4c3d-8e9f-123456789abc#writer@user:auth0|alice\tfalse"), nil
}

// Close closes the mock messaging connection (no-op for testing)
func (m *MockMessagingRepository) Close() error {
	return nil
}

// HealthCheck always returns healthy for mock
func (m *MockMessagingRepository) HealthCheck(_ context.Context) error {
	return nil
}

// ConfigurableMessagingRepository provides a test implementation of MessagingRepository
// with a configurable response function.
type ConfigurableMessagingRepository struct {
	RequestFunc func(ctx context.Context, subject string, data []byte, timeout time.Duration) ([]byte, error)
}

// Request delegates to the configurable function.
func (m *ConfigurableMessagingRepository) Request(ctx context.Context, subject string, data []byte, timeout time.Duration) ([]byte, error) {
	if m.RequestFunc != nil {
		return m.RequestFunc(ctx, subject, data, timeout)
	}
	return []byte("project:a27394a3-7a6c-4d0f-9e0f-692d8753924f#auditor@user:auth0|alice\ttrue\ncommittee:b3c72e18-1a2b-4c3d-8e9f-123456789abc#writer@user:auth0|alice\tfalse"), nil
}

// Close is a no-op for testing.
func (m *ConfigurableMessagingRepository) Close() error {
	return nil
}

// HealthCheck always returns healthy for mock
func (m *ConfigurableMessagingRepository) HealthCheck(_ context.Context) error {
	return nil
}
