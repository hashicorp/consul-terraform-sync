package driver

import (
	"context"
	"errors"
	"testing"

	mocks "github.com/hashicorp/consul-nia/mocks/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestWorkerInit(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		initErr error
	}{
		{
			"happy path",
			nil,
		},
		{
			"error",
			errors.New("error"),
		},
	}

	for _, tc := range cases {
		ctx := context.Background()
		t.Run(tc.name, func(t *testing.T) {
			c := new(mocks.Client)
			c.On("Init", mock.Anything).Return(tc.initErr)
			w := worker{client: c}

			err := w.init(ctx)
			if tc.initErr != nil {
				assert.Error(t, err)
				assert.False(t, w.inited)
				return
			}
			assert.NoError(t, err)
			assert.True(t, w.inited)
		})
	}
}

func TestWorkerApply(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		applyErr error
	}{
		{
			"happy path",
			nil,
		},
		{
			"error",
			errors.New("error"),
		},
	}

	for _, tc := range cases {
		ctx := context.Background()
		t.Run(tc.name, func(t *testing.T) {
			c := new(mocks.Client)
			c.On("Apply", mock.Anything).Return(tc.applyErr)
			w := worker{client: c}

			err := w.apply(ctx)
			if tc.applyErr != nil {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}
