// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// The access-check service.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/linuxfoundation/lfx-v2-access-check/internal/infrastructure/config"
	"github.com/linuxfoundation/lfx-v2-access-check/pkg/log"
	"github.com/linuxfoundation/lfx-v2-access-check/pkg/utils"
)

// Build-time variables set via ldflags
var (
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

const gracefulShutdownSeconds = 25

func init() {
	// slog is the standard library logger, we use it to log errors
	log.InitStructureLogConfig()
}

func main() {
	// Load configuration with CLI flags and environment variables
	cfg := config.LoadConfig()

	// Set up OpenTelemetry SDK.
	// Command-line/environment OTEL_SERVICE_VERSION takes precedence over
	// the build-time Version variable.
	ctx := context.Background()
	otelConfig := utils.OTelConfigFromEnv()
	if otelConfig.ServiceVersion == "" {
		otelConfig.ServiceVersion = Version
	}
	otelShutdown, err := utils.SetupOTelSDKWithConfig(ctx, otelConfig)
	if err != nil {
		slog.With(log.ErrKey, err).Error("error setting up OpenTelemetry SDK")
		os.Exit(1)
	}
	// Handle shutdown properly so nothing leaks.
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), gracefulShutdownSeconds*time.Second)
		defer cancel()
		if shutdownErr := otelShutdown(ctx); shutdownErr != nil {
			slog.With(log.ErrKey, shutdownErr).Error("error shutting down OpenTelemetry SDK")
		}
	}()

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		<-sigChan
		slog.InfoContext(ctx, "Received shutdown signal, shutting down gracefully...")
		cancel()
	}()

	slog.InfoContext(ctx, "Starting LFX Access Check Service",
		"host", cfg.Host,
		"port", cfg.Port,
	)

	if err := StartServer(ctx, cfg); err != nil {
		slog.Error("Failed to start server", "error", err)
		os.Exit(1)
	}
}
