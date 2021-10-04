package notifier

import (
	"fmt"

	"github.com/hashicorp/consul-terraform-sync/logging"
	"github.com/hashicorp/consul-terraform-sync/templates"
	"github.com/hashicorp/hcat/dep"
)

// ConsulKV is a custom notifier expected to be used for a template that
// contains consulKVNotifier template function.
//
// This notifier only notifies on changes to Consul KV pairs and once-mode.
// It suppresses notifications for changes to other tmplfuncs.
type ConsulKV struct {
	templates.Template

	// count all dependencies needed to complete once-mode
	once     bool
	depTotal int
	counter  int
	logger   logging.Logger
}

// NewConsulKV creates a new ConsulKVNotifier.
// serviceCount parameter: the number of services the task is configured with
func NewConsulKV(tmpl templates.Template, serviceCount int) *ConsulKV {
	return &ConsulKV{
		Template: tmpl,
		// expect services and either []*dep.KeyPair or *dep.KeyPair
		depTotal: serviceCount + 1,
		logger:   logging.Global().Named(logSystemName).Named(kvSubsystemName),
	}
}

// Notify notifies when a Consul KV pair or set of pairs changes.
//
// Notifications are sent when:
// A. There is a change in the Consul KV dependency for a single key
//    pair (recurse=false) where the pair is returned (*dep.KeyPair)
// B. There is a change in the Consul KV dependency for a set of key pairs (recurse=true)
//    where a list of key pairs is returned ([]*dep.KeyPair)
// C. All the dependencies have been received for the first time. This is
//    regardless of the dependency type that "completes" having received all the
//    dependencies.
//
// Notification are suppressed when:
//  - Other types of dependencies that are not Consul KV. For example,
//    Services ([]*dep.HealthService).
func (n *ConsulKV) Notify(d interface{}) (notify bool) {
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

	if _, ok := d.(*dep.KeyPair); ok {
		n.logger.Debug("notify Consul KV pair change")
		notify = true
	}

	if _, ok := d.([]*dep.KeyPair); ok {
		n.logger.Debug("notify Consul KV pair list change")
		notify = true
	}

	if notify {
		n.Template.Notify(d)
	}

	return notify
}
