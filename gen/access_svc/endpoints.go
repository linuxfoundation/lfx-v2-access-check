// Code generated by goa v3.21.5, DO NOT EDIT.
//
// access-svc endpoints
//
// Command:
// $ goa gen github.com/linuxfoundation/lfx-v2-access-check/design

package accesssvc

import (
	"context"

	goa "goa.design/goa/v3/pkg"
	"goa.design/goa/v3/security"
)

// Endpoints wraps the "access-svc" service endpoints.
type Endpoints struct {
	CheckAccess goa.Endpoint
	Readyz      goa.Endpoint
	Livez       goa.Endpoint
}

// NewEndpoints wraps the methods of the "access-svc" service with endpoints.
func NewEndpoints(s Service) *Endpoints {
	// Casting service to Auther interface
	a := s.(Auther)
	return &Endpoints{
		CheckAccess: NewCheckAccessEndpoint(s, a.JWTAuth),
		Readyz:      NewReadyzEndpoint(s),
		Livez:       NewLivezEndpoint(s),
	}
}

// Use applies the given middleware to all the "access-svc" service endpoints.
func (e *Endpoints) Use(m func(goa.Endpoint) goa.Endpoint) {
	e.CheckAccess = m(e.CheckAccess)
	e.Readyz = m(e.Readyz)
	e.Livez = m(e.Livez)
}

// NewCheckAccessEndpoint returns an endpoint function that calls the method
// "check-access" of service "access-svc".
func NewCheckAccessEndpoint(s Service, authJWTFn security.AuthJWTFunc) goa.Endpoint {
	return func(ctx context.Context, req any) (any, error) {
		p := req.(*CheckAccessPayload)
		var err error
		sc := security.JWTScheme{
			Name:           "jwt",
			Scopes:         []string{},
			RequiredScopes: []string{},
		}
		ctx, err = authJWTFn(ctx, p.BearerToken, &sc)
		if err != nil {
			return nil, err
		}
		return s.CheckAccess(ctx, p)
	}
}

// NewReadyzEndpoint returns an endpoint function that calls the method
// "readyz" of service "access-svc".
func NewReadyzEndpoint(s Service) goa.Endpoint {
	return func(ctx context.Context, req any) (any, error) {
		return s.Readyz(ctx)
	}
}

// NewLivezEndpoint returns an endpoint function that calls the method "livez"
// of service "access-svc".
func NewLivezEndpoint(s Service) goa.Endpoint {
	return func(ctx context.Context, req any) (any, error) {
		return s.Livez(ctx)
	}
}
