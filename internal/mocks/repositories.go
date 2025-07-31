// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package mocks provides mock implementations for testing.
package mocks

import (
	"context"
	"time"

	"github.com/linuxfoundation/lfx-v2-access-check/internal/domain/contracts"
)

// MockAuthRepository provides a mock implementation of AuthRepository
type MockAuthRepository struct {
	ValidateTokenFunc func(ctx context.Context, token string) (*contracts.HeimdallClaims, error)
}

// NewMockAuthRepository creates a new mock auth repository
func NewMockAuthRepository() *MockAuthRepository {
	return &MockAuthRepository{}
}

// ValidateToken mocks token validation
func (m *MockAuthRepository) ValidateToken(ctx context.Context, token string) (*contracts.HeimdallClaims, error) {
	if m.ValidateTokenFunc != nil {
		return m.ValidateTokenFunc(ctx, token)
	}

	// Default mock behavior
	return &contracts.HeimdallClaims{
		Principal: "mock-user",
		Email:     "mock@example.com",
	}, nil
}

// HealthCheck mocks health check
func (m *MockAuthRepository) HealthCheck(ctx context.Context) error {
	// Default success behavior
	return nil
}

// MockMessagingRepository provides a mock implementation of MessagingRepository
type MockMessagingRepository struct {
	RequestFunc func(ctx context.Context, subject string, data []byte, timeout time.Duration) ([]byte, error)
	CloseFunc   func() error
}

// NewMockMessagingRepository creates a new mock messaging repository
func NewMockMessagingRepository() *MockMessagingRepository {
	return &MockMessagingRepository{}
}

// Request mocks NATS request
func (m *MockMessagingRepository) Request(ctx context.Context, subject string, data []byte, timeout time.Duration) ([]byte, error) {
	if m.RequestFunc != nil {
		return m.RequestFunc(ctx, subject, data, timeout)
	}
	return []byte("allow"), nil
}

// Close mocks connection closing
func (m *MockMessagingRepository) Close() error {
	if m.CloseFunc != nil {
		return m.CloseFunc()
	}
	return nil
}
