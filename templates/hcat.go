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
	// Notify has a different behavior than hcat.Template.Notify and does
	// not mark the template as dirty unless hcat.Template.Notify is called
	// within the implementation
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
	Watch(context.Context, chan string) error
	WaitCh(context.Context) <-chan error
	Buffering(hcat.Notifier) bool
	BufferReset(hcat.Notifier)
	MarkForSweep(notifier hcat.IDer)
	SetBufferPeriod(min, max time.Duration, tmplIDs ...string)
	Size() int
	Stop()
	Sweep(notifier hcat.IDer)
	Register(ns ...hcat.Notifier) error
	Deregister(ns ...hcat.Notifier)
	Clients() hcat.Looker

	// not used but needed to meet the hcat.Watcherer interface
	Complete(hcat.Notifier) bool
	Recaller(hcat.Notifier) hcat.Recaller
}
