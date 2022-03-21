// Package oapigen provides primitives to interact with the openapi HTTP API.
//
// Code generated by github.com/deepmap/oapi-codegen version v1.8.3 DO NOT EDIT.
package oapigen

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/deepmap/oapi-codegen/pkg/runtime"
)

// RequestEditorFn  is the function signature for the RequestEditor callback function
type RequestEditorFn func(ctx context.Context, req *http.Request) error

// Doer performs HTTP requests.
//
// The standard http.Client implements this interface.
type HttpRequestDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Client which conforms to the OpenAPI3 specification for this service.
type Client struct {
	// The endpoint of the server conforming to this interface, with scheme,
	// https://api.deepmap.com for example. This can contain a path relative
	// to the server, such as https://api.deepmap.com/dev-test, and all the
	// paths in the swagger spec will be appended to the server.
	Server string

	// Doer for performing requests, typically a *http.Client with any
	// customized settings, such as certificate chains.
	Client HttpRequestDoer

	// A list of callbacks for modifying requests which are generated before sending over
	// the network.
	RequestEditors []RequestEditorFn
}

// ClientOption allows setting custom parameters during construction
type ClientOption func(*Client) error

// Creates a new Client, with reasonable defaults
func NewClient(server string, opts ...ClientOption) (*Client, error) {
	// create a client with sane default values
	client := Client{
		Server: server,
	}
	// mutate client and add all optional params
	for _, o := range opts {
		if err := o(&client); err != nil {
			return nil, err
		}
	}
	// ensure the server URL always has a trailing slash
	if !strings.HasSuffix(client.Server, "/") {
		client.Server += "/"
	}
	// create httpClient, if not already present
	if client.Client == nil {
		client.Client = &http.Client{}
	}
	return &client, nil
}

// WithHTTPClient allows overriding the default Doer, which is
// automatically created using http.Client. This is useful for tests.
func WithHTTPClient(doer HttpRequestDoer) ClientOption {
	return func(c *Client) error {
		c.Client = doer
		return nil
	}
}

// WithRequestEditorFn allows setting up a callback function, which will be
// called right before sending the request. This can be used to mutate the request.
func WithRequestEditorFn(fn RequestEditorFn) ClientOption {
	return func(c *Client) error {
		c.RequestEditors = append(c.RequestEditors, fn)
		return nil
	}
}

// The interface specification for the client above.
type ClientInterface interface {
	// GetAllTasks request
	GetAllTasks(ctx context.Context, reqEditors ...RequestEditorFn) (*http.Response, error)

	// CreateTask request with any body
	CreateTaskWithBody(ctx context.Context, params *CreateTaskParams, contentType string, body io.Reader, reqEditors ...RequestEditorFn) (*http.Response, error)

	CreateTask(ctx context.Context, params *CreateTaskParams, body CreateTaskJSONRequestBody, reqEditors ...RequestEditorFn) (*http.Response, error)

	// DeleteTaskByName request
	DeleteTaskByName(ctx context.Context, name string, reqEditors ...RequestEditorFn) (*http.Response, error)

	// GetTaskByName request
	GetTaskByName(ctx context.Context, name string, reqEditors ...RequestEditorFn) (*http.Response, error)
}

func (c *Client) GetAllTasks(ctx context.Context, reqEditors ...RequestEditorFn) (*http.Response, error) {
	req, err := NewGetAllTasksRequest(c.Server)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	if err := c.applyEditors(ctx, req, reqEditors); err != nil {
		return nil, err
	}
	return c.Client.Do(req)
}

func (c *Client) CreateTaskWithBody(ctx context.Context, params *CreateTaskParams, contentType string, body io.Reader, reqEditors ...RequestEditorFn) (*http.Response, error) {
	req, err := NewCreateTaskRequestWithBody(c.Server, params, contentType, body)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	if err := c.applyEditors(ctx, req, reqEditors); err != nil {
		return nil, err
	}
	return c.Client.Do(req)
}

func (c *Client) CreateTask(ctx context.Context, params *CreateTaskParams, body CreateTaskJSONRequestBody, reqEditors ...RequestEditorFn) (*http.Response, error) {
	req, err := NewCreateTaskRequest(c.Server, params, body)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	if err := c.applyEditors(ctx, req, reqEditors); err != nil {
		return nil, err
	}
	return c.Client.Do(req)
}

func (c *Client) DeleteTaskByName(ctx context.Context, name string, reqEditors ...RequestEditorFn) (*http.Response, error) {
	req, err := NewDeleteTaskByNameRequest(c.Server, name)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	if err := c.applyEditors(ctx, req, reqEditors); err != nil {
		return nil, err
	}
	return c.Client.Do(req)
}

