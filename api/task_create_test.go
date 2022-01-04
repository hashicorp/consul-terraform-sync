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

	taskConf := config.TaskConfig{
		Name:        config.String(testTaskName),
		Enabled:     config.Bool(true),
		Description: config.String("Writes the service name, id, and IP address to a file"),
		Source:      config.String("./example-module"),
		Services:    []string{"api"},
		Providers:   []string{"local"},
	}
	taskConfVars := *taskConf.Copy()
	taskConfVars.Variables = map[string]string{"filename": "test.txt"}
	cases := []struct {
		name       string
		taskName   string
		request    string
		run        string
		mockReturn config.TaskConfig
		statusCode int
	}{
		{
			name:       "happy_path",
			request:    testCreateTaskRequest,
			run:        "",
			mockReturn: taskConf,
			statusCode: http.StatusCreated,
		},
		{
			name:       "happy_path_run_now",
			request:    testCreateTaskRequest,
			run:        driver.RunOptionNow,
			mockReturn: taskConf,
			statusCode: http.StatusCreated,
		},
		{
			name:       "happy_path_with_variables",
			request:    testCreateTaskRequestVariables,
			run:        driver.RunOptionNow,
			mockReturn: taskConfVars,
			statusCode: http.StatusCreated,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := new(mocks.Server)
			ctrl.On("Task", mock.Anything, mock.Anything).Return(config.TaskConfig{}, fmt.Errorf("DNE")).
				On("TaskCreate", mock.Anything, mock.Anything).Return(tc.mockReturn, nil).
				On("TaskCreateAndRun", mock.Anything, mock.Anything).Return(tc.mockReturn, nil)
			handler := NewTaskLifeCycleHandler(ctrl)

			resp := runTestCreateTask(t, handler, tc.run, tc.statusCode, tc.request)

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
			ctrl := new(mocks.Server)
			ctrl.On("Task", mock.Anything, tc.taskName).Return(config.TaskConfig{}, nil)
			handler := NewTaskLifeCycleHandler(ctrl)

			resp := runTestCreateTask(t, handler, "", tc.statusCode, tc.request)

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

	errMsg := "error initializing new task, mock error"

	ctrl := new(mocks.Server)
	ctrl.On("Task", mock.Anything, testTaskName).Return(config.TaskConfig{}, fmt.Errorf("DNE"))
	ctrl.On("TaskCreate", mock.Anything, mock.Anything).Return(config.TaskConfig{}, fmt.Errorf(errMsg))
	handler := NewTaskLifeCycleHandler(ctrl)

	resp := runTestCreateTask(t, handler, "", http.StatusInternalServerError, testCreateTaskRequest)

	// Check response
	decoder := json.NewDecoder(resp.Body)
	var actual oapigen.ErrorResponse
	err := decoder.Decode(&actual)
	require.NoError(t, err)

	expected := generateErrorResponse("", errMsg)
	assert.Equal(t, expected, actual)
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
