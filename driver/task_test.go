package driver

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"testing"
	"time"

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
			w := worker{
				client: c,
				random: rand.New(rand.NewSource(1)),
			}

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
			w := worker{
				client: c,
				random: rand.New(rand.NewSource(1)),
			}

			err := w.apply(ctx)
			if tc.applyErr != nil {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestWithRetry(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		retry     int
		successOn int
		expected  error
	}{
		{
			"happy path: retry once, success on retry (2)",
			1,
			2,
			nil,
		},
		{
			"no success on retries: retry once",
			1,
			100,
			errors.New("attempt #2 failed 'error on 2': attempt #1 failed 'error on 1'"),
		},
		{
			"happy path: no retry",
			0,
			1,
			nil,
		},
		{
			"no retry, no success",
			0,
			100,
			errors.New("attempt #1 failed 'error on 1'"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// set up fake function
			count := 0
			fxn := func() error {
				count++
				if count == tc.successOn {
					return nil
				}
				return fmt.Errorf("error on %d", count)
			}

			random := rand.New(rand.NewSource(1))
			err := withRetry(fxn, "test fxn", random, tc.retry)
			if tc.expected == nil {
				assert.NoError(t, err)
			} else {
				assert.Equal(t, tc.expected.Error(), err.Error())
			}
		})
	}
}

func TestWaitTime(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		attempt   int
		minReturn float64
		maxReturn float64
	}{
		{
			"first attempt",
			0,
			0,
			0.5,
		},
		{
			"second attempt",
			1,
			1,
			2.5,
		},
		{
			"third attempt",
			2,
			4,
			6.5,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			random := rand.New(rand.NewSource(1))
			a := waitTime(tc.attempt, random)

			actual := float64(a) / float64(time.Second)
			assert.GreaterOrEqual(t, actual, tc.minReturn)
			assert.LessOrEqual(t, actual, tc.maxReturn)
		})
	}
}
