package controller

import (
	"context"
	"testing"
	"time"

	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/driver"
	"github.com/hashicorp/consul-terraform-sync/logging"
	mocksD "github.com/hashicorp/consul-terraform-sync/mocks/driver"
	mocksTmpl "github.com/hashicorp/consul-terraform-sync/mocks/templates"
	"github.com/hashicorp/consul-terraform-sync/state"
	"github.com/hashicorp/consul-terraform-sync/templates"
	"github.com/hashicorp/consul-terraform-sync/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func Test_Daemon_Run_long(t *testing.T) {
	// Only tests long-running mode of Run()
	t.Parallel()

	w := new(mocksTmpl.Watcher)
	w.On("Watch", mock.Anything, mock.Anything).Return(nil)

	port := testutils.FreePort(t)

	ctl := Daemon{once: true}

	tm := newTestTasksManager()
	tm.watcher = w
	tm.state = state.NewInMemoryStore(&config.Config{
		Port: config.Int(port),
	})
	ctl.tasksManager = tm

	t.Run("cancel exits successfully", func(t *testing.T) {
		errCh := make(chan error)
		ctx, cancel := context.WithCancel(context.Background())

		go func() {
			if err := ctl.Run(ctx); err != nil {
				errCh <- err
			}
		}()
		cancel()

		select {
		case err := <-errCh:
			// Confirm that exit is due to context cancel
			assert.Equal(t, err, context.Canceled)
		case <-time.After(time.Second * 15):
			t.Fatal("Run did not exit properly from cancelling context")
		}
	})

	t.Run("error exits successfully", func(t *testing.T) {
		errCh := make(chan error)
		ctx, cancel := context.WithCancel(context.Background())

		go func() {
			if err := ctl.Run(ctx); err != nil {
				errCh <- err
			}
		}()

		// Re-run controller to create an "address already in use" error when
		// trying to serve api at same port
		go func() {
			if err := ctl.Run(ctx); err != nil {
				errCh <- err
			}
		}()
		defer cancel()

		select {
		case err := <-errCh:
			// Confirm error was received and successfully exit
			assert.Contains(t, err.Error(), "address already in use")
		case <-time.After(time.Second * 5):
			t.Fatal("Run did not error and exit properly")
		}
	})
}

func Test_Daemon_Run_once_long_Terraform(t *testing.T) {
	// Tests long-running mode behaves as expected with triggers after once
	// completes
	t.Parallel()

	driverConf := &config.DriverConfig{
		Terraform: &config.TerraformConfig{},
	}

	testOnceThenLong(t, driverConf)
}

func testOnceThenLong(t *testing.T, driverConf *config.DriverConfig) {
	port := testutils.FreePort(t)
	conf := singleTaskConfig()
	conf.Driver = driverConf
	conf.Port = config.Int(port)
	conf.Finalize()

	st := state.NewInMemoryStore(conf)

	rw := Daemon{
		logger: logging.NewNullLogger(),
		state:  st,
	}

	// Setup taskmanager
	tm := newTestTasksManager()
	tm.watcherCh = make(chan string, 5)
	tm.state = st
	rw.tasksManager = tm

	// Mock driver
	tm.factory.newDriver = func(c *config.Config, task *driver.Task, w templates.Watcher) (driver.Driver, error) {
		d := new(mocksD.Driver)
		d.On("Task").Return(task)
		d.On("TemplateIDs").Return([]string{"{{tmpl}}"})
		d.On("RenderTemplate", mock.Anything).Return(true, nil)
		d.On("InitTask", mock.Anything, mock.Anything).Return(nil).Once()
		d.On("ApplyTask", mock.Anything).Return(nil)
		d.On("OverrideNotifier").Return().Once()
		d.On("SetBufferPeriod").Return().Once()
		return d, nil
	}

	// Setup variables for testing
	errCh := make(chan error)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	completedTasksCh := tm.EnableTestMode()

	// Mock watcher
	w := new(mocksTmpl.Watcher)
	waitErrCh := make(chan error)
	var waitErrChRc <-chan error = waitErrCh
	go func() { errCh <- nil }()
	w.On("WaitCh", mock.Anything).Return(waitErrChRc).Once()
	w.On("Size").Return(5)
	w.On("Watch", ctx, tm.watcherCh).Return(nil)
	tm.watcher = w

	go func() {
		err := rw.Run(ctx)
		if err != nil {
			errCh <- err
		}
	}()

	// Emulate triggers to evaluate task completion
	for i := 0; i < 5; i++ {
		tm.watcherCh <- "{{tmpl}}"
		select {
		case taskName := <-completedTasksCh:
			assert.Equal(t, "task", taskName)

		case err := <-errCh:
			require.NoError(t, err)
		case <-ctx.Done():
			assert.NoError(t, ctx.Err(), "Context should not timeout. Once and Run usage of Watcher does not match the expected triggers.")
		}
	}

	// Don't w.AssertExpectations(). Race condition when Watch() is called
	// for rw.Run
	for _, d := range tm.drivers.Map() {
		d.(*mocksD.Driver).AssertExpectations(t)
	}
}
