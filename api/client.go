package api

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/go-rootcerts"
)

//go:generate mockery --name=httpClient  --structname=HttpClient --output=../mocks/api

const (
	HTTPScheme  = "http"
	HTTPSScheme = "https"

	DefaultAddress   = "http://localhost:8558"
	DefaultSSLVerify = true

	// Environment variable names
	EnvAddress = "CTS_ADDRESS" // The address of the CTS daemon, supports http or https by specifying as part of the address (e.g. https://localhost:8558)

	// TLS environment variable names
	EnvTLSCACert     = "CTS_CACERT"      // Path to a directory of CA certificates to use for TLS when communicating with Consul-Terraform-Sync
	EnvTLSCAPath     = "CTS_CAPATH"      // Path to a CA file to use for TLS when communicating with Consul-Terraform-Sync
	EnvTLSClientCert = "CTS_CLIENT_CERT" // Path to a client cert file to use for TLS when verify_incoming is enabled
	EnvTLSClientKey  = "CTS_CLIENT_KEY"  // Path to a client key file to use for TLS when verify_incoming is enabled
	EnvTLSSSLVerify  = "CTS_SSL_VERIFY"  // Boolean to verify SSL or not. Set to true to verify SSL. Default is true
)

// httpClient describes the interface for the client to make http calls
type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// Client to make api requests
type Client struct {
	port    int // remain for backwards compatibility but prefer addr
	addr    string
	version string
	scheme  string
	http    httpClient
}

// ClientConfig configures the client to make api requests
type ClientConfig struct {
	Port      int // Stay for now for backwards compatibility, but prefer Addr
	Addr      string
	Scheme    string
	TLSConfig TLSConfig
}

type TLSConfig struct {
	// TLS variables
	CAPath     string
	CACert     string
	ClientCert string
	ClientKey  string
	SSLVerify  bool
}

type addressComposite struct {
	scheme  string
	address string
}

// DefaultClientConfig returns a default configuration for the client
func DefaultClientConfig() *ClientConfig {
	c := &ClientConfig{
		Port: config.DefaultPort,
		Addr: DefaultAddress,
		TLSConfig: TLSConfig{
			SSLVerify: true,
		},
	}

	// Update configs from env vars
	if value, found := os.LookupEnv(EnvAddress); found {
		c.Addr = value
	}

	// Update TLS configs from env vars
	if value, found := os.LookupEnv(EnvTLSCACert); found {
		c.TLSConfig.CACert = value
	}

	if value, found := os.LookupEnv(EnvTLSCAPath); found {
		c.TLSConfig.CAPath = value
	}

	if value, found := os.LookupEnv(EnvTLSClientCert); found {
		c.TLSConfig.ClientCert = value
	}

	if value, found := os.LookupEnv(EnvTLSClientKey); found {
		c.TLSConfig.ClientKey = value
	}

	if value, found := os.LookupEnv(EnvTLSSSLVerify); found {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			c.TLSConfig.SSLVerify = boolValue
		}
	}

	if value, found := os.LookupEnv(EnvTLSSSLVerify); found {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			c.TLSConfig.SSLVerify = boolValue
		}
	}

	return c
}

// NewClient returns a client to make api requests
func NewClient(c *ClientConfig, httpClient httpClient) (*Client, error) {
	if httpClient == nil {
		tlsConfig, err := setupTLSConfig(c)
		if err != nil {
			return nil, err
		}

		httpClient = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: tlsConfig,
			},
		}
	}

	// Determine the scheme and address without scheme based on the address passed in
	ac, err := parseAddress(c.Addr)
	if err != nil {
		return nil, err
	}

	return &Client{
		port:    c.Port,
		addr:    ac.address,
		version: defaultAPIVersion,
		scheme:  ac.scheme,
		http:    httpClient,
	}, nil
}

