package handler

import (
	"context"
	"errors"
	"fmt"
	"log"
)

// TerraformProviderFake is the name of a fake Terraform provider
const TerraformProviderFake = "fake-sync"

var _ Handler = (*Fake)(nil)

// Fake is the handler for out-of-band actions for a fake Terraform provider.
// Intended to be used for testing and examples.
type Fake struct {
	name         string
	successFirst bool
	err          bool
	next         Handler

	// used by successFirst to determine if this is the first time the handler
	// has been called
	first bool
}

// NewFake configures and returns a new fake handler
func NewFake(config map[string]interface{}) (*Fake, error) {
	h := &Fake{
		first: true,
		next:  nil,
	}

	for k, val := range config {
		switch k {
		case "name":
			if v, ok := val.(string); ok {
				h.name = v
			}
		case "success_first":
			if v, ok := val.(bool); ok {
				h.successFirst = v
			}
		case "err":
			if v, ok := val.(bool); ok {
				h.err = v
			}
		}
	}

	if h.name == "" {
		return nil, errors.New("FakeHandler: missing 'name' configuration")
	}

	log.Printf("[INFO] (handler.fake) creating handler with name: %s", h.name)
	return h, nil
}

// Do executes fake handler, which fmt.Print-s the fake handler's name which
// is the output inspected by handler example. It returns an error if configured
// to do so.
func (h *Fake) Do(ctx context.Context, prevErr error) error {
	fmt.Printf("FakeHandler: '%s'\n", h.name)

	var err error = nil
	if h.err {
		err = fmt.Errorf("error %s", h.name)
	}

	if h.first == true {
		h.first = false
		if h.successFirst {
			err = nil
		}
	}

	return callNext(ctx, h.next, prevErr, err)
}

// SetNext sets the next handler that should be called
func (h *Fake) SetNext(next Handler) {
	h.next = next
}
