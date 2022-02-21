package tftmpl

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServicesTemplate_concatServiceTemplates(t *testing.T) {
	testcases := []struct {
		name string
		tmpl *ServicesTemplate
		exp  string
	}{
		{
			"one name & fully configured & render var",
			&ServicesTemplate{
				Names:      []string{"api"},
				Datacenter: "dc1",
				Namespace:  "ns1",
				Filter:     "filter",
				RenderVar:  true,
			},
			`
{{- with $srv := service "api" "dc=dc1" "ns=ns1" "filter" }}
  {{- range $s := $srv}}
  "{{ joinStrings "." .ID .Node .Namespace .NodeDatacenter }}" = {
{{ HCLService $s | indent 4 }}
  },
  {{- end}}
{{- end}}
`,
		},
		{
			"multi-name & fully configured & render var",
			&ServicesTemplate{
				Names:      []string{"api", "web"},
				Datacenter: "dc1",
				Namespace:  "ns1",
				Filter:     "filter",
				RenderVar:  true,
			},
			`
{{- with $srv := service "api" "dc=dc1" "ns=ns1" "filter" }}
  {{- range $s := $srv}}
  "{{ joinStrings "." .ID .Node .Namespace .NodeDatacenter }}" = {
{{ HCLService $s | indent 4 }}
  },
  {{- end}}
{{- end}}
{{- with $srv := service "web" "dc=dc1" "ns=ns1" "filter" }}
  {{- range $s := $srv}}
  "{{ joinStrings "." .ID .Node .Namespace .NodeDatacenter }}" = {
{{ HCLService $s | indent 4 }}
  },
  {{- end}}
{{- end}}
`},
		{
			"deprecated service fully configure & render var",
			&ServicesTemplate{
				Names: []string{"api", "web"},
				Services: map[string]Service{
					"api": {
						Datacenter: "dc1",
						Namespace:  "ns1",
						Filter:     "filter",
					},
					"web": {
						Datacenter: "dc2",
						Namespace:  "ns2",
						Filter:     "filter2",
					},
				},
				RenderVar: true,
			},
			`
{{- with $srv := service "api" "dc=dc1" "ns=ns1" "filter" }}
  {{- range $s := $srv}}
  "{{ joinStrings "." .ID .Node .Namespace .NodeDatacenter }}" = {
{{ HCLService $s | indent 4 }}
  },
  {{- end}}
{{- end}}
{{- with $srv := service "web" "dc=dc2" "ns=ns2" "filter2" }}
  {{- range $s := $srv}}
  "{{ joinStrings "." .ID .Node .Namespace .NodeDatacenter }}" = {
{{ HCLService $s | indent 4 }}
  },
  {{- end}}
{{- end}}
`,
		},
		{
			"deprecated service some services configured & render var",
			&ServicesTemplate{
				Names: []string{"api", "web"},
				Services: map[string]Service{
					"api": {
						Datacenter: "dc1",
						Namespace:  "ns1",
						Filter:     "filter",
					},
				},
				RenderVar: true,
			},
			`
{{- with $srv := service "api" "dc=dc1" "ns=ns1" "filter" }}
  {{- range $s := $srv}}
  "{{ joinStrings "." .ID .Node .Namespace .NodeDatacenter }}" = {
{{ HCLService $s | indent 4 }}
  },
  {{- end}}
{{- end}}
{{- with $srv := service "web" }}
  {{- range $s := $srv}}
  "{{ joinStrings "." .ID .Node .Namespace .NodeDatacenter }}" = {
{{ HCLService $s | indent 4 }}
  },
  {{- end}}
{{- end}}
`,
		},
		{
			"multi-name & fully configured & no var",
			&ServicesTemplate{
				Names:      []string{"api", "web"},
				Datacenter: "dc1",
				Namespace:  "ns1",
				Filter:     "filter",
				RenderVar:  false,
			},
			`
{{- with $srv := service "api" "dc=dc1" "ns=ns1" "filter" }}
  {{- range $s := $srv}}
  {{- /* Empty template. Detects changes in Services */ -}}
  {{- end}}
{{- end}}
{{- with $srv := service "web" "dc=dc1" "ns=ns1" "filter" }}
  {{- range $s := $srv}}
  {{- /* Empty template. Detects changes in Services */ -}}
  {{- end}}
{{- end}}
`,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := tc.tmpl.concatServiceTemplates()
			require.NoError(t, err)
			assert.Equal(t, tc.exp, actual)
		})
	}
}

