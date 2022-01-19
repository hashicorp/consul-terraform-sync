package notifier

import (
	"github.com/hashicorp/consul-terraform-sync/logging"
	"github.com/hashicorp/consul-terraform-sync/templates"
)

const supSubsystemName = "suppress"

// SuppressNotification is a custom notifier expected to be used for tasks that
// are not triggered by the hcat.watcher and are instead triggered by a
// separate process.
//
// The watcher is shared by all dynamic tasks as they wait on watcher for
// dependency changes. On a dependency change, all dynamic tasks are triggered
// to re-render templates and terraform-apply if there are changes to the
// template
//
// Non-dynamic tasks, such as scheduled tasks, do not wait on the watcher and
// therefore should use a SuppressNotification notify to avoid notifying the
// watcher because it will un-necessarily cause all dynamic tasks to trigger.
type SuppressNotification struct {
	templates.Template
	logger logging.Logger

	// count all dependencies needed to complete once-mode
	once    bool
	tfTotal int
	counter int
}

// NewSuppressNotification creates a new SuppressNotification notifier.
//
// tmplFuncTotal param: the total number of monitored tmplFuncs in the template.
// This is the number of monitored tmplfuncs needed for the scheduled task's
// module inputs. This number is equivalent to the number of hashicat dependencies
//
// Examples:
// - services-regex: 1 tmplfunc
// - services-name: len(services) tmplfuncs
// - consul-kv: 1 tmplfunc
func NewSuppressNotification(tmpl templates.Template, tmplFuncTotal int) *SuppressNotification {
	logger := logging.Global().Named(logSystemName).Named(supSubsystemName)
	logger.Trace("creating notifier", "type", supSubsystemName,
		"tmpl_func_total", tmplFuncTotal)

	return &SuppressNotification{
		Template: tmpl,
		tfTotal:  tmplFuncTotal,
		logger:   logger,
	}
}

// Notify suppresses all notifications on any dependency changes. However,
// it will pass the latest dependency information to the template until the
// task is triggered by another means.
//
// Once-mode requires a notification when all dependencies are received in order
// to trigger CTS. Otherwise it will hang.
func (n *SuppressNotification) Notify(d interface{}) (notify bool) {
	logDependency(n.logger, d)
	notify = false

	if !n.once {
		n.counter++
		// after a dependency for each tmplfunc is received, send notification
		// so that once-mode can complete
		if n.counter >= n.tfTotal {
			n.logger.Debug("notify once-mode complete")
			n.once = true
			notify = true
		}
	}

	// let the template know that its dependencies have updated so that it will
	// re-render when trigger. this does not cause task trigger
	n.Template.Notify(d)

	return notify
}
