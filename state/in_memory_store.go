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
	mu   sync.RWMutex
	conf *config.Config
}

// NewInMemoryStore returns a new in-memory store for CTS state
func NewInMemoryStore(conf *config.Config) *InMemoryStore {
	if conf == nil {
		// expect nil config only for testing
		conf = config.DefaultConfig()
		conf.Finalize()
	}

	return &InMemoryStore{
		conf:   &configStorage{conf: conf.Copy()},
		events: newEventStorage(),
	}
}

func (s *InMemoryStore) GetConfig() config.Config {
	s.conf.mu.RLock()
	defer s.conf.mu.RUnlock()

	return *s.conf.conf.Copy()
}

func (s *InMemoryStore) GetAllTasks() config.TaskConfigs {
	s.conf.mu.RLock()
	defer s.conf.mu.RUnlock()

	taskConfs := s.conf.conf.Tasks
	if taskConfs == nil {
		// not likely to happen. preventative
		return config.TaskConfigs{}
	}

	return *taskConfs.Copy()
}

func (s *InMemoryStore) SetTask(newTaskConf config.TaskConfig) {
	s.conf.mu.Lock()
	defer s.conf.mu.Unlock()

	newTaskName := config.StringVal(newTaskConf.Name)

	taskConfs := s.conf.conf.Tasks
	if taskConfs == nil {
		taskConfs = &config.TaskConfigs{}
	}

	for ix, taskConf := range *taskConfs {
		taskExists := taskConf.Name != nil && *taskConf.Name == newTaskName
		if taskExists {
			// patch update an existing task
			updatedTaskConf := taskConf.Merge(&newTaskConf)
			(*taskConfs)[ix] = updatedTaskConf
			return
		}
	}

	// add as a new task
	*taskConfs = append(*taskConfs, &newTaskConf)
}

func (s *InMemoryStore) GetTask(taskName string) (config.TaskConfig, bool) {
	s.conf.mu.RLock()
	defer s.conf.mu.RUnlock()

	taskConfs := s.conf.conf.Tasks
	if taskConfs == nil {
		// not likely to happen. preventative
		return config.TaskConfig{}, false
	}

	for _, taskConf := range *taskConfs {
		taskExists := taskConf.Name != nil && *taskConf.Name == taskName
		if taskExists {
			return *taskConf.Copy(), true
		}
	}

	return config.TaskConfig{}, false
}

func (s *InMemoryStore) DeleteTask(taskName string) {
	s.conf.mu.Lock()
	defer s.conf.mu.Unlock()

	taskConfs := s.conf.conf.Tasks
	if taskConfs == nil {
		// not likely to happen. preventative
		return
	}

	for ix, taskConf := range *taskConfs {
		taskExists := taskConf.Name != nil && *taskConf.Name == taskName
		if taskExists {
			// delete it
			*taskConfs = append((*taskConfs)[:ix], (*taskConfs)[ix+1:]...)
			return
		}
	}
}

func (s *InMemoryStore) GetTaskEvents(taskName string) map[string][]event.Event {
	return s.events.Read(taskName)
}

func (s *InMemoryStore) DeleteTaskEvents(taskName string) {
	s.events.Delete(taskName)
}

func (s *InMemoryStore) AddTaskEvent(event event.Event) error {
	return s.events.Add(event)
}
