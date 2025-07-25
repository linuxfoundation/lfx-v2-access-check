// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT
package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/linuxfoundation/lfx-v2-access-check/pkg/constants"
)

func TestRequestIDMiddleware_GeneratesNewRequestID(t *testing.T) {
	// Create middleware
	middleware := RequestIDMiddleware()

	// Create a test handler that checks the context
	var capturedRequestID string
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedRequestID = r.Context().Value(requestIDKey).(string)
		w.WriteHeader(http.StatusOK)
	})

	// Wrap the test handler with middleware
	wrappedHandler := middleware(testHandler)

	// Create request without request ID header
	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	// Execute request
	wrappedHandler.ServeHTTP(rec, req)

	// Check that a request ID was generated
	if capturedRequestID == "" {
		t.Error("Expected request ID to be generated, got empty string")
	}

	// Check that request ID is a valid UUID format (36 characters with hyphens)
	if len(capturedRequestID) != 36 {
		t.Errorf("Expected request ID to be UUID format (36 chars), got %d chars: %s", len(capturedRequestID), capturedRequestID)
	}

	// Check that request ID contains hyphens in correct positions (UUID format)
	if !strings.Contains(capturedRequestID, "-") {
		t.Errorf("Expected request ID to be UUID format with hyphens, got: %s", capturedRequestID)
	}

	// Check that response header contains the request ID
	responseRequestID := rec.Header().Get(constants.RequestIDHeader)
	if responseRequestID != capturedRequestID {
		t.Errorf("Expected response header request ID to match context, got '%s' vs '%s'", responseRequestID, capturedRequestID)
	}
}

func TestRequestIDMiddleware_UsesExistingRequestID(t *testing.T) {
	// Create middleware
	middleware := RequestIDMiddleware()

	// Pre-defined request ID
	predefinedRequestID := "test-request-id-123"

	// Create a test handler that checks the context
	var capturedRequestID string
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedRequestID = r.Context().Value(requestIDKey).(string)
		w.WriteHeader(http.StatusOK)
	})

	// Wrap the test handler with middleware
	wrappedHandler := middleware(testHandler)

	// Create request with request ID header
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set(constants.RequestIDHeader, predefinedRequestID)
	rec := httptest.NewRecorder()

	// Execute request
	wrappedHandler.ServeHTTP(rec, req)

	// Check that the predefined request ID was used
	if capturedRequestID != predefinedRequestID {
		t.Errorf("Expected request ID to be '%s', got '%s'", predefinedRequestID, capturedRequestID)
	}

	// Check that response header contains the same request ID
	responseRequestID := rec.Header().Get(constants.RequestIDHeader)
	if responseRequestID != predefinedRequestID {
		t.Errorf("Expected response header request ID to be '%s', got '%s'", predefinedRequestID, responseRequestID)
	}
}

func TestRequestIDMiddleware_EmptyRequestIDHeader(t *testing.T) {
	// Create middleware
	middleware := RequestIDMiddleware()

	// Create a test handler that checks the context
	var capturedRequestID string
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedRequestID = r.Context().Value(requestIDKey).(string)
		w.WriteHeader(http.StatusOK)
	})

	// Wrap the test handler with middleware
	wrappedHandler := middleware(testHandler)

	// Create request with empty request ID header
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set(constants.RequestIDHeader, "")
	rec := httptest.NewRecorder()

	// Execute request
	wrappedHandler.ServeHTTP(rec, req)

	// Check that a new request ID was generated (not empty)
	if capturedRequestID == "" {
		t.Error("Expected request ID to be generated for empty header, got empty string")
	}

	// Check that request ID is a valid UUID format
	if len(capturedRequestID) != 36 {
		t.Errorf("Expected request ID to be UUID format, got: %s", capturedRequestID)
	}
}

func TestRequestIDMiddleware_MultipleRequests(t *testing.T) {
	// Create middleware
	middleware := RequestIDMiddleware()

	// Store request IDs from multiple requests
	var requestIDs []string
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Context().Value(requestIDKey).(string)
		requestIDs = append(requestIDs, requestID)
		w.WriteHeader(http.StatusOK)
	})

	// Wrap the test handler with middleware
	wrappedHandler := middleware(testHandler)

	// Make multiple requests
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		rec := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(rec, req)
	}

	// Check that all request IDs are unique
	if len(requestIDs) != 5 {
		t.Errorf("Expected 5 request IDs, got %d", len(requestIDs))
	}

	for i, id := range requestIDs {
		for j, otherID := range requestIDs {
			if i != j && id == otherID {
				t.Errorf("Found duplicate request IDs at positions %d and %d: %s", i, j, id)
			}
		}
	}

	// Check that all request IDs are valid UUIDs
	for i, id := range requestIDs {
		if len(id) != 36 {
			t.Errorf("Request ID %d is not valid UUID format: %s", i, id)
		}
	}
}

