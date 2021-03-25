package retry

import (
	"context"
	"fmt"
	"log"
	"math"
	"math/rand"
	"time"

	"github.com/pkg/errors"
)

// Retry handles executing and retrying a function
type Retry struct {
	maxRetry uint // doesn't count initial try
	random   *rand.Rand
	testMode bool
}

// NewRetry initializes a retry handler
// maxRetry is *retries*, so maxRetry of 2 means 3 total tries.
func NewRetry(maxRetry uint, seed int64) Retry {
	return Retry{
		maxRetry: maxRetry,
		random:   rand.New(rand.NewSource(seed)),
	}
}

// Do calls a function with exponential retry with a random delay.
func (r Retry) Do(ctx context.Context, f func(context.Context) error, desc string) error {
	var errs error

	err := f(ctx)
	if err == nil || r.maxRetry == 0 {
		return err
	}

	var attempt uint
	wait := r.waitTime(attempt)
	interval := time.NewTicker(time.Duration(wait))
	defer interval.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Printf("[INFO] (task) stopping retry of '%s'", desc)
			return ctx.Err()
		case <-interval.C:
			attempt++
			if attempt > 1 {
				log.Printf("[WARN]: (task) retrying '%s'. attempt #%d", desc, attempt)
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

			wait := r.waitTime(attempt)
			interval = time.NewTicker(time.Duration(wait))
		}
		if attempt >= r.maxRetry {
			return errs
		}
	}
}

// waitTime calculates the wait time based off the attempt number based off
// exponential backoff with a random delay.
func (r Retry) waitTime(attempt uint) int {
	if r.testMode {
		return 1
	}
	a := float64(attempt)
	baseTimeSeconds := math.Exp2(a)
	nextTimeSeconds := math.Exp2(a + 1)
	delayRange := (nextTimeSeconds - baseTimeSeconds) / 2
	delay := r.random.Float64() * delayRange
	total := (baseTimeSeconds + delay) * float64(time.Second)
	return int(total)
}

// Test version, returns retry in test mode (nanosecond retry delay).
func NewTestRetry(maxRetry uint) Retry {
	return Retry{
		maxRetry: maxRetry,
		testMode: true,
	}
}
