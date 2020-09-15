package handler

import "fmt"

// Handler handles additional actions that need to be executed. These can
// be at any level. They are expected to be independent and such that they
// execute, handler errors internal, continue to the next handler.
type Handler interface {

	// Do executes the handler. Any errors that arise should be handled and
	// retried within
	Do()

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
