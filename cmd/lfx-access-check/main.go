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
)

func init() {
	// slog is the standard library logger, we use it to log errors
	log.InitStructureLogConfig()
}

func main() {
	// Load configuration with CLI flags and environment variables
	cfg := config.LoadConfig()

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	ctx, cancel := context.WithCancel(context.Background())
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
