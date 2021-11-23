package event

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStore_Add(t *testing.T) {
	cases := []struct {
		name      string
		event     Event
		expectErr bool
	}{
		{
			"happy path",
			Event{TaskName: "happy"},
			false,
		},
		{
			"error: no taskname",
			Event{},
			true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			store := NewStore()
			err := store.Add(tc.event)
			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				events := store.events[tc.event.TaskName]
				assert.Equal(t, 1, len(events))
				event := events[0]
				assert.Equal(t, tc.event, *event)
			}
		})
	}

	t.Run("limit-and-order", func(t *testing.T) {
		store := NewStore()
		store.limit = 2

		// fill store
		store.Add(Event{ID: "1", TaskName: "task"})
		assert.Equal(t, 1, len(store.events["task"]))

		store.Add(Event{ID: "2", TaskName: "task"})
		assert.Equal(t, 2, len(store.events["task"]))

		// check store did not grow beyond limit
		store.Add(Event{ID: "3", TaskName: "task"})
		assert.Equal(t, 2, len(store.events["task"]))

		// confirm events in store
		event3 := store.events["task"][0]
		assert.Equal(t, "3", event3.ID)
		event2 := store.events["task"][1]
		assert.Equal(t, "2", event2.ID)
	})
}

func TestStore_Read(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		values   []Event
		expected map[string][]Event
	}{
		{
			"read all - no events",
			"",
			[]Event{},
			map[string][]Event{},
		},
		{
			"read all - happy path",
			"",
			[]Event{
				Event{TaskName: "1"},
				Event{TaskName: "2"},
				Event{TaskName: "2"},
				Event{TaskName: "3"},
				Event{TaskName: "3"},
				Event{TaskName: "3"},
			},
			map[string][]Event{
				"1": []Event{
					Event{TaskName: "1"},
				},
				"2": []Event{
					Event{TaskName: "2"},
					Event{TaskName: "2"},
				},
				"3": []Event{
					Event{TaskName: "3"},
					Event{TaskName: "3"},
					Event{TaskName: "3"},
				},
			},
		},
		{
			"read task - happy path",
			"4",
			[]Event{
				Event{TaskName: "4"},
				Event{TaskName: "4"},
				Event{TaskName: "5"},
				Event{TaskName: "5"},
				Event{TaskName: "4"},
				Event{TaskName: "4"},
				Event{TaskName: "5"},
			},
			map[string][]Event{
				"4": []Event{
					Event{TaskName: "4"},
					Event{TaskName: "4"},
					Event{TaskName: "4"},
					Event{TaskName: "4"},
				},
			},
		},
		{
			"read task - no event",
			"4",
			[]Event{},
			map[string][]Event{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			store := NewStore()
			for _, event := range tc.values {
				store.Add(event)
			}

			actual := store.Read(tc.input)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestStore_Delete(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		values   []Event
		expected map[string][]Event
	}{
		{
			"delete - happy path",
			"2",
			[]Event{
				Event{TaskName: "1"},
				Event{TaskName: "2"},
				Event{TaskName: "2"},
			},
			map[string][]Event{
				"1": []Event{
					Event{TaskName: "1"},
				},
			},
		},
		{
			"delete task - no event",
			"4",
			[]Event{},
			map[string][]Event{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			store := NewStore()
			for _, event := range tc.values {
				store.Add(event)
			}
			store.Delete(tc.input)

			after := store.Read("")
			assert.Equal(t, tc.expected, after)
		})
	}
}
