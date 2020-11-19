package templates

import (
	"context"
	"time"

	"github.com/hashicorp/hcat"
	"github.com/hashicorp/hcat/dep"
)

//go:generate mockery --name=template  --structname=Template --output=../mocks/controller
//go:generate mockery --name=resolver --structname=Resolver --output=../mocks/controller
//go:generate mockery --name=watcher --structname=Watcher --output=../mocks/controller

var _ hcat.Templater = (Template)(nil)
var _ hcat.Renderer = (Template)(nil)
var _ Watcher = (*hcat.Watcher)(nil)

// Template describes the interface for hashicat's Template structure
// which implements the interfaces Templater and Renderer
// https://github.com/hashicorp/hcat
type Template interface {
	Render(content []byte) (hcat.RenderResult, error)
	Execute(hcat.Recaller) (*hcat.ExecuteResult, error)
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
	WaitCh(ctx context.Context) <-chan error
	Add(d dep.Dependency) bool
	Changed(tmplID string) bool
	Buffer(tmplID string) bool
	Recall(id string) (interface{}, bool)
	Register(tmplID string, deps ...dep.Dependency)
	SetBufferPeriod(min, max time.Duration, tmplIDs ...string)
	Stop()
}
