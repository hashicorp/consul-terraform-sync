package state

import (
	"sync"

	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/state/event"
)

var (
	_ Store = (*InMemoryStore)(nil)
)

// InMemoryStore implements the CTS state Store interface.
type InMemoryStore struct {
	conf   *configStorage
	events *eventStorage
}

// configStorage is the storage for the configuration with its own mutex lock
type configStorage struct {
	config.Config
	mu sync.RWMutex
}

// NewInMemoryStore returns a new in-memory store for CTS state
func NewInMemoryStore(conf *config.Config) *InMemoryStore {
	if conf == nil {
		// expect nil config only for testing
		conf = config.DefaultConfig()
	}

	return &InMemoryStore{
		conf:   &configStorage{Config: *conf.Copy()},
		events: newEventStorage(),
	}
}

// GetConfig returns a copy of the CTS configuration
func (s *InMemoryStore) GetConfig() config.Config {
	s.conf.mu.RLock()
	defer s.conf.mu.RUnlock()

	return *s.conf.Copy()
}

// GetAllTasks returns a copy of the configs for all the tasks
func (s *InMemoryStore) GetAllTasks() config.TaskConfigs {
	s.conf.mu.RLock()
	defer s.conf.mu.RUnlock()

	taskConfs := s.conf.Tasks
	if taskConfs == nil {
		// expect nil only for testing
		return config.TaskConfigs{}
	}

	return *taskConfs.Copy()
}

// GetTask returns a copy of the task configuration. If the task name does
// not exist, then it returns false
func (s *InMemoryStore) GetTask(taskName string) (config.TaskConfig, bool) {
	s.conf.mu.RLock()
	defer s.conf.mu.RUnlock()

	taskConfs := s.conf.Tasks
	if taskConfs == nil {
		// expect nil only for testing
		return config.TaskConfig{}, false
	}

	for _, taskConf := range *taskConfs {
		if config.StringVal(taskConf.Name) == taskName {
			return *taskConf.Copy(), true
		}
	}

	return config.TaskConfig{}, false
}

// SetTask adds a new task configuration or does a patch update to an
// existing task configuration with the same name
func (s *InMemoryStore) SetTask(newTaskConf config.TaskConfig) error {
	s.conf.mu.Lock()
	defer s.conf.mu.Unlock()

	newTaskName := config.StringVal(newTaskConf.Name)

	taskConfs := s.conf.Tasks
	if taskConfs == nil {
		taskConfs = &config.TaskConfigs{}
	}

	for ix, taskConf := range *taskConfs {
		if config.StringVal(taskConf.Name) == newTaskName {
			// patch update the existing task
			updatedTaskConf := taskConf.Merge(&newTaskConf)
			(*taskConfs)[ix] = updatedTaskConf
			return nil
		}
	}

	// add as a new task
	*taskConfs = append(*taskConfs, &newTaskConf)
	return nil
}

// DeleteTask deletes the task config if it exists
func (s *InMemoryStore) DeleteTask(taskName string) error {
	s.conf.mu.Lock()
	defer s.conf.mu.Unlock()

	taskConfs := s.conf.Tasks
	if taskConfs == nil {
		// expect nil only for testing
		return nil
	}

	for ix, taskConf := range *taskConfs {
		if config.StringVal(taskConf.Name) == taskName {
			// delete it
			*taskConfs = append((*taskConfs)[:ix], (*taskConfs)[ix+1:]...)
			return nil
		}
	}
	return nil
}

// GetTaskEvents returns all the events for a task. If no task name is
// specified, then it returns events for all tasks
func (s *InMemoryStore) GetTaskEvents(taskName string) map[string][]event.Event {
	return s.events.Read(taskName)
}

// DeleteTaskEvents deletes all the events for a given task
func (s *InMemoryStore) DeleteTaskEvents(taskName string) error {
	s.events.Delete(taskName)
	return nil
}

// AddTaskEvent adds an event to the store for the task configured in the
// event
func (s *InMemoryStore) AddTaskEvent(event event.Event) error {
	return s.events.Add(event)
}
