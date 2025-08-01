// Code generated by goa v3.21.5, DO NOT EDIT.
//
// access-svc HTTP client encoders and decoders
//
// Command:
// $ goa gen github.com/linuxfoundation/lfx-v2-access-check/design

package client

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"

	accesssvc "github.com/linuxfoundation/lfx-v2-access-check/gen/access_svc"
	goahttp "goa.design/goa/v3/http"
)

// BuildCheckAccessRequest instantiates a HTTP request object with method and
// path set to call the "access-svc" service "check-access" endpoint
func (c *Client) BuildCheckAccessRequest(ctx context.Context, v any) (*http.Request, error) {
	u := &url.URL{Scheme: c.scheme, Host: c.host, Path: CheckAccessAccessSvcPath()}
	req, err := http.NewRequest("POST", u.String(), nil)
	if err != nil {
		return nil, goahttp.ErrInvalidURL("access-svc", "check-access", u.String(), err)
	}
	if ctx != nil {
		req = req.WithContext(ctx)
	}

	return req, nil
}

// EncodeCheckAccessRequest returns an encoder for requests sent to the
// access-svc check-access server.
func EncodeCheckAccessRequest(encoder func(*http.Request) goahttp.Encoder) func(*http.Request, any) error {
	return func(req *http.Request, v any) error {
		p, ok := v.(*accesssvc.CheckAccessPayload)
		if !ok {
			return goahttp.ErrInvalidType("access-svc", "check-access", "*accesssvc.CheckAccessPayload", v)
		}
		{
			head := p.BearerToken
			if !strings.Contains(head, " ") {
				req.Header.Set("Authorization", "Bearer "+head)
			} else {
				req.Header.Set("Authorization", head)
			}
		}
		values := req.URL.Query()
		values.Add("v", p.Version)
		req.URL.RawQuery = values.Encode()
		body := NewCheckAccessRequestBody(p)
		if err := encoder(req).Encode(&body); err != nil {
			return goahttp.ErrEncodingError("access-svc", "check-access", err)
		}
		return nil
	}
}

// DecodeCheckAccessResponse returns a decoder for responses returned by the
// access-svc check-access endpoint. restoreBody controls whether the response
// body should be restored after having been read.
// DecodeCheckAccessResponse may return the following errors:
//   - "BadRequest" (type *goa.ServiceError): http.StatusBadRequest
//   - "Unauthorized" (type *goa.ServiceError): http.StatusUnauthorized
//   - error: internal error
func DecodeCheckAccessResponse(decoder func(*http.Response) goahttp.Decoder, restoreBody bool) func(*http.Response) (any, error) {
	return func(resp *http.Response) (any, error) {
		if restoreBody {
			b, err := io.ReadAll(resp.Body)
			if err != nil {
				return nil, err
			}
			resp.Body = io.NopCloser(bytes.NewBuffer(b))
			defer func() {
				resp.Body = io.NopCloser(bytes.NewBuffer(b))
			}()
		} else {
			defer resp.Body.Close()
		}
		switch resp.StatusCode {
		case http.StatusOK:
			var (
				body CheckAccessResponseBody
				err  error
			)
			err = decoder(resp).Decode(&body)
			if err != nil {
				return nil, goahttp.ErrDecodingError("access-svc", "check-access", err)
			}
			err = ValidateCheckAccessResponseBody(&body)
			if err != nil {
				return nil, goahttp.ErrValidationError("access-svc", "check-access", err)
			}
			res := NewCheckAccessResultOK(&body)
			return res, nil
		case http.StatusBadRequest:
			var (
				body CheckAccessBadRequestResponseBody
				err  error
			)
			err = decoder(resp).Decode(&body)
			if err != nil {
				return nil, goahttp.ErrDecodingError("access-svc", "check-access", err)
			}
			err = ValidateCheckAccessBadRequestResponseBody(&body)
			if err != nil {
				return nil, goahttp.ErrValidationError("access-svc", "check-access", err)
			}
			return nil, NewCheckAccessBadRequest(&body)
		case http.StatusUnauthorized:
			var (
				body CheckAccessUnauthorizedResponseBody
				err  error
			)
			err = decoder(resp).Decode(&body)
			if err != nil {
				return nil, goahttp.ErrDecodingError("access-svc", "check-access", err)
			}
			err = ValidateCheckAccessUnauthorizedResponseBody(&body)
			if err != nil {
				return nil, goahttp.ErrValidationError("access-svc", "check-access", err)
			}
			return nil, NewCheckAccessUnauthorized(&body)
		default:
			body, _ := io.ReadAll(resp.Body)
			return nil, goahttp.ErrInvalidResponse("access-svc", "check-access", resp.StatusCode, string(body))
		}
	}
}

