// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package contracts defines the interface contracts for domain services and repositories.
package contracts

import (
	"context"
	"time"
)

// MessagingRepository handles NATS communication
type MessagingRepository interface {
	Request(ctx context.Context, subject string, data []byte, timeout time.Duration) ([]byte, error)
	Close() error
	HealthCheck(ctx context.Context) error
}