// setupTLSConfig is used to generate a TLSClientConfig that's useful for talking to
// Consul using TLS.
func setupTLSConfig(c *ClientConfig) (*tls.Config, error) {
	tlsClientConfig := &tls.Config{
		// If verify is false, then we set skip verify to true
		// InsecureSkipVerify will always be the opposite of SSLVerify
		InsecureSkipVerify: !c.TLSConfig.SSLVerify,
	}

	if c.TLSConfig.ClientCert != "" && c.TLSConfig.ClientKey != "" {
		tlsCert, err := tls.LoadX509KeyPair(c.TLSConfig.ClientCert, c.TLSConfig.ClientKey)
		if err != nil {
			return nil, err
		}

		tlsClientConfig.Certificates = []tls.Certificate{tlsCert}
	} else if c.TLSConfig.ClientCert != "" || c.TLSConfig.ClientKey != "" {
		return nil, fmt.Errorf("both client cert and client key must be provided")
	}

	if c.TLSConfig.CACert != "" || c.TLSConfig.CAPath != "" {
		rootConfig := &rootcerts.Config{
			CAFile: c.TLSConfig.CACert,
			CAPath: c.TLSConfig.CAPath,
		}

		if err := rootcerts.ConfigureTLS(tlsClientConfig, rootConfig); err != nil {
			return nil, err
		}
	}

	return tlsClientConfig, nil
}

// Port returns the port being used by the client
func (c *Client) Port() int {
	return c.port
}

// FullAddress returns the client address including the scheme. E.g. http://localhost:8558
func (c *Client) FullAddress() string {
	return fmt.Sprintf("%s://%s", c.scheme, c.addr)
}

// Scheme returns the scheme being used by the client
func (c *Client) Scheme() string {
	return c.scheme
}

// WaitForAPI polls the /v1/status endpoint to check when the CTS API is
// available. The API is started when CTS is run as a daemon and completes
// all tasks once.
func (c *Client) WaitForAPI(timeout time.Duration) error {
	polling := make(chan struct{})
	stopPolling := make(chan struct{})
	statusAPI := c.Status()

	go func() {
		var err error
		for {
			select {
			case <-stopPolling:
				return
			default:
				_, err = statusAPI.Overall()
				if err == nil {
					polling <- struct{}{}
					return
				}
			}
		}
	}()

	select {
	case <-polling:
		return nil
	case <-time.After(timeout):
		close(stopPolling)
		return fmt.Errorf("client timed out waiting for CTS API to start at %s: %v", c.addr, timeout)
	}
}

