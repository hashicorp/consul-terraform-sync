package state

import (
	"testing"

	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/state/event"
	"github.com/stretchr/testify/assert"
)

func Test_NewInMemoryStore(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		conf     *config.Config
		expected InMemoryStore
	}{
		{
			"nil config",
			nil,
			InMemoryStore{
				conf:   *config.DefaultConfig(),
				events: newEventStorage(),
			},
		},
		{
			"non-nil config",
			&config.Config{
				Port: config.Int(1234),
			},
			InMemoryStore{
				conf: config.Config{
					Port: config.Int(1234),
				},
				events: newEventStorage(),
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual := NewInMemoryStore(tc.conf)
			assert.Equal(t, tc.expected, *actual)
		})
	}
}

func Test_InMemoryStore_GetTaskEvents(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		taskName string
		expected map[string][]event.Event
	}{
		{
			"happy path",
			"existing_task",
			map[string][]event.Event{
				"existing_task": {{TaskName: "existing_task"}},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			events := newEventStorage()
			events.Add(event.Event{TaskName: "existing_task"})

			store := InMemoryStore{
				events: events,
			}

			actual := store.GetTaskEvents(tc.taskName)
			assert.Equal(t, tc.expected, actual)
		})
	}
}
func Test_InMemoryStore_DeleteTaskEvents(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		taskName string
	}{
		{
			"happy path",
			"existing_task",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			events := newEventStorage()
			events.Add(event.Event{TaskName: "existing_task"})

			store := InMemoryStore{
				events: events,
			}

			store.DeleteTaskEvents(tc.taskName)
			// confirm task's events deleted
			actual := events.Read(tc.taskName)
			_, exists := actual[tc.taskName]
			assert.False(t, exists)
		})
	}
}
func Test_InMemoryStore_AddTaskEvent(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		event event.Event
	}{
		{
			"happy path",
			event.Event{TaskName: "new_task"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			events := newEventStorage()
			events.Add(event.Event{TaskName: "existing_task"})

			store := InMemoryStore{
				events: events,
			}

			store.AddTaskEvent(tc.event)
			// confirm event is added
			taskName := tc.event.TaskName
			actual := events.Read(taskName)
			actualEvents := actual[taskName]
			exists := false
			for _, actualEvent := range actualEvents {
				if actualEvent == tc.event {
					exists = true
				}
			}
			assert.True(t, exists)
		})
	}
}
