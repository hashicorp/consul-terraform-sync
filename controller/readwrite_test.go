package controller

import (
	"context"
	"testing"
	"time"

	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/driver"
	"github.com/hashicorp/consul-terraform-sync/logging"
	mocks "github.com/hashicorp/consul-terraform-sync/mocks/templates"
	"github.com/hashicorp/consul-terraform-sync/state"
	"github.com/hashicorp/consul-terraform-sync/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func Test_ReadWrite_Run(t *testing.T) {
	w := new(mocks.Watcher)
	w.On("Watch", mock.Anything, mock.Anything).Return(nil).
		On("Size").Return(5).
		On("Stop").Return()

	port := testutils.FreePort(t)

	ctl := ReadWrite{
		tasksManager: &TasksManager{
			driverFactory: &driver.Factory{
				drivers: driver.NewDrivers(),
				watcher: w,
				logger:  logging.NewNullLogger(),
				state: state.NewInMemoryStore(&config.Config{
					Port: config.Int(port),
				}),
			},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error)
	go func() {
		err := ctl.Run(ctx)
		if err != nil {
			errCh <- err
		}
	}()
	cancel()

	select {
	case err := <-errCh:
		assert.Equal(t, err, context.Canceled)
	case <-time.After(time.Second * 15):
		t.Fatal("Run did not exit properly from cancelling context")
	}
}

func Test_ReadWrite_Run_Error(t *testing.T) {
	w := new(mocks.Watcher)
	w.On("Watch", mock.Anything, mock.Anything).Return(nil).
		On("Size").Return(5).
		On("Stop").Return()

	port := testutils.FreePort(t)

	ctl := ReadWrite{
		tasksManager: &TasksManager{
			driverFactory: &driver.Factory{
				drivers: driver.NewDrivers(),
				watcher: w,
				logger:  logging.NewNullLogger(),
				state: state.NewInMemoryStore(&config.Config{
					Port: config.Int(port),
				}),
			},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error)
	go func() {
		err := ctl.Run(ctx)
		if err != nil {
			errCh <- err
		}
	}()
	go func() {
		err := ctl.Run(ctx)
		if err != nil {
			errCh <- err
		}
	}()
	defer cancel()

	select {
	case err := <-errCh:
		assert.Contains(t, err.Error(), "address already in use")
	case <-time.After(time.Second * 5):
		t.Fatal("Run did not error and exit properly")
	}
}
