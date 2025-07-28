// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package constants defines messaging-related constants.
package constants

import "time"

// Messaging constants
const (
	// DefaultNATSTimeout is the default timeout for NATS operations
	DefaultNATSTimeout = 15 * time.Second

	// DefaultNATSReconnectWait is the default wait time between reconnection attempts
	DefaultNATSReconnectWait = 2 * time.Second

	// DefaultNATSDrainTimeout is the default timeout for draining NATS connections
	DefaultNATSDrainTimeout = 25 * time.Second

	// DefaultMessageBufferSizeMultiplier is used to calculate buffer size based on resource count
	DefaultMessageBufferSizeMultiplier = 80

	// DefaultResponseSanityCheckBytes is the number of bytes to check for error detection
	DefaultResponseSanityCheckBytes = 20
)
