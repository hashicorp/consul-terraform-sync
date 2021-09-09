package notifier

import (
	"testing"

	mocks "github.com/hashicorp/consul-terraform-sync/mocks/templates"
	"github.com/hashicorp/hcat/dep"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func Test_SuppressNotification_Notify(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		expected bool
	}{
		{
			"happy path",
			false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tmpl := new(mocks.Template)
			tmpl.On("Notify", mock.Anything).Return(true)

			n := NewSuppressNotification(tmpl, 2)
			actual := n.Notify([]*dep.HealthService{})
			assert.Equal(t, tc.expected, actual)
		})
	}

	t.Run("once-mode", func(t *testing.T) {
		// Test that notifier only notifies when once-mode completes

		// Notifier in test will have two dependencies
		// 1. receive 1st services dependency, no notification
		// 2. receive 2nd services dependency, notify because once-mode complete
		// 3. receive subsequent services dependencies, no more notifications
		// now that once-mode is complete

		tmpl := new(mocks.Template)
		tmpl.On("Notify", mock.Anything).Return(true)
		n := NewSuppressNotification(tmpl, 2)

		// 1. first services dependency does not notify
		notify := n.Notify([]*dep.HealthService{})
		assert.False(t, notify, "first services dep should not have notified")
		assert.False(t, n.once, "got 1/2 deps. once-mode should not be completed")
		assert.Equal(t, 1, n.counter, "first services dep should be 1st dep")

		// 2. second services dependency should notify
		notify = n.Notify([]*dep.HealthService{})
		assert.True(t, notify, "second service dep should have notified")
		assert.True(t, n.once, "got 2/2 deps. once-mode should be completed")
		assert.Equal(t, 2, n.counter, "second services dep should be 2nd dep")

		// 3. subsequent services dependency do not notify
		notify = n.Notify([]*dep.HealthService{})
		assert.False(t, notify, "third services dep should not have notified")
		notify = n.Notify([]*dep.HealthService{})
		assert.False(t, notify, "fourth services dep should not have notified")
		notify = n.Notify([]*dep.HealthService{})
		assert.False(t, notify, "fifth services dep should not have notified")
		notify = n.Notify([]*dep.HealthService{})
		assert.False(t, notify, "sixth services dep should not have notified")
	})
}
