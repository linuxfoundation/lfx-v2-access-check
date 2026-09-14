// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

package contracts

import "context"

// AccessChecker is the domain-level interface for performing access checks and
// reading direct OpenFGA tuples. AccessService depends on this interface;
// AccessCheckClient is the production implementation.
//
// The seam sits at domain level: callers work with principal strings and
// resource slices — no knowledge of NATS wire format, byte encoding, or
// JSON shapes is required.
//
// Result order from CheckAccess is NOT guaranteed. See
// docs/access-check-contract.md for the full unordered-response caveat.
type AccessChecker interface {
	// CheckAccess sends resource-action pairs and returns the result lines.
	CheckAccess(ctx context.Context, principal string, resources []string) ([]string, error)

	// ReadTuples returns the direct OpenFGA tuples for a principal, optionally
	// filtered by objectType.
	ReadTuples(ctx context.Context, principal string, objectType string) ([]string, error)

	// HealthCheck reports whether the underlying messaging transport is healthy.
	HealthCheck(ctx context.Context) error
}