func (c *Client) GetTaskByName(ctx context.Context, name string, reqEditors ...RequestEditorFn) (*http.Response, error) {
	req, err := NewGetTaskByNameRequest(c.Server, name)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	if err := c.applyEditors(ctx, req, reqEditors); err != nil {
		return nil, err
	}
	return c.Client.Do(req)
}

// NewGetAllTasksRequest generates requests for GetAllTasks
func NewGetAllTasksRequest(server string) (*http.Request, error) {
	var err error

	serverURL, err := url.Parse(server)
	if err != nil {
		return nil, err
	}

	operationPath := fmt.Sprintf("/v1/tasks")
	if operationPath[0] == '/' {
		operationPath = "." + operationPath
	}

	queryURL, err := serverURL.Parse(operationPath)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("GET", queryURL.String(), nil)
	if err != nil {
		return nil, err
	}

	return req, nil
}

// NewCreateTaskRequest calls the generic CreateTask builder with application/json body
func NewCreateTaskRequest(server string, params *CreateTaskParams, body CreateTaskJSONRequestBody) (*http.Request, error) {
	var bodyReader io.Reader
	buf, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	bodyReader = bytes.NewReader(buf)
	return NewCreateTaskRequestWithBody(server, params, "application/json", bodyReader)
}

// NewCreateTaskRequestWithBody generates requests for CreateTask with any type of body
func NewCreateTaskRequestWithBody(server string, params *CreateTaskParams, contentType string, body io.Reader) (*http.Request, error) {
	var err error

	serverURL, err := url.Parse(server)
	if err != nil {
		return nil, err
	}

	operationPath := fmt.Sprintf("/v1/tasks")
	if operationPath[0] == '/' {
		operationPath = "." + operationPath
	}

	queryURL, err := serverURL.Parse(operationPath)
	if err != nil {
		return nil, err
	}

	queryValues := queryURL.Query()

	if params.Run != nil {

		if queryFrag, err := runtime.StyleParamWithLocation("form", true, "run", runtime.ParamLocationQuery, *params.Run); err != nil {
			return nil, err
		} else if parsed, err := url.ParseQuery(queryFrag); err != nil {
			return nil, err
		} else {
			for k, v := range parsed {
				for _, v2 := range v {
					queryValues.Add(k, v2)
				}
			}
		}

	}

	queryURL.RawQuery = queryValues.Encode()

	req, err := http.NewRequest("POST", queryURL.String(), body)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", contentType)

	return req, nil
}

// NewDeleteTaskByNameRequest generates requests for DeleteTaskByName
func NewDeleteTaskByNameRequest(server string, name string) (*http.Request, error) {
	var err error

	var pathParam0 string

	pathParam0, err = runtime.StyleParamWithLocation("simple", false, "name", runtime.ParamLocationPath, name)
	if err != nil {
		return nil, err
	}

	serverURL, err := url.Parse(server)
	if err != nil {
		return nil, err
	}

	operationPath := fmt.Sprintf("/v1/tasks/%s", pathParam0)
	if operationPath[0] == '/' {
		operationPath = "." + operationPath
	}

	queryURL, err := serverURL.Parse(operationPath)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("DELETE", queryURL.String(), nil)
	if err != nil {
		return nil, err
	}

	return req, nil
}

// NewGetTaskByNameRequest generates requests for GetTaskByName
func NewGetTaskByNameRequest(server string, name string) (*http.Request, error) {
	var err error

	var pathParam0 string

	pathParam0, err = runtime.StyleParamWithLocation("simple", false, "name", runtime.ParamLocationPath, name)
	if err != nil {
		return nil, err
	}

	serverURL, err := url.Parse(server)
	if err != nil {
		return nil, err
	}

	operationPath := fmt.Sprintf("/v1/tasks/%s", pathParam0)
	if operationPath[0] == '/' {
		operationPath = "." + operationPath
	}

	queryURL, err := serverURL.Parse(operationPath)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("GET", queryURL.String(), nil)
	if err != nil {
		return nil, err
	}

	return req, nil
}

func (c *Client) applyEditors(ctx context.Context, req *http.Request, additionalEditors []RequestEditorFn) error {
	for _, r := range c.RequestEditors {
		if err := r(ctx, req); err != nil {
			return err
		}
	}
	for _, r := range additionalEditors {
		if err := r(ctx, req); err != nil {
			return err
		}
	}
	return nil
}

