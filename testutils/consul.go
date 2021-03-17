package testutils

import (
	"io/ioutil"
	"log"
	"path/filepath"
	"testing"

	"github.com/hashicorp/consul/sdk/testutil"
	"github.com/stretchr/testify/require"
)

func NewTestConsulServerHTTPS(tb testing.TB, relPath string) *testutil.TestServer {
	log.SetOutput(ioutil.Discard)

	path, err := filepath.Abs(relPath)
	require.NoError(tb, err, "unable to get absolute path of test certs")
	certFile := filepath.Join(path, "cert.pem")
	keyFile := filepath.Join(path, "key.pem")

	srv, err := testutil.NewTestServerConfigT(tb,
		func(c *testutil.TestServerConfig) {
			c.LogLevel = "warn"
			c.Stdout = ioutil.Discard
			c.Stderr = ioutil.Discard

			// Support CTS connecting over HTTP2
			c.VerifyIncomingHTTPS = false
			c.CertFile = certFile
			c.KeyFile = keyFile
		})
	require.NoError(tb, err, "unable to start Consul server")

	return srv
}
