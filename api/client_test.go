package api

import (
	"math/rand"
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_DefaultClientConfig_WithEnvVars(t *testing.T) {
	t.Cleanup(func() {
		_ = os.Unsetenv(EnvAddress)
		_ = os.Unsetenv(EnvTLSCACert)
		_ = os.Unsetenv(EnvTLSCAPath)
		_ = os.Unsetenv(EnvTLSClientCert)
		_ = os.Unsetenv(EnvTLSClientKey)
		_ = os.Unsetenv(EnvTLSSSLVerify)
	})

	url := "https://1.2.3.4:5678"
	caCert := "test/path/ca.pem"
	caPath := "test/path"
	clientCert := "test/path/client.pem"
	clientKey := "test/path/key.pem"
	sslVerify := "false"

	require.NoError(t, os.Setenv(EnvAddress, url))
	require.NoError(t, os.Setenv(EnvTLSCACert, caCert))
	require.NoError(t, os.Setenv(EnvTLSCAPath, caPath))
	require.NoError(t, os.Setenv(EnvTLSClientCert, clientCert))
	require.NoError(t, os.Setenv(EnvTLSClientKey, clientKey))
	require.NoError(t, os.Setenv(EnvTLSSSLVerify, sslVerify))

	clientConfig := DefaultClientConfig()

	require.Equal(t, url, clientConfig.Addr)
	require.Equal(t, caCert, clientConfig.TLSConfig.CACert)
	require.Equal(t, caPath, clientConfig.TLSConfig.CAPath)
	require.Equal(t, clientCert, clientConfig.TLSConfig.ClientCert)
	require.Equal(t, clientKey, clientConfig.TLSConfig.ClientKey)

	expectedSSLVerify, err := strconv.ParseBool(sslVerify)
	require.NoError(t, err)
	require.Equal(t, expectedSSLVerify, clientConfig.TLSConfig.SSLVerify)
}

func Test_DefaultClientConfig_Defaults(t *testing.T) {
	caCert := ""
	caPath := ""
	clientCert := ""
	clientKey := ""

	clientConfig := DefaultClientConfig()

	require.Equal(t, DefaultAddress, clientConfig.Addr)
	require.Equal(t, caCert, clientConfig.TLSConfig.CACert)
	require.Equal(t, caPath, clientConfig.TLSConfig.CAPath)
	require.Equal(t, clientCert, clientConfig.TLSConfig.ClientCert)
	require.Equal(t, clientKey, clientConfig.TLSConfig.ClientKey)
	require.Equal(t, DefaultSSLVerify, clientConfig.TLSConfig.SSLVerify)
}

func Test_ClientPort(t *testing.T) {
	expectedPort := rand.Intn(10000)
	c := &Client{port: expectedPort}

	require.Equal(t, expectedPort, c.Port())
}

func Test_ClientScheme(t *testing.T) {
	expectedScheme := "foo"
	c := &Client{scheme: expectedScheme}

	require.Equal(t, expectedScheme, c.Scheme())
}

func Test_ClientFullAddress(t *testing.T) {
	scheme := "foo"
	address := "bar"
	expectedFullAddress := "foo://bar"
	c := &Client{scheme: scheme, addr: address}

	require.Equal(t, expectedFullAddress, c.FullAddress())
}

func Test_ClientTask(t *testing.T) {
	c := &Client{}

	require.NotNil(t, c.Task())
}
