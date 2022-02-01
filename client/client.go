package client

import (
	"context"
	"io"
)

//go:generate mockery --name=Client --filename=client.go  --output=../mocks/client

// Client describes the interface for a driver's client that interacts
// with network infrastructure.
type Client interface {
	// SetEnv Set the environment for the client
	SetEnv(map[string]string) error

	// SetStdout Set the standard out for the client
	SetStdout(w io.Writer)

	// Init initializes the client and environment
	Init(ctx context.Context) error

	// Apply makes a request to apply changes
	Apply(ctx context.Context) error

	// Plan makes a request to generate a plan of proposed changes
	Plan(ctx context.Context) (bool, error)

	// Validate verifies that the generated configurations are valid
	Validate(ctx context.Context) error

	// GoString defines the printable version of the client
	GoString() string
}
