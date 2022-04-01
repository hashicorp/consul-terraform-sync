package retry

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

	r := NewRetry(5, 1)
	fxn := func(context.Context) error {
		return errors.New("test error")
	}

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error)
	go func() {
		err := r.Do(ctx, fxn, "fake fxn that never succeeds")
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
		maxRetry  int
		successOn int
		expected  error
	}{
		{
			"happy path: try twice, success on retry (1)",
			2, // max retries is 2, but will succeed on first retry
			1,
			nil,
		},
		{
			"happy path: infinite retries, success on retry (3)",
			-1, // max retries is infinite, but will succeed on retry 3
			3,
			nil,
		},
		{
			"no success on retries: retry once",
			1,
			100,
			errors.New("retry attempt #1 failed 'error on 1'"),
		},
		{
			"happy path: no retries",
			0,
			0,
			nil,
		},
		{
			"no retries, no success",
			0,
			100,
			errors.New("error on 0"),
		},
	}

	ctx := context.Background()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// set up fake function
			count := 0
			fxn := func(context.Context) error {
				if count == tc.successOn {
					return nil
				}
				err := fmt.Errorf("error on %d", count)
				count++
				return err
			}

			r := NewTestRetry(tc.maxRetry)
			err := r.Do(ctx, fxn, "test fxn")
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

			r := NewRetry(1, 1)
			err := r.Do(ctx, c.Apply, "apply")

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
			1,
			1.5,
		},
		{
			"second attempt",
			1,
			2,
			3,
		},
		{
			"third attempt",
			2,
			4,
			6,
		},
		{
			"max attempt minimum",
			10,
			maxWaitTimeInNs / float64(time.Second),
			maxWaitTimeInNs / float64(time.Second),
		},
		{
			"max attempt high value",
			1000,
			maxWaitTimeInNs / float64(time.Second),
			maxWaitTimeInNs / float64(time.Second),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			random := rand.New(rand.NewSource(time.Now().UnixNano()))
			a := WaitTimeInNs(tc.attempt, random)

			actual := float64(a) / float64(time.Second)
			assert.GreaterOrEqual(t, actual, tc.minReturn)
			assert.LessOrEqual(t, actual, tc.maxReturn)
		})
	}
}
