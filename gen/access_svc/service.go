// Code generated by goa v3.21.5, DO NOT EDIT.
//
// access-svc service
//
// Command:
// $ goa gen github.com/linuxfoundation/lfx-v2-access-check/design

package accesssvc

import (
	"context"

	goa "goa.design/goa/v3/pkg"
	"goa.design/goa/v3/security"
)

// LFX Access Check Service
type Service interface {
	// Check access permissions for resource-action pairs
	CheckAccess(context.Context, *CheckAccessPayload) (res *CheckAccessResult, err error)
	// Check if service is ready
	Readyz(context.Context) (res []byte, err error)
	// Check if service is alive
	Livez(context.Context) (res []byte, err error)
}

// Auther defines the authorization functions to be implemented by the service.
type Auther interface {
	// JWTAuth implements the authorization logic for the JWT security scheme.
	JWTAuth(ctx context.Context, token string, schema *security.JWTScheme) (context.Context, error)
}

// APIName is the name of the API as defined in the design.
const APIName = "access-svc"

// APIVersion is the version of the API as defined in the design.
const APIVersion = "0.0.1"

// ServiceName is the name of the service as defined in the design. This is the
// same value that is set in the endpoint request contexts under the ServiceKey
// key.
const ServiceName = "access-svc"

// MethodNames lists the service method names as defined in the design. These
// are the same values that are set in the endpoint request contexts under the
// MethodKey key.
var MethodNames = [3]string{"check-access", "readyz", "livez"}

// CheckAccessPayload is the payload type of the access-svc service
// check-access method.
type CheckAccessPayload struct {
	// JWT token from Heimdall
	BearerToken string
	// API version
	Version string
	// Resource-action pairs to check
	Requests []string
}

// CheckAccessResult is the result type of the access-svc service check-access
// method.
type CheckAccessResult struct {
	// Access check results
	Results []string
}

// MakeBadRequest builds a goa.ServiceError from an error.
func MakeBadRequest(err error) *goa.ServiceError {
	return goa.NewServiceError(err, "BadRequest", false, false, false)
}

// MakeUnauthorized builds a goa.ServiceError from an error.
func MakeUnauthorized(err error) *goa.ServiceError {
	return goa.NewServiceError(err, "Unauthorized", false, false, false)
}

// MakeNotReady builds a goa.ServiceError from an error.
func MakeNotReady(err error) *goa.ServiceError {
	return goa.NewServiceError(err, "NotReady", false, true, true)
}
