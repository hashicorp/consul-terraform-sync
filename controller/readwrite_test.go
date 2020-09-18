package controller

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/consul-nia/config"
	"github.com/hashicorp/consul-nia/driver"
	"github.com/hashicorp/consul-nia/handler"
	mocks "github.com/hashicorp/consul-nia/mocks/controller"
	mocksD "github.com/hashicorp/consul-nia/mocks/driver"
	"github.com/hashicorp/hcat"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestNewReadWrite(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		expectError bool
		conf        *config.Config
	}{
		{
			"happy path",
			false,
			singleTaskConfig(),
		},
		{
			"unsupported driver error",
			true,
			&config.Config{
				Driver: &config.DriverConfig{},
			},
		},
		{
			"handler error",
			true,
			&config.Config{
				Driver: &config.DriverConfig{
					Terraform: &config.TerraformConfig{},
				},
				Providers: &config.ProviderConfigs{
					&config.ProviderConfig{
						handler.TerraformProviderFake: "malformed-config",
					},
				},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			controller, err := NewReadWrite(tc.conf)
			if tc.expectError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, controller)
		})
	}
}

func TestReadWriteInit(t *testing.T) {
	t.Parallel()

	conf := singleTaskConfig()

	cases := []struct {
		name        string
		expectError bool
		initErr     error
		initTaskErr error
		fileReader  func(string) ([]byte, error)
		config      *config.Config
	}{
		{
			"error on driver.Init()",
			true,
			errors.New("error on driver.Init()"),
			nil,
			func(string) ([]byte, error) { return []byte{}, nil },
			conf,
		},
		{
			"error on driver.InitTask()",
			true,
			nil,
			errors.New("error on driver.InitTask()"),
			func(string) ([]byte, error) { return []byte{}, nil },
			conf,
		},
		{
			"error on newTaskTemplates()",
			true,
			nil,
			nil,
			func(string) ([]byte, error) { return []byte{}, errors.New("error on newTaskTemplates()") },
			conf,
		},
		{
			"happy path",
			false,
			nil,
			nil,
			func(string) ([]byte, error) { return []byte{}, nil },
			conf,
		},
	}

	ctx := context.Background()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := new(mocksD.Driver)
			d.On("Init", mock.Anything).Return(tc.initErr).Once()
			d.On("InitTask", mock.Anything, mock.Anything).Return(tc.initTaskErr).Once()

			controller := ReadWrite{
				newDriver:  func(*config.Config) driver.Driver { return d },
				conf:       tc.config,
				fileReader: tc.fileReader,
			}

			err := controller.Init(ctx)

			if tc.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.name)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

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

			h, err := getPostApplyHandlers(tc.config)
			assert.NoError(t, err)

			controller := ReadWrite{
				watcher:   w,
				resolver:  r,
				postApply: h,
			}
			u := unit{template: tmpl, driver: d}
			ctx := context.Background()

			err = controller.checkApply(u, ctx)
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
		controller := ReadWrite{
			watcher:  w,
			resolver: r,
			units:    []unit{u},
		}

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
		controller := ReadWrite{
			watcher:  w,
			resolver: r,
			units:    []unit{u},
		}

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

	ctl := ReadWrite{
		units:   []unit{},
		watcher: w,
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
