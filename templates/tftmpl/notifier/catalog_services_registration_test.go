package notifier

import (
	"testing"

	"github.com/hashicorp/consul-terraform-sync/logging"
	mocks "github.com/hashicorp/consul-terraform-sync/mocks/templates"
	"github.com/hashicorp/hcat/dep"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func Test_CatalogServicesRegistration_Notify(t *testing.T) {
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
			"don't notify: no change in services",
			[]*dep.CatalogSnippet{},
			false,
		},
		{
			"notify: new service registration",
			[]*dep.CatalogSnippet{{Name: "api"}},
			true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tmpl := new(mocks.Template)
			tmpl.On("Notify", mock.Anything).Return(true)

			n := CatalogServicesRegistration{Template: tmpl, once: true, logger: logging.NewNullLogger()}
			actual := n.Notify(tc.dep)
			assert.Equal(t, tc.expected, actual)
		})
	}

	t.Run("once-mode-services-last", func(t *testing.T) {
		// Test that notifier notifies at the end of once-mode, particularly
		// for the race-condition when the services dependency (which normally
		// does not notify) is received after catalog-service dependency.

		// Notifier has 3 dependencies: 2 services and 1 catalog-service
		// 1. receive first services dependency, no notification
		// 2. receive catalog-service dependency, notify for catalog-service
		// 3. receive second services dependency, notify for once-mode

		tmpl := new(mocks.Template)
		tmpl.On("Notify", mock.Anything).Return(true).Twice()
		n := NewCatalogServicesRegistration(tmpl, 3)

		// 1. first services dependency does not notify
		notify := n.Notify([]*dep.HealthService{})
		assert.False(t, notify, "first services dep should not have notified")
		assert.False(t, n.once, "got 1/3 deps. once-mode should not be completed")
		assert.Equal(t, 1, n.counter, "first services dep should be 1st dep")

		// 2. catalog-service notifies
		notify = n.Notify([]*dep.CatalogSnippet{{Name: "api"}})
		assert.True(t, notify, "catalog-service dep should have notified")
		assert.False(t, n.once, "got 2/3 deps. once-mode should not be completed")
		assert.Equal(t, 2, n.counter, "catalog-service dep should be 2nd dep")
		assert.Equal(t, []string{"api"}, n.services, "api service should be stored")

		// 3. second services notifies
		notify = n.Notify([]*dep.HealthService{})
		assert.True(t, notify, "second services dep should have notified")
		assert.True(t, n.once, "got 3/3 deps. once-mode should be completed")
		assert.Equal(t, 3, n.counter, "second services should be 3rd dep")

		// check mock template was called twice
		tmpl.AssertExpectations(t)
	})

	t.Run("once-mode-catalog-services-last", func(t *testing.T) {
		// Test that notifier notifies at the end of once-mode, particularly for
		// the case when the catalog-services dependency is received last.

		// Notifier in test has 2 dependencies: 1 services and 1 catalog-service
		// 1. receive services dependency, no notification
		// 2. receive catalog-services dependency, notify

		tmpl := new(mocks.Template)
		tmpl.On("Notify", mock.Anything).Return(true).Once()
		n := NewCatalogServicesRegistration(tmpl, 2)

		// 1. services dependency does not notify
		notify := n.Notify([]*dep.HealthService{})
		assert.False(t, notify, "services dep should not have notified")
		assert.False(t, n.once, "got 1/2 deps. once-mode should not be completed")
		assert.Equal(t, 1, n.counter, "services dep should be 1st dep")

		// 2. catalog-service notifies
		notify = n.Notify([]*dep.CatalogSnippet{{Name: "api"}})
		assert.True(t, notify, "catalog-service dep should have notified")
		assert.True(t, n.once, "got 2/2 deps. once-mode should be completed")
		assert.Equal(t, 2, n.counter, "catalog-service dep should be 2nd dep")
		assert.Equal(t, []string{"api"}, n.services, "api service should be stored")

		// check mock template was called once
		tmpl.AssertExpectations(t)
	})
}

func Test_CatalogServicesRegistration_registrationChange(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		notifier *CatalogServicesRegistration
		data     []*dep.CatalogSnippet
		expected bool
	}{
		{
			"change: different number of services",
			&CatalogServicesRegistration{
				once:     true,
				services: []string{"api", "db"},
			},
			[]*dep.CatalogSnippet{{Name: "db"}},
			true,
		},
		{
			"change: same number but different services",
			&CatalogServicesRegistration{
				once:     true,
				services: []string{"api", "db"},
			},
			[]*dep.CatalogSnippet{{Name: "redis"}, {Name: "web"}},
			true,
		},
		{
			"no change",
			&CatalogServicesRegistration{
				once:     true,
				services: []string{"api", "db"},
			},
			[]*dep.CatalogSnippet{{Name: "api"}, {Name: "db"}},
			false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual := tc.notifier.registrationChange(tc.data)
			assert.Equal(t, tc.expected, actual)

			services := make([]string, len(tc.data))
			for ix, s := range tc.data {
				services[ix] = s.Name
			}
			assert.Equal(t, tc.notifier.services, services)
		})
	}
}
