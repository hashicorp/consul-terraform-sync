package driver

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"testing"
	"time"

	mocks "github.com/hashicorp/consul-terraform-sync/mocks/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestWithRetry_context_cancel(t *testing.T) {
	t.Parallel()

	r := retry{
		desc:   "fake fxn that never succeeds",
		retry:  5,
		random: rand.New(rand.NewSource(1)),
		fxn: func() error {
			return errors.New("test error")
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error)
	go func() {
		err := r.do(ctx)
		if err != nil {
			errCh <- err
		}
	}()
	cancel()

	select {
	case err := <-errCh:
		if err != context.Canceled {
			t.Error("wanted 'context canceled', got:", err)
		}
	case <-time.After(time.Second * 5):
		t.Fatal("Run did not exit properly from cancelling context")
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

	ctx := context.Background()
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

			r := retry{
				desc:   "test fxn",
				retry:  tc.retry,
				random: rand.New(rand.NewSource(1)),
				fxn:    fxn,
			}
			err := r.do(ctx)
			if tc.expected == nil {
				assert.NoError(t, err)
			} else {
				assert.Equal(t, tc.expected.Error(), err.Error())
			}
		})
	}
}

func TestWithRetry_client(t *testing.T) {
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
				random: rand.New(rand.NewSource(1)),
			}

			err := w.withRetry(ctx, c.Apply, "apply")
			if tc.applyErr != nil {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
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
			r := retry{
				random: rand.New(rand.NewSource(1)),
			}
			a := r.waitTime(tc.attempt)

			actual := float64(a) / float64(time.Second)
			assert.GreaterOrEqual(t, actual, tc.minReturn)
			assert.LessOrEqual(t, actual, tc.maxReturn)
		})
	}
}
