package controller

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/hashicorp/consul-nia/config"
	"github.com/hashicorp/consul-nia/driver"
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
		name          string
		expectError   bool
		initErr       error
		initTaskErr   error
		initWorkerErr error
		fileReader    func(string) ([]byte, error)
		config        *config.Config
	}{
		{
			"error on driver.Init()",
			true,
			errors.New("error on driver.Init()"),
			nil,
			nil,
			func(string) ([]byte, error) { return []byte{}, nil },
			conf,
		},
		{
			"error on driver.InitTask()",
			true,
			nil,
			errors.New("error on driver.InitTask()"),
			nil,
			func(string) ([]byte, error) { return []byte{}, nil },
			conf,
		},
		{
			"error on driver.InitWorker()",
			true,
			nil,
			nil,
			errors.New("error on driver.InitWorker()"),
			func(string) ([]byte, error) { return []byte{}, nil },
			conf,
		},
		{
			"error on newTaskTemplates()",
			true,
			nil,
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
			d.On("InitWorker", mock.Anything).Return(tc.initWorkerErr).Once()

			controller := ReadWrite{
				newDriver:  func(*config.Config) driver.Driver { return d },
				conf:       tc.config,
				fileReader: tc.fileReader,
			}

			err := controller.Init(ctx)

			if tc.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.name)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestReadWriteRun(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name              string
		expectError       bool
		initWorkErr       error
		applyWorkErr      error
		resolverRunErr    error
		templateRenderErr error
		watcherWaitErr    error
		config            *config.Config
	}{
		{
			"error on resolver.Run()",
			true,
			nil,
			nil,
			errors.New("error on resolver.Run()"),
			nil,
			nil,
			singleTaskConfig(),
		},
		{
			"error on driver.InitWork()",
			true,
			errors.New("error on driver.InitWork()"),
			nil,
			nil,
			nil,
			nil,
			singleTaskConfig(),
		},
		{
			"error on driver.ApplyWork()",
			true,
			nil,
			errors.New("error on driver.ApplyWork()"),
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
			nil,
			singleTaskConfig(),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {

			tmpl := new(mocks.Template)
			tmpl.On("Render", mock.Anything).Return(hcat.RenderResult{}, tc.templateRenderErr).Once()

			r := new(mocks.Resolver)
			r.On("Run", mock.Anything, mock.Anything).Return(hcat.ResolveEvent{Complete: true}, tc.resolverRunErr)

			w := new(mocks.Watcher)
			w.On("Wait", mock.Anything).Return(tc.watcherWaitErr)

			d := new(mocksD.Driver)
			d.On("InitWork", mock.Anything).Return(tc.initWorkErr)
			d.On("ApplyWork", mock.Anything).Return(tc.applyWorkErr)

			u := unit{template: tmpl, driver: d}

			controller := ReadWrite{
				conf:       tc.config,
				fileReader: func(string) ([]byte, error) { return []byte{}, nil },
				units:      []unit{u},
				watcher:    w,
				resolver:   r,
			}

			ctx := context.Background()
			err := controller.run(ctx)

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

func TestReadWriteLoop(t *testing.T) {
	t.Run("loop-wait-exit", func(t *testing.T) {
		w := new(mocks.Watcher)
		w.On("Wait", mock.Anything).Return(errors.New("end-wait"))

		ctl := ReadWrite{
			units:   []unit{},
			watcher: w,
		}
		ctx := context.Background()
		errCh := make(chan error, 1) // buffer lets loop run in foreground
		ctl.loop(ctx, errCh)
		err := <-errCh
		if err != nil && err.Error() != "end-wait" {
			t.Error("wanted 'end-wait', got:", err)
		}
	})
	t.Run("loop-context-cancel", func(t *testing.T) {
		w := new(mocks.Watcher)
		wg := sync.WaitGroup{}
		wg.Add(3)
		count := 0
		w.On("Wait", mock.Anything).Run(
			func(mock.Arguments) {
				if count > 2 {
					wg.Wait()
				} else {
					count++
					wg.Done()
				}
			}).Return(nil)

		ctl := ReadWrite{
			units:   []unit{},
			watcher: w,
		}
		ctx, cancel := context.WithCancel(context.Background())
		errCh := make(chan error)
		go ctl.loop(ctx, errCh)
		wg.Wait()
		cancel()
		err := <-errCh
		if err != nil && err.Error() != "context canceled" {
			t.Error("wanted 'context canceled', got:", err)
		}
	})
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
				SkipVerify: config.Bool(true),
			},
		},
		Tasks: &config.TaskConfigs{
			{
				Description: config.String("automate services for X to do Y"),
				Name:        config.String("task"),
				Services:    []string{"serviceA", "serviceB", "serviceC"},
				Providers:   []string{"X"},
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
		}},
	}

	c.Finalize()
	return c
}
