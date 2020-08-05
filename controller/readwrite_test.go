package controller

import (
	"context"
	"errors"
	"testing"

	"github.com/hashicorp/consul-nia/config"
	"github.com/hashicorp/consul-nia/driver"
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
		config      *config.Config
	}{
		{
			"error on driver.Init()",
			true,
			&driver.MockDriver{
				InitFunc: func() error { return errors.New("error on driver.Init()") },
			},
			conf,
		},
		{
			"error on driver.InitTask()",
			true,
			&driver.MockDriver{
				InitFunc:     func() error { return nil },
				InitTaskFunc: func(driver.Task, bool) error { return errors.New("error on driver.InitTask()") },
			},
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
			conf,
		},
		// {
		// 	"happy path",
		// 	false,
		// 	&driver.MockDriver{
		// 		InitFunc:       func() error { return nil },
		// 		InitTaskFunc:   func(driver.Task, bool) error { return nil },
		// 		InitWorkerFunc: func(driver.Task) error { return nil },
		// 	},
		// 	conf,
		// },
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			controller := ReadWrite{
				driver: tc.mockDriver,
				conf:   tc.config,
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
		name        string
		expectError bool
		mockDriver  *driver.MockDriver
		config      *config.Config
	}{
		// {
		// 	"happy path",
		// 	false,
		// 	&driver.MockDriver{
		// 		InitFunc:       func() error { return nil },
		// 		InitTaskFunc:   func(driver.Task, bool) error { return nil },
		// 		InitWorkerFunc: func(driver.Task) error { return nil },
		// 		InitWorkFunc:   func() error { return nil },
		// 		ApplyWorkFunc:  func() error { return nil },
		// 	},
		// 	singleTaskConfig(),
		// },
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			controller := ReadWrite{
				driver: tc.mockDriver,
				conf:   tc.config,
			}

			err := controller.Init()
			assert.NoError(t, err)

			ctx := context.Background()
			err = controller.Run(ctx)

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
