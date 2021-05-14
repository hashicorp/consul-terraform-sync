package tmplfunc

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCatalogServicesRegistrationQuery(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		opts []string
		exp  *catalogServicesRegistrationQuery
		err  bool
	}{
		{
			"no opts",
			[]string{},
			&catalogServicesRegistrationQuery{},
			false,
		},
		{
			"regexp",
			[]string{"regexp=.*"},
			&catalogServicesRegistrationQuery{
				regexp: regexp.MustCompile(".*"),
			},
			false,
		},
		{
			"dc",
			[]string{"dc=dc1"},
			&catalogServicesRegistrationQuery{
				dc: "dc1",
			},
			false,
		},
		{
			"ns",
			[]string{"ns=namespace"},
			&catalogServicesRegistrationQuery{
				ns: "namespace",
			},
			false,
		},
		{
			"node-meta",
			[]string{"node-meta=k:v", "node-meta=foo:bar"},
			&catalogServicesRegistrationQuery{
				nodeMeta: map[string]string{"k": "v", "foo": "bar"},
			},
			false,
		},
		{
			"multiple",
			[]string{"node-meta=k:v", "ns=namespace", "dc=dc1", "regexp=.*"},
			&catalogServicesRegistrationQuery{
				regexp:   regexp.MustCompile(".*"),
				dc:       "dc1",
				ns:       "namespace",
				nodeMeta: map[string]string{"k": "v"},
			},
			false,
		},
		{
			"invalid query",
			[]string{"invalid=true"},
			nil,
			true,
		},
		{
			"invalid query format",
			[]string{"dc1"},
			nil,
			true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			act, err := newCatalogServicesRegistrationQuery(tc.opts)
			if tc.err {
				assert.Error(t, err)
				return
			}

			if act != nil {
				act.stopCh = nil
			}

			assert.NoError(t, err, err)
			assert.Equal(t, tc.exp, act)
		})
	}
}

func TestCatalogServicesRegistrationQuery_String(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		i    []string
		exp  string
	}{
		{
			"empty",
			[]string{},
			"catalog.services.registration",
		},
		{
			"regexp",
			[]string{"regexp=.*"},
			"catalog.services.registration(regexp=.*)",
		},
		{
			"datacenter",
			[]string{"dc=dc1"},
			"catalog.services.registration(@dc1)",
		},
		{
			"namespace",
			[]string{"ns=namespace"},
			"catalog.services.registration(ns=namespace)",
		},
		{
			"node-meta",
			[]string{"node-meta=k:v", "node-meta=foo:bar"},
			"catalog.services.registration(node-meta=foo:bar&node-meta=k:v)",
		},
		{
			"multiple",
			[]string{"node-meta=k:v", "dc=dc1", "ns=namespace", "regexp=.*"},
			"catalog.services.registration(@dc1&node-meta=k:v&ns=namespace&regexp=.*)",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d, err := newCatalogServicesRegistrationQuery(tc.i)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, tc.exp, d.String())
		})
	}
}
