package controller

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/driver"
	"github.com/hashicorp/consul-terraform-sync/handler"
	mocks "github.com/hashicorp/consul-terraform-sync/mocks/controller"
	mocksD "github.com/hashicorp/consul-terraform-sync/mocks/driver"
	"github.com/hashicorp/hcat"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestReadWriteRun(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name              string
		expectError       bool
		applyTaskErr      error
		resolverRunErr    error
		templateRenderErr error
		watcherWaitErr    error
		config            *config.Config
	}{
		{
			"error on resolver.Run()",
			true,
			nil,
			errors.New("error on resolver.Run()"),
			nil,
			nil,
			singleTaskConfig(),
		},
		{
			"error on driver.ApplyTask()",
			true,
			errors.New("error on driver.ApplyTask()"),
			nil,
			nil,
			nil,
			singleTaskConfig(),
		},
		{
			"happy path",
			false,
			nil,
			nil,
			nil,
			nil,
			singleTaskConfig(),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {

			tmpl := new(mocks.Template)
			tmpl.On("Render", mock.Anything).Return(hcat.RenderResult{}, tc.templateRenderErr).Once()

			r := new(mocks.Resolver)
			r.On("Run", mock.Anything, mock.Anything).
				Return(hcat.ResolveEvent{Complete: true}, tc.resolverRunErr)

			w := new(mocks.Watcher)
			w.On("Wait", mock.Anything).Return(tc.watcherWaitErr)

			d := new(mocksD.Driver)
			d.On("ApplyTask", mock.Anything).Return(tc.applyTaskErr)

			controller := ReadWrite{baseController: &baseController{
				watcher:  w,
				resolver: r,
			}}
			u := unit{template: tmpl, driver: d}
			ctx := context.Background()

			_, err := controller.checkApply(ctx, u)
			if tc.expectError {
				if assert.Error(t, err) {
					assert.Contains(t, err.Error(), tc.name)
				}
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestOnce(t *testing.T) {
	t.Run("init-wraps-units", func(t *testing.T) {
		conf := singleTaskConfig()

		tmpl := new(mocks.Template)
		tmpl.On("Render", mock.Anything).Return(hcat.RenderResult{}, nil).Once()

		r := new(mocks.Resolver)
		var tf bool
		falseThenTrue := func(hcat.Templater, hcat.Watcherer) hcat.ResolveEvent {
			if !tf {
				defer func() { tf = true }()
			}
			return hcat.ResolveEvent{Complete: tf}
		}
		r.On("Run", mock.Anything, mock.Anything).Return(falseThenTrue, nil).Twice()

		w := new(mocks.Watcher)
		errCh := make(chan error)
		var errChRc <-chan error = errCh
		go func() { errCh <- nil }()
		w.On("WaitCh", mock.Anything).Return(errChRc).Once()

		d := new(mocksD.Driver)
		d.On("InitTask", mock.Anything, mock.Anything).Return(nil).Once()
		d.On("ApplyTask", mock.Anything).Return(nil).Once()

		rw := &ReadWrite{baseController: &baseController{
			watcher:  w,
			resolver: r,
			newDriver: func(*config.Config, driver.Task) (driver.Driver, error) {
				return d, nil
			},
			conf:       conf,
			fileReader: func(string) ([]byte, error) { return []byte{}, nil },
		}}

		ctx := context.Background()
		err := rw.Init(ctx)
		assert.NoError(t, err)

		// insert mock template into units
		for i, u := range rw.units {
			u.template = tmpl
			rw.units[i] = u
		}

		// testing really starts here...
		once := Oncer(rw)
		done := make(chan error)
		// running in goroutine so I can timeout
		go func() {
			done <- once.Once(ctx)
		}()
		select {
		case err := <-done:
			if err != nil {
				t.Fatal("Unexpected error in Once():", err)
			}
		case <-time.After(time.Second):
			t.Fatal("Once didn't return in expected time")
		}

		// Not sure about these... to far into the "test implementation" zone?
		tmpl.AssertExpectations(t)
		r.AssertExpectations(t)
		w.AssertExpectations(t)
		d.AssertExpectations(t)
	})
}

func TestReadWriteUnits(t *testing.T) {
	tmpl := new(mocks.Template)
	tmpl.On("Render", mock.Anything).Return(hcat.RenderResult{}, nil)

	r := new(mocks.Resolver)
	r.On("Run", mock.Anything, mock.Anything).
		Return(hcat.ResolveEvent{Complete: true}, nil)

	w := new(mocks.Watcher)
	w.On("Wait", mock.Anything).Return(nil)

	t.Run("simple-success", func(t *testing.T) {
		d := new(mocksD.Driver)
		d.On("InitWork", mock.Anything).Return(nil)
		d.On("ApplyTask", mock.Anything).Return(nil)
		d.On("ApplyTask", mock.Anything).Return(fmt.Errorf("test"))

		u := unit{taskName: "foo", template: tmpl, driver: d}
		controller := ReadWrite{baseController: &baseController{
			watcher:  w,
			resolver: r,
			units:    []unit{u},
		}}

		ctx := context.Background()
		errCh := controller.runUnits(ctx)
		err := <-errCh
		if err != nil {
			t.Error(err)
		}
	})

	t.Run("apply-error", func(t *testing.T) {
		d := new(mocksD.Driver)
		d.On("InitWork", mock.Anything).Return(nil)
		d.On("ApplyTask", mock.Anything).Return(fmt.Errorf("test"))

		u := unit{taskName: "foo", template: tmpl, driver: d}
		controller := ReadWrite{baseController: &baseController{
			watcher:  w,
			resolver: r,
			units:    []unit{u},
		}}

		ctx := context.Background()
		errCh := controller.runUnits(ctx)
		err := <-errCh
		testErr := fmt.Errorf("could not apply: %s", "test")
		if errors.Is(err, testErr) {
			t.Error(
				fmt.Sprintf("bad error, expected '%v', got '%v'", testErr, err))
		}
	})
}

func TestReadWriteRun_context_cancel(t *testing.T) {
	w := new(mocks.Watcher)
	w.On("WaitCh", mock.Anything, mock.Anything).Return(nil).
		On("Stop").Return()

	ctl := ReadWrite{baseController: &baseController{
		units:   []unit{},
		watcher: w,
	}}

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
		if err != context.Canceled {
			t.Error("wanted 'context canceled', got:", err)
		}
	case <-time.After(time.Second * 5):
		t.Fatal("Run did not exit properly from cancelling context")
	}
}

// singleTaskConfig returns a happy path config that has a single task
func singleTaskConfig() *config.Config {
	c := &config.Config{
		Consul: &config.ConsulConfig{
			Address: config.String("consul-example.com"),
		},
		Driver: &config.DriverConfig{
			Terraform: &config.TerraformConfig{
				Log:        config.Bool(true),
				Path:       config.String("path"),
				WorkingDir: config.String("working"),
			},
		},
		Tasks: &config.TaskConfigs{
			{
				Description: config.String("automate services for X to do Y"),
				Name:        config.String("task"),
				Services:    []string{"serviceA", "serviceB", "serviceC"},
				Providers:   []string{"X", handler.TerraformProviderFake},
				Source:      config.String("Y"),
				Version:     config.String("v1"),
			},
		},
		Services: &config.ServiceConfigs{
			{
				ID:          config.String("serviceA_id"),
				Name:        config.String("serviceA"),
				Description: config.String("descriptionA"),
			}, {
				ID:          config.String("serviceB_id"),
				Name:        config.String("serviceB"),
				Namespace:   config.String("teamB"),
				Description: config.String("descriptionB"),
			},
		},
		Providers: &config.ProviderConfigs{{
			"X": map[string]interface{}{},
			handler.TerraformProviderFake: map[string]interface{}{
				"name": "fake-provider",
			},
		}},
	}

	c.Finalize()
	return c
}
