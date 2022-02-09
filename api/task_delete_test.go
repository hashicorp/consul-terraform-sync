package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/consul-terraform-sync/config"
	mocks "github.com/hashicorp/consul-terraform-sync/mocks/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestTaskLifeCycleHandler_DeleteTaskByName(t *testing.T) {
	t.Parallel()
	taskName := "task"
	cases := []struct {
		name       string
		mockServer func(*mocks.Server)
		statusCode int
	}{
		{
			"happy_path",
			func(ctrl *mocks.Server) {
				ctrl.On("Task", mock.Anything, taskName).Return(config.TaskConfig{}, nil)
				ctrl.On("TaskDelete", mock.Anything, taskName).Return(nil)
			},
			http.StatusAccepted,
		},
		{
			"task_not_found",
			func(ctrl *mocks.Server) {
				ctrl.On("Task", mock.Anything, taskName).Return(config.TaskConfig{}, fmt.Errorf("DNE"))
			},
			http.StatusNotFound,
		},
		{
			"task_errored",
			func(ctrl *mocks.Server) {
				err := fmt.Errorf("task deletion error")
				ctrl.On("Task", mock.Anything, taskName).Return(config.TaskConfig{}, nil)
				ctrl.On("TaskDelete", mock.Anything, taskName).Return(err)
			},
			http.StatusInternalServerError,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := new(mocks.Server)
			tc.mockServer(ctrl)
			handler := NewTaskLifeCycleHandler(ctrl)

			path := fmt.Sprintf("/v1/tasks/%s", taskName)
			req, err := http.NewRequest(http.MethodDelete, path, nil)
			require.NoError(t, err)
			resp := httptest.NewRecorder()

			handler.DeleteTaskByName(resp, req, taskName)
			assert.Equal(t, tc.statusCode, resp.Code)
		})
	}
}
