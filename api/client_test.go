package api

import (
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultClientConfig_WithOSEnvSet(t *testing.T) {
	url := "https://1.2.3.4:5678"
	caCert := "test/path/ca.pem"
	caPath := "test/path"
	clientCert := "test/path/client.pem"
	clientKey := "test/path/key.pem"
	sslVerify := "false"

	err := os.Setenv(EnvAddress, url)
	require.NoError(t, err)
	defer os.Setenv(EnvAddress, "")

	err = os.Setenv(EnvTLSCACert, caCert)
	require.NoError(t, err)
	defer os.Setenv(EnvTLSCACert, "")

	err = os.Setenv(EnvTLSCAPath, caPath)
	require.NoError(t, err)
	defer os.Setenv(EnvTLSCAPath, "")

	err = os.Setenv(EnvTLSClientCert, clientCert)
	require.NoError(t, err)
	defer os.Setenv(EnvTLSClientCert, "")

	err = os.Setenv(EnvTLSClientKey, clientKey)
	require.NoError(t, err)
	defer os.Setenv(EnvTLSClientKey, "")

	os.Setenv(EnvTLSSSLVerify, sslVerify)
	require.NoError(t, err)
	defer os.Setenv(EnvTLSSSLVerify, "")

	config := DefaultClientConfig()

	assert.Equal(t, url, config.Addr)
	assert.Equal(t, caCert, config.TLSConfig.CACert)
	assert.Equal(t, caPath, config.TLSConfig.CAPath)
	assert.Equal(t, clientCert, config.TLSConfig.ClientCert)
	assert.Equal(t, clientKey, config.TLSConfig.ClientKey)

	expectedSSLVerify, err := strconv.ParseBool(sslVerify)
	require.NoError(t, err)
	assert.Equal(t, expectedSSLVerify, config.TLSConfig.SSLVerify)
}

func TestDefaultClientConfig_OnlyDefaults(t *testing.T) {
	caCert := ""
	caPath := ""
	clientCert := ""
	clientKey := ""

	config := DefaultClientConfig()

	assert.Equal(t, DefaultAddress, config.Addr)
	assert.Equal(t, caCert, config.TLSConfig.CACert)
	assert.Equal(t, caPath, config.TLSConfig.CAPath)
	assert.Equal(t, clientCert, config.TLSConfig.ClientCert)
	assert.Equal(t, clientKey, config.TLSConfig.ClientKey)
	assert.Equal(t, DefaultSSLVerify, config.TLSConfig.SSLVerify)
}
