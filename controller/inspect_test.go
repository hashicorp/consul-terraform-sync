// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package controller

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/driver"
	"github.com/hashicorp/consul-terraform-sync/logging"
	mocksD "github.com/hashicorp/consul-terraform-sync/mocks/driver"
	mocksTmpl "github.com/hashicorp/consul-terraform-sync/mocks/templates"
	"github.com/hashicorp/consul-terraform-sync/state"
	"github.com/hashicorp/consul-terraform-sync/templates"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func Test_Inspect_Run(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("test error")

	testCases := []struct {
		name           string
		numTasks       int
		setupNewDriver func(*driver.Task) driver.Driver
		expectErr      bool
	}{
		{
			"one task",
			1,
			func(task *driver.Task) driver.Driver {
				return inspectMockDriver(nil)
			},
			false,
		},
		{
			"multiple tasks",
			10,
			func(task *driver.Task) driver.Driver {
				return inspectMockDriver(nil)
			},
			false,
		},
		{
			"error",
			5,
			func(task *driver.Task) driver.Driver {
				if task.Name() == "task_03" {
					// Mock an error during apply for a task
					return inspectMockDriver(expectedErr)
				}
				return inspectMockDriver(nil)
			},
			true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := testInspect(t, tc.numTasks, tc.setupNewDriver)
			if tc.expectErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), expectedErr.Error(),
					"unexpected error in Once")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func Test_Inspect_Run_context_cancel(t *testing.T) {
	// - Controller will try to create and inspect 5 tasks
	// - Mock a task to take 2 seconds to inspect
	// - Cancel context after 1 second. Confirm only 1 task inspected

	t.Parallel()

	conf := multipleTaskConfig(t, 5)
	ss := state.NewInMemoryStore(conf)

	ctrl := Inspect{
		logger: logging.NewNullLogger(),
		state:  ss,
	}

	// Set up tasks manager
	tm := newTestTasksManager()
	tm.state = ss
	ctrl.tasksManager = tm

	// Set up condition monitor
	cm := newTestConditionMonitor(tm)
	ctrl.monitor = cm

	// Mock watcher
	waitErrCh := make(chan error)
	var waitErrChRc <-chan error = waitErrCh
	// Submit nil to the channel in a different goroutine to prevent deadlock.
	go func() { waitErrCh <- nil }()
	w := new(mocksTmpl.Watcher)
	w.On("WaitCh", mock.Anything).Return(waitErrChRc)
	w.On("Size").Maybe().Return(5)
	cm.watcher = w

	// Set up driver factory
	tm.factory.initConf = conf
	drivers := make(map[string]driver.Driver)
	tm.factory.newDriver = func(ctx context.Context, c *config.Config, task *driver.Task, w templates.Watcher) (driver.Driver, error) {
		d := new(mocksD.Driver)
		d.On("RenderTemplate", mock.Anything).Return(true, nil)
		d.On("InitTask", mock.Anything, mock.Anything).Return(nil).Once()
		d.On("InspectTask", mock.Anything).Return(driver.InspectPlan{}, nil)
		drivers[task.Name()] = d

		return d, nil
	}

	errCh := make(chan error)
	go func() {
		// Cancel context before first task execution.
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		err := ctrl.Run(ctx)
		assert.Error(t, err)
		if err != nil {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		assert.Equal(t, err, context.Canceled)
	case <-time.After(time.Second * 5):
		t.Fatal("Run did not exit properly from cancelling context")
	}

	w.AssertExpectations(t)
	for _, d := range drivers {
		d.(*mocksD.Driver).AssertExpectations(t)
	}
}

func Test_Inspect_Run_WatchDep_errors(t *testing.T) {
	// Mock the situation where WatchDep WaitCh errors which can cause
	// driver.RenderTemplate() to always returns false. Confirm on WatchDep
	// error, that creating/running tasks does not hang and exits.
	t.Parallel()

	conf := singleTaskConfig(t)

	ss := state.NewInMemoryStore(conf)

	ctrl := Inspect{
		logger: logging.NewNullLogger(),
		state:  ss,
	}

	// Set up tasks manager
	tm := newTestTasksManager()
	tm.state = ss
	ctrl.tasksManager = tm

	// Set up condition monitor
	cm := newTestConditionMonitor(tm)
	ctrl.monitor = cm

	// Mock watcher
	expectedErr := errors.New("error!")
	waitErrCh := make(chan error)
	var waitErrChRc <-chan error = waitErrCh
	go func() { waitErrCh <- expectedErr }()
	w := new(mocksTmpl.Watcher)
	w.On("WaitCh", mock.Anything).Return(waitErrChRc)
	cm.watcher = w

	// Set up driver factory
	tm.factory.initConf = conf
	tm.factory.newDriver = func(ctx context.Context, c *config.Config, task *driver.Task, w templates.Watcher) (driver.Driver, error) {
		d := new(mocksD.Driver)
		d.On("InitTask", mock.Anything, mock.Anything).Return(nil)
		// Always return false on render template to mock what happens when
		// WaitCh returns an error
		d.On("RenderTemplate", mock.Anything).Return(false, nil)

		return d, nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	errCh := make(chan error)
	go func() {
		err := ctrl.Run(ctx)
		if err != nil {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		require.Error(t, err)
		assert.Equal(t, err, expectedErr)
	case <-time.After(time.Second * 5):
		t.Fatal("Run did not exit properly after WatcherDep errored")
	}

	w.AssertExpectations(t)
	for _, d := range tm.drivers.Map() {
		d.(*mocksD.Driver).AssertExpectations(t)
	}
}

func testInspect(t *testing.T, numTasks int, setupNewDriver func(*driver.Task) driver.Driver) error {

	conf := multipleTaskConfig(t, numTasks)
	ss := state.NewInMemoryStore(conf)

	ctrl := Inspect{
		logger: logging.NewNullLogger(),
		state:  ss,
	}

	// Set up tasks manager
	tm := newTestTasksManager()
	tm.state = ss
	ctrl.tasksManager = tm

	// Set up condition monitor
	cm := newTestConditionMonitor(tm)
	ctrl.monitor = cm

	// Mock watcher
	errCh := make(chan error)
	var errChRc <-chan error = errCh
	go func() { errCh <- nil }()
	w := new(mocksTmpl.Watcher)
	w.On("WaitCh", mock.Anything).Return(errChRc)
	w.On("Size").Return(numTasks)
	cm.watcher = w

	// Set up driver factory
	tm.factory.initConf = conf
	drivers := make(map[string]driver.Driver)
	tm.factory.newDriver = func(ctx context.Context, c *config.Config, task *driver.Task, w templates.Watcher) (driver.Driver, error) {
		d := setupNewDriver(task)
		drivers[task.Name()] = d
		return d, nil
	}

	err := ctrl.Run(context.Background())

	// Don't w.AssertExpectations(). Race condition on inspection completion
	// and if watcher.Size() is called
	for _, d := range drivers {
		d.(*mocksD.Driver).AssertExpectations(t)
	}

	return err
}

// inspectMockDriver mocks the driver with the methods needed for inspect-mode
func inspectMockDriver(inspectTaskErr error) driver.Driver {
	d := new(mocksD.Driver)
	d.On("RenderTemplate", mock.Anything).Return(true, nil)
	d.On("InitTask", mock.Anything, mock.Anything).Return(nil).Once()
	d.On("InspectTask", mock.Anything).Return(driver.InspectPlan{}, inspectTaskErr)
	return d
}
