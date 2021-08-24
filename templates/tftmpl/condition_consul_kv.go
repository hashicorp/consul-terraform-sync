package tftmpl

import (
	"fmt"
	"io"
	"log"
	"strings"

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

func (c ConsulKVCondition) appendTemplate(w io.Writer) error {
	q := c.hcatQuery()

	if c.SourceIncludesVar {
		var baseTmpl string
		if c.Recurse {
			baseTmpl = fmt.Sprintf(consulKVRecurseBaseTmpl, q)
		} else {
			baseTmpl = fmt.Sprintf(consulKVBaseTmpl, q, c.Path)
		}
		_, err := fmt.Fprintf(w, consulKVConditionIncludesVarTmpl, baseTmpl)
		if err != nil {
			log.Printf("[ERR] (templates.tftmpl) unable to write consul-kv" +
				" template to include variable")
			return err
		}
		return nil
	}

	var conditionTmpl string
	if c.Recurse {
		conditionTmpl = consulKVRecurseConditionTmpl
	} else {
		conditionTmpl = consulKVConditionTmpl
	}
	_, err := fmt.Fprintf(w, conditionTmpl, q)
	if err != nil {
		log.Printf("[ERR] (templates.tftmpl) unable to write consul-kv" +
			" empty template")
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
{{- with $kv := key %s }}
	{{- /* Empty template. Detects changes in Consul KV */ -}}
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
{{- with $kv := key %s }}
	"%s" = "{{ $kv }}"
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
