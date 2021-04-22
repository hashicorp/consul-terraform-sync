package tftmpl

import (
	"fmt"
	"io"
	"log"
)

// Condition handles appending a run condition's templating to the tfvars.tmpl
// file
type Condition interface {
	appendTemplate(io.Writer) error
}

var _ Condition = (*ServicesCondition)(nil)

// ServicesCondition handles appending templating for the services run condition
// This is the default run condition
type ServicesCondition struct{}

func (c *ServicesCondition) appendTemplate(w io.Writer) error {
	// no-op: services conditon currently requires no additional condition
	// templating. it relies on the monitoring template as the run condition
	return nil
}

var _ Condition = (*CatalogServicesCondition)(nil)

// CatalogServicesCondition handles appending templating for the catalog-service
// run condition
type CatalogServicesCondition struct {
	Regexp      string
	EnableTfVar bool
	Datacenter  string
	Namespace   string
	NodeMeta    map[string]string
}

func (c *CatalogServicesCondition) appendTemplate(w io.Writer) error {
	_, err := fmt.Fprintf(w, catalogServicesRunConditionTmpl, c.Regexp)
	if err != nil {
		log.Printf("[WARN] (templates.tftmpl) unable to write catalog-service" +
			" trigger condition to template")
		return err
	}
	return nil
}

const catalogServicesRunConditionTmpl = `{{- with $allCatalogServices := services }}
  {{- range $cs := $allCatalogServices }}
    {{- if regexMatch "%s" $cs.Name }}{{/* Empty template. Detects changes in catalog-services */}}
{{- end}}{{- end}}{{- end}}

`
