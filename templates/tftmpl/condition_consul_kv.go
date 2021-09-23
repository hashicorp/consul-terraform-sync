package tftmpl

import (
	"fmt"
	"io"

	"github.com/hashicorp/consul-terraform-sync/logging"
)

var (
	_ Condition = (*ConsulKVCondition)(nil)
)

// ConsulKVCondition handles appending templating for the consul-kv
// run condition
type ConsulKVCondition struct {
	ConsulKVMonitor
	SourceIncludesVar bool
}

// SourceIncludesVariable returns true if the variables are to be included
// and false otherwise
func (c ConsulKVCondition) SourceIncludesVariable() bool {
	return c.SourceIncludesVar
}

// appendTemplate writes the template needed for the Consul KV condition.
// It determines which template to use based on the values of the
// source_includes_var and recurse options. If source_includes_var is set
// to true, include the template as part of the variable consul_kv.
// If recurse is set to true, then use the 'keys' template, otherwise
// use the 'keyExists'/'key' template.
func (c ConsulKVCondition) appendTemplate(w io.Writer) error {
	logger := logging.Global().Named(logSystemName).Named(tftmplSubsystemName)
	var err error
	if c.SourceIncludesVariable() {
		err = c.ConsulKVMonitor.appendTemplate(w)
	} else {
		q := c.hcatQuery()
		var conditionTmpl string
		if c.Recurse {
			conditionTmpl = fmt.Sprintf(consulKVRecurseConditionTmpl, q)
		} else {
			conditionTmpl = fmt.Sprintf(consulKVConditionTmpl, q)
		}
		_, err = w.Write([]byte(conditionTmpl))
		if err != nil {
			logger.Error("unable to write consul-kv empty template", "error", err)
		}
	}

	if err != nil {
		return err
	}
	return nil
}

const consulKVConditionTmpl = `
{{- with $kv := keyExistsGet %s }}
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
