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
	return []byte(`["allow","deny"]`), nil
}

// Close closes the mock messaging connection (no-op for testing)
func (m *MockMessagingRepository) Close() error {
	return nil
}

// HealthCheck always returns healthy for mock
func (m *MockMessagingRepository) HealthCheck(_ context.Context) error {
	return nil
}
