// Code generated by mockery v2.53.3. DO NOT EDIT.

package mocks

import (
	context "context"

	config "github.com/hashicorp/consul-terraform-sync/config"

	event "github.com/hashicorp/consul-terraform-sync/state/event"

	mock "github.com/stretchr/testify/mock"
)

// Server is an autogenerated mock type for the Server type
type Server struct {
	mock.Mock
}

// Config provides a mock function with no fields
func (_m *Server) Config() config.Config {
	ret := _m.Called()

	if len(ret) == 0 {
		panic("no return value specified for Config")
	}

	var r0 config.Config
	if rf, ok := ret.Get(0).(func() config.Config); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(config.Config)
	}

	return r0
}

// Events provides a mock function with given fields: ctx, taskName
func (_m *Server) Events(ctx context.Context, taskName string) (map[string][]event.Event, error) {
	ret := _m.Called(ctx, taskName)

	if len(ret) == 0 {
		panic("no return value specified for Events")
	}

	var r0 map[string][]event.Event
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) (map[string][]event.Event, error)); ok {
		return rf(ctx, taskName)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) map[string][]event.Event); ok {
		r0 = rf(ctx, taskName)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(map[string][]event.Event)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, taskName)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Task provides a mock function with given fields: ctx, taskName
func (_m *Server) Task(ctx context.Context, taskName string) (config.TaskConfig, error) {
	ret := _m.Called(ctx, taskName)

	if len(ret) == 0 {
		panic("no return value specified for Task")
	}

	var r0 config.TaskConfig
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) (config.TaskConfig, error)); ok {
		return rf(ctx, taskName)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) config.TaskConfig); ok {
		r0 = rf(ctx, taskName)
	} else {
		r0 = ret.Get(0).(config.TaskConfig)
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, taskName)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// TaskCreate provides a mock function with given fields: _a0, _a1
func (_m *Server) TaskCreate(_a0 context.Context, _a1 config.TaskConfig) (config.TaskConfig, error) {
	ret := _m.Called(_a0, _a1)

	if len(ret) == 0 {
		panic("no return value specified for TaskCreate")
	}

	var r0 config.TaskConfig
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, config.TaskConfig) (config.TaskConfig, error)); ok {
		return rf(_a0, _a1)
	}
	if rf, ok := ret.Get(0).(func(context.Context, config.TaskConfig) config.TaskConfig); ok {
		r0 = rf(_a0, _a1)
	} else {
		r0 = ret.Get(0).(config.TaskConfig)
	}

	if rf, ok := ret.Get(1).(func(context.Context, config.TaskConfig) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// TaskCreateAndRun provides a mock function with given fields: _a0, _a1
func (_m *Server) TaskCreateAndRun(_a0 context.Context, _a1 config.TaskConfig) (config.TaskConfig, error) {
	ret := _m.Called(_a0, _a1)

	if len(ret) == 0 {
		panic("no return value specified for TaskCreateAndRun")
	}

	var r0 config.TaskConfig
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, config.TaskConfig) (config.TaskConfig, error)); ok {
		return rf(_a0, _a1)
	}
	if rf, ok := ret.Get(0).(func(context.Context, config.TaskConfig) config.TaskConfig); ok {
		r0 = rf(_a0, _a1)
	} else {
		r0 = ret.Get(0).(config.TaskConfig)
	}

	if rf, ok := ret.Get(1).(func(context.Context, config.TaskConfig) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// TaskDelete provides a mock function with given fields: ctx, taskName
func (_m *Server) TaskDelete(ctx context.Context, taskName string) error {
	ret := _m.Called(ctx, taskName)

	if len(ret) == 0 {
		panic("no return value specified for TaskDelete")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string) error); ok {
		r0 = rf(ctx, taskName)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// TaskInspect provides a mock function with given fields: _a0, _a1
func (_m *Server) TaskInspect(_a0 context.Context, _a1 config.TaskConfig) (bool, string, string, error) {
	ret := _m.Called(_a0, _a1)

	if len(ret) == 0 {
		panic("no return value specified for TaskInspect")
	}

	var r0 bool
	var r1 string
	var r2 string
	var r3 error
	if rf, ok := ret.Get(0).(func(context.Context, config.TaskConfig) (bool, string, string, error)); ok {
		return rf(_a0, _a1)
	}
	if rf, ok := ret.Get(0).(func(context.Context, config.TaskConfig) bool); ok {
		r0 = rf(_a0, _a1)
	} else {
		r0 = ret.Get(0).(bool)
	}

	if rf, ok := ret.Get(1).(func(context.Context, config.TaskConfig) string); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Get(1).(string)
	}

	if rf, ok := ret.Get(2).(func(context.Context, config.TaskConfig) string); ok {
		r2 = rf(_a0, _a1)
	} else {
		r2 = ret.Get(2).(string)
	}

	if rf, ok := ret.Get(3).(func(context.Context, config.TaskConfig) error); ok {
		r3 = rf(_a0, _a1)
	} else {
		r3 = ret.Error(3)
	}

	return r0, r1, r2, r3
}

// TaskUpdate provides a mock function with given fields: ctx, updateConf, runOp
func (_m *Server) TaskUpdate(ctx context.Context, updateConf config.TaskConfig, runOp string) (bool, string, string, error) {
	ret := _m.Called(ctx, updateConf, runOp)

	if len(ret) == 0 {
		panic("no return value specified for TaskUpdate")
	}

	var r0 bool
	var r1 string
	var r2 string
	var r3 error
	if rf, ok := ret.Get(0).(func(context.Context, config.TaskConfig, string) (bool, string, string, error)); ok {
		return rf(ctx, updateConf, runOp)
	}
	if rf, ok := ret.Get(0).(func(context.Context, config.TaskConfig, string) bool); ok {
		r0 = rf(ctx, updateConf, runOp)
	} else {
		r0 = ret.Get(0).(bool)
	}

	if rf, ok := ret.Get(1).(func(context.Context, config.TaskConfig, string) string); ok {
		r1 = rf(ctx, updateConf, runOp)
	} else {
		r1 = ret.Get(1).(string)
	}

	if rf, ok := ret.Get(2).(func(context.Context, config.TaskConfig, string) string); ok {
		r2 = rf(ctx, updateConf, runOp)
	} else {
		r2 = ret.Get(2).(string)
	}

	if rf, ok := ret.Get(3).(func(context.Context, config.TaskConfig, string) error); ok {
		r3 = rf(ctx, updateConf, runOp)
	} else {
		r3 = ret.Error(3)
	}

	return r0, r1, r2, r3
}

// Tasks provides a mock function with given fields: _a0
func (_m *Server) Tasks(_a0 context.Context) config.TaskConfigs {
	ret := _m.Called(_a0)

	if len(ret) == 0 {
		panic("no return value specified for Tasks")
	}

	var r0 config.TaskConfigs
	if rf, ok := ret.Get(0).(func(context.Context) config.TaskConfigs); ok {
		r0 = rf(_a0)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(config.TaskConfigs)
		}
	}

	return r0
}

// NewServer creates a new instance of Server. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewServer(t interface {
	mock.TestingT
	Cleanup(func())
}) *Server {
	mock := &Server{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
