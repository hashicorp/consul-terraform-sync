package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hashicorp/consul-terraform-sync/api/oapigen"
	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/driver"
	mocks "github.com/hashicorp/consul-terraform-sync/mocks/server"
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
    "module": "./example-module"
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
    "module": "./example-module"
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
		events     int
	}{
		{
			name:       "happy_path",
			taskName:   testTaskName,
			request:    testCreateTaskRequest,
			run:        "",
			statusCode: http.StatusCreated,
			events:     0,
		},
		{
			name:       "happy_path_run_now",
			taskName:   testTaskName,
			request:    testCreateTaskRequest,
			run:        driver.RunOptionNow,
			statusCode: http.StatusCreated,
			events:     1,
		},
		{
			name:       "happy_path_with_variables",
			taskName:   testTaskName,
			request:    testCreateTaskRequestVariables,
			run:        driver.RunOptionNow,
			statusCode: http.StatusCreated,
			events:     1,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := new(mocks.Server)
			ctrl.On("Task", mock.Anything, tc.taskName).Return(config.TaskConfig{}, fmt.Errorf("DNE")).
				On("TaskCreate", mock.Anything, mock.Anything).Return(nil).
				On("TaskCreateAndRun", mock.Anything, mock.Anything).Return(nil)
			handler := NewTaskLifeCycleHandler(ctrl)

			resp := runTestCreateTask(t, handler, tc.run, tc.statusCode, tc.request)

			// A single event should be registered
			// checkTestEventCount(t, tc.taskName, c.store, 1)

			// Check response
			decoder := json.NewDecoder(resp.Body)
			var actual oapigen.TaskResponse
			err := decoder.Decode(&actual)
			require.NoError(t, err)
			expected := generateExpectedResponse(t, tc.request)
			assert.Equal(t, expected, actual)
		})
	}
}

func TestTaskLifeCycleHandler_CreateTask_RunInspect(t *testing.T) {
	t.Parallel()
	c, d := generateTaskLifecycleHandlerTestDependencies()
	// Expected driver mock calls and returns
	taskName := "inspected_task"
	request := fmt.Sprintf(`{
		"name": "%s",
		"services": ["api"],
		"module": "mkam/hello/cts"
	}`, taskName)
	conf := driver.TaskConfig{
		Name: taskName,
	}
	task, err := driver.NewTask(conf)
	require.NoError(t, err)

	d.On("Task").Return(task)
	d.On("InitTask", mock.Anything).Return(nil)
	d.On("RenderTemplate", mock.Anything).Return(true, nil)
	d.On("InspectTask", mock.Anything).Return(driver.InspectPlan{}, nil)

	resp := runTestCreateTask(t, c, "inspect", http.StatusOK, request)

	// Check response, expect task and run
	decoder := json.NewDecoder(resp.Body)
	var actual taskResponse
	err = decoder.Decode(&actual)
	require.NoError(t, err)
	expected := generateExpectedResponse(t, request)
	plan := ""
	expected.Run = &oapigen.Run{
		Plan: &plan,
	}
	assert.Equal(t, expected, actual)
	d.AssertExpectations(t)

	// Check task not added to driver, no events registered
	_, ok := c.drivers.Get(taskName)
	require.False(t, ok)
	checkTestEventCount(t, taskName, c.store, 0)

	// Run the inspect a second time with same task, expect return 200 OK
	runTestCreateTask(t, c, "inspect", http.StatusOK, request)
}

func TestTaskLifeCycleHandler_CreateTask_BadRequest(t *testing.T) {
	t.Parallel()
	existingTask := "existing_task"
	cases := []struct {
		name       string
		taskName   string
		request    string
		statusCode int
		run        string
		message    string
	}{
		{
			name:     "task already exists",
			taskName: existingTask,
			request: fmt.Sprintf(`{
				"name": "%s",
				"services": ["api"],
				"module": "./example-module"
		}`, existingTask),
			message:    fmt.Sprintf("task with name %s already exists", existingTask),
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
			ctrl := new(mocks.Server)
			ctrl.On("Task", mock.Anything, tc.taskName).Return(config.TaskConfig{}, nil)
			handler := NewTaskLifeCycleHandler(ctrl)

			resp := runTestCreateTask(t, handler, "", tc.statusCode, tc.request)

			// Task should be added to the drivers list
			// _, ok := c.drivers.Get(tc.taskName)
			// require.True(t, ok)

			// No events should be registered
			// checkTestEventCount(t, tc.taskName, c.store, 0)

			// Check response
			decoder := json.NewDecoder(resp.Body)
			var actual oapigen.ErrorResponse
			err := decoder.Decode(&actual)
			require.NoError(t, err)

			expected := generateErrorResponse("", tc.message)
			assert.Equal(t, expected, actual)
		})
	}
}

func TestTaskLifeCycleHandler_CreateTask_InternalError(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name       string
		taskName   string
		request    string
		statusCode int
		run        string
		message    string
		response   oapigen.TaskResponse
	}{
		{
			name:       "task already exists",
			taskName:   testTaskName,
			request:    testCreateTaskRequest,
			message:    "error initializing new task, mock error",
			statusCode: http.StatusInternalServerError,
		},
		{
			name:       "invalid run param",
			taskName:   testTaskName,
			request:    testCreateTaskRequest,
			run:        "invalid",
			message:    "error initializing new task, invalid run option 'invalid'. Please select a valid option",
			statusCode: http.StatusInternalServerError,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := new(mocks.Server)
			ctrl.On("Task", mock.Anything, tc.taskName).Return(config.TaskConfig{}, fmt.Errorf("DNE")).
				On("TaskCreate", mock.Anything, mock.Anything).Return(nil).
				On("TaskCreateAndRun", mock.Anything, mock.Anything).Return(nil)
			handler := NewTaskLifeCycleHandler(ctrl)

			resp := runTestCreateTask(t, handler, tc.run, tc.statusCode, tc.request)

			// Task should not be added to the list
			// _, ok := c.drivers.Get(tc.taskName)
			// require.False(t, ok)

			// Only one event should be registered
			// checkTestEventCount(t, tc.taskName, c.store, 1)

			// Check response
			decoder := json.NewDecoder(resp.Body)
			var actual oapigen.ErrorResponse
			err := decoder.Decode(&actual)
			require.NoError(t, err)

			expected := generateErrorResponse("", tc.message)
			assert.Equal(t, expected, actual)
		})
	}
}

func generateExpectedResponse(t *testing.T, req string) oapigen.TaskResponse {
	var treq oapigen.TaskRequest
	err := json.Unmarshal([]byte(req), &treq)
	require.NoError(t, err)

	task := oapigen.Task(treq)
	return oapigen.TaskResponse{
		Task: &task,
	}
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

func runTestCreateTask(t *testing.T, handler *TaskLifeCycleHandler, run string, expectedStatus int, request string) *httptest.ResponseRecorder {
	path := "/v1/tasks"
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
