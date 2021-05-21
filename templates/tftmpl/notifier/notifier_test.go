package notifier

import (
	"testing"

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

			n := CatalogServicesRegistration{Template: tmpl, once: true}
			actual := n.Notify(tc.dep)
			assert.Equal(t, tc.expected, actual)
		})
	}
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
			"change: once-mode",
			&CatalogServicesRegistration{
				once:     false,
				services: []string{},
			},
			[]*dep.CatalogSnippet{},
			true,
		},
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
