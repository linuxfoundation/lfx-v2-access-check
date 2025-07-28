// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT
package integration

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/linuxfoundation/lfx-v2-access-check/internal/service"

	accesssvc "github.com/linuxfoundation/lfx-v2-access-check/gen/access_svc"
	accesssvcsvr "github.com/linuxfoundation/lfx-v2-access-check/gen/http/access_svc/server"
	goahttp "goa.design/goa/v3/http"
)

func TestHealthEndpoints(t *testing.T) {
	// Create test service with mock dependencies
	mockAuthRepo := &MockAuthRepository{}
	mockMessagingRepo := &MockMessagingRepository{}
	accessService := service.NewAccessService(mockAuthRepo, mockMessagingRepo)

	// Create endpoints from unified service
	endpoints := accesssvc.NewEndpoints(accessService)

	// Create HTTP server
	mux := goahttp.NewMuxer()
	server := accesssvcsvr.New(endpoints, mux,
		goahttp.RequestDecoder,
		goahttp.ResponseEncoder,
		func(_ context.Context, _ http.ResponseWriter, err error) {
			t.Logf("Error handler called: %v", err)
		},
		nil, nil)

	accesssvcsvr.Mount(mux, server)

	// Test server
	testServer := httptest.NewServer(mux)
	defer testServer.Close()

	tests := []struct {
		name           string
		endpoint       string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Liveness check",
			endpoint:       "/livez",
			expectedStatus: http.StatusOK,
			expectedBody:   "OK",
		},
		{
			name:           "Readiness check",
			endpoint:       "/readyz",
			expectedStatus: http.StatusOK,
			expectedBody:   "OK",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := http.Get(testServer.URL + tt.endpoint)
			if err != nil {
				t.Fatalf("Failed to make request: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("Failed to read response body: %v", err)
			}

			if string(body) != tt.expectedBody {
				t.Errorf("Expected body %q, got %q", tt.expectedBody, string(body))
			}
		})
	}
}
