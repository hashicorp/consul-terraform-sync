package driver

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/pkg/errors"
)

// worker manages execution of a function and abstracts error handling and
// retries for the function from the caller and function implementation.
type worker struct {
	random *rand.Rand
	retry  int
}

// retry handles executing and retrying a function
type retry struct {
	desc   string
	retry  int
	random *rand.Rand
	fxn    func() error
}

// newWorker initializes a worker for a task
func newWorker(retry int) *worker {
	return &worker{
		random: rand.New(rand.NewSource(time.Now().UnixNano())),
		retry:  retry,
	}
}

func (w *worker) withRetry(ctx context.Context, f func(context.Context) error, desc string) error {
	r := retry{
		desc:   desc,
		retry:  w.retry,
		random: w.random,
		fxn: func() error {
			return f(ctx)
		},
	}

	return r.do(ctx)
}

// do calls a function with exponential retry with a random delay. First
// call also has random delay.
func (r *retry) do(ctx context.Context) error {
	count := r.retry + 1
	var errs error

	attempt := 0
	wait := r.waitTime(attempt)
	interval := time.NewTicker(time.Duration(wait))
	defer interval.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Printf("[INFO] (task) stopping retry of '%s'", r.desc)
			return ctx.Err()
		case <-interval.C:
			attempt++
			if attempt > 1 {
				log.Printf("[WARN]: (task) retrying '%s'. attempt #%d", r.desc, attempt)
			}
			err := r.fxn()
			if err == nil {
				return nil
			}

			err = fmt.Errorf("attempt #%d failed '%s'", attempt, err)

			if errs == nil {
				errs = err
			} else {
				errs = errors.Wrap(errs, err.Error())
			}

			wait := r.waitTime(attempt)
			interval = time.NewTicker(time.Duration(wait))
		}
		if attempt >= count {
			return errs
		}
	}
}

// waitTime calculates the wait time based off the attempt number based off
// exponential backoff with a random delay.
func (r *retry) waitTime(attempt int) int {
	a := float64(attempt)
	baseTimeSeconds := a * a
	nextTimeSeconds := (a + 1) * (a + 1)
	delayRange := (nextTimeSeconds - baseTimeSeconds) / 2
	delay := r.random.Float64() * delayRange
	total := (baseTimeSeconds + delay) * float64(time.Second)
	return int(total)
}
