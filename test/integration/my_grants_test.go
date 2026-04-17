// Copyright The Linux Foundation and each contributor to LFX.
// SPDX-License-Identifier: MIT
package integration

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	accesssvc "github.com/linuxfoundation/lfx-v2-access-check/gen/access_svc"
	accesssvcsvr "github.com/linuxfoundation/lfx-v2-access-check/gen/http/access_svc/server"
	"github.com/linuxfoundation/lfx-v2-access-check/internal/service"
	"github.com/linuxfoundation/lfx-v2-access-check/pkg/constants"
	goahttp "goa.design/goa/v3/http"
)

func newTestServer(t *testing.T, messagingRepo interface {
	Request(ctx context.Context, subject string, data []byte, timeout time.Duration) ([]byte, error)
	Close() error
	HealthCheck(ctx context.Context) error
}) *httptest.Server {
	t.Helper()
	accessService := service.NewAccessService(&MockAuthRepository{}, messagingRepo)
	endpoints := accesssvc.NewEndpoints(accessService)
	mux := goahttp.NewMuxer()
	svr := accesssvcsvr.New(endpoints, mux,
		goahttp.RequestDecoder,
		goahttp.ResponseEncoder,
		func(_ context.Context, _ http.ResponseWriter, err error) {
			t.Logf("Error handler called: %v", err)
		},
		nil, nil, nil, nil, nil,
	)
	accesssvcsvr.Mount(mux, svr)
	return httptest.NewServer(mux)
}

func TestMyGrantsEndpoint(t *testing.T) {
	tests := []struct {
		name           string
		url            string
		authHeader     string
		natsResponse   []byte
		expectedStatus int
		validateBody   func(t *testing.T, body []byte)
	}{
		{
			name:           "Valid request returns grants",
			url:            "/my-grants?v=1&object_type=project",
			authHeader:     "Bearer valid-token",
			natsResponse:   []byte(`{"results":["project:uuid1#writer@user:test-user","project:uuid2#auditor@user:test-user"]}`),
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, body []byte) {
				var resp map[string]interface{}
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}
				grants, ok := resp["grants"]
				if !ok {
					t.Error("response missing 'grants' field")
				}
				grantsArr, ok := grants.([]interface{})
				if !ok {
					t.Error("grants field is not an array")
				}
				if len(grantsArr) != 2 {
					t.Errorf("expected 2 grants, got %d", len(grantsArr))
				}
			},
		},
		{
			name:           "Valid request with empty results",
			url:            "/my-grants?v=1&object_type=committee",
			authHeader:     "Bearer valid-token",
			natsResponse:   []byte(`{"results":[]}`),
			expectedStatus: http.StatusOK,
			validateBody: func(t *testing.T, body []byte) {
				var resp map[string]interface{}
				if err := json.Unmarshal(body, &resp); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}
				grants, ok := resp["grants"]
				if !ok {
					t.Error("response missing 'grants' field")
				}
				grantsArr, ok := grants.([]interface{})
				if !ok {
					t.Error("grants field is not an array")
				}
				if len(grantsArr) != 0 {
					t.Errorf("expected 0 grants, got %d", len(grantsArr))
				}
			},
		},
		{
			name:           "Missing authorization header",
			url:            "/my-grants?v=1&object_type=project",
			authHeader:     "",
			natsResponse:   nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Invalid token",
			url:            "/my-grants?v=1&object_type=project",
			authHeader:     "Bearer invalid-token",
			natsResponse:   nil,
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Missing version parameter",
			url:            "/my-grants?object_type=project",
			authHeader:     "Bearer valid-token",
			natsResponse:   nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Missing object_type parameter",
			url:            "/my-grants?v=1",
			authHeader:     "Bearer valid-token",
			natsResponse:   nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Invalid object_type — leading underscore",
			url:            "/my-grants?v=1&object_type=_bad",
			authHeader:     "Bearer valid-token",
			natsResponse:   nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Invalid object_type — trailing underscore",
			url:            "/my-grants?v=1&object_type=bad_",
			authHeader:     "Bearer valid-token",
			natsResponse:   nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Invalid object_type — uppercase letters",
			url:            "/my-grants?v=1&object_type=Project",
			authHeader:     "Bearer valid-token",
			natsResponse:   nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Invalid object_type — hyphen",
			url:            "/my-grants?v=1&object_type=foo-bar",
			authHeader:     "Bearer valid-token",
			natsResponse:   nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Valid compound object_type",
			url:            "/my-grants?v=1&object_type=groupsio_mailing_list",
			authHeader:     "Bearer valid-token",
			natsResponse:   []byte(`{"results":[]}`),
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			natsResp := tt.natsResponse
			ts := newTestServer(t, &ConfigurableMessagingRepository{
				RequestFunc: func(_ context.Context, subject string, _ []byte, _ time.Duration) ([]byte, error) {
					if subject != constants.ReadTuplesSubject {
						t.Errorf("unexpected NATS subject: %s", subject)
					}
					return natsResp, nil
				},
			})
			defer ts.Close()

			req, err := http.NewRequest(http.MethodGet, ts.URL+tt.url, nil)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("failed to make request: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}

			if tt.validateBody != nil && resp.StatusCode == http.StatusOK {
				body, err := io.ReadAll(resp.Body)
				if err != nil {
					t.Fatalf("failed to read response body: %v", err)
				}
				tt.validateBody(t, body)
			}
		})
	}
}
