package notifier

import (
	"fmt"
	"time"

	"github.com/hashicorp/consul-terraform-sync/logging"
	"github.com/hashicorp/hcat/dep"
)

// ValidationDep is a dependency used to indicate that Notify was called
// as part of validating the template.
type ValidationDep string

// NewValidationDep creates a new ValidationDep where the value is the ID
// of the template passed to it and the current time.
func NewValidationDep(id string) ValidationDep {
	return ValidationDep(fmt.Sprintf("%s-%s", id, time.Now().String()))
}

// logDependency logs details about the dependencies that the notifiers
// receive
func logDependency(logger logging.Logger, dependency interface{}) {
	switch d := dependency.(type) {
	case []*dep.HealthService:
		serviceNames := make([]string, len(d))
		for ix, hs := range d {
			serviceNames[ix] = hs.Name
		}
		logger.Debug("received dependency",
			"variable", "services", "names", serviceNames)
	case []*dep.CatalogSnippet:
		serviceNames := make([]string, len(d))
		for ix, hs := range d {
			serviceNames[ix] = hs.Name
		}
		logger.Debug("received dependency",
			"variable", "catalog_services", "names", serviceNames)
	case *dep.KeyPair:
		logger.Debug("received dependency",
			"variable", "consul_kv", "recurse", false, "key", d.Key)

	case []*dep.KeyPair:
		keys := make([]string, len(d))
		for ix, kv := range d {
			keys[ix] = kv.Key
		}
		logger.Debug("received dependency",
			"variable", "consul_kv", "recurse", true, "keys", keys)

	case ValidationDep:
		logger.Debug("received dependency for validating template")

	default:
		logger.Debug("received unknown dependency",
			"variable", fmt.Sprintf("%T", dependency))
	}
}
