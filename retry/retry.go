// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package retry

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/hashicorp/consul-terraform-sync/logging"
	"github.com/pkg/errors"
)

const (
	retrySystemName    = "retry"
	DefaultMaxWaitTime = 15 * time.Minute // 15 minutes
)

// NonRetryableError represents an error returned
// if the error should not be retried
type NonRetryableError struct {
	Err error
}

// Error returns an error string
func (e *NonRetryableError) Error() string {
	return e.Err.Error()
}

// Unwrap returns the underlying error
func (e *NonRetryableError) Unwrap() error {
	return e.Err
}

// Retry handles executing and retrying a function
// It uses exponential backoff between retry attempts
type Retry struct {
	maxRetry    int // doesn't count initial try, set to -1 for infinite retries
	random      *rand.Rand
	testMode    bool
	maxWaitTime time.Duration // The maximum wait time between retries
}

// NewRetry initializes a retry handler
// maxRetry is *retries*, so maxRetry of 2 means 3 total tries.
// -1 retries means indefinite retries
func NewRetry(maxRetry int, seed int64) Retry {
	return Retry{
		maxRetry:    maxRetry,
		random:      rand.New(rand.NewSource(seed)),
		maxWaitTime: DefaultMaxWaitTime,
	}
}

// NewRetryWithMaxWaitTime initializes a retry handler
// maxRetry is *retries*, so maxRetry of 2 means 3 total tries.
// -1 retries means indefinite retries
func NewRetryWithMaxWaitTime(maxRetry int, seed int64, maxWaitTime time.Duration) Retry {
	return Retry{
		maxRetry:    maxRetry,
		random:      rand.New(rand.NewSource(seed)),
		maxWaitTime: maxWaitTime,
	}
}

// Do calls a function with exponential retries with a random delay.
func (r Retry) Do(ctx context.Context, f func(context.Context) error, desc string) error {
	var errs error
	logger := logging.FromContext(ctx).Named(retrySystemName)

	err := f(ctx)
	var nonRetryableError *NonRetryableError
	if err == nil || r.maxRetry == 0 {
		return err
	} else if errors.As(err, &nonRetryableError) {
		logger.Debug("received non-retryable error", "description", desc)
		return err
	}

	var attempt int
	wait := r.waitTime(attempt)
	interval := time.NewTicker(wait)
	defer interval.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info("stopping retry", "description", desc)
			if errs != nil {
				return fmt.Errorf("%v: %v", ctx.Err(), errs)
			}
			return ctx.Err()
		case <-interval.C:
			attempt++
			if attempt > 1 {
				logger.Warn("retrying", "attempt_number", attempt, "description", desc)
			}
			err := f(ctx)
			if err == nil {
				return nil
			}

			err = fmt.Errorf("retry attempt #%d failed '%w'", attempt, err)

			if errs == nil {
				errs = err
			} else {
				errs = errors.Wrap(errs, err.Error())
			}

			if errors.As(err, &nonRetryableError) {
				logger.Debug("received non retryable error")
				return errs
			}

			wait := r.waitTime(attempt)
			interval = time.NewTicker(wait)
		}

		if r.maxRetry >= 0 && attempt >= r.maxRetry {
			return errs
		}
	}
}

func (r Retry) waitTime(attempt int) time.Duration {
	if r.testMode {
		return 1 * time.Nanosecond
	}
	return WaitTime(attempt, r.random, r.maxWaitTime)
}

// WaitTime calculates the wait time based off the attempt number based off
// exponential backoff with a random delay. It caps at the passed in maxWaitTime.
func WaitTime(attempt int, random *rand.Rand, maxWaitTime time.Duration) time.Duration {
	a := float64(attempt)

	// Check if max attempts reached, assumes no jitter
	maxBaseTimeWithDelayInS := float64(maxWaitTime) / float64(time.Second)
	maxA := math.Log2(maxBaseTimeWithDelayInS)
	if a >= maxA {
		return maxWaitTime
	}

	// Calculate the wait time
	baseTimeSeconds := math.Exp2(a)
	nextTimeSeconds := math.Exp2(a + 1)
	delayRangeInSeconds := (nextTimeSeconds - baseTimeSeconds) / 2
	delayInSeconds := random.Float64() * delayRangeInSeconds

	waitTimeInSeconds := baseTimeSeconds + delayInSeconds

	return time.Duration(waitTimeInSeconds * float64(time.Second))
}

// NewTestRetry is the test version, returns Retry in test mode (nanosecond retry delay).
func NewTestRetry(maxRetry int) Retry {
	maxWaitTime := 1 * time.Nanosecond
	r := NewRetryWithMaxWaitTime(maxRetry, 1, maxWaitTime)
	return r
}
