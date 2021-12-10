package tftmpl

import (
	"fmt"
	"io"
	"strings"

	"github.com/hashicorp/consul-terraform-sync/logging"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

var (
	_ Monitor = (*ConsulKVMonitor)(nil)
)

// ConsulKVMonitor handles appending templating for the consul-kv
// run monitor
type ConsulKVMonitor struct {
	Path       string
	Recurse    bool
	Datacenter string
	Namespace  string

	SourceIncludesVar bool
}

// isServicesVar returns false because the tmplfunc returns a consul_kv
// variable, not a services variable
func (m ConsulKVMonitor) isServicesVar() bool {
	return false
}

func (m ConsulKVMonitor) SourceIncludesVariable() bool {
	return m.SourceIncludesVar
}

func (m ConsulKVMonitor) appendModuleAttribute(body *hclwrite.Body) {
	body.SetAttributeTraversal("consul_kv", hcl.Traversal{
		hcl.TraverseRoot{Name: "var"},
		hcl.TraverseAttr{Name: "consul_kv"},
	})
}

// appendTemplate writes the template needed for the Consul KV condition.
// It determines which template to use based on the values of the
// source_includes_var and recurse options. If source_includes_var is set
// to true, include the template as part of the variable consul_kv.
// If recurse is set to true, then use the 'keys' template, otherwise
// use the 'keyExists'/'key' template.
func (m ConsulKVMonitor) appendTemplate(w io.Writer) error {
	logger := logging.Global().Named(logSystemName).Named(tftmplSubsystemName)
	q := m.hcatQuery()

	if m.SourceIncludesVar {
		var baseTmpl string
		if m.Recurse {
			baseTmpl = fmt.Sprintf(consulKVRecurseBaseTmpl, q)
		} else {
			baseTmpl = fmt.Sprintf(consulKVBaseTmpl, q)
		}

		if _, err := fmt.Fprintf(w, consulKVIncludesVarTmpl, baseTmpl); err != nil {
			logger.Error("unable to write consul-kv template to include variable", "error", err)
			return err
		}
		return nil
	}

	var emptyTmpl string
	if m.Recurse {
		emptyTmpl = fmt.Sprintf(consulKVRecurseConditionEmptyTmpl, q)
	} else {
		emptyTmpl = fmt.Sprintf(consulKVConditionEmptyTmpl, q)
	}
	if _, err := w.Write([]byte(emptyTmpl)); err != nil {
		logger.Error("unable to write consul-kv empty template", "error", err)
		return err
	}
	return nil
}

func (m ConsulKVMonitor) appendVariable(w io.Writer) error {
	_, err := w.Write(variableConsulKV)
	return err
}

func (m ConsulKVMonitor) hcatQuery() string {
	var opts []string

	opts = append(opts, m.Path)

	if m.Datacenter != "" {
		opts = append(opts, fmt.Sprintf("dc=%s", m.Datacenter))
	}

	if m.Namespace != "" {
		opts = append(opts, fmt.Sprintf("ns=%s", m.Namespace))
	}

	if len(opts) > 0 {
		return `"` + strings.Join(opts, `" "`) + `"`
	}
	return ""
}

var consulKVIncludesVarTmpl = `
consul_kv = {%s}
`

const consulKVBaseTmpl = `
{{- with $kv := keyExistsGet %s }}
  {{- if .Exists }}
  "{{ .Path }}" = "{{ .Value }}"
  {{- end}}
{{- end}}
`

const consulKVRecurseBaseTmpl = `
{{- with $kv := keys %s }}
  {{- range $k := $kv }}
  "{{ .Path }}" = "{{ .Value }}"
  {{- end}}
{{- end}}
`

const consulKVConditionEmptyTmpl = `
{{- with $kv := keyExistsGet %s }}
  {{- /* Empty template. Detects changes in Consul KV */ -}}
{{- end}}
`

const consulKVRecurseConditionEmptyTmpl = `
{{- with $kv := keys %s }}
  {{- range $k, $v := $kv }}
  {{- /* Empty template. Detects changes in Consul KV */ -}}
  {{- end}}
{{- end}}
`

// variableConsulKV is required for modules that include Consul KV
// information. It is versioned to track compatibility between the generated
// root module and modules that include Consul KV.
var variableConsulKV = []byte(`
# Consul KV definition protocol v0
variable "consul_kv" {
  description = "Consul KV pair"
  type        = map(string)
}
`)
