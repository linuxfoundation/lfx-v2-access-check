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
	"github.com/linuxfoundation/lfx-v2-access-check/pkg/constants"
)

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
			return []byte(`{"results":["project:uuid1#writer@user:auth0|testuser","project:uuid2#auditor@user:auth0|testuser"]}`), nil
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

func TestMyGrants_NATSError(t *testing.T) {
	messagingRepo := &mockMessagingRepository{
		requestFunc: func(_ context.Context, _ string, _ []byte, _ time.Duration) ([]byte, error) {
			return nil, errors.New("nats timeout")
		},
	}
	svc := NewAccessService(&mockAuthRepository{}, messagingRepo)

	_, err := svc.MyGrants(contextWithClaims("auth0|user"), &accesssvc.MyGrantsPayload{
		BearerToken: "tok",
		Version:     "1",
		ObjectType:  "project",
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestMyGrants_FgaSyncError(t *testing.T) {
	messagingRepo := &mockMessagingRepository{
		requestFunc: func(_ context.Context, _ string, _ []byte, _ time.Duration) ([]byte, error) {
			return []byte(`{"error":"failed to read tuples: store not found"}`), nil
		},
	}
	svc := NewAccessService(&mockAuthRepository{}, messagingRepo)

	_, err := svc.MyGrants(contextWithClaims("auth0|user"), &accesssvc.MyGrantsPayload{
		BearerToken: "tok",
		Version:     "1",
		ObjectType:  "project",
	})

	if err == nil {
		t.Fatal("expected error, got nil")
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

func TestMyGrants_MalformedResponse(t *testing.T) {
	messagingRepo := &mockMessagingRepository{
		requestFunc: func(_ context.Context, _ string, _ []byte, _ time.Duration) ([]byte, error) {
			return []byte(`not valid json`), nil
		},
	}
	svc := NewAccessService(&mockAuthRepository{}, messagingRepo)

	_, err := svc.MyGrants(contextWithClaims("auth0|user"), &accesssvc.MyGrantsPayload{
		BearerToken: "tok",
		Version:     "1",
		ObjectType:  "project",
	})

	if err == nil {
		t.Fatal("expected error for malformed response, got nil")
	}
}
