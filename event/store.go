package event

import (
	"fmt"
	"sync"
)

const defaultEventCountLimit = 5

// Store stores events
type Store struct {
	mu *sync.RWMutex

	events map[string][]*Event // taskname => events
	limit  int
}

// NewStore returns a new store
func NewStore() *Store {
	return &Store{
		mu:     &sync.RWMutex{},
		events: make(map[string][]*Event),
		limit:  defaultEventCountLimit,
	}
}

// Add adds an event and manages the limit of number of events stored per task.
func (s *Store) Add(e Event) error {
	if e.TaskName == "" {
		return fmt.Errorf("error adding event: taskname cannot be empty %s", e.GoString())
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	events := s.events[e.TaskName]
	events = append([]*Event{&e}, events...) // prepend
	if len(events) > s.limit {
		events = events[:len(events)-1]
	}
	s.events[e.TaskName] = events
	return nil
}

// Read returns events for a task name. If no task name is specified, return
// events for all tasks. Returned events are sorted in reverse chronological
// order based on the end time.
func (s *Store) Read(taskName string) map[string][]Event {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data := make(map[string][]*Event)
	if taskName != "" {
		if e, ok := s.events[taskName]; ok {
			data[taskName] = e
		}
	} else {
		data = s.events
	}

	ret := make(map[string][]Event)
	for k, v := range data {
		events := make([]Event, len(v))
		for ix, event := range v {
			events[ix] = *event
		}
		ret[k] = events
	}
	return ret
}

// Delete removes all events for a task name.
func (s *Store) Delete(taskName string) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if taskName != "" {
		delete(s.events, taskName)
	}
}
