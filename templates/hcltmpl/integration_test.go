// +build integration

package hcltmpl

import (
	"context"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/consul/sdk/testutil"
	"github.com/hashicorp/hcat"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadDynamicConfig_Env(t *testing.T) {
	// Environment cleanup after testing
	cachedTestEnv, ok := os.LookupEnv("CTS_TEST_ENV")
	if ok {
		defer os.Setenv("CTS_TEST_ENV", cachedTestEnv)
	} else {
		defer os.Unsetenv("CTS_TEST_ENV")
	}
	os.Setenv("CTS_TEST_ENV", "foobar")

	cachedTestDNEEnv, ok := os.LookupEnv("CTS_TEST_DNE")
	if ok {
		defer os.Setenv("CTS_TEST_DNE", cachedTestDNEEnv)
	} else {
		defer os.Unsetenv("CTS_TEST_DNE")
	}

	w := hcat.NewWatcher(hcat.WatcherInput{})
	r := hcat.NewResolver()

	testCases := []struct {
		name     string
		config   map[string]interface{}
		expected string
	}{
		{
			"env",
			map[string]interface{}{
				"foo": map[string]interface{}{
					"attr": "{{ env \"CTS_TEST_ENV\" }}",
				},
			},
			"foobar",
		}, {
			"substring",
			map[string]interface{}{
				"foo": map[string]interface{}{
					"attr": "foo_{{ env \"CTS_TEST_ENV\" }}_baz",
				},
			},
			"foo_foobar_baz",
		}, {
			"dne",
			map[string]interface{}{
				"foo": map[string]interface{}{
					"attr": "prefix_{{ env \"CTS_TEST_DNE\" }}",
				},
			},
			"prefix_",
		},
	}

	for _, tc := range testCases {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		block, err := LoadDynamicConfig(ctx, w, r, tc.config)
		assert.NoError(t, err)
		assert.Equal(t, tc.expected, block.Variables["attr"].AsString())
	}
}

func TestLoadDynamicConfig_ConsulKV(t *testing.T) {
	// Setup Consul server and write to KV
	srv, err := testutil.NewTestServerConfig(func(c *testutil.TestServerConfig) {
		c.LogLevel = "warn"
		c.Stdout = ioutil.Discard
		c.Stderr = ioutil.Discard
	})
	require.NoError(t, err)
	clients := hcat.NewClientSet()
	err = clients.AddConsul(hcat.ConsulInput{Address: srv.HTTPAddr})
	require.NoError(t, err)
	srv.SetKVString(t, "cts/test", "foobar")

	w := hcat.NewWatcher(hcat.WatcherInput{Clients: clients})
	r := hcat.NewResolver()

	testCases := []struct {
		name     string
		config   map[string]interface{}
		expected string
		err      bool
	}{
		{
			"key",
			map[string]interface{}{
				"foo": map[string]interface{}{
					"attr": "{{ key \"cts/test\" }}",
				},
			},
			"foobar",
			false,
		}, {
			"substring",
			map[string]interface{}{
				"foo": map[string]interface{}{
					"attr": "foo_{{ key \"cts/test\" }}_baz",
				},
			},
			"foo_foobar_baz",
			false,
		}, {
			"dne",
			map[string]interface{}{
				"foo": map[string]interface{}{
					"attr": "prefix_{{ key \"cts/dne\" }}",
				},
			},
			"prefix_{{ key \"cts/dne\" }}",
			true,
		},
	}

	for _, tc := range testCases {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		block, err := LoadDynamicConfig(ctx, w, r, tc.config)
		if tc.err {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
		}
		assert.Equal(t, tc.expected, block.Variables["attr"].AsString())
	}
}
