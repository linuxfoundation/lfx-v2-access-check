// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package messaging provides NATS-based messaging infrastructure for the access service.
package messaging

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/linuxfoundation/lfx-v2-access-check/internal/domain/contracts"
	"github.com/nats-io/nats.go"
)

type messagingRepository struct {
	conn *nats.Conn
}

// NewMessagingRepository creates a new NATS-based messaging repository
func NewMessagingRepository(natsURL string) (contracts.MessagingRepository, error) {
	slog.Info("Connecting to NATS", "nats_url", natsURL)

	conn, err := nats.Connect(natsURL)
	if err != nil {
		slog.Error("Failed to connect to NATS", "error", err, "nats_url", natsURL)
		return nil, err
	}

	return &messagingRepository{
		conn: conn,
	}, nil
}

// Request sends a request message to the specified subject and waits for a response
func (r *messagingRepository) Request(ctx context.Context, subject string, data []byte, timeout time.Duration) ([]byte, error) {
	msg, err := r.conn.Request(subject, data, timeout)
	if err != nil {
		slog.ErrorContext(ctx, "Failed to send NATS request", "error", err, "subject", subject)
		return nil, err
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
		return errors.New("NATS connection not initialized")
	}

	// Check connection status
	if !r.conn.IsConnected() {
		return errors.New("NATS connection is not active")
	}

	// Check if connection is closed or draining
	if r.conn.IsClosed() {
		return errors.New("NATS connection is closed")
	}

	if r.conn.IsDraining() {
		return errors.New("NATS connection is draining")
	}

	// Send a ping to verify responsiveness
	if err := r.conn.FlushWithContext(ctx); err != nil {
		return fmt.Errorf("NATS connection not responsive: %w", err)
	}

	return nil
}
