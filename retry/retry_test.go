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

func TestRetry_WithNonRetryableError(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name                string
		maxRetry            int
		nonRetryAbleErrorOn int
		expected            error
	}{
		{
			"non retryable error on first attempt",
			2, // max retries is 2, but will not retry
			0,
			&NonRetryableError{
				Err: errors.New("error on 0"),
			},
		},
		{
			"non retryable error after enters retry loop",
			2, // max retries is 2, but will exit due to non retryable error after first retry
			1,
			errors.New("retry attempt #1 failed 'this error is not retryable: error on 1'"),
		},
	}

	ctx := context.Background()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// set up fake function
			count := 0
			fxn := func(context.Context) error {
				if count == tc.nonRetryAbleErrorOn {
					err := fmt.Errorf("error on %d", count)
					return &NonRetryableError{Err: err}
				}
				err := fmt.Errorf("error on %d", count)
				count++
				return err
			}

			r := NewTestRetry(tc.maxRetry)
			err := r.Do(ctx, fxn, "test fxn")
			assert.Equal(t, tc.expected.Error(), err.Error())
		})
	}
}

func TestWaitTime(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		attempt   int
		minReturn time.Duration
		maxReturn time.Duration
	}{
		{
			"first attempt",
			0,
			1 * time.Second,
			time.Duration(1.5 * float64(time.Second)),
		},
		{
			"second attempt",
			1,
			2 * time.Second,
			3 * time.Second,
		},
		{
			"third attempt",
			2,
			4 * time.Second,
			6 * time.Second,
		},
		{
			"maximum attempt before max wait time",
			9,
			8 * time.Minute,
			13 * time.Minute,
		},
		{
			"minimum attempt max wait time",
			10,
			maxWaitTime,
			maxWaitTime,
		},
		{
			"high number attempt max wait time",
			20000000000,
			maxWaitTime,
			maxWaitTime,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			random := rand.New(rand.NewSource(time.Now().UnixNano()))
			a := WaitTime(tc.attempt, random)

			assert.GreaterOrEqual(t, a, tc.minReturn)
			assert.LessOrEqual(t, a, tc.maxReturn)
		})
	}
}

func TestNonRetryableError_Error(t *testing.T) {
	err := NonRetryableError{Err: errors.New("some error")}
	var nonRetryableError *NonRetryableError

	assert.True(t, errors.As(&err, &nonRetryableError))
	assert.Equal(t, "this error is not retryable: some error", err.Error())
}

func TestNonRetryableError_Unwrap(t *testing.T) {
	var terr *testError

	var otherErr testError
	err := NonRetryableError{Err: &otherErr}

	// Assert that the wrapped error is still detectable
	// errors.As is the preferred way to call the underlying Unwrap
	assert.True(t, errors.As(&err, &terr))
}

type testError struct {
}

// Error returns an error string
func (e *testError) Error() string {
	return "this is a test error"
}
