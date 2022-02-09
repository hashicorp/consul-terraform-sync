package event

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func ExampleEvent() {
	examples := []struct {
		name      string
		taskName  string
		expectErr bool
	}{
		{
			"Event captures task erroring",
			"task_fail",
			true,
		},
		{
			"Event captures task succeeding",
			"task_success",
			false,
		},
	}

	for _, ex := range examples {
		fmt.Println("\nExample:", ex.name)

		func() (string, error) {
			// setup capturing event
			event, err := NewEvent(ex.taskName, nil)
			if err != nil {
				fmt.Println(err)
				return "", err
			}

			// capture end result before returning function
			defer func() {
				event.End(err)
				fmt.Println("Task Name:", event.TaskName)
				fmt.Println("Success:", event.Success)
				fmt.Println("Error:", event.EventError)
			}()

			// function body
			ret, err := businessLogic(ex.expectErr)
			if err != nil {
				return "", err
			}
			return ret, nil
		}()
	}

	// Output:
	//
	// Example: Event captures task erroring
	// Task Name: task_fail
	// Success: false
	// Error: &{error}
	//
	// Example: Event captures task succeeding
	// Task Name: task_success
	// Success: true
	// Error: <nil>
}

func TestNewEvent(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		taskName  string
		config    *Config
		expectErr bool
	}{
		{
			"happy path",
			"task_a",
			&Config{
				Providers: []string{"local"},
				Services:  []string{"web", "api"},
				Source:    "/my-module",
			},
			false,
		},
		{
			"nil config",
			"task_b",
			nil,
			false,
		},
		{
			"error: no taskname",
			"",
			nil,
			true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			event, err := NewEvent(tc.taskName, tc.config)
			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, event)
				assert.Equal(t, tc.taskName, event.TaskName)
				assertEqualConfig(t, tc.config, event.Config)
			}
		})
	}
}

func TestEvent_Start(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
	}{
		{
			"happy path",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			event := &Event{}
			event.Start()
			assert.False(t, event.StartTime.IsZero())

			// test that calling Start() again does not reset start time
			firstStartTime := event.StartTime
			event.Start()
			assert.Equal(t, firstStartTime, event.StartTime)
		})
	}
}

func TestEvent_End(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		err  error
	}{
		{
			"task succeeded",
			nil,
		},
		{
			"task failed",
			errors.New("error"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			event := &Event{}
			event.End(tc.err)

			assert.False(t, event.EndTime.IsZero())
			if tc.err == nil {
				assert.True(t, event.Success)
				assert.Nil(t, event.EventError)
			} else {
				assert.False(t, event.Success)
				assert.NotNil(t, event.EventError)
				assert.Equal(t, tc.err.Error(), event.EventError.Message)
			}

			// test that calling End() again does not reset end time
			firstEndTime := event.EndTime
			event.End(tc.err)
			assert.Equal(t, firstEndTime, event.EndTime)
		})
	}
}

func businessLogic(expectError bool) (string, error) {
	if expectError {
		return "", errors.New("error")
	}
	return "mock", nil
}

func TestEvent_GoString(t *testing.T) {
	cases := []struct {
		name     string
		event    *Event
		expected string
	}{
		{
			"nil event",
			nil,
			"(*Event)(nil)",
		},
		{
			"happy path",
			&Event{
				ID:       "123",
				TaskName: "happy",
				Success:  false,
				EventError: &Error{
					Message: "error!",
				},
				Config: &Config{
					Providers: []string{"local"},
					Services:  []string{"web", "api"},
					Source:    "/my-module",
				},
			},
			"&Event{ID:123, TaskName:happy, Success:false, " +
				"StartTime:0001-01-01 00:00:00 +0000 UTC, " +
				"EndTime:0001-01-01 00:00:00 +0000 UTC, EventError:&{error!}, " +
				"Config:&Config{Providers:[local], Services:[web api], Source:/my-module}}",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual := tc.event.GoString()
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func assertEqualConfig(t *testing.T, exp, act *Config) {
	if exp == nil {
		assert.Nil(t, act)
		return
	}
	if act == nil {
		assert.Nil(t, exp)
		return
	}
	assert.Equal(t, exp.Providers, act.Providers)
	assert.Equal(t, exp.Services, act.Services)
	assert.Equal(t, exp.Source, act.Source)
}
