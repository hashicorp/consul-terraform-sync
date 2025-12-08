// Copyright IBM Corp. 2020, 2025
// SPDX-License-Identifier: MPL-2.0

package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/hashicorp/consul-terraform-sync/logging"
	"github.com/hashicorp/consul-terraform-sync/retry"
	"github.com/hashicorp/consul-terraform-sync/testutils"
	consulapi "github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_GetLicense_API_Failure(t *testing.T) {
	t.Parallel()

	var nonRetryableError *retry.NonRetryableError
	var nonEnterpriseConsulError *NonEnterpriseConsulError
	cases := []struct {
		name                 string
		responseCode         int
		isNonRetryableError  bool
		isNonEnterpriseError bool
	}{
		{
			name:         "server error",
			responseCode: http.StatusInternalServerError,
		},
		{
			name:         "request limit reached error",
			responseCode: http.StatusTooManyRequests,
		},
		{
			name:                "bad request non retryable error",
			responseCode:        http.StatusBadRequest,
			isNonRetryableError: true,
		},
		{
			name:                 "status not found non retryable and non enterprise",
			responseCode:         http.StatusNotFound,
			isNonRetryableError:  true,
			isNonEnterpriseError: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := "/v1/operator/license?signed=1"

			intercepts := []*testutils.HttpIntercept{
				{Path: path, ResponseStatusCode: tc.responseCode},
			}

			c := newTestConsulClient(t, testutils.NewHttpClient(t, intercepts), 1)
			_, err := c.GetLicense(context.Background(), nil)
			assert.Error(t, err)

			// Verify the error types
			assert.Equal(t, tc.isNonRetryableError, errors.As(err, &nonRetryableError))
			assert.Equal(t, tc.isNonEnterpriseError, errors.As(err, &nonEnterpriseConsulError))
		})
	}
}

func Test_GetLicense(t *testing.T) {
	t.Parallel()
	path := "/v1/operator/license?signed=1"
	expectedLicense := "foo"

	intercepts := []*testutils.HttpIntercept{
		{Path: path, ResponseStatusCode: http.StatusOK, ResponseData: []byte(expectedLicense)},
	}

	c := newTestConsulClient(t, testutils.NewHttpClient(t, intercepts), 1)
	license, err := c.GetLicense(context.Background(), nil)
	assert.NoError(t, err)
	assert.Equal(t, license, expectedLicense)
}

func TestConsulClient_RegisterService(t *testing.T) {
	t.Parallel()
	path := "/v1/agent/service/register"

	var nonRetryableError *retry.NonRetryableError
	var missingConsulACLError *MissingConsulACLError
	cases := []struct {
		name                string
		responseCode        int
		expectErr           bool
		isNonRetryableError bool
		isMissingAClError   bool
	}{
		{
			name:         "success",
			responseCode: http.StatusOK,
		},
		{
			name:         "request limit reached error retryable",
			responseCode: http.StatusTooManyRequests,
			expectErr:    true,
		},
		{
			name:                "bad request non retryable error",
			responseCode:        http.StatusBadRequest,
			expectErr:           true,
			isNonRetryableError: true,
		},
		{
			name:                "status forbidden non retryable and missing ACL error",
			responseCode:        http.StatusForbidden,
			expectErr:           true,
			isNonRetryableError: true,
			isMissingAClError:   true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			intercepts := []*testutils.HttpIntercept{
				{Path: path, ResponseStatusCode: tc.responseCode},
			}
			c := newTestConsulClient(t, testutils.NewHttpClient(t, intercepts), 1)
			err := c.RegisterService(context.Background(), nil)
			if !tc.expectErr {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				// Verify the error types
				assert.Equal(t, tc.isNonRetryableError, errors.As(err, &nonRetryableError))
				assert.Equal(t, tc.isMissingAClError, errors.As(err, &missingConsulACLError))
			}
		})
	}
}

