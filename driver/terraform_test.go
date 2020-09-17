package driver

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/hashicorp/consul-nia/client"
	mocks "github.com/hashicorp/consul-nia/mocks/client"
	"github.com/stretchr/testify/assert"
)

func TestInitClient(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		clientType  string
		expectError bool
		expect      client.Client
	}{
		{
			"happy path with development client",
			developmentClient,
			false,
			&client.Printer{},
		},
		{
			"happy path with mock client",
			testClient,
			false,
			&mocks.Client{},
		},
		{
			"error when creating Terraform CLI client",
			"",
			true,
			&client.TerraformCLI{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := &Terraform{
				clientType: tc.clientType,
			}

			actual, err := d.initClient(Task{})
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, reflect.TypeOf(tc.expect), reflect.TypeOf(actual))
			}
		})
	}
}

func TestApplyTask(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		expectError bool
		inited      bool
		initReturn  error
		applyReturn error
	}{
		{
			"happy path",
			false,
			false,
			nil,
			nil,
		},
		{
			"already inited",
			false,
			true,
			nil,
			nil,
		},
		{
			"error on init",
			true,
			false,
			errors.New("init error"),
			nil,
		},
		{
			"error on apply",
			true,
			false,
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
					inited: tc.inited,
				},
			}

			err := tf.ApplyTask(ctx)
			if !tc.expectError {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}
