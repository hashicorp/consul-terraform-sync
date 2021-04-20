package testutils

import (
	"io/ioutil"
	"log"
	"path/filepath"
	"testing"

	"github.com/hashicorp/consul/sdk/testutil"
	"github.com/stretchr/testify/require"
)

// TestConsulServerConfig configures a test Consul server
type TestConsulServerConfig struct {
	HTTPSRelPath string
	PortHTTPS    int // random port will be generated if unset
}

// NewTestConsulServer starts a test Consul server as configured
func NewTestConsulServer(tb testing.TB, config TestConsulServerConfig) *testutil.TestServer {
	log.SetOutput(ioutil.Discard)

	var certFile string
	var keyFile string
	if config.HTTPSRelPath != "" {
		path, err := filepath.Abs(config.HTTPSRelPath)
		require.NoError(tb, err, "unable to get absolute path of test certs")
		certFile = filepath.Join(path, "cert.pem")
		keyFile = filepath.Join(path, "key.pem")
	}

	srv, err := testutil.NewTestServerConfigT(tb,
		func(c *testutil.TestServerConfig) {
			c.LogLevel = "warn"
			c.Stdout = ioutil.Discard
			c.Stderr = ioutil.Discard

			// Support CTS connecting over HTTP2
			if config.HTTPSRelPath != "" {
				c.VerifyIncomingHTTPS = false
				c.CertFile = certFile
				c.KeyFile = keyFile
			}

			if config.PortHTTPS != 0 {
				c.Ports.HTTPS = config.PortHTTPS
			}
		})
	require.NoError(tb, err, "unable to start Consul server")

	return srv
}
