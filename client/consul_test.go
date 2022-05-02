package client

import (
	"context"
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
			path := "/v1/operator/license"

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
	path := "/v1/operator/license"
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

	cases := []struct {
		name      string
		response  int
		expectErr bool
	}{
		{
			"success",
			http.StatusOK,
			false,
		},
		{
			"errors",
			http.StatusBadRequest,
			true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			intercepts := []*testutils.HttpIntercept{
				{Path: path, ResponseStatusCode: tc.response},
			}
			c := newTestConsulClient(t, testutils.NewHttpClient(t, intercepts), 1)
			err := c.RegisterService(context.Background(), nil)
			if !tc.expectErr {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestConsulClient_DeregisterService(t *testing.T) {
	t.Parallel()
	id := "cts-123"
	path := fmt.Sprintf("/v1/agent/service/deregister/%s", id)

	cases := []struct {
		name      string
		response  int
		expectErr bool
	}{
		{
			"success",
			http.StatusOK,
			false,
		},
		{
			"errors",
			http.StatusNotFound,
			true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			intercepts := []*testutils.HttpIntercept{
				{Path: path, ResponseStatusCode: tc.response},
			}
			c := newTestConsulClient(t, testutils.NewHttpClient(t, intercepts), 1)
			err := c.DeregisterService(context.Background(), id)
			if !tc.expectErr {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
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