// ClientWithResponses builds on ClientInterface to offer response payloads
type ClientWithResponses struct {
	ClientInterface
}

// NewClientWithResponses creates a new ClientWithResponses, which wraps
// Client with return type handling
func NewClientWithResponses(server string, opts ...ClientOption) (*ClientWithResponses, error) {
	client, err := NewClient(server, opts...)
	if err != nil {
		return nil, err
	}
	return &ClientWithResponses{client}, nil
}

// WithBaseURL overrides the baseURL.
func WithBaseURL(baseURL string) ClientOption {
	return func(c *Client) error {
		newBaseURL, err := url.Parse(baseURL)
		if err != nil {
			return err
		}
		c.Server = newBaseURL.String()
		return nil
	}
}

// ClientWithResponsesInterface is the interface specification for the client with responses above.
type ClientWithResponsesInterface interface {
	// GetAllTasks request
	GetAllTasksWithResponse(ctx context.Context, reqEditors ...RequestEditorFn) (*GetAllTasksResponse, error)

	// CreateTask request with any body
	CreateTaskWithBodyWithResponse(ctx context.Context, params *CreateTaskParams, contentType string, body io.Reader, reqEditors ...RequestEditorFn) (*CreateTaskResponse, error)

	CreateTaskWithResponse(ctx context.Context, params *CreateTaskParams, body CreateTaskJSONRequestBody, reqEditors ...RequestEditorFn) (*CreateTaskResponse, error)

	// DeleteTaskByName request
	DeleteTaskByNameWithResponse(ctx context.Context, name string, reqEditors ...RequestEditorFn) (*DeleteTaskByNameResponse, error)

	// GetTaskByName request
	GetTaskByNameWithResponse(ctx context.Context, name string, reqEditors ...RequestEditorFn) (*GetTaskByNameResponse, error)
}

type GetAllTasksResponse struct {
	Body         []byte
	HTTPResponse *http.Response
	JSON200      *TasksResponse
	JSONDefault  *ErrorResponse
}

// Status returns HTTPResponse.Status
func (r GetAllTasksResponse) Status() string {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.Status
	}
	return http.StatusText(0)
}

// StatusCode returns HTTPResponse.StatusCode
func (r GetAllTasksResponse) StatusCode() int {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.StatusCode
	}
	return 0
}

type CreateTaskResponse struct {
	Body         []byte
	HTTPResponse *http.Response
	JSON200      *TaskResponse
	JSON201      *TaskResponse
	JSONDefault  *ErrorResponse
}

// Status returns HTTPResponse.Status
func (r CreateTaskResponse) Status() string {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.Status
	}
	return http.StatusText(0)
}

// StatusCode returns HTTPResponse.StatusCode
func (r CreateTaskResponse) StatusCode() int {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.StatusCode
	}
	return 0
}

type DeleteTaskByNameResponse struct {
	Body         []byte
	HTTPResponse *http.Response
	JSON202      *TaskDeleteResponse
	JSONDefault  *ErrorResponse
}

// Status returns HTTPResponse.Status
func (r DeleteTaskByNameResponse) Status() string {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.Status
	}
	return http.StatusText(0)
}

// StatusCode returns HTTPResponse.StatusCode
func (r DeleteTaskByNameResponse) StatusCode() int {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.StatusCode
	}
	return 0
}

type GetTaskByNameResponse struct {
	Body         []byte
	HTTPResponse *http.Response
	JSON200      *TaskResponse
	JSONDefault  *ErrorResponse
}

// Status returns HTTPResponse.Status
func (r GetTaskByNameResponse) Status() string {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.Status
	}
	return http.StatusText(0)
}

// StatusCode returns HTTPResponse.StatusCode
func (r GetTaskByNameResponse) StatusCode() int {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.StatusCode
	}
	return 0
}

// GetAllTasksWithResponse request returning *GetAllTasksResponse
func (c *ClientWithResponses) GetAllTasksWithResponse(ctx context.Context, reqEditors ...RequestEditorFn) (*GetAllTasksResponse, error) {
	rsp, err := c.GetAllTasks(ctx, reqEditors...)
	if err != nil {
		return nil, err
	}
	return ParseGetAllTasksResponse(rsp)
}

// CreateTaskWithBodyWithResponse request with arbitrary body returning *CreateTaskResponse
func (c *ClientWithResponses) CreateTaskWithBodyWithResponse(ctx context.Context, params *CreateTaskParams, contentType string, body io.Reader, reqEditors ...RequestEditorFn) (*CreateTaskResponse, error) {
	rsp, err := c.CreateTaskWithBody(ctx, params, contentType, body, reqEditors...)
	if err != nil {
		return nil, err
	}
	return ParseCreateTaskResponse(rsp)
}

