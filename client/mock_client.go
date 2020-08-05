package client

import (
	"context"
	"fmt"
)

var _ Client = (*MockClient)(nil)

// MockClient is a mock implementation of Client for testing purposes
type MockClient struct {
	InitFunc  func() error
	ApplyFunc func() error
	PlanFunc  func() error
}

// NewMockClient configures and initializes a new mock client
func NewMockClient() *MockClient {
	return &MockClient{
		InitFunc:  func() error { return nil },
		ApplyFunc: func() error { return nil },
		PlanFunc:  func() error { return nil },
	}
}

// Init mocks init for testing
func (m *MockClient) Init(ctx context.Context) error {
	return m.InitFunc()
}

// Apply mocks apply for testing
func (m *MockClient) Apply(ctx context.Context) error {
	return m.ApplyFunc()
}

// Plan mocks plan for testing
func (m *MockClient) Plan(ctx context.Context) error {
	return m.PlanFunc()
}

// GoString defines the printable version of this struct
func (m *MockClient) GoString() string {
	if m == nil {
		return "(*MockClient)(nil)"
	}

	return fmt.Sprintf("&MockClient{}")
}
