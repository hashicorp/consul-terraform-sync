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

	// ServicesAppended returns if the services variable has been appended
	// to the template content.
	ServicesAppended() bool

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
type ServicesCondition struct {
	Regexp string
}

func (c ServicesCondition) SourceIncludesVariable() bool {
	return false
}

func (c ServicesCondition) ServicesAppended() bool {
	return c.Regexp != ""
}

func (c ServicesCondition) appendModuleAttribute(body *hclwrite.Body) {}

func (c ServicesCondition) appendTemplate(w io.Writer) error {
	if c.Regexp == "" {
		return nil
	}
	q := c.hcatQuery()
	_, err := fmt.Fprintf(w, servicesRegexTmpl, q)
	if err != nil {
		log.Printf("[ERR] (templates.tftmpl) unable to write service condition template")
		return err
	}
	return nil
}

func (c ServicesCondition) appendVariable(io.Writer) error {
	return nil
}

func (c ServicesCondition) hcatQuery() string {
	var opts []string

	if c.Regexp != "" {
		opts = append(opts, fmt.Sprintf("regexp=%s", c.Regexp))
	}

	if len(opts) > 0 {
		return `"` + strings.Join(opts, `" "`) + `" ` // deliberate space at end
	}
	return ""
}

const servicesRegexTmpl = `
services = {
{{- with $srv := servicesRegex %s }}
  {{- range $s := $srv}}
  "{{ joinStrings "." .ID .Node .Namespace .NodeDatacenter }}" = {
{{ HCLService $s | indent 4 }}
  },
  {{- end}}
{{- end}}
}`

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

func (c CatalogServicesCondition) ServicesAppended() bool {
	return false
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
			log.Printf("[ERR] (templates.tftmpl) unable to write catalog-service" +
				" template to include variable")
			return err
		}
		return nil
	}
	_, err := fmt.Fprintf(w, catalogServicesConditionTmpl, q)
	if err != nil {
		log.Printf("[ERR] (templates.tftmpl) unable to write catalog-service" +
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

type NodesCondition struct {
	Datacenter string
	Filter     string
}

func (n NodesCondition) SourceIncludesVariable() bool {
	return true
}

func (n NodesCondition) ServicesAppended() bool {
	return false
}

func (n NodesCondition) appendModuleAttribute(body *hclwrite.Body) {
	body.SetAttributeTraversal("nodes", hcl.Traversal{
		hcl.TraverseRoot{Name: "var"},
		hcl.TraverseAttr{Name: "nodes"},
	})
}

func (n NodesCondition) appendTemplate(w io.Writer) error {
	q := n.hcatQuery()
	_, err := fmt.Fprintf(w, nodesRegexTmpl, q)
	if err != nil {
		log.Printf("[ERR] (templates.tftmpl) unable to write nodes condition template")
		return err
	}

	return nil
}

func (n NodesCondition) appendVariable(w io.Writer) error {
	_, err := w.Write(variableNodes)
	return err
}

func (n NodesCondition) hcatQuery() string {
	var opts []string

	if n.Datacenter != "" {
		opts = append(opts, fmt.Sprintf("datacenter=%s", n.Datacenter))
	}

	if n.Filter != "" {
		opts = append(opts, fmt.Sprintf("filter=%s", n.Filter))
	}

	if len(opts) > 0 {
		return `"` + strings.Join(opts, `" "`) + `" ` // deliberate space at end
	}
	return ""
}

const nodesRegexTmpl = `
nodes = [
{{- with $nodes := nodes %s }}
	{{- range $node := $nodes}}
	{
{{ HCLNode $node | indent 4 }}
	},
	{{- end}}
{{- end}}
]
`

var variableNodes = []byte(`
# Nodes definition protocol v0
variable "nodes" {
  description = "Consul nodes"
  type = list(object({
	  id               = string
	  node             = string
	  address          = string
	  datacenter       = string
	  tagged_addresses = map(string)
	  meta             = map(string)
  }))
}
`)
