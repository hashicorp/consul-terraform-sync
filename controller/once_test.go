// Copyright IBM Corp. 2020, 2025
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

func Test_Once_Run_Terraform(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		numTasks int
	}{
		{
			"consecutive one task",
			1,
		},
		{
			"consecutive multiple tasks",
			10,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			driverConf := &config.DriverConfig{
				Terraform: &config.TerraformConfig{},
			}

			setupNewDriver := func(task *driver.Task) driver.Driver {
				return onceMockDriver(task, nil)
			}

			mockDrivers, err := testOnce(t, tc.numTasks, driverConf, false, setupNewDriver)
			require.NoError(t, err)
			assert.Len(t, mockDrivers, tc.numTasks)

			for _, mockD := range mockDrivers {
				mockD.AssertExpectations(t)
			}
		})
	}
}

func Test_Once_Run_Terraform_errors(t *testing.T) {
	// Checks test cases where an error occurs. However, Run() itself may
	// not return an error i.e. when we log the error and move on

	t.Parallel()

	expectedErr := errors.New("test error")

	testCases := []struct {
		name      string
		allowFail bool
	}{
		{
			"consecutive allow-fail",
			true,
		},
		{
			"consecutive fail-fast",
			false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			driverConf := &config.DriverConfig{
				Terraform: &config.TerraformConfig{},
			}

			setupNewDriver := func(task *driver.Task) driver.Driver {
				if task.Name() == "task_03" {
					// Mock an error during apply for a task
					return onceMockDriver(task, expectedErr)
				}
				return onceMockDriver(task, nil)
			}

			mockDrivers, err := testOnce(t, 5, driverConf, tc.allowFail, setupNewDriver)

			if tc.allowFail {
				require.NoError(t, err)

				// all drivers should have been created even if 03 errored
				assert.Len(t, mockDrivers, 5)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), expectedErr.Error(),
					"unexpected error in Once")

				// task 00, 01, 02 should have been created before 03 errored
				assert.Len(t, mockDrivers, 3)
			}

			for _, mockD := range mockDrivers {
				mockD.AssertExpectations(t)
			}
		})
	}
}

func Test_Once_Run_WatchDep_errors_Terraform(t *testing.T) {
	// Mock the situation where WatchDep WaitCh errors which can cause
	// driver.RenderTemplate() to always returns false. Confirm on WatchDep
	// error, that creating/running tasks does not hang and exits.
	t.Parallel()

	driverConf := &config.DriverConfig{
		Terraform: &config.TerraformConfig{},
	}

	testOnceWatchDepErrors(t, driverConf)
}

func Test_Once_Run_No_Tasks(t *testing.T) {
	t.Parallel()

	// Create a configuration, do not include any tasks
	conf := &config.Config{}
	ss := state.NewInMemoryStore(conf)

	ctrl := Once{
		logger: logging.NewNullLogger(),
		state:  ss,
	}

	err := ctrl.Run(context.Background())
	require.NoError(t, err)
}

func Test_Once_Stop(t *testing.T) {
	t.Parallel()

	// Craee moock watcher
	w := new(mocksTmpl.Watcher)
	w.On("Stop").Return(mock.Anything).Once()

	ctrl := Once{
		logger:  logging.NewNullLogger(),
		watcher: w,
	}

	// Assert that the watcher is called once and only once
	ctrl.Stop()
	mock.AssertExpectationsForObjects(t, w)
}

