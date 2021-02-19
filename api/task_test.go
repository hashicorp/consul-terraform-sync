package api

import (
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
			h := newTaskHandler(event.NewStore(), map[string]driver.Driver{}, tc.version)
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

	drivers := make(map[string]driver.Driver)
	patchUpdateD := new(mocks.Driver)
	patchUpdateD.On("UpdateTask", mock.Anything, mock.Anything).Return("", nil).Once()
	drivers["task_patch_update"] = patchUpdateD

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
		updateTaskRet error
	}{
		{
			"happy path",
			"/v1/tasks/task_a",
			`{"enabled": true}`,
			http.StatusOK,
			nil,
		},
		{
			"bad path/taskname",
			"/v1/tasks/task/a",
			`{"enabled": true}`,
			http.StatusBadRequest,
			nil,
		},
		{
			"no task specified",
			"/v1/tasks",
			`{"enabled": true}`,
			http.StatusBadRequest,
			nil,
		},
		{
			"task not found",
			"/v1/tasks/task_b",
			`{"enabled": true}`,
			http.StatusNotFound,
			nil,
		},
		{
			"ill formed request body",
			"/v1/tasks/task_a",
			`...???`,
			http.StatusBadRequest,
			nil,
		},
		{
			"error when updating task",
			"/v1/tasks/task_a",
			`{"enabled": true}`,
			http.StatusInternalServerError,
			errors.New("error updating task"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			drivers := make(map[string]driver.Driver)
			d := new(mocks.Driver)
			d.On("UpdateTask", mock.Anything, mock.Anything).
				Return("", tc.updateTaskRet).Once()
			drivers["task_a"] = d

			handler := newTaskHandler(event.NewStore(), drivers, "v1")

			r := strings.NewReader(tc.body)
			req, err := http.NewRequest(http.MethodPatch, tc.path, r)
			require.NoError(t, err)
			resp := httptest.NewRecorder()

			handler.updateTask(resp, req)
			require.Equal(t, tc.statusCode, resp.Code)
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
