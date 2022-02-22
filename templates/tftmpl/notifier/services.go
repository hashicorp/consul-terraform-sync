package notifier

import (
	"sync"

	"github.com/hashicorp/consul-terraform-sync/logging"
	"github.com/hashicorp/consul-terraform-sync/templates"
	"github.com/hashicorp/hcat/dep"
)

const servicesSubsystemName = "services"

// Services is a custom notifier expected to be used for a template that
// contains {{ service }} or {{ servicesRegex }} template function (tmplfuncs)
// for the condition and any other tmplfuncs for module inputs
//
// This notifier only notifies on changes to services instances information and
// once-mode. It suppresses notifications for changes to other tmplfuncs.
type Services struct {
	templates.Template
	logger logging.Logger

	// count all tmplfuncs needed to complete once-mode
	once    bool
	tfTotal int
	counter int

	mu sync.RWMutex
}

func (n *Services) Override() {
	n.mu.Lock()
	defer n.mu.Unlock()

	if !n.once {
		n.once = true
	}
}

// NewServices creates a new Services notifier.
//
// tmplFuncTotal param: the total number of monitored tmplFuncs in the template.
// This is the number of monitored tmplfuncs needed for both the services
// condition and any module inputs. This number is equivalent to the number of
// hashicat dependencies.
//
// Examples:
// - services-regex: 1 tmplfunc
// - services-name: len(services) tmplfuncs
// - consul-kv: 1 tmplfunc
func NewServices(tmpl templates.Template, tmplFuncTotal int) *Services {
	logger := logging.Global().Named(logSystemName).Named(servicesSubsystemName)
	logger.Trace("creating notifier", "type", servicesSubsystemName,
		"tmpl_func_total", tmplFuncTotal)

	return &Services{
		Template: tmpl,
		tfTotal:  tmplFuncTotal,
		logger:   logger,
	}
}

// Notify notifies when service instance changes: existing service instance
// changes, service instance de/registers
//
// Once-mode requires a notification when all dependencies are received in order
// to trigger CTS. Otherwise it will hang.
func (n *Services) Notify(d interface{}) (notify bool) {
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

	// dependency for {{ servicesRegex }} or {{ service }}
	if _, ok := d.([]*dep.HealthService); ok {
		n.logger.Debug("notify services change")
		notify = true
	}

	// let the template know that its dependencies have updated so that it will
	// re-render when trigger. this does not cause task trigger
	n.Template.Notify(d)

	return notify
}
