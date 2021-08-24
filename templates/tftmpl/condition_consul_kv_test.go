package tftmpl

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
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
				Path: "key-path",
			},
			"\"key-path\" ",
		},
		{
			"all_parameters",
			&ConsulKVCondition{
				Path:       "key-path",
				Datacenter: "dc2",
				Namespace:  "test-ns",
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
				Path:              "path",
				Recurse:           false,
				SourceIncludesVar: false,
				Datacenter:        "dc1",
				Namespace:         "test-ns",
			},
			`
{{- with $kv := key "path" "dc=dc1" "ns=test-ns"  }}
	{{- /* Empty template. Detects changes in Consul KV */ -}}
{{- end}}
`,
		},
		{
			"recurse false includes var true",
			&ConsulKVCondition{
				Path:              "path",
				Recurse:           false,
				SourceIncludesVar: true,
				Datacenter:        "dc1",
				Namespace:         "test-ns",
			},
			`
consul_kv = {
{{- with $kv := key "path" "dc=dc1" "ns=test-ns"  }}
	"path" = "{{ $kv }}"
{{- end}}
}
`,
		},
		{
			"recurse true includes var false",
			&ConsulKVCondition{
				Path:              "path",
				Recurse:           true,
				SourceIncludesVar: false,
				Datacenter:        "dc1",
				Namespace:         "test-ns",
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
				Path:              "path",
				Recurse:           true,
				SourceIncludesVar: true,
				Datacenter:        "dc1",
				Namespace:         "test-ns",
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
			tc.c.appendTemplate(w)
			assert.Equal(t, tc.exp, w.String())
		})
	}
}