func Test_Once_onceConsecutive_context_canceled(t *testing.T) {
	// - Controller will try to create and run 5 tasks
	// - Mock a task to take 2 seconds to create and run
	// - Cancel context after 1 second. Confirm only 1 task created and run
	t.Parallel()

	conf := multipleTaskConfig(t, 5)
	ss := state.NewInMemoryStore(conf)

	ctrl := Once{
		logger: logging.NewNullLogger(),
		state:  ss,
	}

	// Set up tasks manager
	tm := newTestTasksManager()
	tm.state = ss
	ctrl.tasksManager = tm

	// Set up driver factory
	tm.factory.initConf = conf
	tm.factory.newDriver = func(ctx context.Context, c *config.Config, task *driver.Task, w templates.Watcher) (driver.Driver, error) {
		d := new(mocksD.Driver)
		d.On("Task").Return(task).Times(3)
		d.On("TemplateIDs").Return(nil)
		d.On("RenderTemplate", mock.Anything).Return(true, nil).Once()
		d.On("InitTask", mock.Anything, mock.Anything).Return(nil).Once()
		d.On("ApplyTask", mock.Anything).Return(nil).Once()
		// Last driver call takes 2 seconds
		d.On("SetBufferPeriod").Return().After(2 * time.Second).Once()
		return d, nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error)
	go func() {
		err := ctrl.onceConsecutive(ctx)
		if err != nil {
			errCh <- err
		}
	}()
	// Cancel context after 1 second (during first task)
	time.Sleep(time.Second)
	cancel()

	select {
	case err := <-errCh:
		assert.Equal(t, err, context.Canceled)
	case <-time.After(time.Second * 5):
		t.Fatal("onceConsecutive did not exit properly from cancelling context")
	}

	assert.Equal(t, 1, tm.drivers.Len(), "only one driver should have been created")
	for _, d := range tm.drivers.Map() {
		d.(*mocksD.Driver).AssertExpectations(t)
	}
}

// testOnce test running once-mode. Returns the mocked drivers for the caller
// to assert expectations
func testOnce(t *testing.T, numTasks int, driverConf *config.DriverConfig, allowFail bool,
	setupNewDriver func(*driver.Task) driver.Driver) ([]*mocksD.Driver, error) {

	conf := multipleTaskConfig(t, numTasks)
	conf.Driver = driverConf
	ss := state.NewInMemoryStore(conf)

	ctrl := Once{
		logger:    logging.NewNullLogger(),
		state:     ss,
		allowFail: allowFail,
	}

	// Set up tasks manager
	tm := newTestTasksManager()
	tm.state = ss
	ctrl.tasksManager = tm

	// Set up driver factory
	tm.factory.initConf = conf
	tm.factory.newDriver = func(ctx context.Context, c *config.Config, task *driver.Task, w templates.Watcher) (driver.Driver, error) {
		return setupNewDriver(task), nil
	}

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

	err := ctrl.Run(context.Background())

	w.AssertExpectations(t)

	mockDrivers := make([]*mocksD.Driver, 0, tm.drivers.Len())
	for _, d := range tm.drivers.Map() {
		mockDriver := d.(*mocksD.Driver)
		mockDrivers = append(mockDrivers, mockDriver)
	}

	return mockDrivers, err
}

func testOnceWatchDepErrors(t *testing.T, driverConf *config.DriverConfig) {
	conf := singleTaskConfig(t)

	if driverConf != nil {
		conf.Driver = driverConf
	}

	ss := state.NewInMemoryStore(conf)

	ctrl := Once{
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
		t.Fatal("Once did not exit properly after WatcherDep errored")
	}

	w.AssertExpectations(t)
	for _, d := range tm.drivers.Map() {
		d.(*mocksD.Driver).AssertExpectations(t)
	}
}

// onceMockDriver mocks the driver with the methods needed for once-mode
func onceMockDriver(task *driver.Task, applyTaskErr error) driver.Driver {
	d := new(mocksD.Driver)
	d.On("Task").Return(task).Times(3)
	d.On("TemplateIDs").Return(nil)
	d.On("RenderTemplate", mock.Anything).Return(false, nil).Once()
	d.On("RenderTemplate", mock.Anything).Return(true, nil).Once()
	d.On("InitTask", mock.Anything, mock.Anything).Return(nil).Once()
	d.On("ApplyTask", mock.Anything).Return(applyTaskErr).Once()
	d.On("SetBufferPeriod").Return().Once()
	return d
}
