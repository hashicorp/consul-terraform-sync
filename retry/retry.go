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
	taskSystemName  = "task"
	maxWaitTimeInNs = 15 * float64(time.Minute) // 15 minutes
)

// Retry handles executing and retrying a function
type Retry struct {
	maxRetry int // doesn't count initial try, set to -1 for infinite retries
	random   *rand.Rand
	testMode bool
	logger   logging.Logger
}

// NewRetry initializes a retry handler
// maxRetry is *retries*, so maxRetry of 2 means 3 total tries.
// -1 retries means indefinite retries
func NewRetry(maxRetry int, seed int64) Retry {
	return Retry{
		maxRetry: maxRetry,
		random:   rand.New(rand.NewSource(seed)),
		logger:   logging.Global().Named(taskSystemName),
	}
}

// Do calls a function with exponential retries with a random delay.
func (r Retry) Do(ctx context.Context, f func(context.Context) error, desc string) error {
	var errs error

	err := f(ctx)
	if err == nil || r.maxRetry == 0 {
		return err
	}

	var attempt int
	wait := r.waitTimeInNs(attempt)
	interval := time.NewTicker(time.Duration(wait))
	defer interval.Stop()

	for {
		select {
		case <-ctx.Done():
			r.logger.Info("stopping retry", "description", desc)
			return ctx.Err()
		case <-interval.C:
			attempt++
			if attempt > 1 {
				r.logger.Warn("retrying", "attempt_number", attempt, "description", desc)
			}
			err := f(ctx)
			if err == nil {
				return nil
			}

			err = fmt.Errorf("retry attempt #%d failed '%s'", attempt, err)

			if errs == nil {
				errs = err
			} else {
				errs = errors.Wrap(errs, err.Error())
			}

			wait := r.waitTimeInNs(attempt)
			interval = time.NewTicker(time.Duration(wait))
		}

		if r.maxRetry >= 0 && attempt >= r.maxRetry {
			return errs
		}
	}
}

func (r Retry) waitTimeInNs(attempt int) int {
	if r.testMode {
		return 1
	}
	return WaitTimeInNs(attempt, r.random)
}

// WaitTimeInNs calculates the wait time in nanoseconds based off the attempt number based off
// exponential backoff with a random delay. It caps at the constant maxWaitTimeInNs.
func WaitTimeInNs(attempt int, random *rand.Rand) int {
	a := float64(attempt)
	baseTimeSeconds := math.Exp2(a)
	nextTimeSeconds := math.Exp2(a + 1)
	delayRange := (nextTimeSeconds - baseTimeSeconds) / 2
	delay := random.Float64() * delayRange
	total := (baseTimeSeconds + delay) * float64(time.Second)

	if total > maxWaitTimeInNs {
		return int(maxWaitTimeInNs)
	}

	return int(total)
}

// NewTestRetry is the test version, returns Retry in test mode (nanosecond retry delay).
func NewTestRetry(maxRetry int) Retry {
	r := NewRetry(maxRetry, 1)
	r.testMode = true
	return r
}
