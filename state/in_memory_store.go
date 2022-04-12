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
	conf config.Config
}

// NewInMemoryStore returns a new in-memory store for CTS state
func NewInMemoryStore(conf *config.Config) *InMemoryStore {
	if conf == nil {
		// expect nil config only for testing
		conf = config.DefaultConfig()
	}

	return &InMemoryStore{
		conf:   &configStorage{conf: *conf.Copy()},
		events: newEventStorage(),
	}
}

func (s *InMemoryStore) GetConfig() config.Config {
	s.conf.mu.RLock()
	defer s.conf.mu.RUnlock()

	return *s.conf.conf.Copy()
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
