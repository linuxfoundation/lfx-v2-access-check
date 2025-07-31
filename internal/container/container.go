// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package container provides dependency injection for the application.
package container

import (
	"log/slog"

	accesssvc "github.com/linuxfoundation/lfx-v2-access-check/gen/access_svc"
	"github.com/linuxfoundation/lfx-v2-access-check/internal/domain/contracts"
	"github.com/linuxfoundation/lfx-v2-access-check/internal/infrastructure/auth"
	"github.com/linuxfoundation/lfx-v2-access-check/internal/infrastructure/config"
	"github.com/linuxfoundation/lfx-v2-access-check/internal/infrastructure/messaging"
	"github.com/linuxfoundation/lfx-v2-access-check/internal/service"
)

// Container holds all application dependencies
type Container struct {
	// Configuration - keep for potential middleware/server needs
	Config *config.Config

	// Services - only expose what consumers actually need
	AccessService accesssvc.Service

	// Private fields for cleanup (not exposed to consumers)
	messagingRepo contracts.MessagingRepository
}

// NewContainer creates a new dependency injection container
func NewContainer(cfg *config.Config) (*Container, error) {
	slog.Info("Initializing dependency container")

	// Initialize repositories
	authRepo, err := auth.NewAuthRepository(cfg.JWKSUrl, cfg.Issuer, cfg.Audience)
	if err != nil {
		slog.Error("Failed to initialize auth repository", "error", err)
		return nil, err
	}

	messagingRepo, err := messaging.NewMessagingRepository(cfg.NATSUrl)
	if err != nil {
		slog.Error("Failed to initialize messaging repository", "error", err)
		return nil, err
	}

	// Initialize services - Create unified access service
	accessService := service.NewAccessService(authRepo, messagingRepo)

	slog.Info("Dependency container initialized successfully")
	return &Container{
		Config:        cfg,
		AccessService: accessService,
		messagingRepo: messagingRepo,
	}, nil
}

// Close cleans up resources
func (c *Container) Close() error {
	if c.messagingRepo != nil {
		err := c.messagingRepo.Close()
		if err != nil {
			slog.Error("Failed to close messaging repository", "error", err)
			return err
		}
		slog.Info("Container resources cleaned up successfully")
	}
	return nil
}