func TestConsulClient_DeregisterService(t *testing.T) {
	t.Parallel()
	id := "cts-123"
	defaultPath := fmt.Sprintf("/v1/agent/service/deregister/%s", id)

	var nonRetryableError *retry.NonRetryableError
	var missingConsulACLError *MissingConsulACLError
	cases := []struct {
		name                string
		responseCode        int
		expectErr           bool
		isNonRetryableError bool
		isMissingAClError   bool
		path                string
		query               *consulapi.QueryOptions
	}{
		{
			name:         "success",
			responseCode: http.StatusOK,
		},
		{
			name:         "success_w_query",
			responseCode: http.StatusOK,
			path:         fmt.Sprintf("%s?ns=test-ns", defaultPath),
			query:        &consulapi.QueryOptions{Namespace: "test-ns"},
		},
		{
			name:         "request limit reached error retryable",
			responseCode: http.StatusTooManyRequests,
			expectErr:    true,
		},
		{
			name:                "bad request non retryable error",
			responseCode:        http.StatusBadRequest,
			expectErr:           true,
			isNonRetryableError: true,
		},
		{
			name:                "status forbidden non retryable and missing ACL error",
			responseCode:        http.StatusForbidden,
			expectErr:           true,
			isNonRetryableError: true,
			isMissingAClError:   true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Configure Consul client with intercepts
			path := tc.path
			if path == "" {
				path = defaultPath
			}
			intercepts := []*testutils.HttpIntercept{
				{Path: path, ResponseStatusCode: tc.responseCode},
			}
			c := newTestConsulClient(t, testutils.NewHttpClient(t, intercepts), 1)

			// Deregister service
			err := c.DeregisterService(context.Background(), id, tc.query)
			if !tc.expectErr {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				// Verify the error types
				assert.Equal(t, tc.isNonRetryableError, errors.As(err, &nonRetryableError))
				assert.Equal(t, tc.isMissingAClError, errors.As(err, &missingConsulACLError))
			}
		})
	}
}

func newTestConsulClient(t *testing.T, httpClient *http.Client, maxRetry int) *ConsulClient {
	c, err := consulapi.NewClient(&consulapi.Config{HttpClient: httpClient})
	assert.NoError(t, err)

	return &ConsulClient{
		Client: c,
		retry:  retry.NewRetry(maxRetry, time.Now().UnixNano()),
		logger: logging.NewNullLogger(),
	}
}

func TestNonEnterpriseConsulError_Error(t *testing.T) {
	err := NonEnterpriseConsulError{Err: errors.New("some error")}
	var nonEnterpriseConsulError *NonEnterpriseConsulError

	assert.True(t, errors.As(&err, &nonEnterpriseConsulError))
	assert.Equal(t, "consul is not consul enterprise: some error", err.Error())
}

func TestNonEnterpriseConsulError_Unwrap(t *testing.T) {
	var terr *testError

	var otherErr testError
	err := NonEnterpriseConsulError{Err: &otherErr}

	// Assert that the wrapped error is still detectable
	// errors.As is the preferred way to call the underlying Unwrap
	assert.True(t, errors.As(&err, &terr))
}

func TestMissingConsulACLError_Error(t *testing.T) {
	err := MissingConsulACLError{Err: errors.New("some error")}
	var missingConsulACLError *MissingConsulACLError

	assert.True(t, errors.As(&err, &missingConsulACLError))
	assert.Equal(t, "missing required Consul ACL: some error", err.Error())
}

func TestMissingConsulACLError_Unwrap(t *testing.T) {
	var terr *testError

	var otherErr testError
	err := MissingConsulACLError{Err: &otherErr}

	// Assert that the wrapped error is still detectable
	// errors.As is the preferred way to call the underlying Unwrap
	assert.True(t, errors.As(&err, &terr))
}

type testError struct {
}

// Error returns an error string
func (e *testError) Error() string {
	return "this is a test error"
}

func TestSessionCreate(t *testing.T) {
	t.Parallel()

	var nonRetryableError *retry.NonRetryableError
	var missingConsulACLError *MissingConsulACLError
	cases := []struct {
		name                string
		responseCode        int
		expectErr           bool
		isNonRetryableError bool
		isMissingAClError   bool
	}{
		{
			name:         "success",
			responseCode: http.StatusOK,
		},
		{
			name:         "retryable_error",
			responseCode: http.StatusInternalServerError,
			expectErr:    true,
		},
		{
			name:                "non_retryable_error",
			responseCode:        http.StatusBadRequest,
			expectErr:           true,
			isNonRetryableError: true,
		},
		{
			name:                "acl_error",
			responseCode:        http.StatusForbidden,
			expectErr:           true,
			isNonRetryableError: true,
			isMissingAClError:   true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Configure Consul client with intercepts
			intercepts := []*testutils.HttpIntercept{
				{
					Path:               "/v1/session/create",
					ResponseStatusCode: tc.responseCode,
					ResponseData:       []byte(`{"ID": "adf4238a-882b-9ddc-4a9d-5b6758e4159e"}`),
				},
			}
			c := newTestConsulClient(t, testutils.NewHttpClient(t, intercepts), 1)

			// Create session
			id, meta, err := c.SessionCreate(context.Background(), &consulapi.SessionEntry{}, nil)
			if !tc.expectErr {
				require.NoError(t, err)
				assert.NotEmpty(t, id)
				assert.NotNil(t, meta)
			} else {
				assert.Error(t, err)
				// Verify the error types
				assert.Equal(t, tc.isNonRetryableError, errors.As(err, &nonRetryableError))
				assert.Equal(t, tc.isMissingAClError, errors.As(err, &missingConsulACLError))
			}
		})
	}
}

