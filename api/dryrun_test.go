package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/driver"
	mocks "github.com/hashicorp/consul-terraform-sync/mocks/driver"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	existingTestTaskName = "existing_task"
)

// generateTestDryRunHandler creates a dry run handler with a mock driver
// creation method and an existing task. The mocked methods for the driver
// take the happy path and do not error.
func generateTestDryRunHandler(t *testing.T) *DryRunTasksHandler {
	drivers := driver.NewDrivers()
	d := new(mocks.Driver)
	conf := driver.TaskConfig{
		Name: "inspected_task",
	}
	task, err := driver.NewTask(conf)
	require.NoError(t, err)
	d.On("Task").Return(task)
	d.On("InitTask", mock.Anything).Return(nil)
	d.On("RenderTemplate", mock.Anything).Return(true, nil)
	d.On("InspectTask", mock.Anything).Return(driver.InspectPlan{}, nil)
	drivers.Add(existingTestTaskName, d)

	createNewTaskDriver := func(taskConfig config.TaskConfig,
		variables map[string]string) (driver.Driver, error) {
		return d, nil
	}
	c := DryRunTasksHandlerConfig{
		drivers:             drivers,
		createNewTaskDriver: createNewTaskDriver,
	}
	return NewDryRunTasksHandler(c)
}

// TestDryRunTask_CreateDryRunTask tests creating dry run tasks
// where the results of the test cases are dependent on the request.
func TestDryRunTask_CreateDryRunTask(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name       string
		statusCode int
		request    string
	}{
		{
			"happy_path",
			http.StatusOK,
			`{
				"name": "inspected_task",
				"services": ["api"],
				"source": "mkam/hello/cts"
			}`,
		},
		{
			"request_decode_error",
			http.StatusBadRequest,
			"not json",
		},
		{
			"task_with_name_exists",
			http.StatusBadRequest,
			fmt.Sprintf(`{
				"name": "%s",
				"services": ["api"],
				"source": "mkam/hello/cts"
			}`, existingTestTaskName),
		},
		{
			"invalid_task_config_buffer_duration",
			http.StatusBadRequest,
			`{
				"name": "inspected_task",
				"services": ["api"],
				"source": "mkam/hello/cts",
				"buffer_period": {
					"min": "100"
				}
			}`,
		},
		{
			"invalid_task_config_name",
			http.StatusBadRequest,
			`{
				"name": "1234",
				"services": ["api"],
				"source": "mkam/hello/cts"
			}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			handler := generateTestDryRunHandler(t)
			r := strings.NewReader(tc.request)
			req, err := http.NewRequest(http.MethodPost, "/v1/dryrun_tasks", r)
			require.NoError(t, err)
			resp := httptest.NewRecorder()
			handler.CreateDryRunTask(resp, req)
			require.Equal(t, tc.statusCode, resp.Code)
		})
	}
}

// TestDryRunTask_DriverErrors tests that the dry run tasks API returns
// a bad request error when there are issues initializing or executing
// the task
func TestDryRunTask_DriverErrors(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name              string
		initTaskErr       error
		renderTemplateErr error
		inspectTaskErr    error
	}{
		{
			name:              "init_task",
			initTaskErr:       fmt.Errorf("tf-init error"),
			renderTemplateErr: nil,
			inspectTaskErr:    nil,
		},
		{
			name:              "render_template_err",
			initTaskErr:       nil,
			renderTemplateErr: fmt.Errorf("render tmpl error"),
			inspectTaskErr:    nil,
		},
		{
			name:              "inspect_task",
			initTaskErr:       nil,
			renderTemplateErr: nil,
			inspectTaskErr:    fmt.Errorf("tf-plan error"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			drivers := driver.NewDrivers()
			d := new(mocks.Driver)
			conf := driver.TaskConfig{
				Name: "inspected_task",
			}
			task, err := driver.NewTask(conf)
			require.NoError(t, err)
			d.On("Task").Return(task)

			// Set mock responses based on use case
			d.On("InitTask", mock.Anything).Return(tc.initTaskErr)
			d.On("RenderTemplate", mock.Anything).Return(true, tc.renderTemplateErr)
			d.On("InspectTask", mock.Anything).Return(driver.InspectPlan{}, tc.inspectTaskErr)

			createNewTaskDriver := func(taskConfig config.TaskConfig,
				variables map[string]string) (driver.Driver, error) {
				return d, nil
			}
			c := DryRunTasksHandlerConfig{
				drivers:             drivers,
				createNewTaskDriver: createNewTaskDriver,
			}
			handler := NewDryRunTasksHandler(c)

			// Request with a valid task config, expect an error on execution
			request := `{
	"name": "inspected_task",
	"services": ["api"],
	"source": "mkam/hello/cts"
}`
			r := strings.NewReader(request)
			req, err := http.NewRequest(http.MethodPost, "/v1/dryrun_tasks", r)
			require.NoError(t, err)
			resp := httptest.NewRecorder()
			handler.CreateDryRunTask(resp, req)
			require.Equal(t, http.StatusBadRequest, resp.Code)
		})
	}
}

// TestDryRunTask_TaskMultipleInspections tests that a dry run task is discarded
// at the end of its inspection and therefore can be recreated multiple times with
// the same task name.
func TestDryRunTask_TaskMultipleInspections(t *testing.T) {
	t.Parallel()
	request := `{
	"name": "inspected_task",
	"services": ["api"],
	"source": "mkam/hello/cts"
}`
	handler := generateTestDryRunHandler(t)

	// First request for inspection
	r := strings.NewReader(request)
	req, err := http.NewRequest(http.MethodPost, "/v1/dryrun_tasks", r)
	require.NoError(t, err)
	resp := httptest.NewRecorder()
	handler.CreateDryRunTask(resp, req)
	require.Equal(t, http.StatusOK, resp.Code)

	// Second request for inspection, no error because task does not persist
	r = strings.NewReader(request)
	req, err = http.NewRequest(http.MethodPost, "/v1/dryrun_tasks", r)
	require.NoError(t, err)
	resp = httptest.NewRecorder()
	handler.CreateDryRunTask(resp, req)
	require.Equal(t, http.StatusOK, resp.Code)
}
