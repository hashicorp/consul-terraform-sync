package tftmpl

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServicesRegexTemplate_appendTemplate(t *testing.T) {
	testcases := []struct {
		name string
		c    *ServicesRegexTemplate
		exp  string
	}{
		{
			"fully configured & includes_var true",
			&ServicesRegexTemplate{
				Regexp:            ".*",
				Datacenter:        "dc1",
				Namespace:         "ns1",
				Filter:            "filter",
				SourceIncludesVar: true,
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
			"fully configured & includes_var false",
			&ServicesRegexTemplate{
				Regexp:            ".*",
				Datacenter:        "dc1",
				Namespace:         "ns1",
				Filter:            "filter",
				SourceIncludesVar: false,
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
		{
			"regexp empty string",
			&ServicesRegexTemplate{
				Regexp:            "",
				SourceIncludesVar: false,
			},
			`
{{- with $srv := servicesRegex "regexp=" }}
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

func TestServicesRegexTemplate_hcatQuery(t *testing.T) {
	testcase := []struct {
		name string
		c    *ServicesRegexTemplate
		exp  string
	}{
		{
			"regexp only",
			&ServicesRegexTemplate{
				Regexp: ".*",
			},
			`"regexp=.*"`,
		},
		{
			"valid regexp empty string",
			&ServicesRegexTemplate{
				Regexp: "",
			},
			`"regexp="`,
		},
		{
			"all_parameters",
			&ServicesRegexTemplate{
				Regexp:     ".*",
				Datacenter: "datacenter",
				Namespace:  "namespace",
				Filter:     "filter",
			},
			`"regexp=.*" "dc=datacenter" "ns=namespace" "filter"`,
		},
	}

	for _, tc := range testcase {
		t.Run(tc.name, func(t *testing.T) {
			actual := tc.c.hcatQuery()
			assert.Equal(t, tc.exp, actual)
		})
	}
}
