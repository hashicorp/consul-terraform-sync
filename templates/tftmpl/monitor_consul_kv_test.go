package tftmpl

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConsulKVMonitor_hcatQuery(t *testing.T) {
	testcase := []struct {
		name string
		c    *ConsulKVMonitor
		exp  string
	}{
		{
			"path only",
			&ConsulKVMonitor{
				Path: "key-path",
			},
			"\"key-path\"",
		},
		{
			"all_parameters",
			&ConsulKVMonitor{
				Path:       "key-path",
				Datacenter: "dc2",
				Namespace:  "test-ns",
			},
			"\"key-path\" \"dc=dc2\" \"ns=test-ns\"",
		},
	}

	for _, tc := range testcase {
		t.Run(tc.name, func(t *testing.T) {
			actual := tc.c.hcatQuery()
			assert.Equal(t, tc.exp, actual)
		})
	}
}

func TestConsulKVMonitor_appendTemplate(t *testing.T) {
	testcases := []struct {
		name string
		c    *ConsulKVMonitor
		exp  string
	}{
		{
			"recurse false & includes_var true",
			&ConsulKVMonitor{
				Path:              "path",
				Recurse:           false,
				Datacenter:        "dc1",
				Namespace:         "test-ns",
				SourceIncludesVar: true,
			},
			`
consul_kv = {
{{- with $kv := keyExistsGet "path" "dc=dc1" "ns=test-ns" }}
  {{- if .Exists }}
  "{{ .Path }}" = "{{ .Value }}"
  {{- end}}
{{- end}}
}
`,
		},
		{
			"recurse true & includes_var true",
			&ConsulKVMonitor{
				Path:              "path",
				Recurse:           true,
				Datacenter:        "dc1",
				Namespace:         "test-ns",
				SourceIncludesVar: true,
			},
			`
consul_kv = {
{{- with $kv := keys "path" "dc=dc1" "ns=test-ns" }}
  {{- range $k := $kv }}
  "{{ .Path }}" = "{{ .Value }}"
  {{- end}}
{{- end}}
}
`,
		},
		{
			"recurse false & includes_var false",
			&ConsulKVMonitor{
				Path:              "path",
				Recurse:           false,
				Datacenter:        "dc1",
				Namespace:         "test-ns",
				SourceIncludesVar: false,
			},
			`
{{- with $kv := keyExistsGet "path" "dc=dc1" "ns=test-ns" }}
  {{- /* Empty template. Detects changes in Consul KV */ -}}
{{- end}}
`,
		},
		{
			"recurse true includes var false",
			&ConsulKVMonitor{
				Path:              "path",
				Recurse:           true,
				Datacenter:        "dc1",
				Namespace:         "test-ns",
				SourceIncludesVar: false,
			},
			`
{{- with $kv := keys "path" "dc=dc1" "ns=test-ns" }}
  {{- range $k, $v := $kv }}
  {{- /* Empty template. Detects changes in Consul KV */ -}}
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
