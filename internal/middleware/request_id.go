// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package middleware provides HTTP middleware for the LFX Access Check service.
package middleware

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/linuxfoundation/lfx-v2-access-check/pkg/constants"
	"github.com/linuxfoundation/lfx-v2-access-check/pkg/log"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

// requestIDKey is the context key for storing request IDs
const requestIDKey contextKey = "request-id"

// RequestIDMiddleware creates a middleware that adds a request ID to the context
func RequestIDMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Try to get request ID from header first
			requestID := r.Header.Get(constants.RequestIDHeader)

			// If no request ID in header, generate a new one
			if requestID == "" {
				requestID = generateRequestID()
			}

			// Add request ID to response header
			w.Header().Set(constants.RequestIDHeader, requestID)

			// Add request ID to context
			ctx := context.WithValue(r.Context(), requestIDKey, requestID)

			// Log the request ID using the context-aware logger
			// This allows the request ID to be included in all logs for this request
			ctx = log.AppendCtx(ctx, slog.String(constants.RequestIDHeader, requestID))

			// Create a new request with the updated context
			r = r.WithContext(ctx)

			// Call the next handler
			next.ServeHTTP(w, r)
		})
	}
}

// generateRequestID generates a new unique request ID
func generateRequestID() string {
	return uuid.New().String()
}
