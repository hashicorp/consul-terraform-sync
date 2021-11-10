package tftmpl

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServicesCondition_appendTemplate(t *testing.T) {
	testcases := []struct {
		name string
		c    *ServicesCondition
		exp  string
	}{
		{
			"source_includes_var=true",
			&ServicesCondition{
				SourceIncludesVar: true,
				ServicesMonitor: ServicesMonitor{
					Regexp:     ".*",
					Datacenter: "dc1",
					Namespace:  "ns1",
					Filter:     "filter",
				},
			},
			`
services = {
{{- with $srv := servicesRegex "regexp=.*" "dc=dc1" "ns=ns1" "filter" }}
  {{- range $s := $srv}}
  "{{ joinStrings "." .ID .Node .Namespace .NodeDatacenter }}" = {
{{ HCLService $s | indent 4 }}
  },
  {{- end}}
{{- end}}
}
`,
		},
		{
			"source_includes_var=false, unconfigured",
			&ServicesCondition{
				SourceIncludesVar: false,
				ServicesMonitor:   ServicesMonitor{},
			},
			"",
		},
		{
			"source_includes_var=false, configured",
			&ServicesCondition{
				SourceIncludesVar: false,
				ServicesMonitor: ServicesMonitor{
					Regexp:     ".*",
					Datacenter: "dc1",
					Namespace:  "ns1",
					Filter:     "filter",
				},
			},
			`
{{- with $srv := servicesRegex "regexp=.*" "dc=dc1" "ns=ns1" "filter" }}
  {{- range $s := $srv}}
  "{{ joinStrings "." .ID .Node .Namespace .NodeDatacenter }}" = {
{{ HCLService $s | indent 4 }}
  },
  {{- end}}
{{- end}}
`,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			w := new(strings.Builder)
			err := tc.c.appendTemplate(w)
			require.NoError(t, err)
			assert.Equal(t, tc.exp, w.String())
		})
	}
}
