package notifier

import (
	"log"

	"github.com/hashicorp/consul-terraform-sync/templates"
	"github.com/hashicorp/hcat/dep"
)

// CatalogServicesRegistration is a custom notifier expected to be used
// for a template that contains catalogServicesRegistration template function
// (tmplfunc) and any other tmplfuncs e.g. services tmplfunc.
//
// This notifier only notifies on changes to Catalog Services registration
// information and once-mode. It suppresses notifications for changes to other
// tmplfuncs and changes to Catalog Services tag data.
type CatalogServicesRegistration struct {
	templates.Template
	services []string

	// count all dependencies needed to complete once-mode
	once     bool
	depTotal int
	counter  int
}

// NewCatalogServicesRegistration creates a new CatalogServicesRegistration
// notifier.
// serviceCount parameter: the number of services the task is configured with
func NewCatalogServicesRegistration(tmpl templates.Template, serviceCount int) *CatalogServicesRegistration {
	return &CatalogServicesRegistration{
		Template: tmpl,
		depTotal: serviceCount + 1, // for additional catalog-services dep
	}
}

// Notify notifies when Catalog Services registration changes.
//
// Note: there is a race condition that can cause once-mode to hang. Once-mode
// requires all dependencies to be retrieved before notifying to render the
// complete template (and then executing task). During once-mode, if the last
// dependency received is not a CatalogSnippet dependency with a registration
// change, we still need to notify or else once-mode will hang
func (n *CatalogServicesRegistration) Notify(d interface{}) (notify bool) {
	log.Printf("[DEBUG] (notifier.cs) received dependency change type %T", d)
	notify = false

	if !n.once {
		n.counter++
		// after all dependencies are received, notify so once-mode can complete
		if n.counter >= n.depTotal {
			log.Printf("[DEBUG] (notifier.cs) notify once-mode complete")
			n.once = true
			notify = true
		}
	}

	if v, ok := d.([]*dep.CatalogSnippet); ok {
		if n.registrationChange(v) {
			log.Printf("[DEBUG] (notifier.cs) notify registration change")
			notify = true
		}
	}

	if notify {
		n.Template.Notify(d)
	}

	return notify
}

// registrationChange determines whether or not the latest Catalog Service
// changes are registration changes i.e. we want to ignore tag changes.
func (n *CatalogServicesRegistration) registrationChange(new []*dep.CatalogSnippet) bool {
	newServices := make([]string, len(new))
	for ix, s := range new {
		newServices[ix] = s.Name
	}
	defer func() { n.services = newServices }()

	// change in list of service names should notify
	if len(n.services) != len(newServices) {
		return true
	}
	for i, v := range n.services {
		if v != newServices[i] {
			return true
		}
	}
	return false
}
