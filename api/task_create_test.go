package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hashicorp/consul-terraform-sync/api/oapigen"
	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/driver"
	"github.com/hashicorp/consul-terraform-sync/event"
	mocks "github.com/hashicorp/consul-terraform-sync/mocks/driver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	testCreateTaskRequest = `{
    "description": "Writes the service name, id, and IP address to a file",
    "enabled": true,
    "name": "api-task",
    "providers": [
        "local"
    ],
    "services": [
        "api"
    ],
    "source": "./example-module"
}`
	testCreateTaskRequestVariables = `{
    "description": "Writes the service name, id, and IP address to a file",
    "enabled": true,
    "name": "api-task",
    "providers": [
        "local"
    ],
    "services": [
        "api"
    ],
    "variables":{
        "filename": "test.txt"
    },
    "source": "./example-module"
}`
	testTaskName         = "api-task"
	testWorkingDirectory = "sync-task"
)

func TestTaskLifeCycleHandler_CreateTask(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name       string
		taskName   string
		request    string
		run        string
		variables  map[string]string
		statusCode int
	}{
		{
			name:       "happy_path",
			taskName:   testTaskName,
			request:    testCreateTaskRequest,
			run:        "",
			statusCode: http.StatusCreated,
		},
		{
			name:       "happy_path_run_now",
			taskName:   testTaskName,
			request:    testCreateTaskRequest,
			run:        driver.RunOptionNow,
			statusCode: http.StatusCreated,
		},
		{
			name:       "happy_path_with_variables",
			taskName:   testTaskName,
			request:    testCreateTaskRequestVariables,
			run:        driver.RunOptionNow,
			statusCode: http.StatusCreated,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, d := generateTaskLifecycleHandlerTestDependencies()

			// Expected driver mock calls and returns
			conf := driver.TaskConfig{
				Name: tc.taskName,
			}
			task, err := driver.NewTask(conf)
			require.NoError(t, err)

			d.On("Task").Return(task)
			d.On("InitTask", mock.Anything).Return(nil)
			d.On("RenderTemplate", mock.Anything).Return(true, nil)
			if tc.run == driver.RunOptionNow {
				d.On("ApplyTask", mock.Anything).Return(nil)
			}

			resp := runTestCreateTask(t, c, tc.run, tc.statusCode, tc.request)

			// Task should be added to the drivers list
			_, ok := c.drivers.Get(tc.taskName)
			require.True(t, ok)

			// A single event should be registered
			checkTestEventCount(t, tc.taskName, c.store, 1)

			// Check response
			decoder := json.NewDecoder(resp.Body)
			var actual taskResponse
			err = decoder.Decode(&actual)
			require.NoError(t, err)
			expected := generateExpectedResponse(t, tc.request)
			assert.Equal(t, expected, actual)
			d.AssertExpectations(t)
		})
	}
}

func TestTaskLifeCycleHandler_CreateTask_BadRequest(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name       string
		taskName   string
		request    string
		statusCode int
		message    string
	}{
		{
			name:       "task already exists",
			taskName:   testTaskName,
			request:    testCreateTaskRequest,
			message:    fmt.Sprintf("task with name %s already exists", testTaskName),
			statusCode: http.StatusBadRequest,
		},
		{
			name:       "empty request",
			taskName:   testTaskName,
			request:    "",
			message:    "error decoding the request: EOF",
			statusCode: http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, d := generateTaskLifecycleHandlerTestDependencies()

			err := c.drivers.Add(tc.taskName, d)
			require.NoError(t, err)

			resp := runTestCreateTask(t, c, "", tc.statusCode, tc.request)

			// Task should be added to the drivers list
			_, ok := c.drivers.Get(tc.taskName)
			require.True(t, ok)

			// No events should be registered
			checkTestEventCount(t, tc.taskName, c.store, 0)

			// Check response
			decoder := json.NewDecoder(resp.Body)
			var actual oapigen.ErrorResponse
			err = decoder.Decode(&actual)
			require.NoError(t, err)

			expected := generateErrorResponse("", tc.message)
			assert.Equal(t, expected, actual)
		})
	}
}

func TestTaskLifeCycleHandler_CreateTask_InitError(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name       string
		taskName   string
		request    string
		statusCode int
		run        string
		message    string
		response   taskResponse
	}{
		{
			name:       "task already exists",
			taskName:   testTaskName,
			request:    testCreateTaskRequest,
			message:    "error initializing new task: mock error",
			statusCode: http.StatusBadRequest,
		},
		{
			name:       "invalid run param",
			taskName:   testTaskName,
			request:    testCreateTaskRequest,
			run:        "invalid",
			message:    "error initializing new task: invalid run option 'invalid'. Please select a valid option",
			statusCode: http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, d := generateTaskLifecycleHandlerTestDependencies()

			d.On("InitTask", mock.Anything).Return(errors.New("mock error"))
			conf := driver.TaskConfig{
				Name: tc.taskName,
			}
			task, err := driver.NewTask(conf)
			require.NoError(t, err)
			d.On("Task").Return(task)

			resp := runTestCreateTask(t, c, tc.run, tc.statusCode, tc.request)

			// Task should not be added to the list
			_, ok := c.drivers.Get(tc.taskName)
			require.False(t, ok)

			// Only one event should be registered
			checkTestEventCount(t, tc.taskName, c.store, 1)

			// Check response
			decoder := json.NewDecoder(resp.Body)
			var actual oapigen.ErrorResponse
			err = decoder.Decode(&actual)
			require.NoError(t, err)

			expected := generateErrorResponse("", tc.message)
			assert.Equal(t, expected, actual)
		})
	}
}

func generateExpectedResponse(t *testing.T, req string) taskResponse {
	var treq taskRequest
	err := json.Unmarshal([]byte(req), &treq)
	require.NoError(t, err)

	trc, err := treq.ToTaskRequestConfig(config.DefaultBufferPeriodConfig(), testWorkingDirectory)
	require.NoError(t, err)

	tresp := taskResponseFromTaskRequestConfig(trc, "")
	require.NoError(t, err)

	return tresp
}

func generateErrorResponse(requestID, message string) oapigen.ErrorResponse {
	errResp := oapigen.ErrorResponse{
		Error: oapigen.Error{
			Message: message,
		},
		RequestId: oapigen.RequestID(requestID),
	}

	return errResp
}

func generateTaskLifecycleHandlerTestDependencies() (TaskLifeCycleHandlerConfig, *mocks.Driver) {
	// Create dependencies
	d := new(mocks.Driver)
	createNewTaskDriver := func(taskConfig config.TaskConfig, variables map[string]string) (driver.Driver, error) {
		return d, nil
	}

	c := TaskLifeCycleHandlerConfig{
		store:               event.NewStore(),
		drivers:             driver.NewDrivers(),
		bufferPeriod:        config.DefaultBufferPeriodConfig(),
		workingDir:          testWorkingDirectory,
		createNewTaskDriver: createNewTaskDriver,
	}

	return c, d
}

func runTestCreateTask(t *testing.T, c TaskLifeCycleHandlerConfig, run string, expectedStatus int, request string) *httptest.ResponseRecorder {
	path := "/v1/tasks"
	handler := NewTaskLifeCycleHandler(c)
	r := strings.NewReader(request)
	req, err := http.NewRequest(http.MethodPost, path, r)
	require.NoError(t, err)
	resp := httptest.NewRecorder()

	runOp := oapigen.CreateTaskParamsRun(run)
	params := oapigen.CreateTaskParams{
		Run: &runOp,
	}

	handler.CreateTask(resp, req, params)
	require.Equal(t, expectedStatus, resp.Code)

	return resp
}

func checkTestEventCount(t *testing.T, taskName string, store *event.Store, eventCount int) {
	data := store.Read(taskName)
	events, ok := data[taskName]

	if eventCount <= 0 {
		assert.False(t, ok)
	} else {
		assert.True(t, ok)
	}
	assert.Equal(t, eventCount, len(events), "event count not expected")
}
