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
				conf: &configStorage{
					Config: *config.DefaultConfig(),
				},
				events: newEventStorage(),
			},
		},
		{
			"non-nil config",
			&config.Config{
				Port: config.Int(1234),
			},
			InMemoryStore{
				conf: &configStorage{
					Config: config.Config{
						Port: config.Int(1234),
					},
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

	t.Run("stored config is dereferenced", func(t *testing.T) {
		finalizedConf := config.DefaultConfig()
		finalizedConf.Finalize()
		actual := NewInMemoryStore(finalizedConf)

		// Confirm that input and stored config have same values
		assert.Equal(t, *finalizedConf, actual.conf.Config)

		// Confrm that input and stored config reference different objects
		assert.NotSame(t, finalizedConf, actual.conf.Config)

		// Confrm that input and stored config fields reference different objects
		assert.NotSame(t, finalizedConf.Tasks, actual.conf.Tasks)
	})
}

func Test_InMemoryStore_GetConfig(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		input *config.Config
	}{
		{
			"happy path",
			&config.Config{
				Port: config.Int(1234),
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			store := NewInMemoryStore(tc.input)

			actual := store.GetConfig()
			assert.Equal(t, *tc.input, actual)
		})
	}

	t.Run("returned config is dereferenced", func(t *testing.T) {
		finalizedConf := config.DefaultConfig()
		finalizedConf.Finalize()
		store := NewInMemoryStore(finalizedConf)

		actual := store.GetConfig()
		storedConf := store.conf.Config

		// Confirm returned config has same value as stored
		assert.Equal(t, storedConf, actual)

		// Confirm returned config references different object from stored
		assert.NotSame(t, storedConf, actual)

		// Confirm returned config field reference different object from stored
		assert.NotSame(t, storedConf.Port, actual.Port)
	})

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
			err := events.Add(event.Event{TaskName: "existing_task"})
			assert.NoError(t, err)

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
			err := events.Add(event.Event{TaskName: "existing_task"})
			assert.NoError(t, err)

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
			err := events.Add(event.Event{TaskName: "existing_task"})
			assert.NoError(t, err)

			store := InMemoryStore{
				events: events,
			}

			err = store.AddTaskEvent(tc.event)
			assert.NoError(t, err)

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
