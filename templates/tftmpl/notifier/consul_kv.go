package notifier

import (
	"sync"

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
	logger logging.Logger

	// count all tmplfuncs needed to complete once-mode
	once    bool
	tfTotal int
	counter int

	mu sync.RWMutex
}

func (n *ConsulKV) Override() {
	n.mu.Lock()
	defer n.mu.Unlock()

	if !n.once {
		n.once = true
	}
}

// NewConsulKV creates a new ConsulKVNotifier.
//
// tmplFuncTotal param: the total number of monitored tmplFuncs in the template.
// This is the number of monitored tmplfuncs needed for both the consul-kv
// condition and any module inputs. This number is equivalent to the number of
// hashicat dependencies.
//
// Examples:
// - consul-kv: 1 tmplfunc
// - services-regex: 1 tmplfunc
// - services-name: len(services) tmplfuncs
func NewConsulKV(tmpl templates.Template, tmplFuncTotal int) *ConsulKV {
	logger := logging.Global().Named(logSystemName).Named(kvSubsystemName)
	logger.Trace("creating notifier", "type", kvSubsystemName,
		"tmpl_func_total", tmplFuncTotal)

	return &ConsulKV{
		Template: tmpl,
		tfTotal:  tmplFuncTotal,
		logger:   logger,
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
	n.mu.Lock()
	defer n.mu.Unlock()

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

	// dependency for {{ keyExistsGet }} i.e. recurse=false
	if _, ok := d.(*dep.KeyPair); ok {
		n.logger.Debug("notify Consul KV pair change")
		notify = true
	}

	// dependency for {{ keys }} i.e. recurse=true
	if _, ok := d.([]*dep.KeyPair); ok {
		n.logger.Debug("notify Consul KV pair list change")
		notify = true
	}

	if notify {
		n.Template.Notify(d)
	}

	return notify
}
