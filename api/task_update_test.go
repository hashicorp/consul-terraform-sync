package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/consul-terraform-sync/config"
	mocks "github.com/hashicorp/consul-terraform-sync/mocks/server"
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
			h := newTaskHandler(new(mocks.Server), tc.version)
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

	ctrl := new(mocks.Server)
	ctrl.On("Task", mock.Anything, mock.Anything).Return(config.TaskConfig{}, nil).
		On("TaskUpdate", mock.Anything, mock.Anything, mock.Anything).Return(true, "", "", nil)
	handler := newTaskHandler(ctrl, "v1")

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
		mockSetup     func(*mocks.Server)
		statusCode    int
		expected      UpdateTaskResponse
		updateTaskErr error
	}{
		{
			"happy path",
			"/v1/tasks/task_a",
			`{"enabled": true}`,
			func(ctrl *mocks.Server) {
				ctrl.On("Task", mock.Anything, "task_a").Return(config.TaskConfig{}, nil).
					On("TaskUpdate", mock.Anything, mock.Anything, "").Return(true, "", "", nil)
			},
			http.StatusOK,
			UpdateTaskResponse{},
			nil,
		},
		{
			"happy path - inspect option",
			"/v1/tasks/task_a?run=inspect",
			`{"enabled": true}`,
			func(ctrl *mocks.Server) {
				ctrl.On("Task", mock.Anything, "task_a").Return(config.TaskConfig{}, nil).
					On("TaskUpdate", mock.Anything, mock.Anything, "inspect").Return(true, "my plan!", "", nil)
			},
			http.StatusOK,
			UpdateTaskResponse{Inspect: &InspectPlan{
				ChangesPresent: true,
				Plan:           "my plan!",
			}},
			nil,
		},
		{
			"happy path - run now option",
			"/v1/tasks/task_a?run=now",
			`{"enabled": true}`,
			func(ctrl *mocks.Server) {
				ctrl.On("Task", mock.Anything, "task_a").Return(config.TaskConfig{}, nil).
					On("TaskUpdate", mock.Anything, mock.Anything, "now").Return(true, "", "", nil)
			},
			http.StatusOK,
			UpdateTaskResponse{},
			nil,
		},
		{
			"bad path/taskname",
			"/v1/tasks/task/a",
			`{"enabled": true}`,
			func(ctrl *mocks.Server) {},
			http.StatusBadRequest,
			UpdateTaskResponse{},
			nil,
		},
		{
			"no task specified",
			"/v1/tasks",
			`{"enabled": true}`,
			func(ctrl *mocks.Server) {},
			http.StatusBadRequest,
			UpdateTaskResponse{},
			nil,
		},
		{
			"task not found",
			"/v1/tasks/task_b",
			`{"enabled": true}`,
			func(ctrl *mocks.Server) {
				ctrl.On("Task", mock.Anything, "task_b").Return(config.TaskConfig{}, fmt.Errorf("DNE"))
			},
			http.StatusNotFound,
			UpdateTaskResponse{},
			nil,
		},
		{
			"ill formed request body",
			"/v1/tasks/task_a",
			`...???`,
			func(ctrl *mocks.Server) {},
			http.StatusBadRequest,
			UpdateTaskResponse{},
			nil,
		},
		{
			"error when updating task",
			"/v1/tasks/task_a",
			`{"enabled": true}`,
			func(ctrl *mocks.Server) {
				ctrl.On("Task", mock.Anything, "task_a").Return(config.TaskConfig{}, nil).
					On("TaskUpdate", mock.Anything, mock.Anything, "").Return(false, "", "", fmt.Errorf("error updating task"))
			},
			http.StatusInternalServerError,
			UpdateTaskResponse{},
			errors.New("error updating task"),
		},
		{
			"error when updating task with run-now",
			"/v1/tasks/task_a?run=now",
			`{"enabled": true}`,
			func(ctrl *mocks.Server) {
				ctrl.On("Task", mock.Anything, "task_a").Return(config.TaskConfig{}, nil).
					On("TaskUpdate", mock.Anything, mock.Anything, "now").Return(false, "", "", fmt.Errorf("update error"))
			},
			http.StatusInternalServerError,
			UpdateTaskResponse{},
			errors.New("error updating task"),
		},
		{
			"invalid run option",
			"/v1/tasks/task_a?run=bad-run-option",
			`{"enabled": true}`,
			func(ctrl *mocks.Server) {},
			http.StatusBadRequest,
			UpdateTaskResponse{},
			nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := new(mocks.Server)
			tc.mockSetup(ctrl)
			handler := newTaskHandler(ctrl, "v1")

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

			assert.Equal(t, tc.expected, actual)
			ctrl.AssertExpectations(t)
		})
	}

	t.Run("cancel", func(t *testing.T) {
		// have the server delay on response, and the client cancel to ensure
		// the handler exits immediately
		ctrl := new(mocks.Server)
		handler := newTaskHandler(ctrl, "v1")

		ctx, cancel := context.WithCancel(context.Background())
		req, err := http.NewRequestWithContext(ctx,
			http.MethodPatch,
			"/v1/tasks/task_a",
			strings.NewReader(`{"enabled": true}`),
		)
		require.NoError(t, err)

		ctrl.On("Task", req.Context(), "task_a").Return(config.TaskConfig{}, nil)
		ctrl.On("TaskUpdate", req.Context(), mock.Anything, "").
			Run(func(mock.Arguments) {
				<-req.Context().Done()
				assert.Equal(t, req.Context().Err(), context.Canceled)
			}).Return(false, "", "", context.Canceled).Once()

		resp := httptest.NewRecorder()
		go func() {
			time.Sleep(time.Second)
			cancel()
		}()
		handler.updateTask(resp, req)
		assert.Equal(t, http.StatusInternalServerError, resp.Code)
	})
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
			RunOptionNow,
			false,
		},
		{
			"happy path run inspect",
			"/v1/tasks/task_a?run=inspect",
			RunOptionInspect,
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
			RunOptionInspect,
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

			actual, err := parseRunOption(req)
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.option, actual)
			}
		})
	}
}
