package driver

import "context"

var _ Driver = (*MockDriver)(nil)

// MockDriver is a mock implementation of Driver for testing purposes
type MockDriver struct {
	InitFunc       func() error
	InitTaskFunc   func(task Task, force bool) error
	InitWorkerFunc func(task Task) error
	InitWorkFunc   func() error
	ApplyWorkFunc  func() error
	VersionFunc    func() string
}

// NewMockDriver configures and initializes a new mock driver
func NewMockDriver() *MockDriver {
	return &MockDriver{
		InitFunc:       func() error { return nil },
		InitTaskFunc:   func(Task, bool) error { return nil },
		InitWorkerFunc: func(Task) error { return nil },
		InitWorkFunc:   func() error { return nil },
		ApplyWorkFunc:  func() error { return nil },
		VersionFunc:    func() string { return "mock-version" },
	}
}

// Init mocks initializing a driver for testing
func (m *MockDriver) Init() error {
	return m.InitFunc()
}

// InitTask mocks initializing a task for testing
func (m *MockDriver) InitTask(task Task, force bool) error {
	return m.InitTaskFunc(task, force)
}

// InitWorker mocks initializing a worker for testing
func (m *MockDriver) InitWorker(task Task) error {
	return m.InitWorkerFunc(task)
}

// InitWork mocks initializing all work for testing
func (m *MockDriver) InitWork(ctx context.Context) error {
	return m.InitWorkFunc()
}

// ApplyWork mocks applying all work for testing
func (m *MockDriver) ApplyWork(ctx context.Context) error {
	return m.ApplyWorkFunc()
}

// Version returns version for testing
func (m *MockDriver) Version() string {
	return m.VersionFunc()
}
