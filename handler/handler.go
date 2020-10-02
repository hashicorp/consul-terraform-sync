package handler

import (
	"fmt"

	"github.com/pkg/errors"
)

// Handler handles additional actions that need to be executed. These can
// be at any level. Handlers can be chained such that they execute and continue
// to the next handler. A chain of handlers will return an aggregate of any
// errors after the handlers are all executed.
type Handler interface {

	// Do executes the handler. Receives previous error and returns previous
	// error wrapped in any new errors.
	Do(error) error

	// SetNext sets the next handler that should be called
	SetNext(Handler)
}

// TerraformProviderHandler returns the handler for providers that require
// post-Apply, out-of-band actions for a Terraform driver.
//
// Returned handler may be nil even if returned err is nil. This happens when
// no providers have a handler.
func TerraformProviderHandler(providerName string, config interface{}) (Handler, error) {
	c, ok := config.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf(
			"Unexpected config type. Want map[string]interface{}. Got %T", config)
	}

	switch providerName {
	case TerraformProviderPanos:
		return NewPanos(c)
	case TerraformProviderFake:
		return NewFake(c)
	default:
		return nil, nil
	}
}

// callNext should be called by a handler's Do() to call the next handler
func callNext(nextH Handler, prevErr, err error) error {
	nextErr := nextError(prevErr, err)

	if nextH != nil {
		return nextH.Do(nextErr)
	}
	return nextErr
}

// nextError uses the previous error and current error to determine the error
// to pass onto the next handler
func nextError(prevErr error, err error) error {
	if prevErr == nil && err == nil {
		return nil
	}
	if prevErr == nil && err != nil {
		return err
	}
	if prevErr != nil && err == nil {
		return prevErr
	}
	// prevErr != nil && err != nil
	return errors.Wrap(prevErr, err.Error())
}
