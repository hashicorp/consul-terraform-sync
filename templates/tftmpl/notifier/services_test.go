package notifier

import (
	"testing"

	"github.com/hashicorp/consul-terraform-sync/logging"
	mocks "github.com/hashicorp/consul-terraform-sync/mocks/templates"
	"github.com/hashicorp/hcat/dep"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func Test_Services_Notify_DaemonMode(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		dep    interface{}
		notify bool
	}{
		{
			"notify: services dep",
			[]*dep.HealthService{},
			true,
		},
		{
			"don't notify: non-services dep",
			&dep.KeyPair{Key: "k", Value: "v"},
			false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tmpl := new(mocks.Template)
			tmpl.On("Notify", mock.Anything).Return(true)

			n := Services{Template: tmpl, once: true, logger: logging.NewNullLogger()}
			actual := n.Notify(tc.dep)
			assert.Equal(t, tc.notify, actual)
		})
	}
}

func Test_Services_Notify_OnceMode(t *testing.T) {
	t.Parallel()

	t.Run("services-not-last", func(t *testing.T) {
		// Test that notifier notifies at the end of once-mode, particularly
		// for the race-condition when a non-service dependency (which normally
		// does not notify) is received after service dependency.

		// Notifier has 2 dependencies: 1 service and 1 consul-kv
		// 1. receive services dependency, notify for service change
		// 2. receive consul-kv dependency, notify for once-mode

		tmpl := new(mocks.Template)
		tmpl.On("Notify", mock.Anything).Return(true)
		n := NewServices(tmpl, 2)

		// 1. service dep notifies
		notify := n.Notify([]*dep.HealthService{})
		assert.True(t, notify, "services dep should cause notification")
		assert.False(t, n.once, "got 1/2 deps. once-mode should not be completed")
		assert.Equal(t, 1, n.counter, "services dep should be 1st dep")

		// 2. consul-kv dep notifies because once-mode (leads to hanging if it doesn't)
		notify = n.Notify([]*dep.KeyPair{})
		assert.True(t, notify, "consul-kv dep should cause notification")
		assert.True(t, n.once, "got 2/2 deps. once-mode should be completed")
		assert.Equal(t, 2, n.counter, "consul-kv dep should be 2nd dep")

		// 3. confirm that future consul-kv dep never notify
		notify = n.Notify([]*dep.KeyPair{})
		assert.False(t, notify, "second consul-kv dep should not have notified")
		notify = n.Notify([]*dep.KeyPair{})
		assert.False(t, notify, "third consul-kv dep should not have notified")
	})

	t.Run("services-last", func(t *testing.T) {
		// Test that notifier notifies at the end of once-mode

		// Notifier has 2 dependencies: 1 service and 1 consul-kv
		// 1. receive consul-kv dependency, don't notify
		// 2. receive services dependency, notify for once-mode / service change

		tmpl := new(mocks.Template)
		tmpl.On("Notify", mock.Anything).Return(true)
		n := NewServices(tmpl, 2)

		// 1. consul-kv dep does not notify
		notify := n.Notify([]*dep.KeyPair{})
		assert.False(t, notify, "consul-kv dep should not cause notification")
		assert.False(t, n.once, "got 1/2 deps. once-mode should not be completed")
		assert.Equal(t, 1, n.counter, "consul-kv dep should be 1st dep")

		// 2. service dep notifies
		notify = n.Notify([]*dep.HealthService{})
		assert.True(t, notify, "services dep should cause notification")
		assert.True(t, n.once, "got 2/2 deps. once-mode should be completed")
		assert.Equal(t, 2, n.counter, "services dep should be 2nd dep")
	})
}
