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

func Test_ReadOnly_Run(t *testing.T) {
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
				return inspectMockDriver(task, nil)
			},
			false,
		},
		{
			"multiple tasks",
			10,
			func(task *driver.Task) driver.Driver {
				return inspectMockDriver(task, nil)
			},
			false,
		},
		{
			"error",
			5,
			func(task *driver.Task) driver.Driver {
				if task.Name() == "task_03" {
					// Mock an error during apply for a task
					return inspectMockDriver(task, expectedErr)
				}
				return inspectMockDriver(task, nil)
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

func testInspect(t *testing.T, numTasks int, setupNewDriver func(*driver.Task) driver.Driver) error {

	conf := multipleTaskConfig(numTasks)
	ss := state.NewInMemoryStore(conf)

	ro := ReadOnly{
		logger: logging.NewNullLogger(),
		state:  ss,
	}

	// Set up tasks manager
	tm := newTestTasksManager()
	tm.state = ss
	ro.tasksManager = &tm

	// Mock watcher
	errCh := make(chan error)
	var errChRc <-chan error = errCh
	go func() { errCh <- nil }()
	w := new(mocksTmpl.Watcher)
	w.On("WaitCh", mock.Anything).Return(errChRc)
	w.On("Size").Return(numTasks)
	tm.watcher = w

	// Set up baseController
	tm.baseController.initConf = conf
	drivers := make(map[string]driver.Driver)
	tm.baseController.newDriver = func(c *config.Config, task *driver.Task, w templates.Watcher) (driver.Driver, error) {
		d := setupNewDriver(task)
		drivers[task.Name()] = d
		return d, nil
	}

	err := ro.Run(context.Background())

	w.AssertExpectations(t)
	for _, d := range drivers {
		d.(*mocksD.Driver).AssertExpectations(t)
	}

	return err
}

// inspectMockDriver mocks the driver with the methods needed for inspect-mode
func inspectMockDriver(task *driver.Task, inspecTaskErr error) driver.Driver {
	d := new(mocksD.Driver)
	d.On("RenderTemplate", mock.Anything).Return(true, nil)
	d.On("InitTask", mock.Anything, mock.Anything).Return(nil).Once()
	d.On("InspectTask", mock.Anything).Return(driver.InspectPlan{}, inspecTaskErr)
	d.On("OverrideNotifier").Return().Once()
	return d
}
