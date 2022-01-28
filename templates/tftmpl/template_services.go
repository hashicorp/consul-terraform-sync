package tftmpl

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/hashicorp/consul-terraform-sync/logging"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

var (
	_ Template = (*ServicesTemplate)(nil)
)

// ServicesTemplate handles the template for the services variable for the
// template function: `{{ service }}`
type ServicesTemplate struct {
	Names []string

	// RenderVar informs whether the template should render the variable or not.
	// Aligns with the task condition configuration `UseAsModuleInput``
	RenderVar bool

	// Introduced in 0.5 - optional overall service filtering configured through
	// the task's condition "services". These configs or Services can be
	// configured but not both.
	Datacenter string
	Namespace  string
	Filter     string

	// Deprecated in 0.5 - optional per service filtering configured through the
	// task's services list. Not all services configured in Names must have
	// filtering configured. Services or the set of {Datacenter,
	// Namespace, Filter} can be configured but not both.
	Services map[string]Service
}

// Service contains additional Consul service filtering information for services
// configured in ServicesTemplate
type Service struct {
	Datacenter string
	Namespace  string
	Filter     string
}

// IsServicesVar returns true because the template is for the services variable
func (t ServicesTemplate) IsServicesVar() bool {
	return true
}

func (t ServicesTemplate) appendModuleAttribute(*hclwrite.Body) {}

func (t ServicesTemplate) appendTemplate(w io.Writer) error {
	tmpl, err := t.concatServiceTemplates()
	if err != nil {
		return err
	}

	if t.RenderVar {
		tmpl = fmt.Sprintf(servicesSetVarTmpl, tmpl)
	}

	if _, err := fmt.Fprint(w, tmpl); err != nil {
		logging.Global().Named(logSystemName).Named(tftmplSubsystemName).Error(
			"unable to write services template", "error", err)
		return err
	}
	return nil
}

// concatServiceTemplate sorts the services alphabetically and concatenates
// a template for each service
func (t ServicesTemplate) concatServiceTemplates() (string, error) {
	// double-check that service query parameter is configured in only one way
	// the current way or the deprecated way
	isCurrent := t.Datacenter != "" || t.Namespace != "" || t.Filter != ""
	isDeprecated := t.Services != nil

	if isCurrent && isDeprecated {
		// Configuration validation should prevent this.
		err := fmt.Errorf("services template query information is configured " +
			"in two ways. use only one")
		logging.Global().Named(logSystemName).Named(tftmplSubsystemName).Error(
			"unable to write services template", "error", err)
		return "", err
	}

	// concatenate sorted templates
	sort.Strings(t.Names)

	tmpl := ""
	for _, n := range t.Names {
		var query string
		if t.Services == nil {
			query = t.hcatQuery(n, t.Datacenter, t.Namespace, t.Filter)
		} else {
			s := t.Services[n]
			query = t.hcatQuery(n, s.Datacenter, s.Namespace, s.Filter)
		}

		if t.RenderVar {
			tmpl += fmt.Sprintf(serviceBaseTmpl, query)
		} else {
			tmpl += fmt.Sprintf(serviceEmptyTmpl, query)
		}
	}

	// special newline handling due to template concatenation
	tmpl += "\n"

	return tmpl, nil
}

func (t ServicesTemplate) appendVariable(io.Writer) error {
	return nil
}

func (t ServicesTemplate) RendersVar() bool {
	return t.RenderVar
}

func (t ServicesTemplate) hcatQuery(name, dc, ns, filter string) string {
	var opts []string

	opts = append(opts, name)

	if dc != "" {
		opts = append(opts, fmt.Sprintf("dc=%s", dc))
	}

	if ns != "" {
		opts = append(opts, fmt.Sprintf("ns=%s", ns))
	}

	if filter != "" {
		filter := strings.ReplaceAll(filter, `"`, `\"`)
		filter = strings.Trim(filter, "\n")
		opts = append(opts, filter)
	}

	if len(opts) > 0 {
		return `"` + strings.Join(opts, `" "`) + `"`
	}
	return ""
}

// servicesSetVarTmpl expects a concatenation of serviceBaseTmpl or
// serviceEmptyTmpl for each monitored service at '%s'
const servicesSetVarTmpl = `
services = {%s}
`

// serviceBaseTmpl is a template for a single monitored service. Multiple
// service requires concatenating multiple base templates. There is no newline
// at the end of this template (unlike other templates) to prevent a gap in the
// templates
const serviceBaseTmpl = `
{{- with $srv := service %s }}
  {{- range $s := $srv}}
  "{{ joinStrings "." .ID .Node .Namespace .NodeDatacenter }}" = {
{{ HCLService $s | indent 4 }}
  },
  {{- end}}
{{- end}}`

// serviceEmptyTmpl is a template for a single monitored service. Multiple
// service requires concatenating multiple empty templates. There is no newline
// at the end of this template (unlike other templates) to prevent a gap in the
// templates
const serviceEmptyTmpl = `
{{- with $srv := service %s }}
  {{- range $s := $srv}}
  {{- /* Empty template. Detects changes in Services */ -}}
  {{- end}}
{{- end}}`
