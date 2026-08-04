// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package service

import (
	"context"
	"testing"
	"time"
)

// BenchmarkBuildMessage measures the NATS message construction path.
func BenchmarkBuildMessage(b *testing.B) {
	client := NewAccessCheckClient(&mockMessagingRepository{})
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
		_ = client.buildMessage(principal, resources)
	}
}

// BenchmarkParseResponse measures the NATS response parsing path.
func BenchmarkParseResponse(b *testing.B) {
	client := NewAccessCheckClient(&mockMessagingRepository{})
	responseData := []byte("true\nfalse\ntrue\nfalse\ntrue\nfalse\ntrue\nfalse\ntrue\nfalse")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = client.parseResponse(responseData)
	}
}

// BenchmarkCheckAccess measures the full CheckAccess path with a mocked NATS response.
func BenchmarkCheckAccess(b *testing.B) {
	client := NewAccessCheckClient(&mockMessagingRepository{
		requestFunc: func(_ context.Context, _ string, _ []byte, _ time.Duration) ([]byte, error) {
			return []byte("true\nfalse\ntrue\nfalse\ntrue"), nil
		},
	})
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
		_, _ = client.CheckAccess(ctx, principal, resources)
	}
}
