// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package contracts defines the interface contracts for domain services and repositories.
package contracts

import (
	"context"
	"log/slog"

	"github.com/linuxfoundation/lfx-v2-access-check/pkg/constants"
)

// HeimdallClaims represents JWT claims from Heimdall
type HeimdallClaims struct {
	Principal string `json:"principal"`
	Email     string `json:"email,omitempty"`
}

// Validate provides validation of HeimdallClaims
func (c *HeimdallClaims) Validate(ctx context.Context) error {
	if c.Principal == "" {
		slog.WarnContext(ctx, "validation failed: principal must be provided")
		return constants.ErrPrincipalRequired
	}
	slog.DebugContext(ctx, "validation successful", "principal", c.Principal)
	return nil
}

// AuthRepository handles JWT validation
type AuthRepository interface {
	ValidateToken(ctx context.Context, token string) (*HeimdallClaims, error)
	HealthCheck(ctx context.Context) error
}
