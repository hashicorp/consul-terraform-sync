package controller

import (
	"time"

	"github.com/hashicorp/hcat"
	"github.com/hashicorp/hcat/dep"
)

var _ template = (*mockTemplate)(nil)

// mockTemplate is a mock implementation of hcat's Template for testing purposes
type mockTemplate struct {
	RenderFunc func([]byte) (hcat.RenderResult, error)
}

// newMockTemplate configures and initializes a new mock hashicat Template
func newMockTemplate() *mockTemplate {
	return &mockTemplate{
		RenderFunc: func([]byte) (hcat.RenderResult, error) {
			return hcat.RenderResult{}, nil
		},
	}
}

// Render mocks render for testing
func (m *mockTemplate) Render(content []byte) (hcat.RenderResult, error) {
	return m.RenderFunc(content)
}

// Execute mocks execute for testing
// Note: function not directly consumed by NIA, therefore not fully mocked out at this time
func (m *mockTemplate) Execute(hcat.Recaller) (*hcat.ExecuteResult, error) {
	return nil, nil
}

// ID mocks ID for testing
// Note: function not directly consumed by NIA, therefore not fully mocked out at this time
func (m *mockTemplate) ID() string {
	return "id"
}

var _ resolver = (*mockResolver)(nil)

// mockResolver is a mock implementation of hcat's Resolver for testing purposes
type mockResolver struct {
	RunFunc func(hcat.Templater, hcat.Watcherer) (hcat.ResolveEvent, error)
}

// newMockResolver configures and initializes a new mock hashicat Resolver
func newMockResolver() *mockResolver {
	return &mockResolver{
		RunFunc: func(hcat.Templater, hcat.Watcherer) (hcat.ResolveEvent, error) {
			return hcat.ResolveEvent{
				Complete: true,
			}, nil
		},
	}
}

// Run mocks run for testing
func (m *mockResolver) Run(tmpl hcat.Templater, w hcat.Watcherer) (hcat.ResolveEvent, error) {
	return m.RunFunc(tmpl, w)
}

var _ watcher = (*mockWatcher)(nil)

// mockWatcher is a mock implementation of Hcat's Watcher for testing purposes
type mockWatcher struct {
	WaitFunc func(time.Duration) error
}

// newMockWatcher configures and initializes a new mock hashicat Watcher
func newMockWatcher() *mockWatcher {
	return &mockWatcher{
		WaitFunc: func(time.Duration) error {
			return nil
		},
	}
}

// Wait mocks wait for testing
func (m *mockWatcher) Wait(timeout time.Duration) error {
	return m.WaitFunc(timeout)
}

// Add mocks add for testing
// Note: function not directly consumed by NIA, therefore not fully mocked out at this time
func (m *mockWatcher) Add(d dep.Dependency) bool {
	return true
}

// Changed mocks changed for testing
// Note: function not directly consumed by NIA, therefore not fully mocked out at this time
func (m *mockWatcher) Changed(tmplID string) bool {
	return true
}

// Recall mocks recall for testing
func (m *mockWatcher) Recall(id string) (interface{}, bool) {
	return "", true
}

// Register mocks register for testing
// Note: function not directly consumed by NIA, therefore not fully mocked out at this time
func (m *mockWatcher) Register(tmplID string, deps ...dep.Dependency) {
	return
}
