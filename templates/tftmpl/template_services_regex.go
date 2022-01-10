package tftmpl

import (
	"fmt"
	"io"
	"strings"

	"github.com/hashicorp/consul-terraform-sync/logging"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

var (
	_ Template = (*ServicesRegexTemplate)(nil)
)

// ServicesRegexTemplate handles the template for the services variable for the
// template function: `{{ servicesRegex }}`
type ServicesRegexTemplate struct {
	Regexp     string
	Datacenter string
	Namespace  string
	Filter     string

	SourceIncludesVar bool
}

// IsServicesVar returns true because the template is for the services variable
func (t ServicesRegexTemplate) IsServicesVar() bool {
	return true
}

func (t ServicesRegexTemplate) appendModuleAttribute(*hclwrite.Body) {}

func (t ServicesRegexTemplate) appendTemplate(w io.Writer) error {
	q := t.hcatQuery()

	tmpl := ""
	if t.SourceIncludesVar {
		tmpl = fmt.Sprintf(servicesRegexIncludesVarTmpl, q)
	} else {
		tmpl = fmt.Sprintf(servicesRegexEmptyTmpl, q)
	}

	if _, err := fmt.Fprint(w, tmpl); err != nil {
		logging.Global().Named(logSystemName).Named(tftmplSubsystemName).Error(
			"unable to write services regex template", "error", err,
			"source_includes_var", t.SourceIncludesVar)
		return err
	}
	return nil
}

func (t ServicesRegexTemplate) appendVariable(io.Writer) error {
	return nil
}

// SourceIncludesVariable returns true if the source variables are to be included in the template.
// For the case of a service monitor, this always returns true and must be overridden to
// return based on other conditions.
func (t ServicesRegexTemplate) SourceIncludesVariable() bool {
	return t.SourceIncludesVar
}

func (t ServicesRegexTemplate) hcatQuery() string {
	var opts []string

	// Support regexp == "" (same as a wildcard)
	opts = append(opts, fmt.Sprintf("regexp=%s", t.Regexp))

	if t.Datacenter != "" {
		opts = append(opts, fmt.Sprintf("dc=%s", t.Datacenter))
	}

	if t.Namespace != "" {
		opts = append(opts, fmt.Sprintf("ns=%s", t.Namespace))
	}

	if t.Filter != "" {
		filter := strings.ReplaceAll(t.Filter, `"`, `\"`)
		filter = strings.Trim(filter, "\n")
		opts = append(opts, filter)
	}

	if len(opts) > 0 {
		return `"` + strings.Join(opts, `" "`) + `"`
	}
	return ""
}

var servicesRegexIncludesVarTmpl = fmt.Sprintf(`
services = {%s}
`, servicesRegexBaseTmpl)

const servicesRegexBaseTmpl = `
{{- with $srv := servicesRegex %s }}
  {{- range $s := $srv}}
  "{{ joinStrings "." .ID .Node .Namespace .NodeDatacenter }}" = {
{{ HCLService $s | indent 4 }}
  },
  {{- end}}
{{- end}}
`

const servicesRegexEmptyTmpl = `
{{- with $srv := servicesRegex %s }}
  {{- range $s := $srv}}
  {{- /* Empty template. Detects changes in Services */ -}}
  {{- end}}
{{- end}}
`
