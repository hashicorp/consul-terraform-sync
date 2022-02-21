package tftmpl

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConsulKVTemplate_hcatQuery(t *testing.T) {
	testcase := []struct {
		name string
		c    *ConsulKVTemplate
		exp  string
	}{
		{
			"path only",
			&ConsulKVTemplate{
				Path: "key-path",
			},
			"\"key-path\"",
		},
		{
			"all_parameters",
			&ConsulKVTemplate{
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

func TestConsulKVTemplate_appendTemplate(t *testing.T) {
	testcases := []struct {
		name string
		c    *ConsulKVTemplate
		exp  string
	}{
		{
			"recurse false & render var",
			&ConsulKVTemplate{
				Path:       "path",
				Recurse:    false,
				Datacenter: "dc1",
				Namespace:  "test-ns",
				RenderVar:  true,
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
			"recurse true & render var",
			&ConsulKVTemplate{
				Path:       "path",
				Recurse:    true,
				Datacenter: "dc1",
				Namespace:  "test-ns",
				RenderVar:  true,
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
			"recurse false & no var",
			&ConsulKVTemplate{
				Path:       "path",
				Recurse:    false,
				Datacenter: "dc1",
				Namespace:  "test-ns",
				RenderVar:  false,
			},
			`
{{- with $kv := keyExistsGet "path" "dc=dc1" "ns=test-ns" }}
  {{- /* Empty template. Detects changes in Consul KV */ -}}
{{- end}}
`,
		},
		{
			"recurse true & no var",
			&ConsulKVTemplate{
				Path:       "path",
				Recurse:    true,
				Datacenter: "dc1",
				Namespace:  "test-ns",
				RenderVar:  false,
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
