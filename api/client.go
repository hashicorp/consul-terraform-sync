package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

//go:generate mockery --name=httpClient  --structname=HttpClient --output=../mocks/api

// httpClient describes the interface for the client to make http calls
type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// Client to make api requests
type Client struct {
	port    int
	version string
	scheme  string
	http    httpClient
}

// ClientConfig configures the client to make api requests
type ClientConfig struct {
	Port int
}

// NewClient returns a client to make api requests
func NewClient(c *ClientConfig, httpClient httpClient) *Client {
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: time.Second * 5,
		}
	}
	return &Client{
		port:    c.Port,
		version: defaultAPIVersion,
		scheme:  "http",
		http:    httpClient,
	}
}

func (c *Client) Port() int {
	return c.port
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
		return fmt.Errorf("client timed out waiting for CTS API to start: %v", timeout)
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
	url := url.URL{
		Scheme:   "http",
		Host:     fmt.Sprintf("localhost:%d", c.port),
		Path:     fmt.Sprintf("%s/%s", c.version, path),
		RawQuery: query,
	}

	r := strings.NewReader(body)
	req, err := http.NewRequest(method, url.String(), r)
	if err != nil {
		return nil, err
	}

	resp, err := c.http.Do(req)
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

// Status can be used to query the status endpoints
type Status struct {
	c *Client
}

// Status returns a handle to the status endpoints
func (c *Client) Status() *Status {
	return &Status{c}
}

// Overall is used to query for overall status
func (s *Status) Overall() (OverallStatus, error) {
	var overallStatus OverallStatus

	resp, err := s.c.request(http.MethodGet, overallStatusPath, "", "")
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
func (s *Status) Task(name string, q *QueryParam) (map[string]TaskStatus, error) {
	var taskStatuses map[string]TaskStatus

	path := taskStatusPath
	if name != "" {
		path += fmt.Sprintf("/%s", name)
	}

	if q == nil {
		q = &QueryParam{}
	}

	resp, err := s.c.request(http.MethodGet, path, q.Encode(), "")
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

// Task can be used to query the task endpoints
type Task struct {
	c *Client
}

// Task returns a handle to the task endpoints
func (c *Client) Task() *Task {
	return &Task{c}
}

// Update is used to patch update task
func (t *Task) Update(name string, config UpdateTaskConfig, q *QueryParam) (UpdateTaskResponse, error) {
	b, err := json.Marshal(config)
	if err != nil {
		return UpdateTaskResponse{}, err
	}

	if q == nil {
		q = &QueryParam{}
	}

	path := fmt.Sprintf("%s/%s", taskPath, name)
	resp, err := t.c.request(http.MethodPatch, path, q.Encode(), string(b))
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
