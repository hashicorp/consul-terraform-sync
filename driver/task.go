package driver

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/hashicorp/consul-nia/client"
	"github.com/pkg/errors"
)

// Service contains service configuration information
type Service struct {
	Datacenter  string
	Description string
	Name        string
	Namespace   string
	Tag         string
}

// Task contains task configuration information
type Task struct {
	Description  string
	Name         string
	Providers    []map[string]interface{}
	ProviderInfo map[string]interface{}
	Services     []Service
	Source       string
	VarFiles     []string
	Version      string
}

// worker executes a unit of work and has a one-to-one relationship with a client
// that will be responsible for executing the work. Currently worker is not safe for
// concurrent use by multiple goroutines
type worker struct {
	client client.Client
	task   Task
	random *rand.Rand

	retry  int
	inited bool
}

func (w *worker) init(ctx context.Context) error {
	r := retry{
		desc:   fmt.Sprintf("Init %s", w.task.Name),
		retry:  w.retry,
		random: w.random,
		fxn: func() error {
			return w.client.Init(ctx)
		},
	}
	if err := r.do(ctx); err != nil {
		return err
	}

	w.inited = true
	return nil
}

func (w *worker) apply(ctx context.Context) error {
	r := retry{
		desc:   fmt.Sprintf("Apply %s", w.task.Name),
		retry:  w.retry,
		random: w.random,
		fxn: func() error {
			return w.client.Apply(ctx)
		},
	}

	return r.do(ctx)
}

// retry handles retries
type retry struct {
	desc   string
	retry  int
	random *rand.Rand
	fxn    func() error
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
