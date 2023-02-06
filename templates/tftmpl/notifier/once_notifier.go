// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package notifier

import (
	"sort"
	"sync"

	"github.com/hashicorp/consul-terraform-sync/templates"
	"github.com/hashicorp/hcat/dep"
)

// TriggerCheck functions will receive any data passed to
// the OnceNotifier and are responsible for returning two
// different booleans to indicate whether a render or task
// execution should occur.
type TriggerCheck func(d interface{}) (render, trigger bool)

// OnceNotifier intercepts `Notify()` calls to templates and controls whether rendering
// or task execution occurs. All renders and task executions will be fired until
// `SetOnceDone()` is called. After once-mode is considered to be done, then the
// TriggerCheck func will take over for deciding whether to render or trigger task
// execution.
//
// Due to the way we interface with hcat, the only safe way to know if a task has all
// data is to check the `hcat.ResolveEvent.Complete` field. Therefore, it is expected
// that some other portion of code will be responsible for calling `SetOnceDone()`.
type OnceNotifier struct {
	templates.Template
	mu           sync.Mutex
	triggerCheck TriggerCheck
	onceDone     bool
}

func NewOnceNotifier(triggerCheck TriggerCheck, template templates.Template) *OnceNotifier {
	return &OnceNotifier{
		Template:     template,
		triggerCheck: triggerCheck,
	}
}

func (n *OnceNotifier) OnceDone() bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.onceDone
}

func (n *OnceNotifier) SetOnceDone() {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.onceDone = true
}

func (n *OnceNotifier) Notify(d interface{}) bool {
	n.mu.Lock()
	defer n.mu.Unlock()
	// Always call the trigger function so that it can track changes.
	render, trigger := n.triggerCheck(d)
	// Render task if once mode is not completed or if the trigger indicates.
	if render || !n.onceDone {
		n.Template.Notify(d)
	}
	// Trigger task if once mode is not completed or if the trigger indicates.
	// The task will check and prevent execution if the template isn't ready.
	return trigger || !n.onceDone
}

// TriggerCheckSuppress never triggers a task execution but renders on every call.
func TriggerCheckSuppress(d interface{}) (render, trigger bool) {
	return true, false
}

// TriggerCheckConsulKV triggers and renders on every KV change.
func TriggerCheckConsulKV(d interface{}) (render, trigger bool) {
	_, ok1 := d.(*dep.KeyPair)
	_, ok2 := d.([]*dep.KeyPair)
	ok := ok1 || ok2
	return ok, ok
}

// TriggerCheckService triggers and renders on every service change.
func TriggerCheckService(d interface{}) (render, trigger bool) {
	_, ok := d.([]*dep.HealthService)
	return ok, ok
}

// MakeTriggerCheckCatalogService creates a function that tracks
// catalog service state between calls. If any change is detected
// to the service names, then it will trigger and render. Otherwise,
// no trigger or render should occur.
func MakeTriggerCheckCatalogService() TriggerCheck {
	var mu sync.Mutex
	var oldServices []string
	return func(d interface{}) (render, trigger bool) {
		new, ok := d.([]*dep.CatalogSnippet)
		if !ok {
			return false, false
		}
		mu.Lock()
		defer mu.Unlock()
		// Get the names of the services so we can compare.
		newServices := make([]string, len(new))
		for ix, s := range new {
			newServices[ix] = s.Name
		}
		sort.Strings(newServices)
		// Different length should immediately notify.
		if len(oldServices) != len(newServices) {
			oldServices = newServices
			return true, true
		}
		// Different values in the list should immediately notify.
		for i := range oldServices {
			if oldServices[i] != newServices[i] {
				oldServices = newServices
				return true, true
			}
		}
		oldServices = newServices
		return false, false
	}
}
