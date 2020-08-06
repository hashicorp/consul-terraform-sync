package controller

import (
	"errors"
	"testing"
	"time"

	"github.com/hashicorp/hcat"
	"github.com/stretchr/testify/assert"
)

func TestMockHcatTemplateRender(t *testing.T) {
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
			m := newMockHcatTemplate()

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

func TestMockHcatTemplateExecute(t *testing.T) {
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
			m := newMockHcatTemplate()

			_, err := m.Execute(hcat.NewStore())
			assert.NoError(t, err)
		})
	}
}

func TestMockHcatTemplateID(t *testing.T) {
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
			m := newMockHcatTemplate()

			id := m.ID()
			assert.NotEmpty(t, id)
		})
	}
}

func TestMockHcatResolverRun(t *testing.T) {
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
			m := newMockHcatResolver()

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

func TestMockHcatWatcherWait(t *testing.T) {
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
			m := newMockHcatWatcher()

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

func TestMockHcatWatcherAdd(t *testing.T) {
	// Skip writing test for mockHcatWatcher.Add(d hcat.Dependency)

	// Dependency currently lives and is used in hcat internally. In order to
	// write a unit test for Add(), we would have to mock Dependency and other
	// internal elements it consumes.
}

func TestMockHcatWatcherChanged(t *testing.T) {
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
			m := newMockHcatWatcher()

			ok := m.Changed("id")
			assert.True(t, ok)
		})
	}
}

func TestMockHcatWatcherRecall(t *testing.T) {
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
			m := newMockHcatWatcher()

			_, ok := m.Recall("id")
			assert.True(t, ok)
		})
	}
}

func TestMockHcatWatcherRegister(t *testing.T) {
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
			m := newMockHcatWatcher()

			m.Register("id")
			// no return value to assert.
			// calling it to test that it runs without problem.
		})
	}
}
