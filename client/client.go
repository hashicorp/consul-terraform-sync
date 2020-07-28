package client

import "context"

// Client describes the interface for a driver's client that interacts
// with network infrastructure.
type Client interface {
	// Init initializes the client and environment
	Init(ctx context.Context) error

	// Apply makes a request to apply changes
	Apply(ctx context.Context) error

	// Plan makes a request to generate a plan of proposed changes
	Plan(ctx context.Context) error

	// GoString defines the printable version of the client
	GoString() string
}
