// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/linuxfoundation/lfx-v2-access-check/pkg/constants"
)

// ===== AccessCheckClient unit tests =====
// These tests exercise the NATS protocol logic in isolation.
// No Goa types are imported here; only the domain contracts and constants.

func newTestClient(requestFunc func(ctx context.Context, subject string, data []byte, timeout time.Duration) ([]byte, error)) *AccessCheckClient {
	return NewAccessCheckClient(&mockMessagingRepository{requestFunc: requestFunc})
}

// ----- CheckAccess -----

func TestAccessCheckClient_CheckAccess_EmptyPrincipal(t *testing.T) {
	client := NewAccessCheckClient(&mockMessagingRepository{})

	_, err := client.CheckAccess(context.Background(), "", []string{"resource1"})
	if err == nil {
		t.Fatal("CheckAccess should fail with empty principal")
	}
	if !errors.Is(err, constants.ErrPrincipalRequired) {
		t.Errorf("expected ErrPrincipalRequired, got %v", err)
	}
}

func TestAccessCheckClient_CheckAccess_EmptyResources(t *testing.T) {
	client := NewAccessCheckClient(&mockMessagingRepository{})

	result, err := client.CheckAccess(context.Background(), "test-user", []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty result for empty resources, got %d items", len(result))
	}
}

func TestAccessCheckClient_CheckAccess_UnexpectedResponse(t *testing.T) {
	client := newTestClient(func(_ context.Context, _ string, _ []byte, _ time.Duration) ([]byte, error) {
		return []byte("error message here"), nil
	})

	_, err := client.CheckAccess(context.Background(), "test-user", []string{"resource1"})
	if err == nil {
		t.Fatal("CheckAccess should fail on space-containing response")
	}
	if !errors.Is(err, constants.ErrUnexpectedResponse) {
		t.Errorf("expected ErrUnexpectedResponse, got %v", err)
	}
}

func TestAccessCheckClient_CheckAccess_NATSFailure(t *testing.T) {
	client := newTestClient(func(_ context.Context, _ string, _ []byte, _ time.Duration) ([]byte, error) {
		return nil, errors.New("NATS connection failed")
	})

	_, err := client.CheckAccess(context.Background(), "test-user", []string{"resource1"})
	if err == nil {
		t.Fatal("expected error on NATS failure")
	}
	if !strings.Contains(err.Error(), "NATS request to subject") {
		t.Errorf("expected NATS transport error, got %v", err)
	}
}

func TestAccessCheckClient_CheckAccess_CorrectSubject(t *testing.T) {
	var gotSubject string
	client := newTestClient(func(_ context.Context, subject string, _ []byte, _ time.Duration) ([]byte, error) {
		gotSubject = subject
		return []byte("true\nfalse"), nil
	})

	_, err := client.CheckAccess(context.Background(), "alice", []string{"project:abc#viewer"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotSubject != constants.AccessCheckSubject {
		t.Errorf("expected subject %q, got %q", constants.AccessCheckSubject, gotSubject)
	}
}

// ----- ReadTuples -----

func TestAccessCheckClient_ReadTuples_Success(t *testing.T) {
	client := newTestClient(func(_ context.Context, subject string, _ []byte, _ time.Duration) ([]byte, error) {
		if subject != constants.ReadTuplesSubject {
			t.Errorf("unexpected NATS subject: %s", subject)
		}
		return []byte(`{"results":["project:abc#auditor@user:alice","committee:xyz#writer@user:alice"]}`), nil
	})

	results, err := client.ReadTuples(context.Background(), "alice", "project")
	if err != nil {
		t.Fatalf("ReadTuples failed: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
}

func TestAccessCheckClient_ReadTuples_NilResultsNormalized(t *testing.T) {
	client := newTestClient(func(_ context.Context, _ string, _ []byte, _ time.Duration) ([]byte, error) {
		return []byte(`{}`), nil
	})

	results, err := client.ReadTuples(context.Background(), "alice", "project")
	if err != nil {
		t.Fatalf("ReadTuples failed: %v", err)
	}
	if results == nil {
		t.Error("expected empty slice, got nil")
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

func TestAccessCheckClient_ReadTuples_NATSFailure(t *testing.T) {
	client := newTestClient(func(_ context.Context, _ string, _ []byte, _ time.Duration) ([]byte, error) {
		return nil, errors.New("nats timeout")
	})

	_, err := client.ReadTuples(context.Background(), "alice", "project")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "NATS request to subject") {
		t.Errorf("expected NATS transport error, got %v", err)
	}
}

func TestAccessCheckClient_ReadTuples_UnmarshalFailure(t *testing.T) {
	client := newTestClient(func(_ context.Context, _ string, _ []byte, _ time.Duration) ([]byte, error) {
		return []byte(`not valid json`), nil
	})

	_, err := client.ReadTuples(context.Background(), "alice", "project")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, constants.ErrUnexpectedResponse) {
		t.Errorf("expected ErrUnexpectedResponse, got %v", err)
	}
}

func TestAccessCheckClient_ReadTuples_BackendError(t *testing.T) {
	client := newTestClient(func(_ context.Context, _ string, _ []byte, _ time.Duration) ([]byte, error) {
		return []byte(`{"error":"store not found"}`), nil
	})

	_, err := client.ReadTuples(context.Background(), "alice", "project")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "FGA error") {
		t.Errorf("expected FGA error message, got %v", err)
	}
	if strings.Contains(err.Error(), "NATS request") {
		t.Errorf("backend error should not look like a NATS transport error, got %v", err)
	}
}

// ----- HealthCheck -----

func TestAccessCheckClient_HealthCheck_NilRepo(t *testing.T) {
	client := NewAccessCheckClient(nil)

	err := client.HealthCheck(context.Background())
	if err == nil {
		t.Fatal("expected error for nil messaging repo")
	}
	if !errors.Is(err, constants.ErrMessagingRepoNotInit) {
		t.Errorf("expected ErrMessagingRepoNotInit, got %v", err)
	}
}

func TestAccessCheckClient_HealthCheck_Healthy(t *testing.T) {
	client := NewAccessCheckClient(&mockMessagingRepository{})

	if err := client.HealthCheck(context.Background()); err != nil {
		t.Errorf("expected healthy, got %v", err)
	}
}

// ----- buildMessage -----

func TestAccessCheckClient_BuildMessage(t *testing.T) {
	client := NewAccessCheckClient(&mockMessagingRepository{})

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

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := client.buildMessage(tc.principal, tc.resources)
			if result != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, result)
			}
		})
	}
}

// ----- parseResponse -----

func TestAccessCheckClient_ParseResponse(t *testing.T) {
	client := NewAccessCheckClient(&mockMessagingRepository{})

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
		},
		{
			name:         "empty response",
			responseData: []byte(""),
			expected:     []string{},
		},
		{
			name:         "response with empty lines",
			responseData: []byte("true\n\nfalse\n"),
			expected:     []string{"true", "false"},
		},
		{
			name:         "response with spaces triggers error",
			responseData: []byte("error message here"),
			expectError:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := client.parseResponse(tc.responseData)

			if tc.expectError {
				if err == nil {
					t.Fatal("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(result) != len(tc.expected) {
				t.Fatalf("expected %d results, got %d", len(tc.expected), len(result))
			}
			for i, want := range tc.expected {
				if result[i] != want {
					t.Errorf("result[%d]: expected %q, got %q", i, want, result[i])
				}
			}
		})
	}
}
