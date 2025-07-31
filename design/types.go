// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package design contains the goa design definitions for the LFX access check service.
package design

import (
	. "goa.design/goa/v3/dsl"
)

// AccessErrorResult defines the error response structure for access check operations.
var AccessErrorResult = Type("AccessErrorResult", func() {
	Description("Standard error response for access check service")
	Attribute("message", String, "Error message", func() {
		Example("Invalid request format")
	})
	Attribute("code", String, "Error code", func() {
		Example("INVALID_REQUEST")
	})
	Required("message")
})
