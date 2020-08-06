package controller

import (
	"time"

	"github.com/hashicorp/hcat"
)

// hcatTemplate describes the interface for hcat's Template
// https://github.com/hashicorp/hcat
type hcatTemplate interface {
	Render(content []byte) (hcat.RenderResult, error)
	Execute(hcat.Recaller) (*hcat.ExecuteResult, error)
	ID() string
}

// hcatResolver describes the interface for hcat's Resolver
// https://github.com/hashicorp/hcat
type hcatResolver interface {
	Run(tmpl hcat.Templater, w hcat.Watcherer) (hcat.ResolveEvent, error)
}

// hcatWatcher describes the interface for hcat's Watcher
// https://github.com/hashicorp/hcat
type hcatWatcher interface {
	Wait(timeout time.Duration) error
	Add(d hcat.Dependency) bool
	Changed(tmplID string) bool
	Recall(id string) (interface{}, bool)
	Register(tmplID string, deps ...hcat.Dependency)
}
