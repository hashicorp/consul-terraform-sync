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

	cases := []struct {
		name        string
		expectError bool
		conf        *config.Config
	}{
		{
			"happy path",
			false,
			singleTaskConfig(),
		},
		{
			"unreachable consul server", // can take >63s locally
			true,
			singleTaskConfig(),
		},
		{
			"unsupported driver error",
			true,
			&config.Config{
				Driver: &config.DriverConfig{},
			},
		},
	}
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

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.expectError == false {
				tc.conf.Consul.Address = &addr
				tc.conf.Finalize()
			}

			t.Run("readwrite", func(t *testing.T) {
				controller, err := NewReadWrite(tc.conf)
				if tc.expectError {
					assert.Error(t, err)
					return
				}
				assert.NoError(t, err)
				assert.NotNil(t, controller)
			})
			t.Run("readonly", func(t *testing.T) {
				controller, err := NewReadOnly(tc.conf)
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
