package controller

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewControllers(t *testing.T) {
	t.Parallel()

	// fake consul server
	ts := httptest.NewUnstartedServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, `"test"`) }))
	var err error
	ts.Listener, err = net.Listen("tcp", ":0")
	require.NoError(t, err)
	port := ts.Listener.Addr().(*net.TCPAddr).Port
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	ts.Start()
	defer ts.Close()

	cases := []struct {
		name        string
		expectError bool
		setupConf   func() *config.Config
	}{
		{
			"happy path",
			false,
			func() *config.Config {
				conf := singleTaskConfig()
				conf.Consul.Address = &addr
				conf.Finalize()
				return conf
			},
		},
		{
			"unreachable consul server", // can take >63s locally
			true,
			func() *config.Config {
				// Consul address not set
				return singleTaskConfig()
			},
		},
		{
			"unsupported driver error",
			true,
			func() *config.Config {
				conf := config.DefaultConfig()
				conf.Finalize()
				// override driver config
				conf.Driver = &config.DriverConfig{}
				return conf
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Run("readwrite", func(t *testing.T) {
				controller, err := NewReadWrite(tc.setupConf())
				if tc.expectError {
					assert.Error(t, err)
					return
				}
				assert.NoError(t, err)
				assert.NotNil(t, controller)
			})
			t.Run("readonly", func(t *testing.T) {
				controller, err := NewReadOnly(tc.setupConf())
				if tc.expectError {
					assert.Error(t, err)
					return
				}
				assert.NoError(t, err)
				assert.NotNil(t, controller)
			})
		})
	}
}
