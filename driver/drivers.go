package driver

import (
	"errors"
	"fmt"
	"sync"
)

// Drivers wraps the map of task-name to associated driver so that the map
// can be accessed concurrently
type Drivers struct {
	mu *sync.RWMutex

	drivers map[string]Driver
}

// NewDrivers returns a new drivers object
func NewDrivers() *Drivers {
	return &Drivers{
		mu:      &sync.RWMutex{},
		drivers: make(map[string]Driver),
	}
}

// Add adds a new driver
func (d *Drivers) Add(taskName string, driver Driver) error {
	if taskName == "" {
		return errors.New("error adding driver: task name cannot be empty")
	}
	if driver == nil {
		return fmt.Errorf("error adding driver: '%s' driver cannot be nil", taskName)
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	if _, ok := d.drivers[taskName]; ok {
		return fmt.Errorf("error adding driver: a driver already exists for '%s'",
			taskName)
	}

	d.drivers[taskName] = driver
	return nil
}

// Get retrieves the driver for a task
func (d *Drivers) Get(taskName string) (Driver, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	driver, ok := d.drivers[taskName]
	if !ok {
		return nil, false
	}

	return driver, true
}

// Map returns a copy of the map containing the drivers
func (d *Drivers) Map() map[string]Driver {
	d.mu.RLock()
	defer d.mu.RUnlock()

	copy := make(map[string]Driver)
	for k, v := range d.drivers {
		copy[k] = v
	}
	return copy
}