// request makes the http request on behalf of the client. Handles status code
// check and any necessary error parsing. Returns a response body only if status
// code is OK. Caller is responsible for closing returned response if error is
// nil i.e. `defer resp.Body.Close()`
//
// path: relative path with no preceding '/' e.g. "status/tasks"
// query: URL encoded query string with no preceding '?'. See QueryParam.Encode()
func (c *Client) request(method, path, query, body string) (*http.Response, error) {
	var serverURL url.URL

	// If port is default, use the address variable instead
	if c.port == config.DefaultPort {
		serverURL = url.URL{
			Scheme:   c.scheme,
			Host:     c.addr,
			Path:     fmt.Sprintf("%s/%s", c.version, path),
			RawQuery: query,
		}
	} else {
		// If port is set, assume using old arguments and append port to localhost
		// assume http scheme
		serverURL = url.URL{
			Scheme:   c.scheme,
			Host:     fmt.Sprintf("localhost:%d", c.port),
			Path:     fmt.Sprintf("%s/%s", c.version, path),
			RawQuery: query,
		}
	}

	r := strings.NewReader(body)
	req, err := http.NewRequest(method, serverURL.String(), r)
	if err != nil {
		return nil, err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	// defer resp.Body.Close() not called for happy path, only called for
	// unhappy path. caller of this method will close if returned err == nil.

	if resp.StatusCode >= 400 {
		defer resp.Body.Close()

		var errResp ErrorResponse
		decoder := json.NewDecoder(resp.Body)
		if err = decoder.Decode(&errResp); err != nil {
			return nil, err
		}

		if msg, ok := errResp.ErrorMessage(); ok && msg != "" {
			return nil, fmt.Errorf("request returned %d status code with error: %s",
				resp.StatusCode, msg)
		}

		return nil, fmt.Errorf("request returned %d status code", resp.StatusCode)
	}

	return resp, nil
}

// QueryParam sets query parameters for the api client
type QueryParam struct {
	IncludeEvents bool
	Status        string
	Run           string
}

// Encode returns QueryParameter values as a URL encoded string. No preceding '?'
// e.g. "include=events&status=critical"
func (q *QueryParam) Encode() string {
	val := url.Values{}
	if q.IncludeEvents {
		val.Set("include", "events") // refactor this out?
	}

	if q.Status != "" {
		val.Set("status", q.Status)
	}

	if q.Run != "" {
		val.Set("run", q.Run)
	}

	return val.Encode()
}

// StatusClient can be used to query the status endpoints
type StatusClient struct {
	*Client
}

// Status returns a handle to the status endpoints
func (c *Client) Status() *StatusClient {
	return &StatusClient{c}
}

// Overall is used to query for overall status
func (s *StatusClient) Overall() (OverallStatus, error) {
	var overallStatus OverallStatus

	resp, err := s.request(http.MethodGet, overallStatusPath, "", "")
	if err != nil {
		return overallStatus, err
	}
	defer resp.Body.Close()

	decoder := json.NewDecoder(resp.Body)
	if err = decoder.Decode(&overallStatus); err != nil {
		return overallStatus, err
	}

	return overallStatus, nil
}

// Task is used to query for task status(es).
//
// name: task name or empty string for all tasks
// q: nil if no query parameters
func (s *StatusClient) Task(name string, q *QueryParam) (map[string]TaskStatus, error) {
	var taskStatuses map[string]TaskStatus

	path := taskStatusPath
	if name != "" {
		path += fmt.Sprintf("/%s", name)
	}

	if q == nil {
		q = &QueryParam{}
	}

	resp, err := s.request(http.MethodGet, path, q.Encode(), "")
	if err != nil {
		return taskStatuses, err
	}
	defer resp.Body.Close()

	decoder := json.NewDecoder(resp.Body)
	if err = decoder.Decode(&taskStatuses); err != nil {
		return taskStatuses, err
	}

	return taskStatuses, nil
}

// TaskClient can be used to query the task endpoints
type TaskClient struct {
	*Client
}

// Task returns a handle to the task endpoints
func (c *Client) Task() *TaskClient {
	return &TaskClient{c}
}

// Update is used to patch update task
func (t *TaskClient) Update(name string, config UpdateTaskConfig, q *QueryParam) (UpdateTaskResponse, error) {
	b, err := json.Marshal(config)
	if err != nil {
		return UpdateTaskResponse{}, err
	}

	if q == nil {
		q = &QueryParam{}
	}

	path := fmt.Sprintf("%s/%s", taskPath, name)
	resp, err := t.request(http.MethodPatch, path, q.Encode(), string(b))
	if err != nil {
		return UpdateTaskResponse{}, err
	}
	defer resp.Body.Close()

	decoder := json.NewDecoder(resp.Body)
	var plan UpdateTaskResponse
	if err = decoder.Decode(&plan); err != nil {
		return UpdateTaskResponse{}, err
	}

	return plan, nil
}

func parseAddress(addr string) (addressComposite, error) {
	ac := addressComposite{}
	ac.scheme = HTTPScheme
	parts := strings.SplitN(addr, "://", 2)
	if len(parts) == 2 {
		switch parts[0] {
		case HTTPScheme:
			ac.scheme = HTTPScheme
		case HTTPSScheme:
			ac.scheme = HTTPSScheme
		default:
			return addressComposite{}, fmt.Errorf("unknown protocol scheme: %s", parts[0])
		}
		ac.address = parts[1]
	}

	return ac, nil
}
