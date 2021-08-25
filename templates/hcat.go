package templates

import (
	"context"
	"time"

	"github.com/hashicorp/hcat"
)

//go:generate mockery --name=Template  --filename=template.go --output=../mocks/templates
//go:generate mockery --name=Resolver --filename=resolver.go --output=../mocks/templates
//go:generate mockery --name=Watcher --filename=watcher.go --output=../mocks/templates

var _ hcat.Templater = (Template)(nil)
var _ hcat.Renderer = (Template)(nil)
var _ Watcher = (*hcat.Watcher)(nil)

// DepSizeWarning is the threshold of dependencies that we warn the user
// of CTS potentially DDoSing their Consul cluster.
const DepSizeWarning = 128

// Template describes the interface for hashicat's Template structure
// which implements the interfaces Templater and Renderer
// https://github.com/hashicorp/hcat
type Template interface {
	Notify(interface{}) bool
	Render(content []byte) (hcat.RenderResult, error)
	Execute(hcat.Recaller) ([]byte, error)
	ID() string
}

// Resolver describes the interface for hashicat's Resolver structure
// which does implement any hashicat interface at time of writing
// https://github.com/hashicorp/hcat
type Resolver interface {
	Run(tmpl hcat.Templater, w hcat.Watcherer) (hcat.ResolveEvent, error)
}

// Watcher describes the interface for hashicat's Watcher structure
// used by this project
// https://github.com/hashicorp/hcat
type Watcher interface {
	WaitCh(context.Context) <-chan error
	Buffer(hcat.Notifier) bool
	Mark(notifier hcat.IDer)
	SetBufferPeriod(min, max time.Duration, tmplIDs ...string)
	Size() int
	Stop()
	Sweep(notifier hcat.IDer)
	// not used but needed to meet the hcat.Watcherer interface
	Complete(hcat.Notifier) bool
	Recaller(hcat.Notifier) hcat.Recaller
	Register(ns ...hcat.Notifier) error
}
