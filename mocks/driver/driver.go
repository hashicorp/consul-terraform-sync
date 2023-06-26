// Code generated by mockery v2.30.16. DO NOT EDIT.

package mocks

import (
	context "context"

	driver "github.com/hashicorp/consul-terraform-sync/driver"
	mock "github.com/stretchr/testify/mock"
)

// Driver is an autogenerated mock type for the Driver type
type Driver struct {
	mock.Mock
}

// ApplyTask provides a mock function with given fields: ctx
func (_m *Driver) ApplyTask(ctx context.Context) error {
	ret := _m.Called(ctx)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context) error); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DestroyTask provides a mock function with given fields: ctx
func (_m *Driver) DestroyTask(ctx context.Context) {
	_m.Called(ctx)
}

// InitTask provides a mock function with given fields: ctx
func (_m *Driver) InitTask(ctx context.Context) error {
	ret := _m.Called(ctx)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context) error); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// InspectTask provides a mock function with given fields: ctx
func (_m *Driver) InspectTask(ctx context.Context) (driver.InspectPlan, error) {
	ret := _m.Called(ctx)

	var r0 driver.InspectPlan
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context) (driver.InspectPlan, error)); ok {
		return rf(ctx)
	}
	if rf, ok := ret.Get(0).(func(context.Context) driver.InspectPlan); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Get(0).(driver.InspectPlan)
	}

	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// RenderTemplate provides a mock function with given fields: ctx
func (_m *Driver) RenderTemplate(ctx context.Context) (bool, error) {
	ret := _m.Called(ctx)

	var r0 bool
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context) (bool, error)); ok {
		return rf(ctx)
	}
	if rf, ok := ret.Get(0).(func(context.Context) bool); ok {
		r0 = rf(ctx)
	} else {
		r0 = ret.Get(0).(bool)
	}

	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// SetBufferPeriod provides a mock function with given fields:
func (_m *Driver) SetBufferPeriod() {
	_m.Called()
}

// Task provides a mock function with given fields:
func (_m *Driver) Task() *driver.Task {
	ret := _m.Called()

	var r0 *driver.Task
	if rf, ok := ret.Get(0).(func() *driver.Task); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*driver.Task)
		}
	}

	return r0
}

// TemplateIDs provides a mock function with given fields:
func (_m *Driver) TemplateIDs() []string {
	ret := _m.Called()

	var r0 []string
	if rf, ok := ret.Get(0).(func() []string); ok {
		r0 = rf()
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]string)
		}
	}

	return r0
}

// UpdateTask provides a mock function with given fields: ctx, task
func (_m *Driver) UpdateTask(ctx context.Context, task driver.PatchTask) (driver.InspectPlan, error) {
	ret := _m.Called(ctx, task)

	var r0 driver.InspectPlan
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, driver.PatchTask) (driver.InspectPlan, error)); ok {
		return rf(ctx, task)
	}
	if rf, ok := ret.Get(0).(func(context.Context, driver.PatchTask) driver.InspectPlan); ok {
		r0 = rf(ctx, task)
	} else {
		r0 = ret.Get(0).(driver.InspectPlan)
	}

	if rf, ok := ret.Get(1).(func(context.Context, driver.PatchTask) error); ok {
		r1 = rf(ctx, task)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// Version provides a mock function with given fields:
func (_m *Driver) Version() string {
	ret := _m.Called()

	var r0 string
	if rf, ok := ret.Get(0).(func() string); ok {
		r0 = rf()
	} else {
		r0 = ret.Get(0).(string)
	}

	return r0
}

// NewDriver creates a new instance of Driver. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewDriver(t interface {
	mock.TestingT
	Cleanup(func())
}) *Driver {
	mock := &Driver{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
