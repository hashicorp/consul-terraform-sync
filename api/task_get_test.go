// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/hashicorp/consul-terraform-sync/api/oapigen"
	"github.com/hashicorp/consul-terraform-sync/config"
	mocks "github.com/hashicorp/consul-terraform-sync/mocks/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestTaskLifeCycleHandler_GetTaskByName(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name          string
		mockServer    func(*mocks.Server)
		statusCode    int
		checkResponse func(*httptest.ResponseRecorder)
	}{
		{
			name: "happy_path",
			mockServer: func(ctrl *mocks.Server) {
				ctrl.On("Task", mock.Anything, testTaskName).Return(testTaskConfig, nil)
			},
			statusCode: http.StatusOK,
			checkResponse: func(resp *httptest.ResponseRecorder) {
				decoder := json.NewDecoder(resp.Body)
				var actual oapigen.TaskResponse
				err := decoder.Decode(&actual)
				require.NoError(t, err)
				expected := generateExpectedResponse(t, testTaskJSON)
				assert.Equal(t, expected, actual)

			},
		},
		{
			name: "not_found",
			mockServer: func(ctrl *mocks.Server) {
				ctrl.On("Task", mock.Anything, testTaskName).Return(config.TaskConfig{}, fmt.Errorf("DNE"))
			},
			statusCode: http.StatusNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := new(mocks.Server)
			tc.mockServer(ctrl)
			handler := NewTaskLifeCycleHandler(ctrl)

			path := fmt.Sprintf("/v1/tasks/%s", testTaskName)
			req, err := http.NewRequest(http.MethodGet, path, nil)
			require.NoError(t, err)
			resp := httptest.NewRecorder()

			handler.GetTaskByName(resp, req, testTaskName)
			assert.Equal(t, tc.statusCode, resp.Code)

			if tc.checkResponse != nil {
				tc.checkResponse(resp)
			}
		})
	}
}

func TestTaskLifeCycleHandler_GetAllTasks(t *testing.T) {
	t.Parallel()

	taskConfigs := config.TaskConfigs{
		&testTaskConfig,
		&testTaskConfig,
	}

	reqID := uuid.New()

	ctrl := new(mocks.Server)
	ctrl.On("Tasks", mock.Anything).Return(taskConfigs)
	handler := NewTaskLifeCycleHandler(ctrl)

	path := fmt.Sprintf("/v1/tasks")
	ctx := requestIDWithContext(context.Background(), reqID.String())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, path, nil)

	require.NoError(t, err)
	resp := httptest.NewRecorder()

	handler.GetAllTasks(resp, req)
	assert.Equal(t, http.StatusOK, resp.Code)

	decoder := json.NewDecoder(resp.Body)
	var actual oapigen.TasksResponse
	err = decoder.Decode(&actual)
	require.NoError(t, err)

	expectedTasksResponse := tasksResponseFromTaskConfigs(taskConfigs, reqID)
	assert.ElementsMatch(t, *expectedTasksResponse.Tasks, *actual.Tasks)
	assert.ElementsMatch(t, expectedTasksResponse.RequestId, reqID)
}
