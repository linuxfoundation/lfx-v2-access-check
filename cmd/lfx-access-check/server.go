// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package main provides consolidated HTTP server implementation following LFX patterns.
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"sync"

	accesssvc "github.com/linuxfoundation/lfx-v2-access-check/gen/access_svc"
	accesssvcsvr "github.com/linuxfoundation/lfx-v2-access-check/gen/http/access_svc/server"
	"github.com/linuxfoundation/lfx-v2-access-check/internal/container"
	"github.com/linuxfoundation/lfx-v2-access-check/internal/infrastructure/config"
	"github.com/linuxfoundation/lfx-v2-access-check/internal/middleware"
	"github.com/linuxfoundation/lfx-v2-access-check/pkg/constants"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"goa.design/clue/debug"
	goahttp "goa.design/goa/v3/http"
)

// StartServer initializes and starts the access check server following LFX pattern
func StartServer(ctx context.Context, cfg *config.Config) error {

	// 1. Initialize dependencies using existing container
	cont, err := container.NewContainer(cfg)
	if err != nil {
		return err
	}

	// 2. Create GOA service implementation (now using unified service)
	accessSvc := cont.AccessService // Direct reference to unified service

	// 3. Create GOA endpoints
	endpoints := accesssvc.NewEndpoints(accessSvc)
	if cfg.Debug {
		endpoints.Use(debug.LogPayloads())
	}

	// 4. Start HTTP server with all functionality embedded
	slog.Info("Starting access check server", "host", cfg.Host, "port", cfg.Port)
	return handleHTTPServer(ctx, cfg, endpoints, cont)
}

// handleHTTPServer follows exact LFX query service pattern for server setup
func handleHTTPServer(ctx context.Context, cfg *config.Config, endpoints *accesssvc.Endpoints, cont *container.Container) error {
	// Build the service HTTP request multiplexer
	var mux goahttp.Muxer
	{
		mux = goahttp.NewMuxer()
		if cfg.Debug {
			// Mount debug endpoints when in debug mode
			debug.MountPprofHandlers(debug.Adapt(mux))
			debug.MountDebugLogEnabler(debug.Adapt(mux))
		}
	}

	// Create and mount GOA server
	var accessSvcServer *accesssvcsvr.Server
	{
		eh := errorHandler(ctx)
		
		// Setup file system for OpenAPI files
		koDatapath := os.Getenv("KO_DATA_PATH")
		if koDatapath == "" {
			koDatapath = "."
		}
		koHttpDir := http.Dir(koDatapath)
		
		accessSvcServer = accesssvcsvr.New(
			endpoints,
			mux,
			goahttp.RequestDecoder,
			goahttp.ResponseEncoder,
			eh,
			nil,       // formatter
			koHttpDir, // file system for openapi.json
			koHttpDir, // file system for openapi.yaml
			koHttpDir, // file system for openapi3.json
			koHttpDir, // file system for openapi3.yaml
		)
	}

	// Mount all endpoints
	accesssvcsvr.Mount(mux, accessSvcServer)

	// Add middleware stack (with request ID first)
	var handler http.Handler = mux
	{
		// Add request ID middleware first
		handler = middleware.RequestIDMiddleware()(handler)

		// Add debug middleware in debug mode
		if cfg.Debug {
			handler = debug.HTTP()(handler)
		}

		// Wrap the handler with OpenTelemetry instrumentation
		handler = otelhttp.NewHandler(handler, "access-check")
	}

	// Create HTTP server
	srv := &http.Server{
		Addr:              cfg.ServerAddress(),
		Handler:           handler,
		ReadHeaderTimeout: constants.DefaultReadHeaderTimeout,
		WriteTimeout:      constants.DefaultWriteTimeout,
		IdleTimeout:       constants.DefaultIdleTimeout,
	}

	// Start server with context-aware lifecycle management
	return runServerWithContext(ctx, srv, cont)
}

// errorHandler provides consistent error handling across all endpoints
func errorHandler(logCtx context.Context) func(context.Context, http.ResponseWriter, error) {
	return func(ctx context.Context, _ http.ResponseWriter, err error) {
		slog.ErrorContext(ctx, "HTTP error occurred",
			"error", err,
			"outer_context", logCtx,
			"request_context", ctx)
	}
}

// runServerWithContext manages HTTP server lifecycle with concurrent graceful shutdown
func runServerWithContext(ctx context.Context, srv *http.Server, cont *container.Container) error {
	// Channel to listen for server errors
	serverErr := make(chan error, 1)

	// Start server in goroutine
	go func() {
		slog.InfoContext(ctx, "Access check server listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	// Wait for context cancellation or server error
	select {
	case err := <-serverErr:
		return err
	case <-ctx.Done():
		slog.InfoContext(ctx, "Shutdown initiated via context cancellation")
	}

	// Enhanced concurrent graceful shutdown (from main.go.txt pattern)
	return performConcurrentShutdown(ctx, srv, cont)
}

// performConcurrentShutdown coordinates HTTP and NATS shutdown concurrently like main.go.txt
func performConcurrentShutdown(ctx context.Context, srv *http.Server, cont *container.Container) error {
	// Create wait group for concurrent shutdown coordination
	var gracefulCloseWG sync.WaitGroup

	// Add to wait group before starting goroutines to avoid race condition
	gracefulCloseWG.Add(2)

	// Start HTTP shutdown in goroutine
	go func() {
		defer gracefulCloseWG.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), constants.DefaultShutdownTimeout)
		defer cancel()

		slog.InfoContext(ctx, "Shutting down HTTP server gracefully...")
		if err := srv.Shutdown(shutdownCtx); err != nil {
			slog.ErrorContext(ctx, "Failed to shutdown HTTP server gracefully", "error", err)
		} else {
			slog.InfoContext(ctx, "HTTP server shutdown completed successfully")
		}
	}()

	// Start container/NATS shutdown in goroutine
	go func() {
		defer gracefulCloseWG.Done()

		slog.InfoContext(ctx, "Closing container resources (NATS draining)...")
		if err := cont.Close(); err != nil {
			slog.ErrorContext(ctx, "Failed to close container resources", "error", err)
		} else {
			slog.InfoContext(ctx, "Container resources closed successfully")
		}
	}()

	// Wait for both HTTP and NATS shutdown to complete
	gracefulCloseWG.Wait()
	slog.InfoContext(ctx, "Concurrent graceful shutdown completed")

	return nil
}
