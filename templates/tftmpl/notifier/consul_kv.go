package notifier

import (
	"log"

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
}

// NewConsulKV creates a new ConsulKVNotifier.
// serviceCount parameter: the number of services the task is configured with
func NewConsulKV(tmpl templates.Template, serviceCount int) *ConsulKV {
	return &ConsulKV{
		Template: tmpl,
		depTotal: serviceCount + 1, // for additional Consul KV dependency
	}
}

// Notify notifies when a Consul KV pair or set of pairs changes.
//
// Notifications are sent when:
// A. There is a change in the Consul KV dependency for whether a single key pair
//    exists or no longer exists (dep.KVExists)
// B. There is a change in the Consul KV dependency for a single key pair where
//    only the value of the key pair is returned (dep.KvValue)
// C. There is a change in the Consul KV dependency for a set of key pairs where
//    a list of key pairs is returned ([]*dep.KeyPair)
// D. All the dependencies have been received for the first time. This is
//    regardless of the dependency type that "completes" having received all the
//    dependencies.
//
// Notification are suppressed when:
//  - Other types of dependencies that are not Consul KV. For example,
//    Services ([]*dep.HealthService).
func (n *ConsulKV) Notify(d interface{}) (notify bool) {
	log.Printf("[DEBUG] (notifier.kv) received dependency change type %T", d)
	notify = false

	if exists, ok := d.(dep.KVExists); ok {
		log.Printf("[DEBUG] (notifier.kv) notify Consul KV pair exists change")
		notify = true

		if !n.once && bool(exists) {
			// expect a KvValue dependency if the key exists
			n.depTotal++
		}
	}

	if !n.once {
		n.counter++
		// after all dependencies are received, notify so once-mode can complete
		if n.counter >= n.depTotal {
			log.Printf("[DEBUG] (notifier.kv) notify once-mode complete")
			n.once = true
			notify = true
		}
	}

	if _, ok := d.(dep.KvValue); ok {
		log.Printf("[DEBUG] (notifier.kv) notify Consul KV pair value change")
		notify = true
	}

	if _, ok := d.([]*dep.KeyPair); ok {
		log.Printf("[DEBUG] (notifier.kv) notify Consul KV pair list change")
		notify = true
	}

	if notify {
		n.Template.Notify(d)
	}

	return notify
}
