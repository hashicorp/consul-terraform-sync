package controller

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hashicorp/consul-nia/config"
	"github.com/hashicorp/consul-nia/driver"
	"github.com/hashicorp/hcat"
	"github.com/stretchr/testify/assert"
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
		name        string
		expectError bool
		mockDriver  *driver.MockDriver
		fileReader  func(string) ([]byte, error)
		config      *config.Config
	}{
		{
			"error on driver.Init()",
			true,
			&driver.MockDriver{
				InitFunc: func() error { return errors.New("error on driver.Init()") },
			},
			func(string) ([]byte, error) { return []byte{}, nil },
			conf,
		},
		{
			"error on driver.InitTask()",
			true,
			&driver.MockDriver{
				InitFunc:     func() error { return nil },
				InitTaskFunc: func(driver.Task, bool) error { return errors.New("error on driver.InitTask()") },
			},
			func(string) ([]byte, error) { return []byte{}, nil },
			conf,
		},
		{
			"error on driver.InitWorker()",
			true,
			&driver.MockDriver{
				InitFunc:       func() error { return nil },
				InitTaskFunc:   func(driver.Task, bool) error { return nil },
				InitWorkerFunc: func(driver.Task) error { return errors.New("error on driver.InitWorker()") },
			},
			func(string) ([]byte, error) { return []byte{}, nil },
			conf,
		},
		{
			"error on newTaskTemplates()",
			true,
			&driver.MockDriver{
				InitFunc:       func() error { return nil },
				InitTaskFunc:   func(driver.Task, bool) error { return nil },
				InitWorkerFunc: func(driver.Task) error { return nil },
			},
			func(string) ([]byte, error) { return []byte{}, errors.New("error on newTaskTemplates()") },
			conf,
		},
		{
			"happy path",
			false,
			&driver.MockDriver{
				InitFunc:       func() error { return nil },
				InitTaskFunc:   func(driver.Task, bool) error { return nil },
				InitWorkerFunc: func(driver.Task) error { return nil },
			},
			func(string) ([]byte, error) { return []byte{}, nil },
			conf,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			controller := ReadWrite{
				driver:     tc.mockDriver,
				conf:       tc.config,
				fileReader: tc.fileReader,
			}

			err := controller.Init()

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
		name         string
		expectError  bool
		mockDriver   *driver.MockDriver
		mockResolver *mockHcatResolver
		mockTemplate *mockHcatTemplate
		mockWatcher  *mockHcatWatcher
		config       *config.Config
	}{
		{
			"error on resolver.Run()",
			true,
			driver.NewMockDriver(),
			&mockHcatResolver{
				RunFunc: func(hcat.Templater, hcat.Watcherer) (hcat.ResolveEvent, error) {
					return hcat.ResolveEvent{}, errors.New("error on resolver.Run()")
				},
			},
			newMockHcatTemplate(),
			newMockHcatWatcher(),
			singleTaskConfig(),
		},
		{
			"error on watcher.Wait()",
			true,
			driver.NewMockDriver(),
			newMockHcatResolver(),
			&mockHcatTemplate{
				RenderFunc: func([]byte) (hcat.RenderResult, error) {
					return hcat.RenderResult{}, errors.New("error on template.Render()")
				},
			},
			&mockHcatWatcher{
				WaitFunc: func(time.Duration) error { return errors.New("error on watcher.Wait()") },
			},
			singleTaskConfig(),
		},
		{
			"error on driver.InitWork()",
			true,
			&driver.MockDriver{
				InitWorkFunc: func() error { return errors.New("error on driver.InitWork()") },
			},
			newMockHcatResolver(),
			newMockHcatTemplate(),
			newMockHcatWatcher(),
			singleTaskConfig(),
		},
		{
			"error on driver.ApplyWork()",
			true,
			&driver.MockDriver{
				InitWorkFunc:  func() error { return nil },
				ApplyWorkFunc: func() error { return errors.New("error on driver.ApplyWork()") },
			},
			newMockHcatResolver(),
			newMockHcatTemplate(),
			newMockHcatWatcher(),
			singleTaskConfig(),
		},
		{
			"happy path",
			false,
			driver.NewMockDriver(),
			newMockHcatResolver(),
			newMockHcatTemplate(),
			newMockHcatWatcher(),
			singleTaskConfig(),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			templates := make(map[string]hcatTemplate)
			templates["test template"] = tc.mockTemplate

			controller := ReadWrite{
				driver:     tc.mockDriver,
				conf:       tc.config,
				fileReader: func(string) ([]byte, error) { return []byte{}, nil },
				templates:  templates,
				watcher:    tc.mockWatcher,
				resolver:   tc.mockResolver,
			}

			ctx := context.Background()
			err := controller.Run(ctx)

			if tc.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.name)
				return
			}
			assert.NoError(t, err)
		})
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
				LogLevel:   config.String("warn"),
				Path:       config.String("path"),
				DataDir:    config.String("data"),
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
