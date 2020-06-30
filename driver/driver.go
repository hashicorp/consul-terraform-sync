package driver

// Driver describes the interface for using an NIA driver to carry out changes
// downstream to update network infrastructure.
type Driver interface {
	// Init initializes the driver and environment
	Init() error

	// Version returns the version of the driver.
	Version() string
}
