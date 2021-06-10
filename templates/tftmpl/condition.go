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
	_ Condition = (*ServicesCondition)(nil)
	_ Condition = (*CatalogServicesCondition)(nil)
)

// Condition handles appending a run condition's relevant templating for Terraform
// generated files
type Condition interface {
	// SourceIncludesVariable returns if the module source expects to
	// include the condition variable.
	SourceIncludesVariable() bool

	// appendModuleAttribute writes to an HCL module body the condition variable
	// as a module argument in main.tf file.
	// module "name" {
	//   catalog_services = var.catalog_services
	// }
	appendModuleAttribute(*hclwrite.Body)

	// appendTemplate writes the generated variable template for the condition
	// based on whether the source includes the condition variable.
	appendTemplate(io.Writer) error

	// appendVariable writes the corresponding Terraform variable block to
	// the variables.tf file.
	appendVariable(io.Writer) error
}

// ServicesCondition handles appending templating for the services run condition
// This is the default run condition
type ServicesCondition struct{}

func (c ServicesCondition) SourceIncludesVariable() bool {
	return false
}

func (c ServicesCondition) appendModuleAttribute(body *hclwrite.Body) {}

func (c ServicesCondition) appendTemplate(io.Writer) error {
	// no-op: services condition currently requires no additional condition
	// templating. it relies on the monitoring template as the run condition
	return nil
}

func (c ServicesCondition) appendVariable(io.Writer) error {
	return nil
}

// CatalogServicesCondition handles appending templating for the catalog-service
// run condition
type CatalogServicesCondition struct {
	Regexp            string
	SourceIncludesVar bool
	Datacenter        string
	Namespace         string
	NodeMeta          map[string]string
}

func (c CatalogServicesCondition) SourceIncludesVariable() bool {
	return c.SourceIncludesVar
}

func (c CatalogServicesCondition) appendModuleAttribute(body *hclwrite.Body) {
	body.SetAttributeTraversal("catalog_services", hcl.Traversal{
		hcl.TraverseRoot{Name: "var"},
		hcl.TraverseAttr{Name: "catalog_services"},
	})
}

func (c CatalogServicesCondition) appendTemplate(w io.Writer) error {
	q := c.hcatQuery()
	if c.SourceIncludesVar {
		_, err := fmt.Fprintf(w, catalogServicesConditionIncludesVarTmpl, q)
		if err != nil {
			log.Printf("[WARN] (templates.tftmpl) unable to write catalog-service" +
				" template to include variable")
			return err
		}
		return nil
	}
	_, err := fmt.Fprintf(w, catalogServicesConditionTmpl, q)
	if err != nil {
		log.Printf("[WARN] (templates.tftmpl) unable to write catalog-service" +
			" empty template")
		return err
	}
	return nil
}

func (c CatalogServicesCondition) appendVariable(w io.Writer) error {
	_, err := w.Write(variableCatalogServices)
	return err
}

func (c CatalogServicesCondition) hcatQuery() string {
	var opts []string

	if c.Regexp != "" {
		opts = append(opts, fmt.Sprintf("regexp=%s", c.Regexp))
	}

	if c.Datacenter != "" {
		opts = append(opts, fmt.Sprintf("dc=%s", c.Datacenter))
	}

	if c.Namespace != "" {
		opts = append(opts, fmt.Sprintf("ns=%s", c.Namespace))
	}

	for k, v := range c.NodeMeta {
		opts = append(opts, fmt.Sprintf("node-meta=%s:%s", k, v))
	}

	if len(opts) > 0 {
		return `"` + strings.Join(opts, `" "`) + `" ` // deliberate space at end
	}
	return ""
}

const catalogServicesConditionTmpl = `
{{- with $catalogServices := catalogServicesRegistration %s}}
  {{- range $cs := $catalogServices }}
    {{- /* Empty template. Detects changes in catalog-services */ -}}
{{- end}}{{- end}}
`

var catalogServicesConditionIncludesVarTmpl = fmt.Sprintf(`
catalog_services = {%s}
`, catalogServicesBaseTmpl)

const catalogServicesBaseTmpl = `
{{- with $catalogServices := catalogServicesRegistration %s}}
  {{- range $cs := $catalogServices }}
  "{{ $cs.Name }}" = {{ HCLServiceTags $cs.Tags }}
{{- end}}{{- end}}
`

// variableCatalogServices is required for modules that include catalog-services
// information. It is versioned to track compatibility between the generated
// root module and modules that include catalog-services.
var variableCatalogServices = []byte(`
# Catalog Services definition protocol v0
variable "catalog_services" {
  description = "Consul catalog service names and list of all known tags for a given service"
  type = map(list(string))
}
`)
