package state

import (
	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/state/event"
)

var (
	_ Store = (*InMemoryStore)(nil)
)

// InMemoryStore implements the CTS state Store interface.
type InMemoryStore struct {
	conf   config.Config
	events *eventStore
}

func NewInMemoryStore(conf *config.Config) *InMemoryStore {
	if conf == nil {
		// expect nil config only for testing
		conf = config.DefaultConfig()
	}

	return &InMemoryStore{
		conf:   *conf,
		events: newEventStore(),
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
