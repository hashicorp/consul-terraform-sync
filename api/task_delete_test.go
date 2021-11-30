package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/consul-terraform-sync/driver"
	"github.com/hashicorp/consul-terraform-sync/event"
	mocks "github.com/hashicorp/consul-terraform-sync/mocks/driver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTaskDelete_DeleteTaskByName(t *testing.T) {
	t.Parallel()
	existingTask := "task_a"
	cases := []struct {
		name       string
		taskName   string
		active     bool
		deleted    bool
		statusCode int
	}{
		{
			"happy_path",
			existingTask,
			false,
			true,
			http.StatusOK,
		},
		{
			"task_not_found",
			"task_b",
			false,
			true,
			http.StatusNotFound,
		},
		{
			"task_is_running",
			existingTask,
			true,
			false,
			http.StatusConflict,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			drivers := driver.NewDrivers()
			d := new(mocks.Driver)
			drivers.Add(existingTask, d)
			if tc.active {
				drivers.SetActive(existingTask)
			}

			store := event.NewStore()
			store.Add(event.Event{TaskName: existingTask})
			handler := NewTaskLifeCycleHandler(store, drivers)

			path := fmt.Sprintf("/v1/tasks/%s", tc.taskName)
			req, err := http.NewRequest(http.MethodDelete, path, nil)
			require.NoError(t, err)
			resp := httptest.NewRecorder()

			handler.DeleteTaskByName(resp, req, tc.taskName)
			require.Equal(t, tc.statusCode, resp.Code)

			_, ok := drivers.Get(tc.taskName)
			if tc.deleted {
				assert.False(t, ok, "task should have been deleted")
			} else {
				assert.True(t, ok, "task should not have been deleted")
			}

			data := store.Read(tc.taskName)
			events, ok := data[tc.taskName]

			if tc.deleted {
				require.False(t, ok, "task should have been deleted")
			} else {
				require.True(t, ok, "task should not have been deleted")
				assert.Equal(t, 1, len(events))
			}
		})
	}
}
