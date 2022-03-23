package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/hashicorp/consul-terraform-sync/api/oapigen"
)

// TaskLifecycleClient defines a client for task lifecycle requests
// Currently non task lifecycle requests use the client in api/client.go, but eventually all endpoint
// may use this new client. In that case TaskLifecycleClient should be renamed
type TaskLifecycleClient struct {
	url *url.URL
	*oapigen.ClientWithResponses
}

var _ oapigen.ClientWithResponsesInterface = (*TaskLifecycleClient)(nil)

//go:generate mockery --recursive --name=ClientWithResponsesInterface --filename=oapigen_client.go --output=../mocks/api
//go:generate oapi-codegen  -package oapigen -generate client -o oapigen/client.go openapi.yaml

// NewTaskLifecycleClient returns a client to make api requests
func NewTaskLifecycleClient(c *ClientConfig, httpClient httpClient) (*TaskLifecycleClient, error) {
	if httpClient == nil {
		h, err := newHTTPClient(&c.TLSConfig)
		if err != nil {
			return nil, err
		}

		httpClient = NewTaskLifecycleHTTPClient(h)
	}

	u, err := parseURL(c.URL)
	if err != nil {
		return nil, err
	}

	gc := &TaskLifecycleClient{url: u}

	// Create the new underlying client based on generated code
	oc, err := oapigen.NewClientWithResponses(gc.url.String(), oapigen.WithHTTPClient(httpClient))
	if err != nil {
		return nil, err
	}

	gc.ClientWithResponses = oc

	return gc, nil
}

// Scheme returns the scheme being used by the client
func (c *TaskLifecycleClient) Scheme() string {
	return c.url.Scheme
}

// CreateTaskSimple takes a task request and run option and sends this information to the client. It then returns
// a task response object and any errors to the caller.
// TODO: remove this to conform to interface
func (c *TaskLifecycleClient) CreateTask(ctx context.Context, runOption string, req TaskRequest) (*oapigen.CreateTaskResponse, error) {
	var run oapigen.CreateTaskParamsRun
	switch runOption {
	case RunOptionInspect:
		run = RunOptionInspect
	case RunOptionNow:
		run = RunOptionNow
	case "":
		run = ""
	default:
		err := errors.New("invalid run option provided")
		return nil, err
	}

	resp, err := c.CreateTaskWithResponse(ctx, &oapigen.CreateTaskParams{Run: &run}, oapigen.CreateTaskJSONRequestBody(req))
	if err != nil {
		return nil, err
	}

	return resp, nil
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

	if checkStatusCodeCategory(ClientErrorResponseCategory, resp.StatusCode) ||
		checkStatusCodeCategory(ServerErrorResponseCategory, resp.StatusCode) {
		defer resp.Body.Close()

		// Nominal scenario is an error will be application/json type, if not application/json assume that the error
		// is plaintext
		var errMsg string
		if resp.Header.Get("Content-Type") == "application/json" {
			var errResp oapigen.ErrorResponse
			decoder := json.NewDecoder(resp.Body)
			if err = decoder.Decode(&errResp); err != nil {
				return nil, err
			}
			errMsg = fmt.Sprintf("%s, see logs for more details (Request ID: %s)", errResp.Error.Message, errResp.RequestId)
		} else {
			b, err := io.ReadAll(resp.Body)
			if err != nil {
				return nil, err
			}
			errMsg = string(b)
		}

		if errMsg == "" {
			return nil, errors.New(resp.Status)
		} else {
			return nil, fmt.Errorf("%s: %s", resp.Status, errMsg)
		}
	}

	return resp, nil
}
