package tftmpl

import (
	"fmt"
	"io"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

var (
	_ Template = (*CatalogServicesTemplate)(nil)
)

// CatalogServicesTemplate handles the template for the catalog_services
// variable for the template function: `{{ catalogServicesRegistration }}`
type CatalogServicesTemplate struct {
	Regexp     string
	Datacenter string
	Namespace  string
	NodeMeta   map[string]string

	// RenderVar informs whether the template should render the variable or not.
	// Aligns with the task condition configuration `UseAsModuleInput``
	RenderVar bool
}

// IsServicesVar returns false because the template returns a catalog_services
// variable, not a services variable
func (t CatalogServicesTemplate) IsServicesVar() bool {
	return false
}

func (t CatalogServicesTemplate) RendersVar() bool {
	return t.RenderVar
}

func (m CatalogServicesTemplate) appendModuleAttribute(body *hclwrite.Body) {
	body.SetAttributeTraversal("catalog_services", hcl.Traversal{
		hcl.TraverseRoot{Name: "var"},
		hcl.TraverseAttr{Name: "catalog_services"},
	})
}

func (t CatalogServicesTemplate) appendTemplate(w io.Writer) error {
	q := t.hcatQuery()

	if t.RenderVar {
		_, err := fmt.Fprintf(w, catalogServicesSetVarTmpl, q)
		if err != nil {
			err = fmt.Errorf("unable to write catalog-service template with variable, error: %v", err)
			return err
		}
		return nil
	}

	if _, err := fmt.Fprintf(w, catalogServicesEmptyTmpl, q); err != nil {
		err = fmt.Errorf("unable to write catalog-service empty template, error %v", err)
		return err
	}
	return nil
}

func (t CatalogServicesTemplate) appendVariable(w io.Writer) error {
	_, err := w.Write(variableCatalogServices)
	return err
}

func (t CatalogServicesTemplate) hcatQuery() string {
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

	for k, v := range t.NodeMeta {
		opts = append(opts, fmt.Sprintf("node-meta=%s:%s", k, v))
	}

	if len(opts) > 0 {
		return `"` + strings.Join(opts, `" "`) + `" ` // deliberate space at end
	}
	return ""
}

var catalogServicesSetVarTmpl = fmt.Sprintf(`
catalog_services = {%s}
`, catalogServicesBaseTmpl)

const catalogServicesBaseTmpl = `
{{- with $catalogServices := catalogServicesRegistration %s}}
  {{- range $cs := $catalogServices }}
  "{{ $cs.Name }}" = {{ HCLServiceTags $cs.Tags }}
{{- end}}{{- end}}
`

const catalogServicesEmptyTmpl = `
{{- with $catalogServices := catalogServicesRegistration %s}}
  {{- range $cs := $catalogServices }}
    {{- /* Empty template. Detects changes in catalog-services */ -}}
{{- end}}{{- end}}
`

// variableCatalogServices is required for modules that include catalog-services
// information. It is versioned to track compatibility between the generated
// root module and modules that include catalog-services.
var variableCatalogServices = []byte(`
# Catalog Services definition protocol v0
variable "catalog_services" {
  description = "Consul catalog service names and list of all known tags for a given service"
  type        = map(list(string))
}
`)
