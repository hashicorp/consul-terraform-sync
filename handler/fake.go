package handler

import (
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
	name string
	err  bool
	next Handler
}

// NewFake configures and returns a new fake handler
func NewFake(config map[string]interface{}) (*Fake, error) {
	h := &Fake{
		next: nil,
	}

	for k, val := range config {
		switch k {
		case "name":
			if v, ok := val.(string); ok {
				h.name = v
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
func (h *Fake) Do(prevErr error) error {
	fmt.Printf("FakeHandler: '%s'\n", h.name)

	var err error = nil
	if h.err {
		err = fmt.Errorf("error %s", h.name)
	}

	return callNext(h.next, prevErr, err)
}

// SetNext sets the next handler that should be called
func (h *Fake) SetNext(next Handler) {
	h.next = next
}
