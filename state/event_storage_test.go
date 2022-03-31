package state

import (
	"testing"

	"github.com/hashicorp/consul-terraform-sync/state/event"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_eventStorage_Add(t *testing.T) {
	cases := []struct {
		name      string
		event     event.Event
		expectErr bool
	}{
		{
			"happy path",
			event.Event{TaskName: "happy"},
			false,
		},
		{
			"error: no taskname",
			event.Event{},
			true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			storage := newEventStorage()
			err := storage.Add(tc.event)
			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				events := storage.events[tc.event.TaskName]
				assert.Equal(t, 1, len(events))
				event := events[0]
				assert.Equal(t, tc.event, *event)
			}
		})
	}

	t.Run("limit-and-order", func(t *testing.T) {
		storage := newEventStorage()
		storage.limit = 2

		// fill storage
		err := storage.Add(event.Event{ID: "1", TaskName: "task"})
		require.NoError(t, err)
		assert.Equal(t, 1, len(storage.events["task"]))

		err = storage.Add(event.Event{ID: "2", TaskName: "task"})
		require.NoError(t, err)
		assert.Equal(t, 2, len(storage.events["task"]))

		// check storage did not grow beyond limit
		err = storage.Add(event.Event{ID: "3", TaskName: "task"})
		require.NoError(t, err)
		assert.Equal(t, 2, len(storage.events["task"]))

		// confirm events in storage
		event3 := storage.events["task"][0]
		assert.Equal(t, "3", event3.ID)
		event2 := storage.events["task"][1]
		assert.Equal(t, "2", event2.ID)
	})
}

func Test_eventStorage_Read(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		values   []event.Event
		expected map[string][]event.Event
	}{
		{
			"read all - no events",
			"",
			[]event.Event{},
			map[string][]event.Event{},
		},
		{
			"read all - happy path",
			"",
			[]event.Event{
				{TaskName: "1"},
				{TaskName: "2"},
				{TaskName: "2"},
				{TaskName: "3"},
				{TaskName: "3"},
				{TaskName: "3"},
			},
			map[string][]event.Event{
				"1": {{TaskName: "1"}},
				"2": {{TaskName: "2"}, {TaskName: "2"}},
				"3": {{TaskName: "3"}, {TaskName: "3"}, {TaskName: "3"}},
			},
		},
		{
			"read task - happy path",
			"4",
			[]event.Event{
				{TaskName: "4"},
				{TaskName: "4"},
				{TaskName: "5"},
				{TaskName: "5"},
				{TaskName: "4"},
				{TaskName: "4"},
				{TaskName: "5"},
			},
			map[string][]event.Event{
				"4": {{TaskName: "4"}, {TaskName: "4"}, {TaskName: "4"}, {TaskName: "4"}},
			},
		},
		{
			"read task - no event",
			"4",
			[]event.Event{},
			map[string][]event.Event{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			storage := newEventStorage()
			for _, event := range tc.values {
				err := storage.Add(event)
				require.NoError(t, err)
			}

			actual := storage.Read(tc.input)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func Test_eventStorage_Delete(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		values   []event.Event
		expected map[string][]event.Event
	}{
		{
			"delete - happy path",
			"2",
			[]event.Event{
				{TaskName: "1"},
				{TaskName: "2"},
				{TaskName: "2"},
			},
			map[string][]event.Event{
				"1": {{TaskName: "1"}},
			},
		},
		{
			"delete task - no event",
			"4",
			[]event.Event{},
			map[string][]event.Event{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			storage := newEventStorage()
			for _, event := range tc.values {
				err := storage.Add(event)
				require.NoError(t, err)
			}
			storage.Delete(tc.input)

			after := storage.Read("")
			assert.Equal(t, tc.expected, after)
		})
	}
}