func TestServicesTemplate_concatServiceTemplates_error(t *testing.T) {
	t.Parallel()
	// handles test cases when concatServiceTemplates() errors

	testcases := []struct {
		name string
		tmpl *ServicesTemplate
	}{
		{
			"datacenter & services configured",
			&ServicesTemplate{
				Names:      []string{"api"},
				Datacenter: "dc1",
				Services:   map[string]Service{},
			},
		},
		{
			"namespace & services configured",
			&ServicesTemplate{
				Names:     []string{"api"},
				Namespace: "ns1",
				Services:  map[string]Service{},
			},
		},
		{
			"filter & services configured",
			&ServicesTemplate{
				Names:    []string{"api"},
				Filter:   "filter",
				Services: map[string]Service{},
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := tc.tmpl.concatServiceTemplates()
			require.Error(t, err)
		})
	}
}

func TestServicesTemplate_appendTemplate(t *testing.T) {
	testcases := []struct {
		name     string
		tmpl     *ServicesTemplate
		expError bool
		exp      string
	}{
		{
			name: "error",
			tmpl: &ServicesTemplate{
				Names:      []string{"api"},
				Datacenter: "dc1",
				Services: map[string]Service{
					"api": {Datacenter: "dc1"},
				},
			},
			expError: true,
		},
		{
			name: "happy path",
			tmpl: &ServicesTemplate{
				Names:     []string{"api"},
				RenderVar: true,
			},
			exp: `
services = {
{{- with $srv := service "api" }}
  {{- range $s := $srv}}
  "{{ joinStrings "." .ID .Node .Namespace .NodeDatacenter }}" = {
{{ HCLService $s | indent 4 }}
  },
  {{- end}}
{{- end}}
}
`,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			w := new(strings.Builder)
			err := tc.tmpl.appendTemplate(w)
			if tc.expError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.exp, w.String())
		})
	}
}

func TestServicesTemplate_hcatQuery(t *testing.T) {
	testCases := []struct {
		name string

		serviceName string
		dc          string
		ns          string
		filter      string

		expected string
	}{
		{
			name:     "empty",
			expected: `""`,
		},
		{
			name:        "base",
			serviceName: "app",
			expected:    `"app"`,
		},
		{
			name:        "datacenter",
			serviceName: "app",
			dc:          "dc1",
			expected:    `"app" "dc=dc1"`,
		},
		{
			name:        "namespace",
			serviceName: "app",
			ns:          "namespace",
			expected:    `"app" "ns=namespace"`,
		},
		{
			name:        "filter",
			serviceName: "filtered-app",
			filter:      `"test" in Service.Tags or Service.Tags is empty`,
			expected:    `"filtered-app" "\"test\" in Service.Tags or Service.Tags is empty"`,
		},
		{
			name:        "all",
			serviceName: "app",
			dc:          "dc1",
			ns:          "namespace",
			filter:      `Service.Meta["meta-key"] contains "test"`,
			expected:    `"app" "dc=dc1" "ns=namespace" "Service.Meta[\"meta-key\"] contains \"test\""`,
		},
	}
	for _, tc := range testCases {
		tmpl := ServicesTemplate{}
		actual := tmpl.hcatQuery(tc.serviceName, tc.dc, tc.ns, tc.filter)
		assert.Equal(t, tc.expected, actual)
	}
}
