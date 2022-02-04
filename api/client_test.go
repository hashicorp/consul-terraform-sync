package api

import (
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_ParseDefaultURL(t *testing.T) {
	// ensures the default URL constant is a valid URL
	u, err := url.ParseRequestURI(DefaultURL)
	assert.NotNil(t, u)
	assert.NoError(t, err)
}

func Test_BaseClientConfig_Defaults(t *testing.T) {
	caCert := ""
	caPath := ""
	clientCert := ""
	clientKey := ""

	clientConfig := BaseClientConfig()

	assert.Equal(t, DefaultURL, clientConfig.URL)
	assert.Equal(t, caCert, clientConfig.TLSConfig.CACert)
	assert.Equal(t, caPath, clientConfig.TLSConfig.CAPath)
	assert.Equal(t, clientCert, clientConfig.TLSConfig.ClientCert)
	assert.Equal(t, clientKey, clientConfig.TLSConfig.ClientKey)
	assert.Equal(t, DefaultSSLVerify, clientConfig.TLSConfig.SSLVerify)
}

func Test_BaseClientConfig_WithEnvVars(t *testing.T) {
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

	clientConfig := BaseClientConfig()

	assert.Equal(t, urlString, clientConfig.URL)
	assert.Equal(t, caCert, clientConfig.TLSConfig.CACert)
	assert.Equal(t, caPath, clientConfig.TLSConfig.CAPath)
	assert.Equal(t, clientCert, clientConfig.TLSConfig.ClientCert)
	assert.Equal(t, clientKey, clientConfig.TLSConfig.ClientKey)

	expectedSSLVerify, err := strconv.ParseBool(sslVerify)
	assert.NoError(t, err)
	assert.Equal(t, expectedSSLVerify, clientConfig.TLSConfig.SSLVerify)
}

func Test_Client_Port(t *testing.T) {
	expectedPort := 1234
	c := &Client{url: &url.URL{Host: fmt.Sprintf("localhost:%d", expectedPort)}}

	assert.Equal(t, expectedPort, c.Port())
}

func Test_Client_MissingPort(t *testing.T) {
	c := &Client{url: &url.URL{Host: "localhost"}}
	assert.Equal(t, -1, c.Port())
}

func Test_Client_Scheme(t *testing.T) {
	expectedScheme := "foo"
	c := &Client{url: &url.URL{Scheme: expectedScheme}}

	assert.Equal(t, expectedScheme, c.Scheme())
}

func Test_Client_FullAddress(t *testing.T) {
	scheme := "foo"
	host := "bar:1234"
	u := &url.URL{Scheme: scheme, Host: host}
	c := &Client{url: u}

	assert.Equal(t, u.String(), c.FullAddress())
}

func Test_Client_Task(t *testing.T) {
	c := &Client{}
	assert.NotNil(t, c.Task())
}

func Test_NewClient(t *testing.T) {
	clientConfig := BaseClientConfig()
	c, err := NewClient(clientConfig, nil)

	assert.NotNil(t, c)
	assert.NoError(t, err)
}

func Test_NewClient_InvalidScheme(t *testing.T) {
	clientConfig := &ClientConfig{URL: "foo://bar"}
	c, err := NewClient(clientConfig, nil)

	assert.Nil(t, c)
	assert.Error(t, err)
}

func Test_NewClient_InvalidURL(t *testing.T) {
	clientConfig := &ClientConfig{URL: "invalid URL"}
	c, err := NewClient(clientConfig, nil)

	assert.Nil(t, c)
	assert.Error(t, err)
}

func Test_NewClient_MissingHost(t *testing.T) {
	clientConfig := &ClientConfig{URL: "http://"}
	c, err := NewClient(clientConfig, nil)

	assert.Nil(t, c)
	assert.Error(t, err)
}

func Test_Client_Status(t *testing.T) {
	c := &Client{}
	assert.NotNil(t, c.Status())
}

func Test_QueryParam_Encode(t *testing.T) {
	type fields struct {
		IncludeEvents bool
		Status        string
		Run           string
	}

	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name:   "empty",
			fields: fields{},
			want:   "",
		},
		{
			name:   "single field",
			fields: fields{Status: "foo"},
			want:   "status=foo",
		},
		{
			name:   "multiple fields",
			fields: fields{Status: "foo", Run: "bar"},
			want:   "run=bar&status=foo",
		},
		{
			name:   "multiple fields with include events",
			fields: fields{Status: "foo", Run: "bar", IncludeEvents: true},
			want:   "include=events&run=bar&status=foo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := &QueryParam{
				IncludeEvents: tt.fields.IncludeEvents,
				Status:        tt.fields.Status,
				Run:           tt.fields.Run,
			}
			assert.Equalf(t, tt.want, q.Encode(), "Encode()")
		})
	}
}

func Test_StatusClient_Overall_BadResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := fmt.Fprintf(w, "foo")
		assert.NoError(t, err)
	}))

	defer server.Close()

	clientConfig := BaseClientConfig()
	clientConfig.URL = server.URL
	c, err := NewClient(clientConfig, nil)
	assert.NoError(t, err)

	o, err := c.Status().Overall()
	assert.Error(t, err)
	assert.EqualValues(t, OverallStatus{}, o)
}

func Test_StatusClient_Overall_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))

	defer server.Close()

	clientConfig := BaseClientConfig()
	clientConfig.URL = server.URL
	c, err := NewClient(clientConfig, nil)
	assert.NoError(t, err)

	o, err := c.Status().Overall()
	assert.Error(t, err)
	assert.EqualValues(t, OverallStatus{}, o)
}

func Test_StatusClient_Overall(t *testing.T) {
	expectedOverallStatus := OverallStatus{TaskSummary: TaskSummary{
		Status:  StatusSummary{Successful: 1},
		Enabled: EnabledSummary{True: 2},
	}}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bytes, err := json.Marshal(&expectedOverallStatus)
		assert.NotEmpty(t, bytes)
		assert.NoError(t, err)

		_, err = fmt.Fprintf(w, string(bytes))
		assert.NoError(t, err)
	}))

	defer server.Close()

	clientConfig := BaseClientConfig()
	clientConfig.URL = server.URL
	c, err := NewClient(clientConfig, nil)
	assert.NoError(t, err)

	o, err := c.Status().Overall()
	assert.Nil(t, err)
	assert.Equal(t, expectedOverallStatus, o)
}
