package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"

	"github.com/hashicorp/consul-terraform-sync/driver"
	"github.com/hashicorp/consul-terraform-sync/event"
	mocks "github.com/hashicorp/consul-terraform-sync/mocks/driver"
	"github.com/hashicorp/consul-terraform-sync/templates/hcltmpl"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestTaskStatus_New(t *testing.T) {
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
			h := newTaskStatusHandler(event.NewStore(), nil, tc.version)
			assert.Equal(t, tc.version, h.version)
		})
	}
}

func TestTaskStatus_ServeHTTP(t *testing.T) {
	t.Parallel()

	// set up store and handler
	store := event.NewStore()

	// task A is in successful state
	eventsA := createTaskEvents("task_a", []bool{true})
	addEvents(store, eventsA)

	// task B is in critical state
	eventsB := createTaskEvents("task_b", []bool{false, false, true})
	addEvents(store, eventsB)

	// task C is in errored state
	eventsC := createTaskEvents("task_c", []bool{false, true, true})
	addEvents(store, eventsC)

	drivers := driver.NewDrivers()
	drivers.Add("task_a", createDriver("task_a", true))
	drivers.Add("task_b", createDriver("task_b", true))
	drivers.Add("task_c", createDriver("task_c", true))

	disabledD := new(mocks.Driver)
	disabledD.On("Task").Return(driver.Task{
		Name:    "task_d",
		Enabled: false,
		Providers: driver.NewTerraformProviderBlocks(
			hcltmpl.NewNamedBlocksTest([]map[string]interface{}{
				{"null": map[string]interface{}{}},
			})),
		Services: []driver.Service{
			driver.Service{Name: "web"},
		},
	})
	drivers.Add("task_d", disabledD)

	handler := newTaskStatusHandler(store, drivers, "v1")

	cases := []struct {
		name       string
		path       string
		statusCode int
		expected   map[string]TaskStatus
	}{
		{
			"all task statuses",
			"/v1/status/tasks",
			http.StatusOK,
			map[string]TaskStatus{
				"task_a": TaskStatus{
					TaskName:  "task_a",
					Status:    StatusSuccessful,
					Enabled:   true,
					Providers: []string{},
					Services:  []string{},
					EventsURL: "/v1/status/tasks/task_a?include=events",
				},
				"task_b": TaskStatus{
					TaskName:  "task_b",
					Status:    StatusCritical,
					Enabled:   true,
					Providers: []string{},
					Services:  []string{},
					EventsURL: "/v1/status/tasks/task_b?include=events",
				},
				"task_c": TaskStatus{
					TaskName:  "task_c",
					Status:    StatusErrored,
					Enabled:   true,
					Providers: []string{},
					Services:  []string{},
					EventsURL: "/v1/status/tasks/task_c?include=events",
				},
				"task_d": TaskStatus{
					TaskName:  "task_d",
					Status:    StatusUnknown,
					Enabled:   false,
					Providers: []string{"null"},
					Services:  []string{"web"},
					EventsURL: "",
				},
			},
		},
		{
			"all task statuses with events",
			"/v1/status/tasks?include=events",
			http.StatusOK,
			map[string]TaskStatus{
				"task_a": TaskStatus{
					TaskName:  "task_a",
					Status:    StatusSuccessful,
					Enabled:   true,
					Providers: []string{},
					Services:  []string{},
					EventsURL: "/v1/status/tasks/task_a?include=events",
					Events:    eventsA,
				},
				"task_b": TaskStatus{
					TaskName:  "task_b",
					Status:    StatusCritical,
					Enabled:   true,
					Providers: []string{},
					Services:  []string{},
					EventsURL: "/v1/status/tasks/task_b?include=events",
					Events:    eventsB,
				},
				"task_c": TaskStatus{
					TaskName:  "task_c",
					Status:    StatusErrored,
					Enabled:   true,
					Providers: []string{},
					Services:  []string{},
					EventsURL: "/v1/status/tasks/task_c?include=events",
					Events:    eventsC,
				},
				"task_d": TaskStatus{
					TaskName:  "task_d",
					Status:    StatusUnknown,
					Enabled:   false,
					Providers: []string{"null"},
					Services:  []string{"web"},
					EventsURL: "",
					Events:    nil,
				},
			},
		},
		{
			"all task statuses filtered by status critical",
			"/v1/status/tasks?status=critical",
			http.StatusOK,
			map[string]TaskStatus{
				"task_b": TaskStatus{
					TaskName:  "task_b",
					Status:    StatusCritical,
					Enabled:   true,
					Providers: []string{},
					Services:  []string{},
					EventsURL: "/v1/status/tasks/task_b?include=events",
				},
			},
		},
		{
			"all task statuses filtered by status unknown",
			"/v1/status/tasks?status=unknown",
			http.StatusOK,
			map[string]TaskStatus{
				"task_d": TaskStatus{
					TaskName:  "task_d",
					Status:    StatusUnknown,
					Enabled:   false,
					Providers: []string{"null"},
					Services:  []string{"web"},
					EventsURL: "",
				},
			},
		},
		{
			"single task",
			"/v1/status/tasks/task_b",
			http.StatusOK,
			map[string]TaskStatus{
				"task_b": TaskStatus{
					TaskName:  "task_b",
					Status:    StatusCritical,
					Enabled:   true,
					Providers: []string{},
					Services:  []string{},
					EventsURL: "/v1/status/tasks/task_b?include=events",
				},
			},
		},
		{
			"single task with events",
			"/v1/status/tasks/task_b?include=events",
			http.StatusOK,
			map[string]TaskStatus{
				"task_b": TaskStatus{
					TaskName:  "task_b",
					Status:    StatusCritical,
					Enabled:   true,
					Providers: []string{},
					Services:  []string{},
					EventsURL: "/v1/status/tasks/task_b?include=events",
					Events:    eventsB,
				},
			},
		},
		{
			"single task that has no event data",
			"/v1/status/tasks/task_d",
			http.StatusOK,
			map[string]TaskStatus{
				"task_d": TaskStatus{
					TaskName:  "task_d",
					Status:    StatusUnknown,
					Enabled:   false,
					Providers: []string{"null"},
					Services:  []string{"web"},
					EventsURL: "",
				},
			},
		},
		{
			"non-existent task",
			"/v1/status/tasks/task_nonexistent",
			http.StatusNotFound,
			map[string]TaskStatus{},
		},
		{
			"non-existent task with events",
			"/v1/status/tasks/task_nonexistent?include=events",
			http.StatusNotFound,
			map[string]TaskStatus{},
		},
		{
			"bad include parameter",
			"/v1/status/tasks?include=wrongparam",
			http.StatusBadRequest,
			map[string]TaskStatus{},
		},
		{
			"bad status parameter",
			"/v1/status/tasks?status=invalidparam",
			http.StatusBadRequest,
			map[string]TaskStatus{},
		},
		{
			"bad url path",
			"/v1/status/tasks/task_b/events",
			http.StatusBadRequest,
			map[string]TaskStatus{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", tc.path, nil)
			require.NoError(t, err)
			resp := httptest.NewRecorder()

			handler.ServeHTTP(resp, req)

			require.Equal(t, tc.statusCode, resp.Code)
			if tc.statusCode != http.StatusOK {
				return
			}

			decoder := json.NewDecoder(resp.Body)
			var actual map[string]TaskStatus
			err = decoder.Decode(&actual)
			require.NoError(t, err)

			assert.Equal(t, tc.expected, actual)
		})
	}

}

