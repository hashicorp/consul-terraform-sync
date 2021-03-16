package testutils

import (
	"io/ioutil"
	"log"
	"path/filepath"
	"testing"

	"github.com/hashicorp/consul/sdk/testutil"
)

func NewTestConsulServerHTTPS(tb testing.TB, path string) *testutil.TestServer {
	log.SetOutput(ioutil.Discard)

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
	if err != nil {
		tb.Fatalf("unable to start Consul test server: %s", err)
	}

	return srv
}
