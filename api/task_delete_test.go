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

func TestTaskDelete_DeleteTaskByName(t *testing.T) {
	t.Parallel()
	taskName := "task"
	cases := []struct {
		name       string
		mock       func(*mocks.Server)
		active     bool
		deleted    bool
		statusCode int
	}{
		{
			"happy_path",
			func(ctrl *mocks.Server) {
				ctrl.On("Task", mock.Anything, taskName).Return(config.TaskConfig{}, nil)
				ctrl.On("TaskDelete", mock.Anything, taskName).Return(nil)
			},
			false,
			true,
			http.StatusOK,
		},
		{
			"task_not_found",
			func(ctrl *mocks.Server) {
				ctrl.On("Task", mock.Anything, taskName).Return(config.TaskConfig{}, fmt.Errorf("DNE"))
			},
			false,
			true,
			http.StatusNotFound,
		},
		{
			"task_is_running",
			func(ctrl *mocks.Server) {
				err := fmt.Errorf("task '%s' is currently running and cannot be deleted at this time", taskName)
				ctrl.On("Task", mock.Anything, taskName).Return(config.TaskConfig{}, nil)
				ctrl.On("TaskDelete", mock.Anything, taskName).Return(err)
			},
			true,
			false,
			http.StatusConflict,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := new(mocks.Server)
			tc.mock(ctrl)
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
