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

const testTaskName = "api-task"

var (
	// testTaskJSON and testTaskConfig are a set of json and config for the
	// same task "api-task"
	testTaskJSON = fmt.Sprintf(`{
		"task": {
			"description": "Writes the service name, id, and IP address to a file",
			"enabled": true,
			"name": "%s",
			"providers": [
				"local"
			],
			"condition": {
				"services": {"names": ["api"]}
			},
			"module_input": {
				"consul_kv": {
					"path": "key-path",
					"recurse": true,
					"datacenter": "dc2",
					"namespace": "ns2"
				}
			},
			"variables":{
				"filename": "test.txt"
			},
			"module": "./example-module"
		}
	}`, testTaskName)

	testTaskConfig = config.TaskConfig{
		Name:        config.String(testTaskName),
		Enabled:     config.Bool(true),
		Description: config.String("Writes the service name, id, and IP address to a file"),
		Module:      config.String("./example-module"),
		Condition: &config.ServicesConditionConfig{
			ServicesMonitorConfig: config.ServicesMonitorConfig{Names: []string{"api"}},
		},
		ModuleInputs: &config.ModuleInputConfigs{
			&config.ConsulKVModuleInputConfig{
				ConsulKVMonitorConfig: config.ConsulKVMonitorConfig{
					Path:       config.String("key-path"),
					Recurse:    config.Bool(true),
					Datacenter: config.String("dc2"),
					Namespace:  config.String("ns2"),
				},
			},
		},
		Providers: []string{"local"},
		Variables: map[string]string{"filename": "test.txt"},
	}
)

func TestTaskLifeCycleHandler_CreateTask(t *testing.T) {
	t.Parallel()

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
			request:    testTaskJSON,
			run:        "",
			mockReturn: testTaskConfig,
			statusCode: http.StatusCreated,
		},
		{
			name:       "happy_path_run_now",
			request:    testTaskJSON,
			run:        "now",
			mockReturn: testTaskConfig,
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
	ctrl := new(mocks.Server)
	ctrl.On("Task", mock.Anything, testTaskName).Return(config.TaskConfig{}, fmt.Errorf("DNE")).
		On("TaskInspect", mock.Anything, testTaskConfig).Return(true, "foobar-plan", "", nil)
	handler := NewTaskLifeCycleHandler(ctrl)

	resp := runTestCreateTask(t, handler, "inspect", http.StatusOK, testTaskJSON)

	// Check response, expect task and run
	decoder := json.NewDecoder(resp.Body)
	var actual TaskResponse
	require.NoError(t, decoder.Decode(&actual))
	expected := generateExpectedResponse(t, testTaskJSON)
	expected.Run = &oapigen.Run{
		Plan:           config.String("foobar-plan"),
		ChangesPresent: config.Bool(true),
	}
	assert.Equal(t, expected, oapigen.TaskResponse(actual))
	ctrl.AssertExpectations(t)

	// Run inspect a second time with same task, expect return 200 OK
	runTestCreateTask(t, handler, "inspect", http.StatusOK, testTaskJSON)
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
				"task": {
					"name": "%s",
					"condition": {
						"services": {
							"names": [
								"api"
							]
						}
					},
					"module": "./example-module"
				}
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

	resp := runTestCreateTask(t, handler, "", http.StatusInternalServerError, testTaskJSON)

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

	// Set cts_user_defined_meta to an empty map if nil
	services := treq.Task.Condition.Services
	if services != nil && services.CtsUserDefinedMeta == nil {
		services.CtsUserDefinedMeta = &oapigen.ServicesCondition_CtsUserDefinedMeta{}
	}

	return oapigen.TaskResponse{
		Task: &treq.Task,
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
