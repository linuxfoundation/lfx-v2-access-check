// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package service

import (
	"context"
	"testing"
	"time"
)

// Simple benchmark for the refactored buildAccessCheckMessage method
func BenchmarkBuildAccessCheckMessage(b *testing.B) {
	authRepo := &mockAuthRepository{}
	messagingRepo := &mockMessagingRepository{}
	service := NewAccessService(authRepo, messagingRepo)

	principal := "test-user-with-long-name"
	resources := []string{
		"repository/project1",
		"repository/project2",
		"repository/project3",
		"repository/project4",
		"repository/project5",
		"organization/org1",
		"organization/org2",
		"team/team1",
		"team/team2",
		"project/project1",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = service.buildAccessCheckMessage(principal, resources)
	}
}

// Benchmark for the parseAccessCheckResponse method
func BenchmarkParseAccessCheckResponse(b *testing.B) {
	authRepo := &mockAuthRepository{}
	messagingRepo := &mockMessagingRepository{}
	service := NewAccessService(authRepo, messagingRepo)

	ctx := context.Background()
	responseData := []byte("true\nfalse\ntrue\nfalse\ntrue\nfalse\ntrue\nfalse\ntrue\nfalse")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.parseAccessCheckResponse(ctx, responseData)
	}
}

// End-to-end benchmark for the entire performAccessCheck flow (mocked NATS)
func BenchmarkPerformAccessCheck(b *testing.B) {
	authRepo := &mockAuthRepository{}
	messagingRepo := &mockMessagingRepository{
		requestFunc: func(_ context.Context, _ string, _ []byte, _ time.Duration) ([]byte, error) {
			return []byte("true\nfalse\ntrue\nfalse\ntrue"), nil
		},
	}
	service := NewAccessService(authRepo, messagingRepo)

	ctx := context.Background()
	principal := "test-user-with-long-name"
	resources := []string{
		"repository/project1",
		"repository/project2",
		"repository/project3",
		"repository/project4",
		"repository/project5",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.performAccessCheck(ctx, principal, resources)
	}
}
