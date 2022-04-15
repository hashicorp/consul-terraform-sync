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
	t.Parallel()

	w := new(mocks.Watcher)
	w.On("Watch", mock.Anything, mock.Anything).Return(nil)

	port := testutils.FreePort(t)

	ctl := ReadWrite{
		tasksManager: &TasksManager{
			baseController: &baseController{
				drivers: driver.NewDrivers(),
				watcher: w,
				logger:  logging.NewNullLogger(),
				state: state.NewInMemoryStore(&config.Config{
					Port: config.Int(port),
				}),
			},
		},
	}

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
