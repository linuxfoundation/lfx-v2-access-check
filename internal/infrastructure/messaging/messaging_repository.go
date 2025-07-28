// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package messaging provides NATS-based messaging infrastructure for the access service.
package messaging

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/linuxfoundation/lfx-v2-access-check/internal/domain/contracts"
	"github.com/linuxfoundation/lfx-v2-access-check/pkg/constants"
	"github.com/nats-io/nats.go"
)

type messagingRepository struct {
	conn *nats.Conn
}

// NewMessagingRepository creates a new NATS-based messaging repository
func NewMessagingRepository(natsURL string) (contracts.MessagingRepository, error) {
	slog.Info("Connecting to NATS", "nats_url", natsURL)

	// Configure NATS connection options with basic error handling from reference
	opts := []nats.Option{
		nats.MaxReconnects(3),
		nats.ReconnectWait(constants.DefaultNATSReconnectWait),
		nats.DrainTimeout(constants.DefaultNATSDrainTimeout),
		nats.ErrorHandler(func(_ *nats.Conn, s *nats.Subscription, err error) {
			if s != nil {
				slog.Error("async NATS error", "error", err, "subject", s.Subject, "queue", s.Queue)
			} else {
				slog.Error("async NATS error outside subscription", "error", err)
			}
		}),
		nats.ClosedHandler(func(_ *nats.Conn) {
			// This handler means that max reconnect attempts have been exhausted.
			slog.Error(constants.ErrMsgNATSMaxReconnects)
			// In a full implementation, this would coordinate with graceful shutdown
		}),
	}

	conn, err := nats.Connect(natsURL, opts...)
	if err != nil {
		slog.Error("Failed to connect to NATS", "error", err, "nats_url", natsURL)
		return nil, err
	}

	slog.Info("Successfully connected to NATS", "connected_url", conn.ConnectedUrl())

	return &messagingRepository{
		conn: conn,
	}, nil
}

// Request sends a request message to the specified subject and waits for a response
func (r *messagingRepository) Request(ctx context.Context, subject string, data []byte, timeout time.Duration) ([]byte, error) {
	if r.conn == nil {
		slog.ErrorContext(ctx, "NATS connection not initialized")
		return nil, constants.ErrNATSConnNotInit
	}

	msg, err := r.conn.Request(subject, data, timeout)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to send NATS request", "error", err, "subject", subject)
		return nil, fmt.Errorf("%s: %w", constants.ErrMsgNATSRequestFailed, err)
	}
	return msg.Data, nil
}

// Close closes the NATS connection gracefully
func (r *messagingRepository) Close() error {
	if r.conn != nil {
		err := r.conn.Drain()
		if err != nil {
			slog.Error("Failed to close NATS connection", "error", err)
			return err
		}
	}
	return nil
}

// HealthCheck verifies the NATS connection is healthy and responsive
func (r *messagingRepository) HealthCheck(ctx context.Context) error {
	if r.conn == nil {
		return constants.ErrNATSConnNotInit
	}

	// Check connection status
	if !r.conn.IsConnected() {
		return constants.ErrNATSConnNotActive
	}

	// Check if connection is closed or draining
	if r.conn.IsClosed() {
		return constants.ErrNATSConnClosed
	}

	if r.conn.IsDraining() {
		return constants.ErrNATSConnDraining
	}

	// Send a ping to verify responsiveness
	if err := r.conn.FlushWithContext(ctx); err != nil {
		return fmt.Errorf("%s: %w", constants.ErrMsgNATSConnNotResponsive, err)
	}

	return nil
}
