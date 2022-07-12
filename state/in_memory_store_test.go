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

func Test_InMemoryStore_GetAllTasks(t *testing.T) {
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

			storedConfig := store.conf.Tasks
			assert.Equal(t, *storedConfig, actual)
		})
	}

	t.Run("returned config is dereferenced", func(t *testing.T) {
		finalizedConf := &config.Config{
			Tasks: &config.TaskConfigs{
				{Name: config.String("task_a")},
			},
		}
		finalizedConf.Finalize()
		store := NewInMemoryStore(finalizedConf)

		actual := store.GetAllTasks()
		storedConf := store.conf.Tasks

		// Confirm returned config has same value as stored
		assert.Equal(t, *storedConf, actual)

		// Confirm returned config references different object from stored
		assert.NotSame(t, storedConf, actual)

		// Confirm returned config field reference different object from stored
		assert.NotSame(t, (*storedConf)[0], actual[0])
	})
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
		})
	}
	t.Run("returned config is dereferenced", func(t *testing.T) {
		finalizedConf := &config.Config{
			Tasks: &config.TaskConfigs{
				{Name: config.String("task_a")},
			},
		}
		finalizedConf.Finalize()
		store := NewInMemoryStore(finalizedConf)

		actual, exists := store.GetTask("task_a")
		assert.True(t, exists)

		assert.Len(t, *store.conf.Tasks, 1)
		storedConf := (*store.conf.Tasks)[0]

		// Confirm returned config has same value as stored
		assert.Equal(t, *storedConf, actual)

		// Confirm returned config references different object from stored
		assert.NotSame(t, storedConf, &actual)

		// Confirm returned config field reference different object from stored
		assert.NotSame(t, storedConf.Name, actual.Name)
	})
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

			if err := store.SetTask(tc.input); err != nil {
				assert.NoError(t, err, "unexpected error while setting task state")
			}
			assert.Equal(t, tc.expected, *store.conf.Tasks)
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
			"existing_task_a",
			config.TaskConfigs{
				{Name: config.String("existing_task_b")},
			},
		},
		{
			"task does not exist",
			"non_existing_task",
			config.TaskConfigs{
				{Name: config.String("existing_task_a")},
				{Name: config.String("existing_task_b")},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			conf := &config.Config{
				Tasks: &config.TaskConfigs{
					{Name: config.String("existing_task_a")},
					{Name: config.String("existing_task_b")},
				},
			}
			store := NewInMemoryStore(conf)

			store.DeleteTask(tc.input)
			assert.Equal(t, tc.expected, *store.conf.Tasks)
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

			if err = store.DeleteTaskEvents(tc.taskName); err != nil {
				assert.NoError(t, err, "unexpected error while deleting task events")
			}
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
