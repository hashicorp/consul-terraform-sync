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
	vaultAPI "github.com/hashicorp/vault/api"
	vaultHTTP "github.com/hashicorp/vault/http"
	vaultTest "github.com/hashicorp/vault/vault"
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
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			block, err := LoadDynamicConfig(ctx, w, r, tc.config)
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, block.Variables["attr"].AsString())
		})
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
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			block, err := LoadDynamicConfig(ctx, w, r, tc.config)
			if tc.err {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tc.expected, block.Variables["attr"].AsString())
		})
	}
}

func TestLoadDynamicConfig_Vault(t *testing.T) {
	// Setup Vault server, client, and write secret
	core, _, token := vaultTest.TestCoreUnsealed(t)
	ln, addr := vaultHTTP.TestServer(t, core)
	defer ln.Close()

	vaultConf := vaultAPI.DefaultConfig()
	vaultConf.Address = addr
	vClient, err := vaultAPI.NewClient(vaultConf)
	require.NoError(t, err)
	vClient.SetToken(token)

	// Vault kv1
	req := vClient.NewRequest("PUT", "/v1/secret/cts1")
	req.SetJSONBody(vaultAPI.Secret{Data: map[string]interface{}{"foo": "foobar1"}})
	resp, err := vClient.RawRequest(req)
	require.NoError(t, err)
	require.Equal(t, 204, resp.StatusCode)

	// Vault kv2
	req = vClient.NewRequest("POST", "/v1/secret/data/cts2")
	req.SetJSONBody(vaultAPI.Secret{Data: map[string]interface{}{"foo": "foobar2"}})
	resp, err = vClient.RawRequest(req)
	require.NoError(t, err)
	require.Equal(t, 204, resp.StatusCode)

	// Setup Vault client for hcat templates
	clients := hcat.NewClientSet()
	err = clients.AddVault(hcat.VaultInput{
		Address: addr,
		Token:   token,
	})
	require.NoError(t, err)

	w := hcat.NewWatcher(hcat.WatcherInput{Clients: clients})
	r := hcat.NewResolver()

	testCases := []struct {
		name     string
		config   map[string]interface{}
		expected string
		err      bool
	}{
		{
			"secret kv1",
			map[string]interface{}{
				"foo": map[string]interface{}{
					"attr": "{{ with secret \"secret/cts1\" }}{{ .Data.data.foo }}{{ end }}",
				},
			},
			"foobar1",
			false,
		}, {
			"secret kv2",
			map[string]interface{}{
				"foo": map[string]interface{}{
					"attr": "{{ with secret \"secret/data/cts2\" }}{{ .Data.data.foo }}{{ end }}",
				},
			},
			"foobar2",
			false,
		}, {
			"substring",
			map[string]interface{}{
				"foo": map[string]interface{}{
					"attr": "{{ with secret \"secret/data/cts2\" }}foo_{{ .Data.data.foo }}_baz{{ end }}",
				},
			},
			"foo_foobar2_baz",
			false,
		}, {
			"key dne",
			map[string]interface{}{
				"foo": map[string]interface{}{
					"attr": "{{ with secret \"secret/data/cts2\" }}{{ .Data.data.dne }}{{ end }}",
				},
			},
			"<no value>",
			false,
		}, {
			"path dne",
			map[string]interface{}{
				"foo": map[string]interface{}{
					"attr": "{{ with secret \"path/dne\" }}{{ . }}{{ end }}",
				},
			},
			"{{ with secret \"path/dne\" }}{{ . }}{{ end }}",
			true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			block, err := LoadDynamicConfig(ctx, w, r, tc.config)
			if tc.err {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tc.expected, block.Variables["attr"].AsString())
		})
	}
}
