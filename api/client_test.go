package api

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"net/url"
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

	urlString := "https://1.2.3.4:5678"
	caCert := "test/path/ca.pem"
	caPath := "test/path"
	clientCert := "test/path/client.pem"
	clientKey := "test/path/key.pem"
	sslVerify := "false"

	require.NoError(t, os.Setenv(EnvAddress, urlString))
	require.NoError(t, os.Setenv(EnvTLSCACert, caCert))
	require.NoError(t, os.Setenv(EnvTLSCAPath, caPath))
	require.NoError(t, os.Setenv(EnvTLSClientCert, clientCert))
	require.NoError(t, os.Setenv(EnvTLSClientKey, clientKey))
	require.NoError(t, os.Setenv(EnvTLSSSLVerify, sslVerify))

	clientConfig := DefaultClientConfig()
	u, err := url.Parse(DefaultURL)
	require.NoError(t, err)

	assert.Equal(t, u, clientConfig.URL)
	assert.Equal(t, caCert, clientConfig.TLSConfig.CACert)
	assert.Equal(t, caPath, clientConfig.TLSConfig.CAPath)
	assert.Equal(t, clientCert, clientConfig.TLSConfig.ClientCert)
	assert.Equal(t, clientKey, clientConfig.TLSConfig.ClientKey)

	expectedSSLVerify, err := strconv.ParseBool(sslVerify)
	assert.NoError(t, err)
	assert.Equal(t, expectedSSLVerify, clientConfig.TLSConfig.SSLVerify)
}

func Test_DefaultClientConfig_Defaults(t *testing.T) {
	caCert := ""
	caPath := ""
	clientCert := ""
	clientKey := ""

	clientConfig := DefaultClientConfig()
	u, err := url.Parse(DefaultURL)
	require.NoError(t, err)

	assert.Equal(t, u, clientConfig.URL)
	assert.Equal(t, caCert, clientConfig.TLSConfig.CACert)
	assert.Equal(t, caPath, clientConfig.TLSConfig.CAPath)
	assert.Equal(t, clientCert, clientConfig.TLSConfig.ClientCert)
	assert.Equal(t, clientKey, clientConfig.TLSConfig.ClientKey)
	assert.Equal(t, DefaultSSLVerify, clientConfig.TLSConfig.SSLVerify)
}

func Test_DefaultClientConfig_InvalidAddressEnv(t *testing.T) {
	t.Cleanup(func() {
		_ = os.Unsetenv(EnvAddress)
	})

	configBeforeEnvVarSet := DefaultClientConfig()
	assert.NoError(t, os.Setenv(EnvAddress, "invalid address"))
	configAfterEnvVarSet := DefaultClientConfig()

	assert.EqualValues(t, configBeforeEnvVarSet.URL, configAfterEnvVarSet.URL)
}

func Test_ParseDefaultURL(t *testing.T) {
	u, err := url.Parse(DefaultURL)
	assert.NotNil(t, u)
	assert.NoError(t, err)
}

func Test_ClientPort(t *testing.T) {
	expectedPort := 1234
	c := &Client{url: &url.URL{Scheme: "http", Host: fmt.Sprintf("localhost:%d", expectedPort)}}

	assert.Equal(t, expectedPort, c.Port())
}

func Test_ClientScheme(t *testing.T) {
	expectedScheme := "foo"
	c := &Client{url: &url.URL{Scheme: expectedScheme}}

	assert.Equal(t, expectedScheme, c.Scheme())
}

func Test_ClientFullAddress(t *testing.T) {
	scheme := "foo"
	host := "bar"
	u := &url.URL{Scheme: scheme, Host: host}
	c := &Client{url: u}

	assert.Equal(t, u.String(), c.FullAddress())
}

func Test_ClientTask(t *testing.T) {
	c := &Client{}
	assert.NotNil(t, c.Task())
}

func Test_NewClient_InvalidScheme(t *testing.T) {
	clientConfig := DefaultClientConfig()
	clientConfig.URL.Scheme = "foo"
	c, err := NewClient(clientConfig, nil)

	assert.Nil(t, c)
	assert.Error(t, err)
}
