package controller

import (
	"time"

	"github.com/hashicorp/hcat"
)

var _ hcatTemplate = (*mockHcatTemplate)(nil)

// mockHcatTemplate is a mock implementation of hcat's Template for testing purposes
type mockHcatTemplate struct {
	RenderFunc func([]byte) (hcat.RenderResult, error)
}

// newMockHcatTemplate configures and initializes a new mock hcat Template
func newMockHcatTemplate() *mockHcatTemplate {
	return &mockHcatTemplate{
		RenderFunc: func([]byte) (hcat.RenderResult, error) {
			return hcat.RenderResult{}, nil
		},
	}
}

// Render mocks render for testing
func (m *mockHcatTemplate) Render(content []byte) (hcat.RenderResult, error) {
	return m.RenderFunc(content)
}

// Execute mocks execute for testing
// Note: function not directly consumed by NIA, therefore not fully mocked out at this time
func (m *mockHcatTemplate) Execute(hcat.Recaller) (*hcat.ExecuteResult, error) {
	return nil, nil
}

// ID mocks ID for testing
// Note: function not directly consumed by NIA, therefore not fully mocked out at this time
func (m *mockHcatTemplate) ID() string {
	return "id"
}

var _ hcatResolver = (*mockHcatResolver)(nil)

// mockHcatResolver is a mock implementation of hcat's Resolver for testing purposes
type mockHcatResolver struct {
	RunFunc func(hcat.Templater, hcat.Watcherer) (hcat.ResolveEvent, error)
}

// newMockHcatResolver configures and initializes a new mock hcat Resolver
func newMockHcatResolver() *mockHcatResolver {
	return &mockHcatResolver{
		RunFunc: func(hcat.Templater, hcat.Watcherer) (hcat.ResolveEvent, error) {
			return hcat.ResolveEvent{
				Complete: true,
			}, nil
		},
	}
}

// Run mocks run for testing
func (m *mockHcatResolver) Run(tmpl hcat.Templater, w hcat.Watcherer) (hcat.ResolveEvent, error) {
	return m.RunFunc(tmpl, w)
}

var _ hcatWatcher = (*mockHcatWatcher)(nil)

// mockHcatWatcher is a mock implementation of Hcat's Watcher for testing purposes
type mockHcatWatcher struct {
	WaitFunc func(time.Duration) error
}

// newMockHcatWatcher configures and initializes a new mock hcat Watcher
func newMockHcatWatcher() *mockHcatWatcher {
	return &mockHcatWatcher{
		WaitFunc: func(time.Duration) error {
			return nil
		},
	}
}

// Wait mocks wait for testing
func (m *mockHcatWatcher) Wait(timeout time.Duration) error {
	return m.WaitFunc(timeout)
}

// Add mocks add for testing
// Note: function not directly consumed by NIA, therefore not fully mocked out at this time
func (m *mockHcatWatcher) Add(d hcat.Dependency) bool {
	return true
}

// Changed mocks changed for testing
// Note: function not directly consumed by NIA, therefore not fully mocked out at this time
func (m *mockHcatWatcher) Changed(tmplID string) bool {
	return true
}

// Recall mocks recall for testing
func (m *mockHcatWatcher) Recall(id string) (interface{}, bool) {
	return "", true
}

// Register mocks register for testing
// Note: function not directly consumed by NIA, therefore not fully mocked out at this time
func (m *mockHcatWatcher) Register(tmplID string, deps ...hcat.Dependency) {
	return
}
