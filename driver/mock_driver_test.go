package driver

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMockDriverInit(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		returnErr error
	}{
		{
			"default, no error",
			nil,
		},
		{
			"error on custom init",
			errors.New("error on init"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := NewMockDriver()

			// setup Init() return values
			if tc.returnErr != nil {
				d.InitFunc = func() error {
					return tc.returnErr
				}
			}

			err := d.Init()
			if tc.returnErr != nil {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestMockDriverInitTask(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name         string
		expectError  bool
		initTaskFunc func(task Task, force bool) error
		task         Task
	}{
		{
			"default, no error",
			false,
			nil,
			Task{},
		},
		{
			"no error on custom init task",
			false,
			func(task Task, force bool) error {
				if task.Name == "invalid task" {
					return fmt.Errorf("error init task named '%s'", task.Name)
				}
				return nil
			},
			Task{Name: "my task"},
		},
		{
			"error on custom init task",
			true,
			func(task Task, force bool) error {
				if task.Name == "invalid task" {
					return fmt.Errorf("error init task named '%s'", task.Name)
				}
				return nil
			},
			Task{Name: "invalid task"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := NewMockDriver()

			// setup InitTask() return values
			if tc.initTaskFunc != nil {
				d.InitTaskFunc = tc.initTaskFunc
			}

			// currently not meaningfully testing 'force', passing in true for now
			err := d.InitTask(tc.task, true)
			if tc.expectError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestMockDriverInitWorker(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name           string
		expectError    bool
		initWorkerFunc func(task Task) error
		task           Task
	}{
		{
			"default, no error",
			false,
			nil,
			Task{},
		},
		{
			"no error on custom init worker",
			false,
			func(task Task) error {
				if task.Name == "invalid task" {
					return fmt.Errorf("error init worker for task named '%s'", task.Name)
				}
				return nil
			},
			Task{Name: "my task"},
		},
		{
			"error on custom init worker",
			true,
			func(task Task) error {
				if task.Name == "invalid task" {
					return fmt.Errorf("error init worker for task named '%s'", task.Name)
				}
				return nil
			},
			Task{Name: "invalid task"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := NewMockDriver()

			// setup InitWorker() return values
			if tc.initWorkerFunc != nil {
				d.InitWorkerFunc = tc.initWorkerFunc
			}

			err := d.InitWorker(tc.task)
			if tc.expectError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestMockDriverInitWork(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		expectError bool
		errs        []error
	}{
		{
			"default, no error",
			false,
			nil,
		},
		{
			"channel input with mixed errors",
			false,
			[]error{
				nil,
				errors.New("2 error"),
				errors.New("3 error"),
				errors.New("4 error"),
				nil,
				nil,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := NewMockDriver()

			// setup InitWork() return values
			if len(tc.errs) > 0 {
				errChan := make(chan error, len(tc.errs))
				for _, err := range tc.errs {
					errChan <- err
				}

				d.InitWorkFunc = func() error {
					return <-errChan
				}
			}

			ctx := context.Background()
			for _, expectedErr := range tc.errs {
				actualErr := d.InitWork(ctx)

				if expectedErr == nil {
					assert.NoError(t, actualErr)
					continue
				}

				assert.Error(t, actualErr)
				assert.Equal(t, expectedErr, actualErr)
			}

			// make sure to check default test where no expected errors
			if len(tc.errs) == 0 {
				actualErr := d.InitWork(ctx)
				assert.NoError(t, actualErr)
			}
		})
	}
}

func TestMockDriverApplyWork(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		expectError bool
		errs        []error
	}{
		{
			"default, no error",
			false,
			[]error{},
		},
		{
			"channel input with mixed errors",
			false,
			[]error{
				nil,
				errors.New("2 error"),
				errors.New("3 error"),
				errors.New("4 error"),
				nil,
				nil,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := NewMockDriver()

			// setup ApplyWork() return values
			if len(tc.errs) > 0 {
				errChan := make(chan error, len(tc.errs))
				for _, err := range tc.errs {
					errChan <- err
				}

				d.ApplyWorkFunc = func() error {
					return <-errChan
				}
			}

			ctx := context.Background()
			for _, expectedErr := range tc.errs {
				actualErr := d.ApplyWork(ctx)

				if expectedErr == nil {
					assert.NoError(t, actualErr)
					continue
				}

				assert.Error(t, actualErr)
				assert.Equal(t, expectedErr, actualErr)
			}

			// make sure to check default test where no expected errors
			if len(tc.errs) == 0 {
				actualErr := d.ApplyWork(ctx)
				assert.NoError(t, actualErr)
			}
		})
	}
}

func TestMockDriverVersion(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		version string
	}{
		{
			"default",
			"",
		},
		{
			"custom version",
			"v0.0.1-custom",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := NewMockDriver()

			// setup Version() return values
			if len(tc.version) > 0 {
				d.VersionFunc = func() string {
					return tc.version
				}
			}

			actualVersion := d.Version()
			if len(tc.version) > 0 {
				assert.Equal(t, tc.version, actualVersion)
				return
			}
			assert.Equal(t, "mock-version", actualVersion)
		})
	}
}
