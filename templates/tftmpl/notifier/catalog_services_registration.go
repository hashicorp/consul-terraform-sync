package notifier

import (
	"github.com/hashicorp/consul-terraform-sync/logging"
	"github.com/hashicorp/consul-terraform-sync/templates"
	"github.com/hashicorp/hcat/dep"
)

const (
	logSystemName   = "notifier"
	csSubsystemName = "cs"
	kvSubsystemName = "kv"
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
	logger   logging.Logger

	// count all tmplfuncs needed to complete once-mode
	once    bool
	tfTotal int
	counter int
}

// NewCatalogServicesRegistration creates a new CatalogServicesRegistration
// notifier.
//
// tmplFuncTotal param: the total number of monitored tmplFuncs in the template.
// This is the number of monitored tmplfuncs needed for both the catalog-services
// condition and any module inputs. This number is equivalent to the number of
// hashicat dependencies.
//
// Examples:
// - catalog-services: 1 tmplfunc
// - services-regex: 1 tmplfunc
// - services-name: len(services) tmplfuncs
// - consul-kv: 1 tmplfunc
func NewCatalogServicesRegistration(tmpl templates.Template, tmplFuncTotal int) *CatalogServicesRegistration {
	logger := logging.Global().Named(logSystemName).Named(servicesSubsystemName)
	logger.Trace("creating notifier", "type", csSubsystemName,
		"tmpl_func_total", tmplFuncTotal)

	return &CatalogServicesRegistration{
		Template: tmpl,
		tfTotal:  tmplFuncTotal,
		logger:   logger,
	}
}

// Notify notifies when Catalog Services registration changes.
//
// Notifications are sent when:
// A. There is a change in the Catalog Service's dependency ([]*dep.CatalogSnippet)
//    that is specifically a service _registration_ change.
// B. All the dependencies have been received for the first time. This is
//    regardless of the dependency type that "completes" having received all the
//    dependencies. Note: this is a special notification sent to handle a race
//    condition that causes hanging during once-mode (details below)
//
// Notification are suppressed when:
//  - There is a change in the Catalog Service's dependency ([]*dep.CatalogSnippet)
//    that is specifically a service _tag_ change.
//  - Other types of dependencies that are not Catalog Service. For example,
//    Services ([]*dep.HealthService).
//
// Race condition: Once-mode requires a notification when all dependencies are
// received in order to trigger CTS. It will hang otherwise. This notifier only
// notifies on Catalog Service registration changes. The dependencies are
// received by the notifier in any order. Therefore sometimes the last
// dependency to "complete all dependencies" is a Health Service change, which
// can lead to no notification even though once-mode requires a notification
// when all dependencies are received.
// Resolved by sending a special notification for once-mode. Bullet B above.
func (n *CatalogServicesRegistration) Notify(d interface{}) (notify bool) {
	logDependency(n.logger, d)
	notify = false

	if !n.once {
		n.counter++
		// after a dependency is received for each tmplfunc, send notification
		// so that once-mode can complete
		if n.counter >= n.tfTotal {
			n.logger.Debug("notify once-mode complete")
			n.once = true
			notify = true
		}
	}

	// dependency for {{ catalogServicesRegistration}}
	if v, ok := d.([]*dep.CatalogSnippet); ok {
		if n.registrationChange(v) {
			n.logger.Debug("notify registration change")
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
