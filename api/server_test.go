package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/event"
	apiMocks "github.com/hashicorp/consul-terraform-sync/mocks/api"
	mocks "github.com/hashicorp/consul-terraform-sync/mocks/server"
	"github.com/hashicorp/consul-terraform-sync/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestRequest(t *testing.T) {
	cases := []struct {
		name             string
		httpStatus       int
		httpResponseBody interface{}
		httpError        error
		expectError      bool
	}{
		{
			"happy path",
			http.StatusOK,
			"expected response",
			nil,
			false,
		},
		{
			"error: request failed",
			0,
			nil,
			errors.New("error"),
			true,
		},
		{
			"error: decoding error",
			http.StatusBadRequest,
			"request failed string",
			nil,
			true,
		},
		{
			"error: response map missing error",
			http.StatusBadRequest,
			map[string]string{
				"unexpected-field": "request failed",
			},
			nil,
			true,
		},
		{
			"error: response map has error info",
			http.StatusBadRequest,
			map[string]string{
				"error": "helpful error messasge",
			},
			nil,
			true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			hc := new(apiMocks.HttpClient)

			// set up return response on mock
			b, err := json.Marshal(tc.httpResponseBody)
			require.NoError(t, err)
			bytesR := bytes.NewBuffer(b)
			mockResp := &http.Response{
				Body:       ioutil.NopCloser(bytesR),
				StatusCode: tc.httpStatus,
			}
			hc.On("Do", mock.Anything).Return(mockResp, tc.httpError).Once()

			c, err := NewClient(createTestClientConfig(8558), hc)
			require.NoError(t, err)
			resp, err := c.request("GET", "v1/some/endpoint", "test=true", "body")
			if tc.expectError {
				assert.Error(t, err)
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.NoError(t, resp.Body.Close())
			}
		})
	}
}

func TestStatus(t *testing.T) {
	t.Parallel()

	// setup store + events
	store := event.NewStore()
	// successful
	eventsA := createTaskEvents("task_a", []bool{true})
	addEvents(store, eventsA)
	// critical
	eventsB := createTaskEvents("task_b", []bool{false, false, true})
	addEvents(store, eventsB)
	eventsC := createTaskEvents("task_c", []bool{false, false, true})
	addEvents(store, eventsC)

	ctrl := new(mocks.Server)
	var confs []config.TaskConfig
	for _, taskName := range []string{"task_a", "task_b", "task_c"} {
		conf := createTaskConf(taskName, true)
		confs = append(confs, conf)
		ctrl.On("Task", mock.Anything, taskName).Return(conf, nil).
			On("Events", mock.Anything, taskName).Return(store.Read(taskName), nil)
	}
	ctrl.On("Tasks", mock.Anything).Return(confs, nil)
	ctrl.On("Events", mock.Anything, "").Return(store.Read(""), nil)

	// start up server
	port := testutils.FreePort(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	api, err := NewAPI(Config{
		Controller: ctrl,
		Port:       port,
	})
	require.NoError(t, err)
	go api.Serve(ctx)

	c, err := NewClient(createTestClientConfig(port), nil)
	require.NoError(t, err)
	err = c.WaitForAPI(3 * time.Second) // in case tests run before server is ready
	require.NoError(t, err)

	t.Run("overall-status", func(t *testing.T) {
		actual, err := c.Status().Overall()
		require.NoError(t, err)
		expect := OverallStatus{
			TaskSummary: TaskSummary{
				Status: StatusSummary{
					Successful: 1,
					Errored:    0,
					Critical:   2,
				},
				Enabled: EnabledSummary{
					True:  3,
					False: 0,
				},
			},
		}
		assert.Equal(t, expect, actual)
	})

	t.Run("task-status", func(t *testing.T) {
		cases := []struct {
			name        string
			taskName    string
			q           *QueryParam
			expectError bool
			expect      map[string]TaskStatus
		}{
			{
				"all tasks",
				"",
				nil,
				false,
				map[string]TaskStatus{
					"task_a": {
						TaskName:  "task_a",
						Enabled:   true,
						Status:    StatusSuccessful,
						Providers: []string{},
						Services:  []string{},
						EventsURL: "/v1/status/tasks/task_a?include=events",
					},
					"task_b": {
						TaskName:  "task_b",
						Enabled:   true,
						Status:    StatusCritical,
						Providers: []string{},
						Services:  []string{},
						EventsURL: "/v1/status/tasks/task_b?include=events",
					},
					"task_c": {
						TaskName:  "task_c",
						Enabled:   true,
						Status:    StatusCritical,
						Providers: []string{},
						Services:  []string{},
						EventsURL: "/v1/status/tasks/task_c?include=events",
					},
				},
			},
			{
				"specific task",
				"task_a",
				nil,
				false,
				map[string]TaskStatus{
					"task_a": {
						TaskName:  "task_a",
						Enabled:   true,
						Status:    StatusSuccessful,
						Providers: []string{},
						Services:  []string{},
						EventsURL: "/v1/status/tasks/task_a?include=events",
					},
				},
			},
			{
				"include events",
				"task_b",
				&QueryParam{IncludeEvents: true},
				false,
				map[string]TaskStatus{
					"task_b": {
						TaskName:  "task_b",
						Status:    StatusCritical,
						Enabled:   true,
						Providers: []string{},
						Services:  []string{},
						EventsURL: "/v1/status/tasks/task_b?include=events",
						Events:    eventsB,
					},
				},
			},
			{
				"filter by status",
				"",
				&QueryParam{Status: StatusCritical},
				false,
				map[string]TaskStatus{
					"task_b": {
						TaskName:  "task_b",
						Enabled:   true,
						Status:    StatusCritical,
						Providers: []string{},
						Services:  []string{},
						EventsURL: "/v1/status/tasks/task_b?include=events",
					},
					"task_c": {
						TaskName:  "task_c",
						Enabled:   true,
						Status:    StatusCritical,
						Providers: []string{},
						Services:  []string{},
						EventsURL: "/v1/status/tasks/task_c?include=events",
					},
				},
			},
			{
				"error",
				"invalid/taskname/",
				nil,
				true,
				map[string]TaskStatus{},
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				actual, err := c.Status().Task(tc.taskName, tc.q)
				if tc.expectError {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
					assert.Equal(t, tc.expect, actual)
				}
			})
		}
	})
}

func TestWaitForAPI(t *testing.T) {
	t.Parallel()

	t.Run("timeout", func(t *testing.T) {
		cts, err := NewClient(createTestClientConfig(0), nil)
		require.NoError(t, err)
		err = cts.WaitForAPI(time.Second)
		assert.Error(t, err, "No CTS API server running, test is expected to timeout")
	})

	t.Run("available", func(t *testing.T) {
		ctrl := new(mocks.Server)
		ctrl.On("Tasks", mock.Anything).Return([]config.TaskConfig{}, nil).
			On("Events", mock.Anything, "").Return(map[string][]event.Event{}, nil)

		// start up server
		port := testutils.FreePort(t)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		api, err := NewAPI(Config{
			Controller: ctrl,
			Port:       port,
		})
		require.NoError(t, err)
		go api.Serve(ctx)

		cts, err := NewClient(createTestClientConfig(port), nil)
		require.NoError(t, err)
		err = cts.WaitForAPI(3 * time.Second)
		assert.NoError(t, err, "CTS API server should be available")
	})
}

func createTestClientConfig(port int) *ClientConfig {
	return &ClientConfig{
		URL: fmt.Sprintf("http://localhost:%d", port),
	}
}
