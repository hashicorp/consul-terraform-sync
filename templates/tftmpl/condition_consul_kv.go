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
	_ Condition = (*ConsulKVCondition)(nil)
)

// ConsulKVCondition handles appending templating for the consul-kv
// run condition
type ConsulKVCondition struct {
	Path              string
	SourceIncludesVar bool
	Recurse           bool
	Datacenter        string
	Namespace         string
}

func (c ConsulKVCondition) SourceIncludesVariable() bool {
	return c.SourceIncludesVar
}

func (c ConsulKVCondition) ServicesAppended() bool {
	return false
}

func (c ConsulKVCondition) appendModuleAttribute(body *hclwrite.Body) {
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
func (c ConsulKVCondition) appendTemplate(w io.Writer) error {
	q := c.hcatQuery()

	logger := logging.Global().Named(logSystemName).Named(tftmplSubsystemName)
	if c.SourceIncludesVar {
		var baseTmpl string
		if c.Recurse {
			baseTmpl = fmt.Sprintf(consulKVRecurseBaseTmpl, q)
		} else {
			baseTmpl = fmt.Sprintf(consulKVBaseTmpl, q, q, c.Path)
		}
		_, err := fmt.Fprintf(w, consulKVConditionIncludesVarTmpl, baseTmpl)
		if err != nil {
			logger.Error("unable to write consul-kv template to include variable", "error", err)
			return err
		}
		return nil
	}

	var conditionTmpl string
	if c.Recurse {
		conditionTmpl = fmt.Sprintf(consulKVRecurseConditionTmpl, q)
	} else {
		conditionTmpl = fmt.Sprintf(consulKVConditionTmpl, q, q)
	}
	_, err := w.Write([]byte(conditionTmpl))
	if err != nil {
		logger.Error("unable to write consul-kv empty template", "error", err)
		return err
	}
	return nil
}

func (c ConsulKVCondition) appendVariable(w io.Writer) error {
	_, err := w.Write(variableConsulKV)
	return err
}

func (c ConsulKVCondition) hcatQuery() string {
	var opts []string

	opts = append(opts, c.Path)

	if c.Datacenter != "" {
		opts = append(opts, fmt.Sprintf("dc=%s", c.Datacenter))
	}

	if c.Namespace != "" {
		opts = append(opts, fmt.Sprintf("ns=%s", c.Namespace))
	}

	if len(opts) > 0 {
		return `"` + strings.Join(opts, `" "`) + `" ` // deliberate space at end
	}
	return ""
}

const consulKVConditionTmpl = `
{{- if keyExists %s }}
	{{- with $kv := key %s }}
		{{- /* Empty template. Detects changes in Consul KV */ -}}
	{{- end}}
{{- end}}
`
const consulKVRecurseConditionTmpl = `
{{- with $kv := keys %s }}
	{{- range $k, $v := $kv }}
		{{- /* Empty template. Detects changes in Consul KV */ -}}
	{{- end}}
{{- end}}
`

var consulKVConditionIncludesVarTmpl = `
consul_kv = {%s}
`

const consulKVBaseTmpl = `
{{- if keyExists %s }}
	{{- with $kv := key %s }}
		"%s" = "{{ $kv }}"
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

// variableConsulKV is required for modules that include Consul KV
// information. It is versioned to track compatibility between the generated
// root module and modules that include Consul KV.
var variableConsulKV = []byte(`
# Consul KV definition protocol v0
variable "consul_kv" {
	description = "Consul KV pair"
	type = map(string)
}
`)
