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
		task        Task
	}{
		{
			"happy path with development client",
			developmentClient,
			false,
			Task{Name: "development-client task"},
		},
		{
			"happy path with mock client",
			testClient,
			false,
			Task{Name: "mock-client task"},
		},
		{
			"error when creating Terraform CLI client",
			"",
			true,
			Task{Name: "task"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tf := &Terraform{
				workingDir: "test/working/dir",
				path:       "exec/path",
				clientType: tc.clientType,
			}

			err := tf.InitWorker(tc.task)
			if tc.expectError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tc.task, tf.worker.task)
		})
	}
}

func TestApplyWork(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		expectError bool
		initReturn  error
		applyReturn error
	}{
		{
			"happy path",
			false,
			nil,
			nil,
		},
		{
			"error on init",
			true,
			errors.New("init error"),
			nil,
		},
		{
			"error on apply",
			true,
			nil,
			errors.New("apply error"),
		},
	}
	ctx := context.Background()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := new(mocks.Client)
			c.On("Init", ctx).Return(tc.initReturn).Once()
			c.On("Apply", ctx).Return(tc.applyReturn).Once()

			tf := &Terraform{
				worker: &worker{
					client: c,
					task:   Task{},
				},
			}

			err := tf.ApplyWork(ctx)
			if !tc.expectError {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}
