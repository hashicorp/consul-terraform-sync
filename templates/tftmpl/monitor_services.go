package tftmpl

import (
	"fmt"
	"io"
	"strings"

	"github.com/hashicorp/consul-terraform-sync/logging"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

var (
	_ Monitor = (*ServicesMonitor)(nil)
)

const (
	logSystemName       = "templates"
	tftmplSubsystemName = "tftmpl"
)

// Monitor handles appending a run monitor's relevant templating for Terraform
// generated files
type Monitor interface {
	// isServicesVar returns whether or not the template function returns a
	// variable of type services
	isServicesVar() bool

	// SourceIncludesVariable returns if the module source expects to
	// include the monitor variable.
	SourceIncludesVariable() bool

	// appendModuleAttribute writes to an HCL module body the monitor variable
	// as a module argument in main.tf file.
	// module "name" {
	//   catalog_services = var.catalog_services
	// }
	appendModuleAttribute(*hclwrite.Body)

	// appendTemplate writes the generated variable template for the monitor
	// based on whether the source includes the monitor variable.
	appendTemplate(io.Writer) error

	// appendVariable writes the corresponding Terraform variable block to
	// the variables.tf file.
	appendVariable(io.Writer) error
}

// ServicesMonitor handles appending templating for the services run monitor
type ServicesMonitor struct {
	Regexp     string
	Datacenter string
	Namespace  string
	Filter     string

	SourceIncludesVar bool
}

// isServicesVar returns true because the tmplfunc returns a services variable
func (m ServicesMonitor) isServicesVar() bool {
	return true
}

func (m ServicesMonitor) appendModuleAttribute(*hclwrite.Body) {}

func (m ServicesMonitor) appendTemplate(w io.Writer) error {
	if m.Regexp == "" {
		return nil
	}
	q := m.hcatQuery()

	if m.SourceIncludesVar {
		if _, err := fmt.Fprintf(w, servicesRegexIncludesVarTmpl, q); err != nil {
			return err
		}
		return nil
	}

	if _, err := fmt.Fprintf(w, servicesRegexBaseTmpl, q); err != nil {
		logging.Global().Named(logSystemName).Named(tftmplSubsystemName).Error(
			"unable to write service condition empty template", "error", err)
		return err
	}
	return nil
}

func (m ServicesMonitor) appendVariable(io.Writer) error {
	return nil
}

// SourceIncludesVariable returns true if the source variables are to be included in the template.
// For the case of a service monitor, this always returns true and must be overridden to
// return based on other conditions.
func (m ServicesMonitor) SourceIncludesVariable() bool {
	return m.SourceIncludesVar
}

func (m ServicesMonitor) hcatQuery() string {
	var opts []string

	if m.Regexp != "" {
		opts = append(opts, fmt.Sprintf("regexp=%s", m.Regexp))
	}

	if m.Datacenter != "" {
		opts = append(opts, fmt.Sprintf("dc=%s", m.Datacenter))
	}

	if m.Namespace != "" {
		opts = append(opts, fmt.Sprintf("ns=%s", m.Namespace))
	}

	if m.Filter != "" {
		filter := strings.ReplaceAll(m.Filter, `"`, `\"`)
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
