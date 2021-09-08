package notifier

import (
	"testing"

	"github.com/hashicorp/consul-terraform-sync/logging"
	mocks "github.com/hashicorp/consul-terraform-sync/mocks/templates"
	"github.com/hashicorp/hcat/dep"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func Test_ConsulKV_Notify(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		dep      interface{}
		expected bool
	}{
		{
			"don't notify: other type of change",
			[]*dep.HealthService{},
			false,
		},
		{
			"notify: key value",
			dep.KvValue("value"),
			true,
		},
		{
			"notify: key value exists",
			dep.KVExists(true),
			true,
		},
		{
			"notify: list of key pairs",
			[]*dep.KeyPair{{Key: "key", Value: "value"}},
			true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tmpl := new(mocks.Template)
			tmpl.On("Notify", mock.Anything).Return(true)

			n := ConsulKV{Template: tmpl, once: true, logger: logging.NewNullLogger()}
			actual := n.Notify(tc.dep)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func Test_ConsulKV_Notify_Once_Mode_Key_Pairs(t *testing.T) {
	t.Run("services-last", func(t *testing.T) {
		// Test that notifier notifies at the end of once-mode, particularly
		// for the race-condition when the services dependency (which normally
		// does not notify) is received after consul-kv dependency.

		// Notifier has 3 dependencies: 2 services and 1 consul-kv
		// 1. receive first services dependency, no notification
		// 2. receive consul-kv dependency, notify for consul-kv
		// 3. receive second services dependency, notify for once-mode

		tmpl := new(mocks.Template)
		tmpl.On("Notify", mock.Anything).Return(true).Twice()
		n := NewConsulKV(tmpl, 2)

		// 1. first services dependency does not notify
		notify := n.Notify([]*dep.HealthService{})
		assert.False(t, notify, "first services dep should not have notified")
		assert.False(t, n.once, "got 1/3 deps. once-mode should not be completed")
		assert.Equal(t, 1, n.counter, "first services dep should be 1st dep")

		// 2. consul-kv notifies
		notify = n.Notify([]*dep.KeyPair{{Key: "key", Value: "value"}})
		assert.True(t, notify, "consul-kv dep should have notified")
		assert.False(t, n.once, "got 2/3 deps. once-mode should not be completed")
		assert.Equal(t, 2, n.counter, "consul-kv dep should be 2nd dep")

		// 3. second services notifies
		notify = n.Notify([]*dep.HealthService{})
		assert.True(t, notify, "second services dep should have notified")
		assert.True(t, n.once, "got 3/3 deps. once-mode should be completed")
		assert.Equal(t, 3, n.counter, "second services should be 3rd dep")

		// check mock template was called twice
		tmpl.AssertExpectations(t)
	})

	t.Run("consul-kv-last", func(t *testing.T) {
		// Test that notifier notifies at the end of once-mode, particularly for
		// the case when the consul-kv dependency is received last.

		// Notifier in test has 2 dependencies: 1 services and 1 consul-kv
		// 1. receive services dependency, no notification
		// 2. receive consul-kv dependency, notify

		tmpl := new(mocks.Template)
		tmpl.On("Notify", mock.Anything).Return(true).Once()
		n := NewConsulKV(tmpl, 1)

		// 1. services dependency does not notify
		notify := n.Notify([]*dep.HealthService{})
		assert.False(t, notify, "services dep should not have notified")
		assert.False(t, n.once, "got 1/2 deps. once-mode should not be completed")
		assert.Equal(t, 1, n.counter, "services dep should be 1st dep")

		// 2. consul-kv notifies
		notify = n.Notify([]*dep.KeyPair{{Key: "key", Value: "value"}})
		assert.True(t, notify, "consul-kv dep should have notified")
		assert.True(t, n.once, "got 2/2 deps. once-mode should be completed")
		assert.Equal(t, 2, n.counter, "consul-kv dep should be 2nd dep")

		// check mock template was called once
		tmpl.AssertExpectations(t)
	})
}

func Test_ConsulKV_Notify_Once_Mode_Single_Key(t *testing.T) {

	t.Run("services-last-with-existing-key", func(t *testing.T) {
		// Test that notifier notifies at the end of once-mode, particularly
		// for the race-condition when the services dependency (which normally
		// does not notify) is received after consul-kv dependency.

		// Notifier has 4 dependencies: 2 services, 1 Consul KV exists, 1 Consul KV value
		// 1. receive first services dependency, no notification
		// 2. receive Consul KV exists (true) dependency, notify for consul-kv
		// 3. receive Consul KV value dependency, notify for consul-kv
		// 4. receive second services dependency, notify for once-mode

		tmpl := new(mocks.Template)
		tmpl.On("Notify", mock.Anything).Return(true).Times(3)
		n := NewConsulKV(tmpl, 2)

		// 1. first services dependency does not notify
		notify := n.Notify([]*dep.HealthService{})
		assert.False(t, notify, "first services dep should not have notified")
		assert.False(t, n.once, "got 1/4 deps. once-mode should not be completed")
		assert.Equal(t, 1, n.counter, "first services dep should be 1st dep")

		// 2. consul-kv exists notifies
		notify = n.Notify(dep.KVExists(true))
		assert.True(t, notify, "consul-kv dep should have notified")
		assert.False(t, n.once, "got 2/4 deps. once-mode should not be completed")
		assert.Equal(t, 2, n.counter, "consul-kv dep should be 2nd dep")

		// 3. consul-kv value change notifies
		notify = n.Notify(dep.KvValue("test"))
		assert.True(t, notify, "consul-kv dep should have notified")
		assert.False(t, n.once, "got 3/4 deps. once-mode should not be completed")
		assert.Equal(t, 3, n.counter, "consul-kv dep should be 3rd dep")

		// 4. second services notifies
		notify = n.Notify([]*dep.HealthService{})
		assert.True(t, notify, "second services dep should have notified")
		assert.True(t, n.once, "got 4/4 deps. once-mode should be completed")
		assert.Equal(t, 4, n.counter, "second services should be 4th dep")

		// check mock template was called three times
		tmpl.AssertExpectations(t)
	})

	t.Run("services-last-with-nonexistent-key", func(t *testing.T) {
		// Test that notifier notifies at the end of once-mode, particularly
		// for the race-condition when the services dependency (which normally
		// does not notify) is received after consul-kv dependency.

		// Notifier has 3 dependencies: 2 services, 1 Consul KV exists
		// 1. receive first services dependency, no notification
		// 2. receive Consul KV exists (false) dependency, notify for consul-kv
		// 3. receive second services dependency, notify for once-mode

		tmpl := new(mocks.Template)
		tmpl.On("Notify", mock.Anything).Return(true).Times(2)
		n := NewConsulKV(tmpl, 2)

		// 1. first services dependency does not notify
		notify := n.Notify([]*dep.HealthService{})
		assert.False(t, notify, "first services dep should not have notified")
		assert.False(t, n.once, "got 1/3 deps. once-mode should not be completed")
		assert.Equal(t, 1, n.counter, "first services dep should be 1st dep")

		// 2. consul-kv exists notifies
		notify = n.Notify(dep.KVExists(false))
		assert.True(t, notify, "consul-kv dep should have notified")
		assert.False(t, n.once, "got 2/3 deps. once-mode should not be completed")
		assert.Equal(t, 2, n.counter, "consul-kv dep should be 2nd dep")

		// 3. second services notifies
		notify = n.Notify([]*dep.HealthService{})
		assert.True(t, notify, "second services dep should have notified")
		assert.True(t, n.once, "got 3/3 deps. once-mode should be completed")
		assert.Equal(t, 3, n.counter, "second services should be 3rd dep")

		// check mock template was called twice
		tmpl.AssertExpectations(t)
	})

	t.Run("consul-kv-last-with-existing-key", func(t *testing.T) {
		// Test that notifier notifies at the end of once-mode, particularly for
		// the case when the consul-kv dependency is received last.

		// Notifier in test has 3 dependencies: 1 services, 1 Consul KV exists, 1 Consul KV value
		// 1. receive services dependency, no notification
		// 2. receive consul-kv exists (true) dependency, notify
		// 3. receive consul-kv value dependency, notify

		tmpl := new(mocks.Template)
		tmpl.On("Notify", mock.Anything).Return(true).Twice()
		n := NewConsulKV(tmpl, 1)

		// 1. services dependency does not notify
		notify := n.Notify([]*dep.HealthService{})
		assert.False(t, notify, "services dep should not have notified")
		assert.False(t, n.once, "got 1/3 deps. once-mode should not be completed")
		assert.Equal(t, 1, n.counter, "services dep should be 1st dep")

		// 2. consul-kv exists notifies
		notify = n.Notify(dep.KVExists(true))
		assert.True(t, notify, "consul-kv dep should have notified")
		assert.False(t, n.once, "got 2/3 deps. once-mode should not be completed")
		assert.Equal(t, 2, n.counter, "consul-kv dep should be 2nd dep")

		notify = n.Notify(dep.KvValue("test"))
		assert.True(t, notify, "consul-kv dep should have notified")
		assert.True(t, n.once, "got 3/3 deps. once-mode should be completed")
		assert.Equal(t, 3, n.counter, "consul-kv dep should be 3rd dep")

		// check mock template was called once
		tmpl.AssertExpectations(t)
	})

	t.Run("consul-kv-last-with-nonexistent-key", func(t *testing.T) {
		// Test that notifier notifies at the end of once-mode, particularly for
		// the case when the consul-kv dependency is received last.

		// Notifier in test has 2 dependencies: 1 services and 1 Consul KV exists
		// 1. receive services dependency, no notification
		// 2. receive consul-kv exists (false) dependency, notify

		tmpl := new(mocks.Template)
		tmpl.On("Notify", mock.Anything).Return(true).Once()
		n := NewConsulKV(tmpl, 1)

		// 1. services dependency does not notify
		notify := n.Notify([]*dep.HealthService{})
		assert.False(t, notify, "services dep should not have notified")
		assert.False(t, n.once, "got 1/2 deps. once-mode should not be completed")
		assert.Equal(t, 1, n.counter, "services dep should be 1st dep")

		// 2. consul-kv exists notifies
		notify = n.Notify(dep.KVExists(false))
		assert.True(t, notify, "consul-kv dep should have notified")
		assert.True(t, n.once, "got 2/2 deps. once-mode should be completed")
		assert.Equal(t, 2, n.counter, "consul-kv dep should be 2nd dep")

		// check mock template was called once
		tmpl.AssertExpectations(t)
	})
}