func TestTaskStatus_MakeStatus(t *testing.T) {
	cases := []struct {
		name     string
		events   []event.Event
		task     driver.Task
		expected TaskStatus
	}{
		{
			"happy path",
			[]event.Event{
				event.Event{
					Success: true,
					Config: &event.Config{
						Providers: []string{"local", "null"},
						Services:  []string{"api", "web"},
					},
				},
				event.Event{
					Success: false,
					Config: &event.Config{
						Providers: []string{"local"},
					},
				},
				event.Event{
					Success: false,
					Config: &event.Config{
						Providers: []string{"f5"},
						Services:  []string{"db"},
					},
				},
			},
			driver.Task{Name: "test_task", Enabled: true},
			TaskStatus{
				TaskName:  "test_task",
				Enabled:   true,
				Status:    StatusSuccessful,
				Providers: []string{"local", "null", "f5"},
				Services:  []string{"api", "web", "db"},
				EventsURL: "/v1/status/tasks/test_task?include=events",
			},
		},
		{
			"no events",
			[]event.Event{},
			driver.Task{Name: "test_task", Enabled: true},
			TaskStatus{
				TaskName:  "test_task",
				Enabled:   true,
				Status:    StatusUnknown,
				Providers: []string{},
				Services:  []string{},
				EventsURL: "",
			},
		},
		{
			"no config",
			[]event.Event{
				event.Event{
					Success: false,
					Config:  nil,
				},
				event.Event{
					Success: false,
					Config:  nil,
				},
			},
			driver.Task{Name: "test_task", Enabled: true},
			TaskStatus{
				TaskName:  "test_task",
				Enabled:   true,
				Status:    StatusCritical,
				Providers: []string{},
				Services:  []string{},
				EventsURL: "/v1/status/tasks/test_task?include=events",
			},
		},
		{
			"disabled task",
			[]event.Event{
				event.Event{
					Success: true,
					Config: &event.Config{
						Providers: []string{"local"},
						Services:  []string{"api"},
					},
				},
			},
			driver.Task{Name: "test_task", Enabled: false},
			TaskStatus{
				TaskName:  "test_task",
				Enabled:   false,
				Status:    StatusSuccessful,
				Providers: []string{"local"},
				Services:  []string{"api"},
				EventsURL: "/v1/status/tasks/test_task?include=events",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual := makeTaskStatus(tc.events, tc.task, "v1")
			sort.Strings(tc.expected.Providers)
			sort.Strings(tc.expected.Services)
			sort.Strings(actual.Providers)
			sort.Strings(actual.Services)

			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestTaskStatus_MapKeyToArray(t *testing.T) {
	cases := []struct {
		name     string
		input    map[string]bool
		expected []string
	}{
		{
			"happy path",
			map[string]bool{
				"api":     true,
				"web":     true,
				"service": true,
			},
			[]string{"api", "service", "web"},
		},
		{
			"empty map",
			map[string]bool{},
			[]string{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual := mapKeyToArray(tc.input)
			sort.Strings(actual)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestTaskStatus_SuccessToStatus(t *testing.T) {
	cases := []struct {
		name      string
		successes []bool
		status    string
	}{
		{
			"all successes",
			[]bool{true, true, true, true, true},
			StatusSuccessful,
		},
		{
			"lastest successful",
			[]bool{true, false, false, false, false},
			StatusSuccessful,
		},
		{
			"latest errored",
			[]bool{false, true, true, true, true},
			StatusErrored,
		},
		{
			"latest errored with prior error",
			[]bool{false, true, false, true, true},
			StatusCritical,
		},
		{
			"no successes",
			[]bool{false, false, false, false, false},
			StatusCritical,
		},
		{
			"no data",
			[]bool{},
			StatusUnknown,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual := successToStatus(tc.successes)
			assert.Equal(t, tc.status, actual)
		})
	}
}

func TestTaskStatus_MakeEventsURL(t *testing.T) {
	cases := []struct {
		name     string
		events   []event.Event
		expected string
	}{
		{
			"no events",
			[]event.Event{},
			"",
		},
		{
			"events",
			[]event.Event{event.Event{}},
			"/v1/status/tasks/my_task?include=events",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual := makeEventsURL(tc.events, "v1", "my_task")
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestTaskStatus_Include(t *testing.T) {
	cases := []struct {
		name        string
		path        string
		include     bool
		expectError bool
	}{
		{
			"happy path include",
			"/v1/status?include=events",
			true,
			false,
		},
		{
			"happy path include with other parameters",
			"/v1/status?include=events&status=critical",
			true,
			false,
		},
		{
			"happy path don't include",
			"/v1/status",
			false,
			false,
		},
		{
			"bad include parameter",
			"/v1/status?include=badparam",
			false,
			true,
		},
		{
			"missing include value",
			"/v1/status?include=",
			false,
			true,
		},
		{
			"too many include parameters",
			"/v1/status?include=stuff&include=morestuff",
			false,
			true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", tc.path, nil)
			require.NoError(t, err)

			actual, err := include(req)
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.include, actual)
			}
		})
	}
}

func TestTaskStatus_StatusFilter(t *testing.T) {
	cases := []struct {
		name        string
		path        string
		status      string
		expectError bool
	}{
		{
			"happy path status",
			"/v1/status/tasks?status=successful",
			StatusSuccessful,
			false,
		},
		{
			"happy path status with other parameters",
			"/v1/status/tasks?&status=successful&include=events",
			StatusSuccessful,
			false,
		},
		{
			"happy path no status",
			"/v1/status/tasks",
			"",
			false,
		},
		{
			"not lower case",
			"/v1/status/tasks?status=SUCCESSFUL",
			StatusSuccessful,
			false,
		},
		{
			"unknown status",
			"/v1/status/tasks?status=badstatus",
			"",
			true,
		},
		{
			"too many status parameters",
			"/v1/status/tasks?status=successful&status=critical",
			"",
			true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest("GET", tc.path, nil)
			require.NoError(t, err)

			actual, err := statusFilter(req)
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.status, actual)
			}
		})
	}
}

// createTaskEvents is a test helper function to create a list of events for a
// task in reverse chronological order, resembling what is returned from
// store.Read().
func createTaskEvents(taskName string, successes []bool) []event.Event {
	events := make([]event.Event, len(successes))
	for i, s := range successes {
		events[i] = event.Event{TaskName: taskName, Success: s}
	}
	return events
}

// addEvents is a test helper function to add events in chronological
// order from a list of events sorted by latest first.
func addEvents(store *event.Store, events []event.Event) {
	for i := len(events) - 1; i >= 0; i-- {
		store.Add(events[i])
	}
}

func createDriver(taskName string, enabled bool) driver.Driver {
	d := new(mocks.Driver)
	d.On("UpdateTask", mock.Anything, mock.Anything).Return("", nil).Once()
	d.On("Task").Return(driver.Task{
		Name:    taskName,
		Enabled: enabled,
	})
	return d
}
