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

	"github.com/linuxfoundation/lfx-v2-access-check/internal/infrastructure/config"
	"github.com/linuxfoundation/lfx-v2-access-check/pkg/log"
	"github.com/linuxfoundation/lfx-v2-access-check/pkg/utils"
)

func init() {
	// slog is the standard library logger, we use it to log errors
	log.InitStructureLogConfig()
}

func main() {
	// Load configuration with CLI flags and environment variables
	cfg := config.LoadConfig()

	// Set up OpenTelemetry SDK.
	ctx := context.Background()
	otelConfig := utils.OTelConfigFromEnv()
	otelShutdown, err := utils.SetupOTelSDKWithConfig(ctx, otelConfig)
	if err != nil {
		slog.Error("error setting up OpenTelemetry SDK", "error", err)
		os.Exit(1)
	}
	defer func() {
		if shutdownErr := otelShutdown(context.Background()); shutdownErr != nil {
			slog.Error("error shutting down OpenTelemetry SDK", "error", shutdownErr)
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
