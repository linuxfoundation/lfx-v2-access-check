// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT
package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/linuxfoundation/lfx-v2-access-check/internal/service"

	accesssvc "github.com/linuxfoundation/lfx-v2-access-check/gen/access_svc"
	accesssvcsvr "github.com/linuxfoundation/lfx-v2-access-check/gen/http/access_svc/server"
	goahttp "goa.design/goa/v3/http"
)

func TestAccessCheckEndpoint(t *testing.T) {
	// Create test service with mock dependencies
	mockAuthRepo := &MockAuthRepository{}
	mockMessagingRepo := &MockMessagingRepository{}
	accessService := service.NewAccessService(mockAuthRepo, mockMessagingRepo)

	// Create endpoints
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
		method         string
		url            string
		headers        map[string]string
		body           interface{}
		expectedStatus int
		expectedError  bool
	}{
		{
			name:   "Valid access check request",
			method: "POST",
			url:    "/access-check?v=1",
			headers: map[string]string{
				"Authorization": "Bearer valid-token",
				"Content-Type":  "application/json",
			},
			body: map[string]interface{}{
				"requests": []string{"project:123:read", "committee:456:write"},
			},
			expectedStatus: http.StatusOK,
			expectedError:  false,
		},
		{
			name:   "Missing authorization header",
			method: "POST",
			url:    "/access-check?v=1",
			headers: map[string]string{
				"Content-Type": "application/json",
			},
			body: map[string]interface{}{
				"requests": []string{"project:123:read"},
			},
			expectedStatus: http.StatusBadRequest, // GOA validates required headers first
			expectedError:  true,
		},
		{
			name:   "Invalid authorization token",
			method: "POST",
			url:    "/access-check?v=1",
			headers: map[string]string{
				"Authorization": "Bearer invalid-token",
				"Content-Type":  "application/json",
			},
			body: map[string]interface{}{
				"requests": []string{"project:123:read"},
			},
			expectedStatus: http.StatusUnauthorized,
			expectedError:  true,
		},
		{
			name:   "Missing version parameter",
			method: "POST",
			url:    "/access-check",
			headers: map[string]string{
				"Authorization": "Bearer valid-token",
				"Content-Type":  "application/json",
			},
			body: map[string]interface{}{
				"requests": []string{"project:123:read"},
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  true,
		},
		{
			name:   "Empty requests array",
			method: "POST",
			url:    "/access-check?v=1",
			headers: map[string]string{
				"Authorization": "Bearer valid-token",
				"Content-Type":  "application/json",
			},
			body: map[string]interface{}{
				"requests": []string{},
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Prepare request body
			var bodyReader *bytes.Reader
			if tt.body != nil {
				bodyBytes, err := json.Marshal(tt.body)
				if err != nil {
					t.Fatalf("Failed to marshal request body: %v", err)
				}
				bodyReader = bytes.NewReader(bodyBytes)
			} else {
				bodyReader = bytes.NewReader([]byte{})
			}

			// Create request
			req, err := http.NewRequest(tt.method, testServer.URL+tt.url, bodyReader)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			// Set headers
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			// Make request
			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("Failed to make request: %v", err)
			}
			defer resp.Body.Close()

			// Check status code
			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}

			// For successful requests, validate response structure
			if !tt.expectedError && resp.StatusCode == http.StatusOK {
				var response map[string]interface{}
				if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}

				results, ok := response["results"]
				if !ok {
					t.Error("Response missing 'results' field")
				}

				resultsArray, ok := results.([]interface{})
				if !ok {
					t.Error("Results field is not an array")
				}

				if len(resultsArray) == 0 {
					t.Error("Results array is empty")
				}

				t.Logf("Successful response: %+v", response)
			}
		})
	}
}
