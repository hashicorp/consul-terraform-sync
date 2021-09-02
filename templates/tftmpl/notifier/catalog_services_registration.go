package notifier

import (
	"fmt"

	"github.com/hashicorp/consul-terraform-sync/logging"
	"github.com/hashicorp/consul-terraform-sync/templates"
	"github.com/hashicorp/hcat/dep"
)

const (
	logSystemName   = "notifier"
	csSubsystemName = "cs"
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
	logger   logging.Logger
}

// NewCatalogServicesRegistration creates a new CatalogServicesRegistration
// notifier.
// serviceCount parameter: the number of services the task is configured with
func NewCatalogServicesRegistration(tmpl templates.Template, serviceCount int) *CatalogServicesRegistration {
	return &CatalogServicesRegistration{
		Template: tmpl,
		depTotal: serviceCount + 1, // for additional catalog-services dep
		logger:   logging.Global().Named(logSystemName).Named(csSubsystemName),
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
	n.logger.Debug("received dependency change", "dependency_type", fmt.Sprintf("%T", d))
	notify = false

	if !n.once {
		n.counter++
		// after all dependencies are received, notify so once-mode can complete
		if n.counter >= n.depTotal {
			n.logger.Debug("notify once-mode complete")
			n.once = true
			notify = true
		}
	}

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
