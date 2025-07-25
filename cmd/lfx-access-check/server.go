// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package main provides consolidated HTTP server implementation following LFX patterns.
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	accesssvc "github.com/linuxfoundation/lfx-v2-access-check/gen/access_svc"
	accesssvcsvr "github.com/linuxfoundation/lfx-v2-access-check/gen/http/access_svc/server"
	"github.com/linuxfoundation/lfx-v2-access-check/internal/container"
	"github.com/linuxfoundation/lfx-v2-access-check/internal/infrastructure/config"
	"github.com/linuxfoundation/lfx-v2-access-check/internal/middleware"

	"goa.design/clue/debug"
	goahttp "goa.design/goa/v3/http"
)

// StartServer initializes and starts the access check server following LFX pattern
func StartServer(cfg *config.Config) error {
	ctx := context.Background()

	// 1. Initialize dependencies using existing container
	cont, err := container.NewContainer(cfg)
	if err != nil {
		return err
	}
	defer func() {
		if err := cont.Close(); err != nil {
			slog.ErrorContext(ctx, "Failed to close container", "error", err)
		}
	}()

	// 2. Create GOA service implementation (now using unified service)
	accessSvc := cont.AccessService // Direct reference to unified service

	// 3. Create GOA endpoints
	endpoints := accesssvc.NewEndpoints(accessSvc)
	if cfg.Debug {
		endpoints.Use(debug.LogPayloads())
	}

	// 4. Start HTTP server with all functionality embedded
	slog.Info("Starting access check server", "host", cfg.Host, "port", cfg.Port)
	return handleHTTPServer(ctx, cfg, endpoints)
}

// handleHTTPServer follows exact LFX query service pattern for server setup
func handleHTTPServer(ctx context.Context, cfg *config.Config, endpoints *accesssvc.Endpoints) error {
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
		accessSvcServer = accesssvcsvr.New(
			endpoints,
			mux,
			goahttp.RequestDecoder,
			goahttp.ResponseEncoder,
			eh,
			nil, // formatter
			nil, // file system for OpenAPI
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
	}

	// Create HTTP server
	srv := &http.Server{
		Addr:              cfg.Host + ":" + cfg.Port,
		Handler:           handler,
		ReadHeaderTimeout: time.Second * 60,
		WriteTimeout:      time.Second * 60,
		IdleTimeout:       time.Second * 90,
	}

	// Start server with graceful shutdown
	return startServerWithGracefulShutdown(ctx, srv)
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

// startServerWithGracefulShutdown manages server lifecycle with graceful shutdown
func startServerWithGracefulShutdown(ctx context.Context, srv *http.Server) error {
	// Channel to listen for interrupt/terminate signals
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Channel to listen for server errors
	serverErr := make(chan error, 1)

	// Start server in goroutine
	go func() {
		slog.Info("Access check server listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	// Wait for interrupt signal or server error
	select {
	case err := <-serverErr:
		return err
	case sig := <-stop:
		slog.Info("Shutdown signal received", "signal", sig.String())
	}

	// Graceful shutdown with timeout
	shutdownCtx, cancel := context.WithTimeout(ctx, 25*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.ErrorContext(ctx, "Failed to shutdown server gracefully", "error", err)
		return err
	}

	slog.Info("Access check server shutdown complete")
	return nil
}
