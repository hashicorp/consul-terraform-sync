package tftmpl

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConsulKVCondition_hcatQuery(t *testing.T) {
	testcase := []struct {
		name string
		c    *ConsulKVCondition
		exp  string
	}{
		{
			"path only",
			&ConsulKVCondition{
				ConsulKVMonitor{
					Path: "key-path",
				},
				false,
			},
			"\"key-path\" ",
		},
		{
			"all_parameters",
			&ConsulKVCondition{
				ConsulKVMonitor{
					Path:       "key-path",
					Datacenter: "dc2",
					Namespace:  "test-ns",
				},
				false,
			},
			"\"key-path\" \"dc=dc2\" \"ns=test-ns\" ",
		},
	}

	for _, tc := range testcase {
		t.Run(tc.name, func(t *testing.T) {
			actual := tc.c.hcatQuery()
			assert.Equal(t, tc.exp, actual)
		})
	}
}

func TestConsulKVCondition_appendTemplate(t *testing.T) {
	testcases := []struct {
		name string
		c    *ConsulKVCondition
		exp  string
	}{
		{
			"recurse false includes var false",
			&ConsulKVCondition{
				ConsulKVMonitor: ConsulKVMonitor{
					Path:       "path",
					Recurse:    false,
					Datacenter: "dc1",
					Namespace:  "test-ns",
				},
				SourceIncludesVar: false,
			},
			`
{{- if keyExists "path" "dc=dc1" "ns=test-ns"  }}
	{{- with $kv := key "path" "dc=dc1" "ns=test-ns"  }}
		{{- /* Empty template. Detects changes in Consul KV */ -}}
	{{- end}}
{{- end}}
`,
		},
		{
			"recurse false includes var true",
			&ConsulKVCondition{
				ConsulKVMonitor: ConsulKVMonitor{
					Path:       "path",
					Recurse:    false,
					Datacenter: "dc1",
					Namespace:  "test-ns",
				},
				SourceIncludesVar: true,
			},
			`
consul_kv = {
{{- if keyExists "path" "dc=dc1" "ns=test-ns"  }}
	{{- with $kv := key "path" "dc=dc1" "ns=test-ns"  }}
		"path" = "{{ $kv }}"
	{{- end}}
{{- end}}
}
`,
		},
		{
			"recurse true includes var false",
			&ConsulKVCondition{
				ConsulKVMonitor: ConsulKVMonitor{
					Path:       "path",
					Recurse:    true,
					Datacenter: "dc1",
					Namespace:  "test-ns",
				},
				SourceIncludesVar: false,
			},
			`
{{- with $kv := keys "path" "dc=dc1" "ns=test-ns"  }}
	{{- range $k, $v := $kv }}
		{{- /* Empty template. Detects changes in Consul KV */ -}}
	{{- end}}
{{- end}}
`,
		},
		{
			"recurse true includes var true",
			&ConsulKVCondition{
				ConsulKVMonitor: ConsulKVMonitor{
					Path:       "path",
					Recurse:    true,
					Datacenter: "dc1",
					Namespace:  "test-ns",
				},
				SourceIncludesVar: true,
			},
			`
consul_kv = {
{{- with $kv := keys "path" "dc=dc1" "ns=test-ns"  }}
	{{- range $k := $kv }}
	"{{ .Path }}" = "{{ .Value }}"
	{{- end}}
{{- end}}
}
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
