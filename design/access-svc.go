// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT

// Package design contains the goa design definitions for the LFX access check service.
package design

import (
	"github.com/linuxfoundation/lfx-v2-access-check/pkg/constants"
	. "goa.design/goa/v3/dsl"
)

// API defines the metadata for the access check service API.
var _ = API("access-svc", func() {
	Title("LFX V2 - Access Check Service")
	Description("LFX Access Check Service for bulk access checks")
})

// JWTAuth defines the JWT authentication security scheme for the service.
var JWTAuth = JWTSecurity("jwt", func() {
	Description("Heimdall authorization")
})

var _ = Service("access-svc", func() {
	Description("LFX Access Check Service")

	Method("check-access", func() {
		Description("Check access permissions for resource-action pairs")
		Security(JWTAuth)

		Payload(func() {
			Token("bearer_token", String, "JWT token from Heimdall")
			Attribute("version", String, "API version", func() {
				Enum("1")
				Example("1")
			})
			Attribute("requests", ArrayOf(String), "Resource-action pairs to check", func() {
				Example([]string{constants.ExampleProjectAction, constants.ExampleCommitteeAction})
				MinLength(1)
			})
			Required("bearer_token", "version", "requests")
		})

		Result(func() {
			Attribute("results", ArrayOf(String), "Access check results", func() {
				Example([]string{constants.AccessAllow, constants.AccessDeny})
			})
			Required("results")
		})

		Error("BadRequest", ErrorResult, "Bad request")
		Error("Unauthorized", ErrorResult, "Unauthorized")

		HTTP(func() {
			POST("/access-check")
			Param("version:v")
			Header("bearer_token:Authorization")
			Response(StatusOK)
			Response("BadRequest", StatusBadRequest)
			Response("Unauthorized", StatusUnauthorized)
		})
	})

	Method("readyz", func() {
		Description("Check if service is ready")
		Result(Bytes, func() {
			Example("OK")
		})
		Error("NotReady", func() {
			Description("Service not ready")
			Temporary()
			Fault()
		})
		HTTP(func() {
			GET("/readyz")
			Response(StatusOK, func() {
				ContentType("text/plain")
			})
			Response("NotReady", StatusServiceUnavailable)
		})
		Meta("swagger:generate", "false")
	})

	Method("livez", func() {
		Description("Check if service is alive")
		Result(Bytes, func() {
			Example("OK")
		})
		HTTP(func() {
			GET("/livez")
			Response(StatusOK, func() {
				ContentType("text/plain")
			})
		})
		Meta("swagger:generate", "false")
	})

	Files("/_access-check/openapi.json", "gen/http/openapi.json", func() {
		Meta("swagger:generate", "false")
	})
	Files("/_access-check/openapi.yaml", "gen/http/openapi.yaml", func() {
		Meta("swagger:generate", "false")
	})
	Files("/_access-check/openapi3.json", "gen/http/openapi3.json", func() {
		Meta("swagger:generate", "false")
	})
	Files("/_access-check/openapi3.yaml", "gen/http/openapi3.yaml", func() {
		Meta("swagger:generate", "false")
	})
})
