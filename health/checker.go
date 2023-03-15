// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package health

import "fmt"

//go:generate mockery --name=Checker --filename=checker.go --output=../mocks/health

// UnhealthySystemError represents an error returned
// if the system is not healthy. This error can be returned from a health manager
// to indicate that the response code returned should be a `503`
type UnhealthySystemError struct {
	Err error
}

// Error returns an error string
func (e *UnhealthySystemError) Error() string {
	return fmt.Sprintf("CTS is not healthy: %v", e.Err)
}

// Unwrap returns the underlying error
func (e *UnhealthySystemError) Unwrap() error {
	return e.Err
}

// Checker includes methods necessary for managing and determining the health of the system
type Checker interface {
	Check() error
}

var _ Checker = (*BasicChecker)(nil)

// BasicChecker supports a simple health check, which always returns nil
type BasicChecker struct{}

// Check always returns nil
func (h *BasicChecker) Check() error {
	return nil
}
