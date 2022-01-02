package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hashicorp/consul-terraform-sync/api/oapigen"
	"github.com/hashicorp/consul-terraform-sync/config"
)

// TaskLifecycleClient defines a client for task lifecycle requests
// Currently non task lifecycle requests use the client in api/client.go, but eventually all endpoint
// may use this new client. In that case TaskLifecycleClient should be renamed
type TaskLifecycleClient struct {
	port   int
	scheme string
	addr   string
	*oapigen.Client
}

//go:generate oapi-codegen  -package oapigen -generate client -o oapigen/client.go openapi.yaml

// NewTaskLifecycleClient returns a client to make api requests
func NewTaskLifecycleClient(c *ClientConfig, httpClient httpClient) (*TaskLifecycleClient, error) {
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
		httpClient = NewTaskLifecycleHTTPClient(httpClient)
	}

	// Determine the scheme and address without scheme based on the address passed in
	ac, err := parseAddress(c.Addr)
	if err != nil {
		return nil, err
	}

	gc := &TaskLifecycleClient{
		port:   c.Port,
		scheme: ac.scheme,
		addr:   ac.address,
	}

	// If port is set, assume using old arguments and append port to localhost
	// assume http scheme
	server := gc.FullAddress()
	if c.Port != config.DefaultPort {
		server = fmt.Sprintf("localhost:%d", c.Port)
	}

	// Create the new underlying client based on generated code
	oc, err := oapigen.NewClient(server, oapigen.WithHTTPClient(httpClient))
	if err != nil {
		return nil, err
	}

	gc.Client = oc

	return gc, nil
}

// Port returns the port being used by the client
func (c *TaskLifecycleClient) Port() int {
	return c.port
}

// FullAddress returns the client address including the scheme. eg. http://localhost:8558
func (c *TaskLifecycleClient) FullAddress() string {
	return fmt.Sprintf("%s://%s", c.scheme, c.addr)
}

// Scheme returns the scheme being used by the client
func (c *TaskLifecycleClient) Scheme() string {
	return fmt.Sprintf(c.scheme)
}

var _ httpClient = (*TaskLifecycleHTTPClient)(nil)

// TaskLifecycleHTTPClient is an httpClient for task life cycle requests and
// allows for wrapping the base http.Do function
type TaskLifecycleHTTPClient struct {
	http httpClient
}

// NewTaskLifecycleHTTPClient creates a new http client wrapper
// around the passed httpClient
func NewTaskLifecycleHTTPClient(httpClient httpClient) *TaskLifecycleHTTPClient {
	return &TaskLifecycleHTTPClient{
		http: httpClient,
	}
}

// Do performs the http request and parses the response to report any errors
func (d *TaskLifecycleHTTPClient) Do(req *http.Request) (*http.Response, error) {
	resp, err := d.http.Do(req)
	if err != nil {
		return nil, err
	}
	// defer resp.Body.Close() not called for happy path, only called for
	// unhappy path. caller of this method will close if returned err == nil.

	if resp.StatusCode != http.StatusOK {
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
