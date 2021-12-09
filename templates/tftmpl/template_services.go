package tftmpl

import (
	"fmt"
	"io"
	"strings"

	"github.com/hashicorp/consul-terraform-sync/logging"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

var (
	_ Template = (*ServicesTemplate)(nil)
)

const (
	logSystemName       = "templates"
	tftmplSubsystemName = "tftmpl"
)

// Template handles templates for different template functions to monitor
// different types of variables
type Template interface {
	// isServicesVar returns whether or not the template function returns a
	// variable of type services
	isServicesVar() bool

	// SourceIncludesVariable returns if the module source expects to
	// include the monitored variable.
	SourceIncludesVariable() bool

	// appendModuleAttribute writes to an HCL module body the monitored variable
	// as a module argument in main.tf file.
	// module "name" {
	//   catalog_services = var.catalog_services
	// }
	appendModuleAttribute(*hclwrite.Body)

	// appendTemplate writes the generated variable template to the
	// terrafort.tfvars.tmpl file based on whether the source includes the
	// monitored variable.
	appendTemplate(io.Writer) error

	// appendVariable writes the corresponding Terraform variable block to
	// the variables.tf file.
	appendVariable(io.Writer) error
}

// ServicesTemplate handles the template for the services variable for the
// template function: `{{ servicesRegex }}`
type ServicesTemplate struct {
	Regexp     string
	Datacenter string
	Namespace  string
	Filter     string

	SourceIncludesVar bool
}

// isServicesVar returns true because the template is for the services variable
func (t ServicesTemplate) isServicesVar() bool {
	return true
}

func (t ServicesTemplate) appendModuleAttribute(*hclwrite.Body) {}

func (t ServicesTemplate) appendTemplate(w io.Writer) error {
	if t.Regexp == "" {
		return nil
	}
	q := t.hcatQuery()

	if t.SourceIncludesVar {
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

func (t ServicesTemplate) appendVariable(io.Writer) error {
	return nil
}

// SourceIncludesVariable returns true if the source variables are to be included in the template.
// For the case of a service monitor, this always returns true and must be overridden to
// return based on other conditions.
func (t ServicesTemplate) SourceIncludesVariable() bool {
	return t.SourceIncludesVar
}

func (t ServicesTemplate) hcatQuery() string {
	var opts []string

	if t.Regexp != "" {
		opts = append(opts, fmt.Sprintf("regexp=%s", t.Regexp))
	}

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