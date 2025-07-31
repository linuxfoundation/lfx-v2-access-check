// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT
package messaging

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestNewMessagingRepository_InvalidURL(t *testing.T) {
	natsURL := "invalid://url"
	_, err := NewMessagingRepository(natsURL)
	if err == nil {
		t.Error("Expected error for invalid URL, got none")
	}
	t.Logf("Got expected error: %v", err)
}

func TestNewMessagingRepository_UnreachableHost(t *testing.T) {
	natsURL := "nats://non-existent-host:4222"
	_, err := NewMessagingRepository(natsURL)
	if err == nil {
		t.Error("Expected error for unreachable host, got none")
	}
	t.Logf("Got expected error: %v", err)
}

func TestNewMessagingRepository_InvalidPort(t *testing.T) {
	natsURL := "nats://localhost:99999"
	_, err := NewMessagingRepository(natsURL)
	if err == nil {
		t.Error("Expected error for invalid port, got none")
	}
	t.Logf("Got expected error: %v", err)
}

func TestNewMessagingRepository_EmptyURL(t *testing.T) {
	natsURL := ""
	_, err := NewMessagingRepository(natsURL)
	if err == nil {
		t.Error("Expected error for empty URL, got none")
	}
	t.Logf("Got expected error: %v", err)
}

func TestNewMessagingRepository_LocalhostNoServer(t *testing.T) {
	natsURL := "nats://localhost:4222"
	_, err := NewMessagingRepository(natsURL)
	if err == nil {
		t.Error("Expected error when no NATS server is running, got none")
	}
	t.Logf("Got expected connection error: %v", err)
}

func TestMessagingRepository_Close_NilConnection(_ *testing.T) {
	repo := &messagingRepository{conn: nil}
	_ = repo.Close() // Should not panic
}

func TestNewMessagingRepository_URLFormats(t *testing.T) {
	testCases := []struct {
		name string
		url  string
		desc string
	}{
		{"valid_format_but_no_server", "nats://localhost:4222", "should fail - no server"},
		{"missing_protocol", "localhost:4222", "should fail - missing nats://"},
		{"wrong_protocol", "http://localhost:4222", "should fail - wrong protocol"},
		{"invalid_host", "nats://999.999.999.999:4222", "should fail - invalid IP"},
		{"no_port", "nats://localhost", "should fail - no port specified"},
		{"invalid_characters", "nats://local host:4222", "should fail - space in hostname"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewMessagingRepository(tc.url)
			if err == nil {
				t.Errorf("Expected error for %s, got none", tc.desc)
			}
			t.Logf("%s: %v", tc.desc, err)
		})
	}
}

func TestMessagingRepository_Request_WithNilConnection(t *testing.T) {
	repo := &messagingRepository{conn: nil}
	ctx := context.Background()
	_, err := repo.Request(ctx, "test.subject", []byte("test data"), 1*time.Second)
	if err == nil {
		t.Error("Expected error with nil connection, got none")
	}
	t.Logf("Got expected error with nil connection: %v", err)
}

func TestMessagingRepository_Timeout(t *testing.T) {
	natsURL := "nats://127.0.0.1:4223"
	_, err := NewMessagingRepository(natsURL)
	if err == nil {
		t.Error("Expected connection error, got none")
	}
	t.Logf("Got expected connection error: %v", err)
}

func TestMessagingRepository_Structure(t *testing.T) {
	repo := &messagingRepository{}
	// Test repository structure
	if repo.conn == nil {
		t.Log("conn is nil as expected for uninitialized repository")
	}
}

func TestMessagingRepository_MultipleConnections(t *testing.T) {
	testCases := []string{
		"nats://127.0.0.1:4222",
		"nats://localhost:4223",
		"nats://127.0.0.1:4224",
	}

	for _, url := range testCases {
		t.Run(fmt.Sprintf("url_%s", url), func(t *testing.T) {
			repo, err := NewMessagingRepository(url)
			if err == nil {
				t.Errorf("Expected error for unavailable NATS server at %s, got none", url)
			}
			if repo != nil {
				t.Errorf("Expected nil repository when connection fails, got %v", repo)
			}
		})
	}
}

func TestMessagingRepository_ErrorHandling(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "unsupported_scheme",
			url:     "tcp://localhost:4222",
			wantErr: true,
			errMsg:  "no servers available",
		},
		{
			name:    "hostname_with_spaces",
			url:     "nats://bad host:4222",
			wantErr: true,
			errMsg:  "invalid character",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, err := NewMessagingRepository(tt.url)
			if !tt.wantErr {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
				if repo == nil {
					t.Error("Expected repository, got nil")
				}
			} else {
				if err == nil {
					t.Error("Expected error, got none")
				}
				if repo != nil {
					t.Errorf("Expected nil repository, got %v", repo)
				}
				if !strings.Contains(err.Error(), tt.errMsg) && !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tt.errMsg)) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errMsg, err.Error())
				}
			}
		})
	}
}

func TestMessagingRepository_Request_EdgeCases(t *testing.T) {
	repo := &messagingRepository{conn: nil}
	ctx := context.Background()

	tests := []struct {
		name    string
		subject string
		data    []byte
		wantErr bool
	}{
		{
			name:    "empty_subject",
			subject: "",
			data:    []byte("test"),
			wantErr: true,
		},
		{
			name:    "nil_data",
			subject: "test.subject",
			data:    nil,
			wantErr: true,
		},
		{
			name:    "empty_data",
			subject: "test.subject",
			data:    []byte{},
			wantErr: true,
		},
		{
			name:    "very_long_subject",
			subject: strings.Repeat("very.long.subject.", 100),
			data:    []byte("test"),
			wantErr: true,
		},
		{
			name:    "large_data",
			subject: "test.subject",
			data:    make([]byte, 1024*1024),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := repo.Request(ctx, tt.subject, tt.data, 1*time.Second)
			if tt.wantErr && err == nil {
				t.Error("Expected error, got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Expected no error, got %v", err)
			}
		})
	}
}
