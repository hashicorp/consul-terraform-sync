package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/consul-terraform-sync/driver"
	"github.com/hashicorp/consul-terraform-sync/event"
	mocks "github.com/hashicorp/consul-terraform-sync/mocks/driver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTaskDelete_deleteTask(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name       string
		path       string
		active     bool
		deleted    bool
		statusCode int
	}{
		{
			"happy path",
			"/v1/tasks/task_a",
			false,
			true,
			http.StatusOK,
		},
		{
			"bad path/taskname",
			"/v1/tasks/task/a",
			false,
			false,
			http.StatusBadRequest,
		},
		{
			"no task specified",
			"/v1/tasks",
			false,
			false,
			http.StatusBadRequest,
		},
		{
			"task not found",
			"/v1/tasks/task_b",
			false,
			false,
			http.StatusNotFound,
		},
		{
			"task_is_running",
			"/v1/tasks/task_a",
			true,
			false,
			http.StatusConflict,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			taskName := "task_a"
			drivers := driver.NewDrivers()
			d := new(mocks.Driver)
			drivers.Add(taskName, d)
			if tc.active {
				drivers.SetActive(taskName)
			}

			store := event.NewStore()
			store.Add(event.Event{TaskName: taskName})
			handler := newTaskHandler(store, drivers, "v1")

			req, err := http.NewRequest(http.MethodDelete, tc.path, nil)
			require.NoError(t, err)
			resp := httptest.NewRecorder()

			handler.deleteTask(resp, req)
			require.Equal(t, tc.statusCode, resp.Code)

			_, ok := drivers.Get(taskName)
			if tc.deleted {
				assert.False(t, ok)
			} else {
				assert.True(t, ok)
			}

			data := store.Read(taskName)
			events, ok := data[taskName]

			if tc.deleted {
				require.False(t, ok)
			} else {
				require.True(t, ok)
				assert.Equal(t, 1, len(events))
			}
		})
	}
}
