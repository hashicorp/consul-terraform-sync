package tftmpl

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntentionsTemplate_appendTemplate(t *testing.T) {
	testcases := []struct {
		name string
		c    *IntentionsTemplate
		exp  string
	}{
		{
			"fully configured & no var",
			&IntentionsTemplate{
				Datacenter: "dc1",
				Namespace:  "test-ns",
				SourceServices: &intentionServices{
					Names: []string{"api", "^web.*"},
				},
				DestinationServices: &intentionServices{
					Names: []string{"api2", "^web2.*"},
				},
			},
			`
{{- with $srv := servicesRegex "regexp=.*" "dc=dc1" "ns=ns1" "filter" }}
  {{- range $s := $srv}}
  {{- /* Empty template. Detects changes in Services */ -}}
  {{- end}}
{{- end}}
`,
		},
		{
			"source and destination only",
			&IntentionsTemplate{
				SourceServices: &intentionServices{
					Regexp: "^web.*",
				},
				DestinationServices: &intentionServices{
					Regexp: "^api.*",
				},
			},
			`
{{- with $srv := servicesRegex "regexp=" }}
  {{- range $s := $srv}}
  {{- /* Empty template. Detects changes in Services */ -}}
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

func TestIntentionsVTemplate_hcatQuery(t *testing.T) {
	testcase := []struct {
		name string
		c    *IntentionsTemplate
		exp  string
	}{
		{
			"all parameters",
			&IntentionsTemplate{
				Datacenter: "dc1",
				Namespace:  "test-ns",
				SourceServices: &intentionServices{
					Regexp: "^web.*",
				},
				DestinationServices: &intentionServices{
					Regexp: "^api.*",
				},
			},
			`"dc=dc1" "ns=test-ns source"`,
		},
		{
			"source and destination only",
			&IntentionsTemplate{
				SourceServices: &intentionServices{
					Regexp: "^web.*",
				},
				DestinationServices: &intentionServices{
					Regexp: "^api.*",
				},
			},
			`"regexp=.*"`,
		},
	}

	for _, tc := range testcase {
		t.Run(tc.name, func(t *testing.T) {
			actual := tc.c.hcatQuery()
			assert.Equal(t, tc.exp, actual)
		})
	}

}
