// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package state

import (
	"fmt"
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
				assert.Len(t, events, 1)
				e := events[0]
				assert.Equal(t, tc.event, e)
			}
		})
	}

	t.Run("limit-and-order", func(t *testing.T) {
		storage := newEventStorage()
		storage.limit = 2

		// fill storage
		err := storage.Add(event.Event{ID: "1", TaskName: "task"})
		require.NoError(t, err)
		assert.Len(t, storage.events["task"], 1)

		err = storage.Add(event.Event{ID: "2", TaskName: "task"})
		require.NoError(t, err)
		assert.Len(t, storage.events["task"], 2)

		// check storage did not grow beyond limit
		err = storage.Add(event.Event{ID: "3", TaskName: "task"})
		require.NoError(t, err)
		assert.Len(t, storage.events["task"], 2)

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
			for _, e := range tc.values {
				err := storage.Add(e)
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
			for _, e := range tc.values {
				err := storage.Add(e)
				require.NoError(t, err)
			}

			storage.Delete(tc.input)
			after := storage.Read("")
			assert.Equal(t, tc.expected, after)
		})
	}
}

func Test_eventStorage_Set(t *testing.T) {

	makeEvents := func(taskName string, numEvents int) []event.Event {
		events := make([]event.Event, numEvents)
		for i := range events {
			events[i].TaskName = taskName
			events[i].ID = fmt.Sprintf("%v-%v", taskName, i)
		}
		return events
	}

	cases := []struct {
		name         string
		inputTask    string
		inputValues  []event.Event
		initialState map[string][]event.Event
		expected     map[string][]event.Event
	}{
		{
			"set with no existing data",
			"1",
			[]event.Event{
				{TaskName: "1"},
				{TaskName: "1"},
				{TaskName: "1"},
			},
			map[string][]event.Event{},
			map[string][]event.Event{
				"1": {{TaskName: "1"}, {TaskName: "1"}, {TaskName: "1"}},
			},
		},
		{
			"set with existing data",
			"1",
			[]event.Event{
				{TaskName: "1", ID: "i1-1"},
				{TaskName: "1", ID: "i1-2"},
			},
			map[string][]event.Event{
				"1": {{TaskName: "1", ID: "i1"}},
				"2": {{TaskName: "2", ID: "i2"}},
			},
			map[string][]event.Event{
				"1": {{TaskName: "1", ID: "i1-1"}, {TaskName: "1", ID: "i1-2"}},
				"2": {{TaskName: "2", ID: "i2"}},
			},
		},
		{
			"set and exceed limit",
			"1",
			makeEvents("1", defaultEventCountLimit+1),
			map[string][]event.Event{},
			map[string][]event.Event{
				"1": makeEvents("1", defaultEventCountLimit),
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			storage := newEventStorage()
			storage.events = tc.initialState
			storage.Set(tc.inputTask, tc.inputValues)
			after := storage.Read("")
			assert.Equal(t, tc.expected, after)
			for _, v := range after {
				assert.LessOrEqual(t, len(v), storage.limit)
			}
		})
	}
}
