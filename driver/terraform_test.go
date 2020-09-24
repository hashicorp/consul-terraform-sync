package driver

import (
	"context"
	"errors"
	"math/rand"
	"testing"

	mocks "github.com/hashicorp/consul-nia/mocks/client"
	"github.com/stretchr/testify/assert"
)

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
					task:   Task{Name: "ApplyTaskTest"},
					inited: tc.inited,
					random: rand.New(rand.NewSource(1)),
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
