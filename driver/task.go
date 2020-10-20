package driver

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"path/filepath"
	"time"

	"github.com/hashicorp/consul-terraform-sync/client"
	mocks "github.com/hashicorp/consul-terraform-sync/mocks/client"
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
	Providers    []map[string]interface{} // task.providers config info
	ProviderInfo map[string]interface{}   // driver.required_provider config info
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

// workerConfig configures a worker
type workerConfig struct {
	task       Task
	clientType string
	log        bool
	persistLog bool
	path       string
	workingDir string
	retry      int
}

// newWorker initializes a worker for a task
func newWorker(c *workerConfig) (*worker, error) {
	client, err := initClient(c)
	if err != nil {
		log.Printf("[ERR] (task) init client type '%s' error: %s", c.clientType, err)
		return nil, err
	}

	return &worker{
		client: client,
		task:   c.task,
		random: rand.New(rand.NewSource(time.Now().UnixNano())),
		retry:  c.retry,
	}, nil
}

// initClient initializes a specific type of client given a task
func initClient(conf *workerConfig) (client.Client, error) {
	var err error
	var c client.Client
	taskName := conf.task.Name

	switch conf.clientType {
	case developmentClient:
		log.Printf("[TRACE] (task) creating development client for task '%s'", taskName)
		c, err = client.NewPrinter(&client.PrinterConfig{
			LogLevel:   "debug",
			ExecPath:   conf.path,
			WorkingDir: filepath.Join(conf.workingDir, taskName),
			Workspace:  taskName,
		})
	case testClient:
		log.Printf("[TRACE] (task) creating mock client for task '%s'", taskName)
		c = new(mocks.Client)
	default:
		log.Printf("[TRACE] (task) creating terraform cli client for task '%s'", taskName)
		c, err = client.NewTerraformCLI(&client.TerraformCLIConfig{
			Log:        conf.log,
			PersistLog: conf.persistLog,
			ExecPath:   conf.path,
			WorkingDir: filepath.Join(conf.workingDir, taskName),
			Workspace:  taskName,
			VarFiles:   conf.task.VarFiles,
		})
	}

	return c, err
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

// retry handles executing and retrying a function
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
