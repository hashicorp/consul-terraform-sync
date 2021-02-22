package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/driver"
	"github.com/hashicorp/consul-terraform-sync/event"
	mocks "github.com/hashicorp/consul-terraform-sync/mocks/api"
	mocksD "github.com/hashicorp/consul-terraform-sync/mocks/driver"
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
			hc := new(mocks.HttpClient)

			// set up return response on mock
			b, err := json.Marshal(tc.httpResponseBody)
			require.NoError(t, err)
			bytesR := bytes.NewBuffer(b)
			mockResp := &http.Response{
				Body:       ioutil.NopCloser(bytesR),
				StatusCode: tc.httpStatus,
			}
			hc.On("Do", mock.Anything).Return(mockResp, tc.httpError).Once()

			c := NewClient(&ClientConfig{Port: 8558}, hc)
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

	// setup drivers
	drivers := make(map[string]driver.Driver)
	drivers["task_a"] = createDriver("task_a", true)
	drivers["task_b"] = createDriver("task_b", true)
	drivers["task_c"] = createDriver("task_c", true)

	// start up server
	port, err := FreePort()
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	api := NewAPI(store, drivers, port)
	go api.Serve(ctx)
	time.Sleep(3 * time.Second) // in case tests run before server is ready

	c := NewClient(&ClientConfig{Port: port}, nil)

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
					"task_a": TaskStatus{
						TaskName:  "task_a",
						Enabled:   true,
						Status:    StatusSuccessful,
						Providers: []string{},
						Services:  []string{},
						EventsURL: "/v1/status/tasks/task_a?include=events",
					},
					"task_b": TaskStatus{
						TaskName:  "task_b",
						Enabled:   true,
						Status:    StatusCritical,
						Providers: []string{},
						Services:  []string{},
						EventsURL: "/v1/status/tasks/task_b?include=events",
					},
					"task_c": TaskStatus{
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
					"task_a": TaskStatus{
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
					"task_b": TaskStatus{
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
					"task_b": TaskStatus{
						TaskName:  "task_b",
						Enabled:   true,
						Status:    StatusCritical,
						Providers: []string{},
						Services:  []string{},
						EventsURL: "/v1/status/tasks/task_b?include=events",
					},
					"task_c": TaskStatus{
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

func Test_Task_Update(t *testing.T) {
	t.Parallel()

	// start up server
	port, err := FreePort()
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	drivers := make(map[string]driver.Driver)
	api := NewAPI(event.NewStore(), drivers, port)
	go api.Serve(ctx)
	time.Sleep(3 * time.Second) // in case tests run before server is ready

	c := NewClient(&ClientConfig{Port: port}, nil)

	t.Run("disable-then-enable", func(t *testing.T) {
		// setup temp dir
		tempDir := "disable-enable"
		delete, err := testutils.MakeTempDir(tempDir)
		require.NoError(t, err)
		defer delete()

		// add a driver
		d, err := driver.NewTerraform(&driver.TerraformConfig{
			Task:       driver.Task{Enabled: true},
			WorkingDir: tempDir,
			ClientType: "test",
		})
		require.NoError(t, err)
		drivers["task_a"] = d

		assert.True(t, d.Task().Enabled)
		plan, err := c.Task().Update("task_a", UpdateTaskConfig{
			Enabled: config.Bool(false),
		}, nil)
		require.NoError(t, err)
		assert.False(t, d.Task().Enabled)
		assert.Empty(t, plan)
	})
	t.Run("task-not-found-error", func(t *testing.T) {
		plan, err := c.Task().Update("non-existent-task", UpdateTaskConfig{
			Enabled: config.Bool(false),
		}, nil)
		require.Error(t, err)
		assert.Empty(t, plan)
	})
	t.Run("task-run-option", func(t *testing.T) {
		expectedPlan := driver.InspectPlan{
			ChangesPresent: true,
			Plan:           "plan!",
		}
		// add a driver
		d := new(mocksD.Driver)
		d.On("UpdateTask", mock.Anything, mock.Anything).
			Return(expectedPlan, nil).Once()
		drivers["task_a"] = d

		actualPlan, err := c.Task().Update("task_a", UpdateTaskConfig{
			Enabled: config.Bool(false),
		}, &QueryParam{Run: driver.RunOptionInspect})

		require.NoError(t, err)
		assert.Equal(t, expectedPlan, actualPlan)
	})
}