func TestKVGet(t *testing.T) {
	t.Parallel()

	var nonRetryableError *retry.NonRetryableError
	var missingConsulACLError *MissingConsulACLError
	cases := []struct {
		name                string
		responseCode        int
		responseBody        string
		expectErr           bool
		isNonRetryableError bool
		isMissingAClError   bool
		query               *consulapi.QueryOptions
	}{
		{
			name:         "success",
			responseCode: http.StatusOK,
			responseBody: `[
  {
    "LockIndex": 0,
    "Key": "test",
    "Flags": 0,
    "Value": "dGVzdA==",
    "CreateIndex": 2154,
    "ModifyIndex": 2154
  }
]`,
		},
		{
			name:         "key_does_not_exist",
			responseCode: http.StatusNotFound,
			// do not expect error since KV().Get() does not error
		},
		{
			name:                "non_retryable_error",
			responseCode:        http.StatusBadRequest,
			expectErr:           true,
			isNonRetryableError: true,
		},
		{
			name:         "retryable_error",
			responseCode: http.StatusInternalServerError,
			expectErr:    true,
		},
		{
			name:                "acl_error",
			responseCode:        http.StatusForbidden,
			expectErr:           true,
			isNonRetryableError: true,
			isMissingAClError:   true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			key := "test"
			// Configure Consul client with intercepts
			intercepts := []*testutils.HttpIntercept{
				{
					Path:               "/v1/kv/" + key,
					ResponseStatusCode: tc.responseCode,
					ResponseData:       []byte(tc.responseBody),
				},
			}
			c := newTestConsulClient(t, testutils.NewHttpClient(t, intercepts), 1)

			// Get KV pair
			_, meta, err := c.KVGet(context.Background(), key, nil)
			if !tc.expectErr {
				require.NoError(t, err)
				assert.NotNil(t, meta)
			} else {
				assert.Error(t, err)
				// Verify the error types
				assert.Equal(t, tc.isNonRetryableError, errors.As(err, &nonRetryableError))
				assert.Equal(t, tc.isMissingAClError, errors.As(err, &missingConsulACLError))
			}
		})
	}
}

func TestConsulClient_QueryServices(t *testing.T) {
	t.Parallel()
	path := "/v1/agent/services"

	var nonRetryableError *retry.NonRetryableError
	var missingConsulACLError *MissingConsulACLError
	cases := []struct {
		name                string
		responseCode        int
		responseBody        string
		expectedServices    []*consulapi.AgentService
		expectErr           bool
		isNonRetryableError bool
		isMissingAClError   bool
	}{
		{
			name:             "success_empty",
			responseCode:     http.StatusOK,
			expectErr:        false,
			responseBody:     makeAgentQueryServicesResponse(),
			expectedServices: []*consulapi.AgentService{},
		},
		{
			name:             "success_single",
			responseCode:     http.StatusOK,
			expectErr:        false,
			responseBody:     makeAgentQueryServicesResponse(makeService("foo", "foo-1")),
			expectedServices: []*consulapi.AgentService{{Service: "foo", ID: "foo-1"}},
		},
		{
			name:         "success_multiple",
			responseCode: http.StatusOK,
			expectErr:    false,
			responseBody: makeAgentQueryServicesResponse(makeService("foo", "foo-1"),
				makeService("bar", "bar-1")),
			expectedServices: []*consulapi.AgentService{
				{Service: "foo", ID: "foo-1"},
				{Service: "bar", ID: "bar-1"},
			},
		},
		{
			name:         "request limit reached error retryable",
			responseCode: http.StatusTooManyRequests,
			expectErr:    true,
		},
		{
			name:                "bad request non retryable error",
			responseCode:        http.StatusBadRequest,
			expectErr:           true,
			isNonRetryableError: true,
		},
		{
			name:                "status forbidden non retryable and missing ACL error",
			responseCode:        http.StatusForbidden,
			expectErr:           true,
			isNonRetryableError: true,
			isMissingAClError:   true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			intercepts := []*testutils.HttpIntercept{
				{Path: path, ResponseStatusCode: tc.responseCode, ResponseData: []byte(tc.responseBody)},
			}
			c := newTestConsulClient(t, testutils.NewHttpClient(t, intercepts), 1)
			services, err := c.QueryServices(context.Background(), "", nil)
			if !tc.expectErr {
				assert.NoError(t, err)

				assert.NotNil(t, services)
				assert.ElementsMatch(t, tc.expectedServices, services)
			} else {
				assert.Error(t, err)
				// Verify the error types
				assert.Equal(t, tc.isNonRetryableError, errors.As(err, &nonRetryableError))
				assert.Equal(t, tc.isMissingAClError, errors.As(err, &missingConsulACLError))

				assert.Nil(t, services)
			}
		})
	}
}

