package client

import (
	"context"
	"errors"
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

func newTestConsulClient(t *testing.T, httpClient *http.Client, maxRetry int) *ConsulClient {
	c, err := consulapi.NewClient(&consulapi.Config{HttpClient: httpClient})
	assert.NoError(t, err)

	return &ConsulClient{
		Client: c,
		retry:  retry.NewRetry(maxRetry, time.Now().UnixNano()),
		logger: logging.NewNullLogger(),
	}
}
