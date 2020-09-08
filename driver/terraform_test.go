package driver

import (
	"context"
	"errors"
	"testing"

	mocks "github.com/hashicorp/consul-nia/mocks/client"
	"github.com/stretchr/testify/assert"
)

func TestInitWorker(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		clientType  string
		expectError bool
		tasks       []Task
	}{
		{
			"happy path with development client",
			developmentClient,
			false,
			[]Task{
				Task{Name: "first"},
			},
		},
		{
			"happy path with mock client",
			testClient,
			false,
			[]Task{
				Task{Name: "first"},
				Task{Name: "second"},
				Task{Name: "third"},
			},
		},
		{
			"error when creating Terraform CLI client",
			"",
			true,
			[]Task{
				Task{Name: "task"},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tf := &Terraform{
				workingDir: "test/working/dir",
				path:       "exec/path",
				clientType: tc.clientType,
			}

			for _, task := range tc.tasks {
				err := tf.InitWorker(task)
				if tc.expectError {
					assert.Error(t, err)
					return
				}
				assert.NoError(t, err)
			}
			assert.Equal(t, len(tc.tasks), len(tf.workers))
		})
	}
}

func TestInitWork(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		expectError bool
		errs        []error
	}{
		{
			"single worker, no error",
			false,
			[]error{
				nil,
			},
		},
		{
			"multiple workers, no errors",
			false,
			[]error{
				nil,
				nil,
				nil,
				nil,
			},
		},
		{
			"single worker, with error",
			true,
			[]error{
				errors.New("first task error"),
			},
		},
		{
			"multiple workers, mixed error",
			true,
			[]error{
				errors.New("first task error"),
				nil,
				errors.New("third task error"),
				errors.New("fourth task error"),
				nil,
				nil,
				errors.New("seventh task error"),
				errors.New("eighth task error"),
				errors.New("nineth task error"),
				nil,
				nil,
			},
		},
	}
	ctx := context.Background()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// create workers for the driver
			workers := make([]*worker, len(tc.errs))
			for ix, err := range tc.errs {
				c := new(mocks.Client)
				c.On("Init", ctx).Return(err)
				workers[ix] = &worker{
					client: c,
					task:   Task{},
				}
			}

			tf := &Terraform{
				workers: workers,
			}
			err := tf.InitWork(ctx)
			if !tc.expectError {
				assert.NoError(t, err)
				return
			}

			assert.Error(t, err)
			// confirm all the error strings are within error
			for _, e := range tc.errs {
				if e == nil {
					continue
				}
				assert.Contains(t, err.Error(), e.Error())
			}
		})
	}
}

func TestApplyWork(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		expectError bool
		errs        []error
	}{
		{
			"single worker, no error",
			false,
			[]error{
				nil,
			},
		},
		{
			"multiple workers, no errors",
			false,
			[]error{
				nil,
				nil,
				nil,
				nil,
			},
		},
		{
			"single worker, with error",
			true,
			[]error{
				errors.New("first task error"),
			},
		},
		{
			"multiple workers, mixed error",
			true,
			[]error{
				errors.New("first task error"),
				nil,
				errors.New("third task error"),
				errors.New("fourth task error"),
				nil,
				nil,
				errors.New("seventh task error"),
				errors.New("eighth task error"),
				errors.New("nineth task error"),
				nil,
				nil,
			},
		},
	}
	ctx := context.Background()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// create workers for the driver
			workers := make([]*worker, len(tc.errs))
			for ix, err := range tc.errs {
				c := new(mocks.Client)
				c.On("Apply", ctx).Return(err)
				workers[ix] = &worker{
					client: c,
					task:   Task{},
				}
			}

			tf := &Terraform{
				workers: workers,
			}
			err := tf.ApplyWork(ctx)
			if !tc.expectError {
				assert.NoError(t, err)
				return
			}

			assert.Error(t, err)
			// confirm all the error strings are within error
			for _, e := range tc.errs {
				if e == nil {
					continue
				}
				assert.Contains(t, err.Error(), e.Error())
			}
		})
	}
}
