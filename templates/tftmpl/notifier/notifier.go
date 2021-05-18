package notifier

import (
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
	once     bool
	services []string
}

// Notify notifies when Catalog Services registration changes
func (n *CatalogServicesRegistration) Notify(d interface{}) (notify bool) {
	if v, ok := d.([]*dep.CatalogSnippet); ok {
		if n.registrationChange(v) {
			n.Template.Notify(d)
			return true
		}
	}
	return false
}

// registrationChange determines whether or not the latest Catalog Service
// changes are registration changes i.e. we want to ignore tag changes.
// Note: this can cause once-mode to hang if there are no registration changes,
// so return true for this edge-case
func (n *CatalogServicesRegistration) registrationChange(new []*dep.CatalogSnippet) bool {
	newServices := make([]string, len(new))
	for ix, s := range new {
		newServices[ix] = s.Name
	}
	defer func() { n.services = newServices }()

	// first time through should notify (once-mode)
	if !n.once {
		n.once = true
		return true
	}

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
