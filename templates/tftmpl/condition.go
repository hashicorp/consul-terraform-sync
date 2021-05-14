package tftmpl

import (
	"fmt"
	"io"
	"log"
	"strings"
)

var (
	_ Condition = (*ServicesCondition)(nil)
	_ Condition = (*CatalogServicesCondition)(nil)
)

// Condition handles appending a run condition's templating to the tfvars.tmpl
// file
type Condition interface {
	appendTemplate(io.Writer) error
}

// ServicesCondition handles appending templating for the services run condition
// This is the default run condition
type ServicesCondition struct{}

func (c *ServicesCondition) appendTemplate(w io.Writer) error {
	// no-op: services conditon currently requires no additional condition
	// templating. it relies on the monitoring template as the run condition
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

func (c *CatalogServicesCondition) appendTemplate(w io.Writer) error {
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

func (c *CatalogServicesCondition) hcatQuery() string {
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

const catalogServicesConditionTmpl = `{{- with $catalogServices := catalogServicesRegistration %s}}
  {{- range $cs := $catalogServices }}
    {{- /* Empty template. Detects changes in catalog-services */ -}}
{{- end}}{{- end}}

`

const catalogServicesConditionIncludesVarTmpl = `catalog_services = {
{{- with $catalogServices := catalogServicesRegistration %s}}
  {{- range $cs := $catalogServices }}
  "{{ $cs.Name }}" = {{ HCLServiceTags $cs.Tags }}
{{- end}}{{- end}}
}

`
