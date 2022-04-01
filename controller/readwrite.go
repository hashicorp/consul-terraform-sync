package controller

import (
	"context"
	"time"

	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/driver"
	"github.com/hashicorp/consul-terraform-sync/retry"
)

var (
	_ Controller = (*ReadWrite)(nil)

	// Number of times to retry attempts
	defaultRetry uint = 2
)

// ReadWrite is the controller to run in read-write mode
type ReadWrite struct {
	*baseController
	retry retry.Retry

	watcherCh chan string

	// scheduleStartCh is used to coordinate scheduled tasks created via the API
	scheduleStartCh chan driver.Driver
	// scheduleStopChs is a map of channels used to stop scheduled tasks
	scheduleStopChs map[string](chan struct{})

	// deleteCh is used to coordinate task deletion via the API
	deleteCh chan string

	// taskNotify is only initialized if EnableTestMode() is used. It provides
	// tests insight into which tasks were triggered and had completed
	taskNotify chan string
}

// NewReadWrite configures and initializes a new ReadWrite controller
func NewReadWrite(conf *config.Config) (*ReadWrite, error) {
	baseCtrl, err := newBaseController(conf)
	if err != nil {
		return nil, err
	}

	return &ReadWrite{
		baseController:  baseCtrl,
		retry:           retry.NewRetry(defaultRetry, time.Now().UnixNano()),
		scheduleStartCh: make(chan driver.Driver, 10), // arbitrarily chosen size
		deleteCh:        make(chan string, 10),        // arbitrarily chosen size
		scheduleStopChs: make(map[string](chan struct{})),
	}, nil
}

// Init initializes the controller before it can be run. Ensures that
// driver is initializes, works are created for each task.
func (rw *ReadWrite) Init(ctx context.Context) error {
	return rw.init(ctx)
}

// EnableTestMode is a helper for testing which tasks were triggered and
// executed. Callers of this method must consume from TaskNotify channel to
// prevent the buffered channel from filling and causing a dead lock.
func (rw *ReadWrite) EnableTestMode() <-chan string {
	rw.taskNotify = make(chan string, rw.drivers.Len())
	return rw.taskNotify
}
