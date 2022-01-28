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
	_ Template = (*ConsulKVTemplate)(nil)
)

// ConsulKVTemplate handles the template for the consul_kv variable for the
// template functions: `{{ key }}` and `{{ keyExistsGet }}`
type ConsulKVTemplate struct {
	Path       string
	Recurse    bool
	Datacenter string
	Namespace  string

	// RenderVar informs whether the template should render the variable or not.
	// Aligns with the task condition configuration `UseAsModuleInput``
	RenderVar bool
}

// IsServicesVar returns false because the template returns a consul_kv
// variable, not a services variable
func (t ConsulKVTemplate) IsServicesVar() bool {
	return false
}

func (t ConsulKVTemplate) RendersVar() bool {
	return t.RenderVar
}

func (t ConsulKVTemplate) appendModuleAttribute(body *hclwrite.Body) {
	body.SetAttributeTraversal("consul_kv", hcl.Traversal{
		hcl.TraverseRoot{Name: "var"},
		hcl.TraverseAttr{Name: "consul_kv"},
	})
}

// appendTemplate writes the template needed for the Consul KV condition.
// It determines which template to use based on the values of the RenderVar and
// recurse options. If RenderVar is true, then set the consul_kv variable to
// the template. If recurse is set to true, then use the 'keys' template,
// otherwise use the 'keyExists'/'key' template.
func (t ConsulKVTemplate) appendTemplate(w io.Writer) error {
	logger := logging.Global().Named(logSystemName).Named(tftmplSubsystemName)
	q := t.hcatQuery()

	if t.RenderVar {
		var baseTmpl string
		if t.Recurse {
			baseTmpl = fmt.Sprintf(consulKVRecurseBaseTmpl, q)
		} else {
			baseTmpl = fmt.Sprintf(consulKVBaseTmpl, q)
		}

		if _, err := fmt.Fprintf(w, consulKVSetVarTmpl, baseTmpl); err != nil {
			logger.Error("unable to write consul-kv template with variable", "error", err)
			return err
		}
		return nil
	}

	var emptyTmpl string
	if t.Recurse {
		emptyTmpl = fmt.Sprintf(consulKVRecurseEmptyTmpl, q)
	} else {
		emptyTmpl = fmt.Sprintf(consulKVEmptyTmpl, q)
	}
	if _, err := w.Write([]byte(emptyTmpl)); err != nil {
		logger.Error("unable to write consul-kv empty template", "error", err)
		return err
	}
	return nil
}

func (t ConsulKVTemplate) appendVariable(w io.Writer) error {
	_, err := w.Write(variableConsulKV)
	return err
}

func (t ConsulKVTemplate) hcatQuery() string {
	var opts []string

	opts = append(opts, t.Path)

	if t.Datacenter != "" {
		opts = append(opts, fmt.Sprintf("dc=%s", t.Datacenter))
	}

	if t.Namespace != "" {
		opts = append(opts, fmt.Sprintf("ns=%s", t.Namespace))
	}

	if len(opts) > 0 {
		return `"` + strings.Join(opts, `" "`) + `"`
	}
	return ""
}

var consulKVSetVarTmpl = `
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

const consulKVEmptyTmpl = `
{{- with $kv := keyExistsGet %s }}
  {{- /* Empty template. Detects changes in Consul KV */ -}}
{{- end}}
`

const consulKVRecurseEmptyTmpl = `
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
