package tftmpl

import (
	"fmt"
	"io"

	"github.com/hashicorp/consul-terraform-sync/logging"
)

var (
	_ Condition = (*CatalogServicesCondition)(nil)
)

// CatalogServicesCondition handles appending templating for the catalog-service
// run condition
type CatalogServicesCondition struct {
	CatalogServicesMonitor
	SourceIncludesVar bool
}

// SourceIncludesVariable returns true if the variables are to be included
// and false otherwise
func (c CatalogServicesCondition) SourceIncludesVariable() bool {
	return c.SourceIncludesVar
}

func (c CatalogServicesCondition) appendTemplate(w io.Writer) error {
	logger := logging.Global().Named(logSystemName).Named(tftmplSubsystemName)
	var err error
	if c.SourceIncludesVariable() {
		err = c.CatalogServicesMonitor.appendTemplate(w)
	} else {
		q := c.hcatQuery()
		_, err = fmt.Fprintf(w, catalogServicesConditionTmpl, q)
		if err != nil {
			err = fmt.Errorf("unable to write catalog-service empty template, error %v", err)
		}
	}

	if err != nil {
		logger.Error("an error occurred appending the template", "error", err)
		return err
	}

	return nil
}

const catalogServicesConditionTmpl = `
{{- with $catalogServices := catalogServicesRegistration %s}}
  {{- range $cs := $catalogServices }}
    {{- /* Empty template. Detects changes in catalog-services */ -}}
{{- end}}{{- end}}
`
