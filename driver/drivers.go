package driver

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/hashicorp/consul-terraform-sync/logging"
)

// Drivers wraps the map of task-name to associated driver so that the map
// can be accessed concurrently
type Drivers struct {
	mu *sync.RWMutex

	// Map of task name to driver
	drivers map[string]Driver

	// Map of template ID to task name
	driverTemplates map[string]string

	// Tracks which driver is currently active
	active sync.Map

	// Tracks if a driver is marked for deletion
	deletion map[string]bool
}

// NewDrivers returns a new drivers object
func NewDrivers() *Drivers {
	return &Drivers{
		mu:              &sync.RWMutex{},
		drivers:         make(map[string]Driver),
		driverTemplates: make(map[string]string),
		deletion:        make(map[string]bool),
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
	for _, id := range driver.TemplateIDs() {
		d.driverTemplates[id] = taskName
	}
	return nil
}

// Get retrieves the driver for a task by task name
func (d *Drivers) Get(taskName string) (Driver, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	driver, ok := d.drivers[taskName]
	if !ok {
		return nil, false
	}

	return driver, true
}

// GetTaskByTemplate retrieves the driver for a task by template ID
func (d *Drivers) GetTaskByTemplate(tmplID string) (Driver, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	taskName, ok := d.driverTemplates[tmplID]
	if !ok {
		return nil, false
	}

	driver, ok := d.drivers[taskName]
	return driver, ok
}

func (d *Drivers) Reset() {
	d.mu.Lock()
	defer d.mu.Unlock()

	for k := range d.drivers {
		delete(d.drivers, k)
		d.active.Delete(k)
	}
}

func (d *Drivers) Len() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return len(d.drivers)
}

// Map returns a copy of the map containing the drivers
func (d *Drivers) Map() map[string]Driver {
	d.mu.RLock()
	defer d.mu.RUnlock()

	copy := make(map[string]Driver, len(d.drivers))
	for k, v := range d.drivers {
		copy[k] = v
	}
	return copy
}

func (d *Drivers) SetBufferPeriod() {
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, driver := range d.drivers {
		driver.SetBufferPeriod()
	}
}

func (d *Drivers) SetActive(name string) bool {
	d.active.Store(name, struct{}{})
	return true
}

func (d *Drivers) SetInactive(name string) bool {
	_, ok := d.active.Load(name)
	if ok {
		d.active.Delete(name)
	}
	return ok
}

func (d *Drivers) IsActive(name string) bool {
	_, ok := d.active.Load(name)
	return ok
}

// Delete removes the driver for the given task name from
// the map of drivers.
func (d *Drivers) Delete(taskName string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if taskName == "" {
		return errors.New("task name cannot be empty")
	}

	driver, ok := d.drivers[taskName]

	if ok {
		driver.DestroyTask(context.Background())
	} else {
		logging.Global().Debug("attempted to destroy a non-existent task", taskNameLogKey, taskName)
	}

	// delete driver templates associated with task
	for val, id := range d.driverTemplates {
		if val == taskName {
			delete(d.driverTemplates, id)
		}
	}

	delete(d.drivers, taskName)
	delete(d.deletion, taskName)
	return nil
}

func (d *Drivers) MarkForDeletion(name string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.deletion[name] = true
}

func (d *Drivers) IsMarkedForDeletion(name string) bool {
	d.mu.RLock()
	defer d.mu.RUnlock()

	mark, ok := d.deletion[name]
	if !ok {
		return false
	}
	return mark
}
