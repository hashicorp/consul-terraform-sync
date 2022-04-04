package state

import (
	"fmt"
	"sync"

	"github.com/hashicorp/consul-terraform-sync/state/event"
)

const defaultEventCountLimit = 5

// eventStorage is the storage for events
type eventStorage struct {
	mu *sync.RWMutex

	events map[string][]*event.Event // taskname => events
	limit  int
}

// newEventStorage returns a new storage for event
func newEventStorage() *eventStorage {
	return &eventStorage{
		mu:     &sync.RWMutex{},
		events: make(map[string][]*event.Event),
		limit:  defaultEventCountLimit,
	}
}

// Add adds an event and manages the limit of number of events stored per task.
func (s *eventStorage) Add(e event.Event) error {
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
func (s *eventStorage) Read(taskName string) map[string][]event.Event {
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
		for ix, e := range v {
			events[ix] = *e
		}
		ret[k] = events
	}
	return ret
}

// Delete removes all events for a task name.
func (s *eventStorage) Delete(taskName string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if taskName != "" {
		delete(s.events, taskName)
	}
}
