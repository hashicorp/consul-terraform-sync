package client

import (
	"context"
	"encoding/json"
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
	path := "/v1/operator/license"
	ctx := logging.WithContext(context.Background(), logging.NewNullLogger())

	intercepts := []*testutils.HttpIntercept{
		{Path: path, ResponseStatusCode: http.StatusInternalServerError},
	}

	c := newTestConsulClient(t, testutils.NewHttpClient(t, intercepts), 1)
	_, err := c.GetLicense(ctx, nil)
	assert.Error(t, err)
}

func Test_GetLicense(t *testing.T) {
	t.Parallel()
	path := "/v1/operator/license"
	ctx := logging.WithContext(context.Background(), logging.NewNullLogger())

	expectedLicense := "foo"
	intercepts := []*testutils.HttpIntercept{
		{Path: path, ResponseStatusCode: http.StatusOK, ResponseData: []byte(expectedLicense)},
	}

	c := newTestConsulClient(t, testutils.NewHttpClient(t, intercepts), 1)
	license, err := c.GetLicense(ctx, nil)
	assert.NoError(t, err)
	assert.Equal(t, license, expectedLicense)
}

func Test_IsEnterprise_API_Failure(t *testing.T) {
	t.Parallel()
	path := "/v1/agent/self"
	ctx := logging.WithContext(context.Background(), logging.NewNullLogger())

	intercepts := []*testutils.HttpIntercept{
		{Path: path, ResponseStatusCode: http.StatusInternalServerError},
	}

	c := newTestConsulClient(t, testutils.NewHttpClient(t, intercepts), 1)
	_, err := c.IsEnterprise(ctx)
	assert.Error(t, err)
}

func Test_IsEnterprise(t *testing.T) {
	t.Parallel()
	path := "/v1/agent/self"
	ctx := logging.WithContext(context.Background(), logging.NewNullLogger())

	cases := []struct {
		name           string
		response       ConsulAgentConfig
		expectedResult bool
		expectError    bool
	}{
		{
			"oss",
			ConsulAgentConfig{"Config": {"Version": "v1.9.5"}},
			false,
			false,
		},
		{
			"oss dev",
			ConsulAgentConfig{"Config": {"Version": "v1.9.5-dev"}},
			false,
			false,
		},
		{
			"ent",
			ConsulAgentConfig{"Config": {"Version": "v1.9.5+ent"}},
			true,
			false,
		},
		{
			"ent dev",
			ConsulAgentConfig{"Config": {"Version": "v1.9.5+ent-dev"}},
			true,
			false,
		},
		{
			"missing",
			ConsulAgentConfig{"Config": {}},
			false,
			true,
		},
		{
			"malformed",
			ConsulAgentConfig{"Config": {"Version": "***"}},
			false,
			true,
		},
		{
			"bad key 1",
			ConsulAgentConfig{"NoConfig": {"Version": "***"}},
			false,
			true,
		},
		{
			"bad key 2",
			ConsulAgentConfig{"Config": {"NoVersion": "v1.9.5"}},
			false,
			true,
		},
		{
			"not a string",
			ConsulAgentConfig{"Config": {"Version": 123}},
			false,
			true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			infoBytes, err := json.Marshal(tc.response)
			assert.NoError(t, err)

			intercepts := []*testutils.HttpIntercept{
				{Path: path, ResponseStatusCode: http.StatusOK, ResponseData: infoBytes},
			}

			c := newTestConsulClient(t, testutils.NewHttpClient(t, intercepts), 1)

			isEnterprise, err := c.IsEnterprise(ctx)
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedResult, isEnterprise)
			}
		})
	}
}

func newTestConsulClient(t *testing.T, httpClient *http.Client, maxRetry int) ConsulClientInterface {
	c, err := consulapi.NewClient(&consulapi.Config{HttpClient: httpClient})
	assert.NoError(t, err)

	return &ConsulClient{
		Client: c,
		retry:  retry.NewRetry(maxRetry, time.Now().UnixNano()),
		logger: logging.Global().Named(consulSubsystemName),
	}
}
