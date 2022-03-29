package state

import (
	"fmt"
	"sync"

	"github.com/hashicorp/consul-terraform-sync/state/event"
)

const defaultEventCountLimit = 5

// eventStore stores events
type eventStore struct {
	mu *sync.RWMutex

	events map[string][]*event.Event // taskname => events
	limit  int
}

// newEventStore returns a new store for event
func newEventStore() *eventStore {
	return &eventStore{
		mu:     &sync.RWMutex{},
		events: make(map[string][]*event.Event),
		limit:  defaultEventCountLimit,
	}
}

// Add adds an event and manages the limit of number of events stored per task.
func (s *eventStore) Add(e event.Event) error {
	if e.TaskName == "" {
		return fmt.Errorf("error adding event: taskname cannot be empty %s", e.GoString())
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	events := s.events[e.TaskName]
	events = append([]*event.Event{&e}, events...) // prepend
	if len(events) > s.limit {
		events = events[:len(events)-1]
	}
	s.events[e.TaskName] = events
	return nil
}

// Read returns events for a task name. If no task name is specified, return
// events for all tasks. Returned events are sorted in reverse chronological
// order based on the end time.
func (s *eventStore) Read(taskName string) map[string][]event.Event {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data := make(map[string][]*event.Event)
	if taskName != "" {
		if e, ok := s.events[taskName]; ok {
			data[taskName] = e
		}
	} else {
		data = s.events
	}

	ret := make(map[string][]event.Event)
	for k, v := range data {
		events := make([]event.Event, len(v))
		for ix, event := range v {
			events[ix] = *event
		}
		ret[k] = events
	}
	return ret
}

// Delete removes all events for a task name.
func (s *eventStore) Delete(taskName string) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if taskName != "" {
		delete(s.events, taskName)
	}
}
