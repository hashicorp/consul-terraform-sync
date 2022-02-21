//go:build integration && vault
// +build integration,vault

package hcltmpl

import (
	"context"
	"testing"
	"time"

	"github.com/hashicorp/consul-terraform-sync/testutils"
	"github.com/hashicorp/hcat"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestVault_Compile confirms that the Vault integration tests are compilable.
// Vault integration tests are only run weekly. This test is intended to run
// with each change (vs. weekly) to do a basic check that the tests are still
// in a compilable state.
func TestVaultIntegration_Compile(t *testing.T) {
	// no-op
}

func TestLoadDynamicConfig_Vault(t *testing.T) {
	vClient, stopVault := testutils.NewTestVaultServer(t, testutils.TestVaultServerConfig{})
	defer stopVault(t)

	// Vault kv1
	_, err := vClient.Logical().Write("/secret/data/cts1",
		map[string]interface{}{"data": map[string]interface{}{"foo": "foobar1"}},
	)
	require.NoError(t, err)

	// Vault kv2
	_, err = vClient.Logical().Write("/secret/data/cts2",
		map[string]interface{}{"data": map[string]interface{}{"foo": "foobar2"}},
	)
	require.NoError(t, err)

	// Setup Vault client for hcat templates
	clients := hcat.NewClientSet()
	err = clients.AddVault(hcat.VaultInput{
		Address: vClient.Address(),
		Token:   vClient.Token(),
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
