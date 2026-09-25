// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package constants defines access control constants.
package constants

// Access control response values — each result is a tab-separated tuple
// of the form: object#relation@user\ttrue  or  object#relation@user\tfalse.
const (
	// AccessTrue indicates permission is granted.
	AccessTrue = "true"
	// AccessFalse indicates permission is denied.
	AccessFalse = "false"
)