func TestRequestIDMiddleware_ContextPropagation(t *testing.T) {
	// Create middleware
	middleware := RequestIDMiddleware()

	// Create a nested handler that checks context propagation
	var outerRequestID, innerRequestID string
	innerHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		innerRequestID = r.Context().Value(requestIDKey).(string)
		w.WriteHeader(http.StatusOK)
	})

	outerHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		outerRequestID = r.Context().Value(requestIDKey).(string)
		innerHandler.ServeHTTP(w, r)
	})

	// Wrap the outer handler with middleware
	wrappedHandler := middleware(outerHandler)

	// Create request
	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	// Execute request
	wrappedHandler.ServeHTTP(rec, req)

	// Check that request ID is consistent across nested handlers
	if outerRequestID != innerRequestID {
		t.Errorf("Request ID not consistent: outer='%s', inner='%s'", outerRequestID, innerRequestID)
	}

	if outerRequestID == "" {
		t.Error("Expected request ID to be set in context")
	}
}

func TestRequestIDMiddleware_ResponseHeaders(t *testing.T) {
	// Create middleware
	middleware := RequestIDMiddleware()

	// Simple test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Wrap the test handler with middleware
	wrappedHandler := middleware(testHandler)

	testCases := []struct {
		name           string
		inputRequestID string
		expectGenerate bool
	}{
		{
			name:           "no_request_id_header",
			inputRequestID: "",
			expectGenerate: true,
		},
		{
			name:           "with_request_id_header",
			inputRequestID: "custom-request-id-456",
			expectGenerate: false,
		},
		{
			name:           "uuid_request_id",
			inputRequestID: "550e8400-e29b-41d4-a716-446655440000",
			expectGenerate: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			if tc.inputRequestID != "" {
				req.Header.Set(constants.RequestIDHeader, tc.inputRequestID)
			}
			rec := httptest.NewRecorder()

			wrappedHandler.ServeHTTP(rec, req)

			responseRequestID := rec.Header().Get(constants.RequestIDHeader)

			if tc.expectGenerate {
				// Should have generated a new UUID
				if responseRequestID == "" {
					t.Error("Expected generated request ID in response header")
				}
				if len(responseRequestID) != 36 {
					t.Errorf("Expected UUID format in response header, got: %s", responseRequestID)
				}
			} else {
				// Should use the provided request ID
				if responseRequestID != tc.inputRequestID {
					t.Errorf("Expected response header to contain '%s', got '%s'", tc.inputRequestID, responseRequestID)
				}
			}
		})
	}
}

func TestGenerateRequestID(t *testing.T) {
	// Test the generateRequestID function directly
	id1 := generateRequestID()
	id2 := generateRequestID()

	// Check that IDs are not empty
	if id1 == "" || id2 == "" {
		t.Error("Generated request IDs should not be empty")
	}

	// Check that IDs are unique
	if id1 == id2 {
		t.Errorf("Generated request IDs should be unique, both were: %s", id1)
	}

	// Check UUID format (36 characters)
	if len(id1) != 36 || len(id2) != 36 {
		t.Errorf("Generated request IDs should be 36 characters, got %d and %d", len(id1), len(id2))
	}

	// Check UUID format (contains hyphens)
	if !strings.Contains(id1, "-") || !strings.Contains(id2, "-") {
		t.Error("Generated request IDs should contain hyphens (UUID format)")
	}
}

func TestRequestIDMiddleware_HTTPMethods(t *testing.T) {
	// Create middleware
	middleware := RequestIDMiddleware()

	// Test handler that captures method and request ID
	var results []struct {
		method    string
		requestID string
	}

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Context().Value(requestIDKey).(string)
		results = append(results, struct {
			method    string
			requestID string
		}{r.Method, requestID})
		w.WriteHeader(http.StatusOK)
	})

	// Wrap the test handler with middleware
	wrappedHandler := middleware(testHandler)

	// Test different HTTP methods
	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"}

	for _, method := range methods {
		req := httptest.NewRequest(method, "/test", nil)
		rec := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(rec, req)
	}

	// Check that request IDs were generated for all methods
	if len(results) != len(methods) {
		t.Errorf("Expected %d results, got %d", len(methods), len(results))
	}

	for i, result := range results {
		if result.requestID == "" {
			t.Errorf("Request ID should be generated for method %s", result.method)
		}
		if result.method != methods[i] {
			t.Errorf("Expected method %s, got %s", methods[i], result.method)
		}
	}
}

func TestRequestIDMiddleware_ErrorHandling(t *testing.T) {
	// Create middleware
	middleware := RequestIDMiddleware()

	// Test handler that returns an error
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Context().Value(requestIDKey).(string)
		if requestID == "" {
			t.Error("Request ID should be available even when handler returns error")
		}
		w.WriteHeader(http.StatusInternalServerError)
	})

	// Wrap the test handler with middleware
	wrappedHandler := middleware(testHandler)

	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	// Execute request
	wrappedHandler.ServeHTTP(rec, req)

	// Check that request ID is still set in response header even with error
	responseRequestID := rec.Header().Get(constants.RequestIDHeader)
	if responseRequestID == "" {
		t.Error("Request ID should be set in response header even when handler returns error")
	}

	// Check status code
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("Expected status code %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}
