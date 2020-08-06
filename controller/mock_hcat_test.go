package controller

import (
	"errors"
	"testing"
	"time"

	"github.com/hashicorp/hcat"
	"github.com/stretchr/testify/assert"
)

func TestMockTemplateRender(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		returnErr error
	}{
		{
			"default, no error",
			nil,
		},
		{
			"error on custom render",
			errors.New("error on render"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := newMockTemplate()

			// setup Render() return values
			if tc.returnErr != nil {
				m.RenderFunc = func([]byte) (hcat.RenderResult, error) {
					return hcat.RenderResult{}, tc.returnErr
				}
			}

			_, err := m.Render([]byte("content"))
			if tc.returnErr != nil {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestMockTemplateExecute(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
	}{
		{
			"default, no error",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := newMockTemplate()

			_, err := m.Execute(hcat.NewStore())
			assert.NoError(t, err)
		})
	}
}

func TestMockTemplateID(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
	}{
		{
			"default",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := newMockTemplate()

			id := m.ID()
			assert.NotEmpty(t, id)
		})
	}
}

func TestMockResolverRun(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		returnErr error
	}{
		{
			"default, no error",
			nil,
		},
		{
			"error on custom run",
			errors.New("error on run"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := newMockResolver()

			// setup Run() return values
			if tc.returnErr != nil {
				m.RunFunc = func(hcat.Templater, hcat.Watcherer) (hcat.ResolveEvent, error) {
					return hcat.ResolveEvent{}, tc.returnErr
				}
			}

			_, err := m.Run(&hcat.Template{}, &hcat.Watcher{})
			if tc.returnErr != nil {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestMockWatcherWait(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		returnErr error
	}{
		{
			"default, no error",
			nil,
		},
		{
			"error on custom wait",
			errors.New("error on wait"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := newMockWatcher()

			// setup Wait() return values
			if tc.returnErr != nil {
				m.WaitFunc = func(time.Duration) error {
					return tc.returnErr
				}
			}

			err := m.Wait(1 * time.Second)
			if tc.returnErr != nil {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestMockWatcherAdd(t *testing.T) {
	// Skip writing test for mockWatcher.Add(d hcat.Dependency)

	// Dependency currently lives and is used in hashicat internally. In order to
	// write a unit test for Add(), we would have to mock Dependency and other
	// internal elements it consumes.
}

func TestMockWatcherChanged(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
	}{
		{
			"default, no error",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := newMockWatcher()

			ok := m.Changed("id")
			assert.True(t, ok)
		})
	}
}

func TestMockWatcherRecall(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
	}{
		{
			"default, no error",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := newMockWatcher()

			_, ok := m.Recall("id")
			assert.True(t, ok)
		})
	}
}

func TestMockWatcherRegister(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
	}{
		{
			"default, no error",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := newMockWatcher()

			m.Register("id")
			// no return value to assert.
			// calling it to test that it runs without problem.
		})
	}
}