// BuildReadyzRequest instantiates a HTTP request object with method and path
// set to call the "access-svc" service "readyz" endpoint
func (c *Client) BuildReadyzRequest(ctx context.Context, v any) (*http.Request, error) {
	u := &url.URL{Scheme: c.scheme, Host: c.host, Path: ReadyzAccessSvcPath()}
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, goahttp.ErrInvalidURL("access-svc", "readyz", u.String(), err)
	}
	if ctx != nil {
		req = req.WithContext(ctx)
	}

	return req, nil
}

// DecodeReadyzResponse returns a decoder for responses returned by the
// access-svc readyz endpoint. restoreBody controls whether the response body
// should be restored after having been read.
// DecodeReadyzResponse may return the following errors:
//   - "NotReady" (type *goa.ServiceError): http.StatusServiceUnavailable
//   - error: internal error
func DecodeReadyzResponse(decoder func(*http.Response) goahttp.Decoder, restoreBody bool) func(*http.Response) (any, error) {
	return func(resp *http.Response) (any, error) {
		if restoreBody {
			b, err := io.ReadAll(resp.Body)
			if err != nil {
				return nil, err
			}
			resp.Body = io.NopCloser(bytes.NewBuffer(b))
			defer func() {
				resp.Body = io.NopCloser(bytes.NewBuffer(b))
			}()
		} else {
			defer resp.Body.Close()
		}
		switch resp.StatusCode {
		case http.StatusOK:
			var (
				body []byte
				err  error
			)
			err = decoder(resp).Decode(&body)
			if err != nil {
				return nil, goahttp.ErrDecodingError("access-svc", "readyz", err)
			}
			return body, nil
		case http.StatusServiceUnavailable:
			var (
				body ReadyzNotReadyResponseBody
				err  error
			)
			err = decoder(resp).Decode(&body)
			if err != nil {
				return nil, goahttp.ErrDecodingError("access-svc", "readyz", err)
			}
			err = ValidateReadyzNotReadyResponseBody(&body)
			if err != nil {
				return nil, goahttp.ErrValidationError("access-svc", "readyz", err)
			}
			return nil, NewReadyzNotReady(&body)
		default:
			body, _ := io.ReadAll(resp.Body)
			return nil, goahttp.ErrInvalidResponse("access-svc", "readyz", resp.StatusCode, string(body))
		}
	}
}

// BuildLivezRequest instantiates a HTTP request object with method and path
// set to call the "access-svc" service "livez" endpoint
func (c *Client) BuildLivezRequest(ctx context.Context, v any) (*http.Request, error) {
	u := &url.URL{Scheme: c.scheme, Host: c.host, Path: LivezAccessSvcPath()}
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, goahttp.ErrInvalidURL("access-svc", "livez", u.String(), err)
	}
	if ctx != nil {
		req = req.WithContext(ctx)
	}

	return req, nil
}

// DecodeLivezResponse returns a decoder for responses returned by the
// access-svc livez endpoint. restoreBody controls whether the response body
// should be restored after having been read.
func DecodeLivezResponse(decoder func(*http.Response) goahttp.Decoder, restoreBody bool) func(*http.Response) (any, error) {
	return func(resp *http.Response) (any, error) {
		if restoreBody {
			b, err := io.ReadAll(resp.Body)
			if err != nil {
				return nil, err
			}
			resp.Body = io.NopCloser(bytes.NewBuffer(b))
			defer func() {
				resp.Body = io.NopCloser(bytes.NewBuffer(b))
			}()
		} else {
			defer resp.Body.Close()
		}
		switch resp.StatusCode {
		case http.StatusOK:
			var (
				body []byte
				err  error
			)
			err = decoder(resp).Decode(&body)
			if err != nil {
				return nil, goahttp.ErrDecodingError("access-svc", "livez", err)
			}
			return body, nil
		default:
			body, _ := io.ReadAll(resp.Body)
			return nil, goahttp.ErrInvalidResponse("access-svc", "livez", resp.StatusCode, string(body))
		}
	}
}
