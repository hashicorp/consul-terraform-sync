package notifier

import (
	"fmt"

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
	once     bool
	depTotal int
	counter  int
}

// NewSuppressNotification creates a new SuppressNotification notifier.
// serviceCount parameter: the number of services the task is configured with
func NewSuppressNotification(tmpl templates.Template, dependencyCount int) *SuppressNotification {
	return &SuppressNotification{
		Template: tmpl,
		depTotal: dependencyCount,
		logger:   logging.Global().Named(logSystemName).Named(supSubsystemName),
	}
}

// Notify suppresses all notifications on any dependency changes. However,
// it will pass the latest dependency information to the template until the
// task is triggered by another means.
//
// Once-mode requires a notification when all dependencies are received in order
// to trigger CTS. It will hang otherwise.
func (n *SuppressNotification) Notify(d interface{}) (notify bool) {
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

	// still let the template know that its dependencies have been updated so
	// that it will re-render when triggered, even if we do not notify watcher.
	n.Template.Notify(d)

	return notify
}
