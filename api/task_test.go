package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/driver"
	"github.com/hashicorp/consul-terraform-sync/event"
	mocks "github.com/hashicorp/consul-terraform-sync/mocks/driver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestTask_New(t *testing.T) {
	cases := []struct {
		name    string
		version string
	}{
		{
			"happy path",
			"v1",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newTaskHandler(event.NewStore(), driver.NewDrivers(), tc.version)
			assert.Equal(t, tc.version, h.version)
		})
	}
}

func TestTask_ServeHTTP(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name       string
		path       string
		method     string
		body       string
		statusCode int
	}{
		{
			"patch update task",
			"/v1/tasks/task_patch_update",
			http.MethodPatch,
			`{"enabled": true}`,
			http.StatusOK,
		},
		{
			"put update task",
			"/v1/tasks/xyz",
			http.MethodPut,
			``,
			http.StatusMethodNotAllowed,
		},
	}

	drivers := driver.NewDrivers()
	patchUpdateD := new(mocks.Driver)
	patchUpdateD.On("UpdateTask", mock.Anything, mock.Anything).
		Return(driver.InspectPlan{}, nil).Once()
	drivers.Add("task_patch_update", patchUpdateD)

	handler := newTaskHandler(event.NewStore(), drivers, "v1")

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := strings.NewReader(tc.body)
			req, err := http.NewRequest(tc.method, tc.path, r)
			require.NoError(t, err)
			resp := httptest.NewRecorder()

			handler.ServeHTTP(resp, req)

			require.Equal(t, tc.statusCode, resp.Code)
		})
	}
}

func TestTask_updateTask(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name          string
		path          string
		body          string
		statusCode    int
		updateTaskRet driver.InspectPlan
		updateTaskErr error
	}{
		{
			"happy path",
			"/v1/tasks/task_a",
			`{"enabled": true}`,
			http.StatusOK,
			driver.InspectPlan{},
			nil,
		},
		{
			"happy path - inspect option",
			"/v1/tasks/task_a?run=inspect",
			`{"enabled": true}`,
			http.StatusOK,
			driver.InspectPlan{
				ChangesPresent: true,
				Plan:           "my plan!",
			},
			nil,
		},
		{
			"happy path - run now option",
			"/v1/tasks/task_a?run=now",
			`{"enabled": true}`,
			http.StatusOK,
			driver.InspectPlan{},
			nil,
		},
		{
			"bad path/taskname",
			"/v1/tasks/task/a",
			`{"enabled": true}`,
			http.StatusBadRequest,
			driver.InspectPlan{},
			nil,
		},
		{
			"no task specified",
			"/v1/tasks",
			`{"enabled": true}`,
			http.StatusBadRequest,
			driver.InspectPlan{},
			nil,
		},
		{
			"task not found",
			"/v1/tasks/task_b",
			`{"enabled": true}`,
			http.StatusNotFound,
			driver.InspectPlan{},
			nil,
		},
		{
			"ill formed request body",
			"/v1/tasks/task_a",
			`...???`,
			http.StatusBadRequest,
			driver.InspectPlan{},
			nil,
		},
		{
			"error when updating task",
			"/v1/tasks/task_a",
			`{"enabled": true}`,
			http.StatusInternalServerError,
			driver.InspectPlan{},
			errors.New("error updating task"),
		},
		{
			"invalid run option",
			"/v1/tasks/task_a?run=bad-run-option",
			`{"enabled": true}`,
			http.StatusBadRequest,
			driver.InspectPlan{},
			nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			drivers := driver.NewDrivers()
			d := new(mocks.Driver)
			d.On("UpdateTask", mock.Anything, mock.Anything).
				Return(tc.updateTaskRet, tc.updateTaskErr).Once()
			drivers.Add("task_a", d)

			handler := newTaskHandler(event.NewStore(), drivers, "v1")

			r := strings.NewReader(tc.body)
			req, err := http.NewRequest(http.MethodPatch, tc.path, r)
			require.NoError(t, err)
			resp := httptest.NewRecorder()

			handler.updateTask(resp, req)
			require.Equal(t, tc.statusCode, resp.Code)

			decoder := json.NewDecoder(resp.Body)
			var actual UpdateTaskResponse
			err = decoder.Decode(&actual)
			require.NoError(t, err)

			emptyPlan := driver.InspectPlan{}
			if tc.updateTaskRet == emptyPlan {
				assert.Nil(t, actual.Inspect)
			} else {
				assert.Equal(t, tc.updateTaskRet, *actual.Inspect)
			}
		})
	}
}

func TestTask_decodeJSON(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		body      string
		expected  UpdateTaskConfig
		expectErr bool
	}{
		{
			"no enabled field",
			`{"unsupported": true}`,
			UpdateTaskConfig{Enabled: nil},
			true,
		},
		{
			"enabled true",
			`{"enabled": true}`,
			UpdateTaskConfig{Enabled: config.Bool(true)},
			false,
		},
		{
			"enable false",
			`{"enabled": false}`,
			UpdateTaskConfig{Enabled: config.Bool(false)},
			false,
		},
		{
			"unmarshal error",
			`sdfsdf`,
			UpdateTaskConfig{Enabled: nil},
			true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := decodeBody([]byte(tc.body))
			assert.Equal(t, tc.expected, actual)

			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTask_RunOption(t *testing.T) {
	cases := []struct {
		name        string
		path        string
		option      string
		expectError bool
	}{
		{
			"happy path run now",
			"/v1/tasks/task_a?run=now",
			driver.RunOptionNow,
			false,
		},
		{
			"happy path run inspect",
			"/v1/tasks/task_a?run=inspect",
			driver.RunOptionInspect,
			false,
		},
		{
			"happy path no run option",
			"/v1/tasks/task_a",
			"",
			false,
		},
		{
			"not lower case",
			"/v1/tasks/task_a?run=INSPECT",
			driver.RunOptionInspect,
			false,
		},
		{
			"unknown run option",
			"/v1/tasks/task_a?run=badoption",
			"",
			true,
		},
		{
			"too many run parameters",
			"/v1/tasks/task_a?run=now&run=inspect",
			"",
			true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodPatch, tc.path, nil)
			require.NoError(t, err)

			actual, err := runOption(req)
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.option, actual)
			}
		})
	}
}
