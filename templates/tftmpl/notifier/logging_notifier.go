// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package notifier

import (
	"fmt"

	"github.com/hashicorp/consul-terraform-sync/logging"
	"github.com/hashicorp/hcat/dep"
)

// logDependency logs details about the dependencies that the notifiers
// receive
func logDependency(logger logging.Logger, dependency interface{}) {
	switch d := dependency.(type) {
	case []*dep.HealthService:
		serviceIDs := make([]string, len(d))
		for ix, hs := range d {
			serviceIDs[ix] = hs.ID
		}
		logger.Debug("received dependency",
			"variable", "services", "ids", serviceIDs)
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
	default:
		logger.Debug("received unknown dependency",
			"variable", fmt.Sprintf("%T", dependency))
	}
}
