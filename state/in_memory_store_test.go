package state

import (
	"testing"

	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/state/event"
	"github.com/stretchr/testify/assert"
)

func Test_NewInMemoryStore(t *testing.T) {
	t.Parallel()

	finalizedConf := config.DefaultConfig()
	finalizedConf.Finalize()

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
					conf: finalizedConf,
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
					conf: &config.Config{
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
		actual := NewInMemoryStore(finalizedConf)

		// Confirm that input and stored config have same values
		assert.Equal(t, *finalizedConf, *actual.conf.conf)

		// Confrm that input and stored config reference different objects
		assert.NotSame(t, finalizedConf, actual.conf.conf)

		// Confrm that input and stored config fields reference different objects
		assert.NotSame(t, finalizedConf.Tasks, actual.conf.conf.Tasks)
	})
}

func Test_InMemoryStore_GetConfig(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		expected *config.Config
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
			store := NewInMemoryStore(tc.expected)
			actual := store.GetConfig()

			storedConfig := store.conf.conf

			// Confirm returned config has same value as stored
			assert.Equal(t, *storedConfig, actual)

			// Confirm returned config references different object from stored
			assert.NotSame(t, storedConfig, actual)

			// Confirm returned config children reference different object
			// from stored
			assert.NotSame(t, storedConfig.Port, actual.Port)
		})
	}
}

func Test_InMemoryStore_GetAssignedTasks(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		input config.Config
	}{
		{
			"nil tasks",
			config.Config{},
		},
		{
			"happy path",
			config.Config{
				Tasks: &config.TaskConfigs{
					{Name: config.String("task_a")},
					{Name: config.String("task_b")},
					{Name: config.String("task_c")},
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.input.Finalize()
			store := NewInMemoryStore(&tc.input)
			actual := store.GetAllTasks()

			storedConfig := store.conf.conf.Tasks

			// Confirm returned task configs has same value as stored
			assert.Equal(t, *storedConfig, actual)

			// Confirm returned config references different object from stored
			assert.NotSame(t, storedConfig, actual)

			if len(*tc.input.Tasks) > 0 {
				// Confirm returned config children reference different object
				// from stored
				assert.NotSame(t, (*storedConfig)[0], actual[0])
			}
		})
	}
}

func Test_InMemoryStore_SetTask(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		stateConf *config.Config
		input     config.TaskConfig
		expected  config.TaskConfigs
	}{
		{
			name: "update existing task",
			stateConf: &config.Config{
				Tasks: &config.TaskConfigs{
					{Name: config.String("existing_task")},
				},
			},
			input: config.TaskConfig{
				Name:        config.String("existing_task"),
				Description: config.String("new description"),
			},
			expected: config.TaskConfigs{
				&config.TaskConfig{
					Name:        config.String("existing_task"),
					Description: config.String("new description"),
				},
			},
		},
		{
			name: "add new task",
			stateConf: &config.Config{
				Tasks: &config.TaskConfigs{
					{Name: config.String("existing_task")},
				},
			},
			input: config.TaskConfig{
				Name: config.String("new_task"),
			},
			expected: config.TaskConfigs{
				&config.TaskConfig{
					Name: config.String("existing_task"),
				},
				&config.TaskConfig{
					Name: config.String("new_task"),
				},
			},
		},
		{
			name:      "no prior existing tasks",
			stateConf: &config.Config{},
			input: config.TaskConfig{
				Name: config.String("new_task"),
			},
			expected: config.TaskConfigs{
				&config.TaskConfig{
					Name: config.String("new_task"),
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.stateConf.Finalize()
			store := NewInMemoryStore(tc.stateConf)

			// finalize the task configs
			bp := tc.stateConf.BufferPeriod
			wd := config.StringVal(tc.stateConf.WorkingDir)
			tc.input.Finalize(bp, wd)
			tc.expected.Finalize(bp, wd)

			store.SetTask(tc.input)
			assert.Equal(t, tc.expected, *store.conf.conf.Tasks)
		})
	}
}

func Test_InMemoryStore_GetTask(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		input    string
		exists   bool
		expected config.TaskConfig
	}{
		{
			"task exists",
			"existing_task",
			true,
			config.TaskConfig{Name: config.String("existing_task")},
		},
		{
			"task doesn't exist",
			"non_existing_task",
			false,
			config.TaskConfig{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			conf := &config.Config{
				Tasks: &config.TaskConfigs{
					{Name: config.String("existing_task")},
				},
			}
			store := NewInMemoryStore(conf)

			actual, exists := store.GetTask(tc.input)
			assert.Equal(t, tc.exists, exists)
			assert.Equal(t, tc.expected, actual)

			if exists {
				assert.Len(t, *store.conf.conf.Tasks, 1)
				storedConfig := (*store.conf.conf.Tasks)[0]

				// Confirm returned task config references different object from stored
				assert.NotSame(t, *storedConfig, actual)

				// Confirm returned config children reference different object
				// from stored
				assert.NotSame(t, storedConfig.Name, actual.Name)
			}
		})
	}
}
func Test_InMemoryStore_DeleteTask(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		input    string
		expected config.TaskConfigs
	}{
		{
			"task exists",
			"existing_task",
			config.TaskConfigs{},
		},
		{
			"task does not exists",
			"non_existing_task",
			config.TaskConfigs{
				{Name: config.String("existing_task")},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			conf := &config.Config{
				Tasks: &config.TaskConfigs{
					{Name: config.String("existing_task")},
				},
			}
			store := NewInMemoryStore(conf)

			store.DeleteTask(tc.input)

			assert.Equal(t, tc.expected, *store.conf.conf.Tasks)
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
