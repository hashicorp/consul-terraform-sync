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
    "condition": {
			"services": {"names": ["api"]}
		},
    "module": "./example-module"
}`
	testCreateTaskRequestVariables = `{
    "description": "Writes the service name, id, and IP address to a file",
    "enabled": true,
    "name": "api-task",
    "providers": [
        "local"
    ],
    "condition": {
			"services": {"names": ["api"]}
		},
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

	taskConf := config.TaskConfig{
		Name:        config.String(testTaskName),
		Enabled:     config.Bool(true),
		Description: config.String("Writes the service name, id, and IP address to a file"),
		Module:      config.String("./example-module"),
		Condition: &config.ServicesConditionConfig{
			ServicesMonitorConfig: config.ServicesMonitorConfig{Names: []string{"api"}},
		},
		Providers: []string{"local"},
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
			run:        "now",
			mockReturn: taskConf,
			statusCode: http.StatusCreated,
		},
		{
			name:       "happy_path_with_variables",
			request:    testCreateTaskRequestVariables,
			run:        "now",
			mockReturn: taskConfVars,
			statusCode: http.StatusCreated,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := new(mocks.Server)
			ctrl.On("Task", mock.Anything, testTaskName).Return(config.TaskConfig{}, fmt.Errorf("DNE")).
				On("TaskCreate", mock.Anything, tc.mockReturn).Return(tc.mockReturn, nil).
				On("TaskCreateAndRun", mock.Anything, tc.mockReturn).Return(tc.mockReturn, nil)
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

func TestTaskLifeCycleHandler_CreateTask_RunInspect(t *testing.T) {
	t.Parallel()

	// Expected ctrl mock calls and returns
	taskName := "inspected_task"
	request := fmt.Sprintf(`{
		"name": "%s",
		"enabled": true,
		"condition": {"services": {"names": ["api"]}},
		"module": "mkam/hello/cts"
	}`, taskName)
	taskConf := config.TaskConfig{
		Name:    config.String(taskName),
		Enabled: config.Bool(true),
		Module:  config.String("mkam/hello/cts"),
		Condition: &config.ServicesConditionConfig{
			ServicesMonitorConfig: config.ServicesMonitorConfig{Names: []string{"api"}},
		},
	}

	ctrl := new(mocks.Server)
	ctrl.On("Task", mock.Anything, taskName).Return(config.TaskConfig{}, fmt.Errorf("DNE")).
		On("TaskInspect", mock.Anything, taskConf).Return(true, "foobar-plan", "", nil)
	handler := NewTaskLifeCycleHandler(ctrl)

	resp := runTestCreateTask(t, handler, "inspect", http.StatusOK, request)

	// Check response, expect task and run
	decoder := json.NewDecoder(resp.Body)
	var actual TaskResponse
	require.NoError(t, decoder.Decode(&actual))
	expected := generateExpectedResponse(t, request)
	expected.Run = &oapigen.Run{
		Plan:           config.String("foobar-plan"),
		ChangesPresent: config.Bool(true),
	}
	assert.Equal(t, expected, oapigen.TaskResponse(actual))
	ctrl.AssertExpectations(t)

	// Run the inspect a second time with same task, expect return 200 OK
	runTestCreateTask(t, handler, "inspect", http.StatusOK, request)
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