func (c *ClientWithResponses) CreateTaskWithResponse(ctx context.Context, params *CreateTaskParams, body CreateTaskJSONRequestBody, reqEditors ...RequestEditorFn) (*CreateTaskResponse, error) {
	rsp, err := c.CreateTask(ctx, params, body, reqEditors...)
	if err != nil {
		return nil, err
	}
	return ParseCreateTaskResponse(rsp)
}

// DeleteTaskByNameWithResponse request returning *DeleteTaskByNameResponse
func (c *ClientWithResponses) DeleteTaskByNameWithResponse(ctx context.Context, name string, reqEditors ...RequestEditorFn) (*DeleteTaskByNameResponse, error) {
	rsp, err := c.DeleteTaskByName(ctx, name, reqEditors...)
	if err != nil {
		return nil, err
	}
	return ParseDeleteTaskByNameResponse(rsp)
}

// GetTaskByNameWithResponse request returning *GetTaskByNameResponse
func (c *ClientWithResponses) GetTaskByNameWithResponse(ctx context.Context, name string, reqEditors ...RequestEditorFn) (*GetTaskByNameResponse, error) {
	rsp, err := c.GetTaskByName(ctx, name, reqEditors...)
	if err != nil {
		return nil, err
	}
	return ParseGetTaskByNameResponse(rsp)
}

// ParseGetAllTasksResponse parses an HTTP response from a GetAllTasksWithResponse call
func ParseGetAllTasksResponse(rsp *http.Response) (*GetAllTasksResponse, error) {
	bodyBytes, err := ioutil.ReadAll(rsp.Body)
	defer func() { _ = rsp.Body.Close() }()
	if err != nil {
		return nil, err
	}

	response := &GetAllTasksResponse{
		Body:         bodyBytes,
		HTTPResponse: rsp,
	}

	switch {
	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && rsp.StatusCode == 200:
		var dest TasksResponse
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON200 = &dest

	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && true:
		var dest ErrorResponse
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSONDefault = &dest

	}

	return response, nil
}

// ParseCreateTaskResponse parses an HTTP response from a CreateTaskWithResponse call
func ParseCreateTaskResponse(rsp *http.Response) (*CreateTaskResponse, error) {
	bodyBytes, err := ioutil.ReadAll(rsp.Body)
	defer func() { _ = rsp.Body.Close() }()
	if err != nil {
		return nil, err
	}

	response := &CreateTaskResponse{
		Body:         bodyBytes,
		HTTPResponse: rsp,
	}

	switch {
	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && rsp.StatusCode == 200:
		var dest TaskResponse
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON200 = &dest

	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && rsp.StatusCode == 201:
		var dest TaskResponse
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON201 = &dest

	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && true:
		var dest ErrorResponse
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSONDefault = &dest

	}

	return response, nil
}

// ParseDeleteTaskByNameResponse parses an HTTP response from a DeleteTaskByNameWithResponse call
func ParseDeleteTaskByNameResponse(rsp *http.Response) (*DeleteTaskByNameResponse, error) {
	bodyBytes, err := ioutil.ReadAll(rsp.Body)
	defer func() { _ = rsp.Body.Close() }()
	if err != nil {
		return nil, err
	}

	response := &DeleteTaskByNameResponse{
		Body:         bodyBytes,
		HTTPResponse: rsp,
	}

	switch {
	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && rsp.StatusCode == 202:
		var dest TaskDeleteResponse
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON202 = &dest

	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && true:
		var dest ErrorResponse
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSONDefault = &dest

	}

	return response, nil
}

// ParseGetTaskByNameResponse parses an HTTP response from a GetTaskByNameWithResponse call
func ParseGetTaskByNameResponse(rsp *http.Response) (*GetTaskByNameResponse, error) {
	bodyBytes, err := ioutil.ReadAll(rsp.Body)
	defer func() { _ = rsp.Body.Close() }()
	if err != nil {
		return nil, err
	}

	response := &GetTaskByNameResponse{
		Body:         bodyBytes,
		HTTPResponse: rsp,
	}

	switch {
	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && rsp.StatusCode == 200:
		var dest TaskResponse
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON200 = &dest

	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && true:
		var dest ErrorResponse
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSONDefault = &dest

	}

	return response, nil
}