func TestConsulClient_GetHealthChecks(t *testing.T) {
	t.Parallel()
	pathPrefix := "/v1/health/checks"

	var nonRetryableError *retry.NonRetryableError
	var missingConsulACLError *MissingConsulACLError
	cases := []struct {
		name                 string
		responseCode         int
		responseBody         string
		expectedHealthChecks consulapi.HealthChecks
		expectErr            bool
		serviceName          string
		isNonRetryableError  bool
		isMissingAClError    bool
	}{
		{
			name:                 "success_empty",
			responseCode:         http.StatusOK,
			expectErr:            false,
			responseBody:         makeHealthCheckResponse(),
			expectedHealthChecks: []*consulapi.HealthCheck{},
		},
		{
			name:         "success_single",
			responseCode: http.StatusOK,
			expectErr:    false,
			responseBody: makeHealthCheckResponse(makeHealthCheck("foo", "foo-01", "passing")),
			expectedHealthChecks: []*consulapi.HealthCheck{
				{
					ServiceName: "foo",
					ServiceID:   "foo-01",
					Status:      "passing",
				},
			},
		},
		{
			name:         "success_single_service_multiple_instances",
			responseCode: http.StatusOK,
			expectErr:    false,
			responseBody: makeHealthCheckResponse(makeHealthCheck("foo", "foo-01", "passing"),
				makeHealthCheck("foo", "foo-02", "passing")),
			expectedHealthChecks: []*consulapi.HealthCheck{
				{
					ServiceName: "foo",
					ServiceID:   "foo-01",
					Status:      "passing",
				},
				{
					ServiceName: "foo",
					ServiceID:   "foo-02",
					Status:      "passing",
				},
			},
		},
		{
			name:         "success_multiple_services_multiple_instances",
			responseCode: http.StatusOK,
			expectErr:    false,
			responseBody: makeHealthCheckResponse(makeHealthCheck("foo", "foo-01", "passing"),
				makeHealthCheck("bar", "bar-01", "passing")),
			expectedHealthChecks: []*consulapi.HealthCheck{
				{
					ServiceName: "foo",
					ServiceID:   "foo-01",
					Status:      "passing",
				},
				{
					ServiceName: "bar",
					ServiceID:   "bar-01",
					Status:      "passing",
				},
			},
		},
		{
			name:         "request limit reached error retryable",
			responseCode: http.StatusTooManyRequests,
			expectErr:    true,
		},
		{
			name:                "bad request non retryable error",
			responseCode:        http.StatusBadRequest,
			expectErr:           true,
			isNonRetryableError: true,
		},
		{
			name:                "status forbidden non retryable and missing ACL error",
			responseCode:        http.StatusForbidden,
			expectErr:           true,
			isNonRetryableError: true,
			isMissingAClError:   true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			intercepts := []*testutils.HttpIntercept{
				{
					Path:               fmt.Sprintf("%s/%s", pathPrefix, tc.serviceName),
					ResponseStatusCode: tc.responseCode,
					ResponseData:       []byte(tc.responseBody),
				},
			}
			c := newTestConsulClient(t, testutils.NewHttpClient(t, intercepts), 1)
			healthChecks, err := c.GetHealthChecks(context.Background(), tc.serviceName, nil)
			if !tc.expectErr {
				assert.NotNil(t, healthChecks)
				assert.NoError(t, err)

				assert.NotNil(t, healthChecks)
				assert.ElementsMatch(t, tc.expectedHealthChecks, healthChecks)
			} else {
				assert.Error(t, err)
				// Verify the error types
				assert.Equal(t, tc.isNonRetryableError, errors.As(err, &nonRetryableError))
				assert.Equal(t, tc.isMissingAClError, errors.As(err, &missingConsulACLError))

				assert.Nil(t, healthChecks)
			}
		})
	}
}

func makeService(name, id string) *consulapi.AgentService {
	return &consulapi.AgentService{
		ID:      id,
		Service: name,
	}
}

func makeAgentQueryServicesResponse(services ...*consulapi.AgentService) string {
	m := make(map[string]*consulapi.AgentService)
	for _, service := range services {
		m[service.ID] = service
	}

	bytes, _ := json.Marshal(m)
	return string(bytes)
}

func makeHealthCheck(serviceName, serviceId, status string) *consulapi.HealthCheck {
	return &consulapi.HealthCheck{
		ServiceName: serviceName,
		ServiceID:   serviceId,
		Status:      status,
	}
}

func makeHealthCheckResponse(healthChecks ...*consulapi.HealthCheck) string {
	hc := make([]*consulapi.HealthCheck, 0, len(healthChecks))
	hc = append(hc, healthChecks...)
	bytes, _ := json.Marshal(hc)

	return string(bytes)
}
