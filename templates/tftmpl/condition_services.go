package tftmpl

import (
	"fmt"
	"io"

	"github.com/hashicorp/consul-terraform-sync/logging"
)

var (
	_ Condition = (*ServicesCondition)(nil)
)

// Condition handles appending a run condition's relevant templating for Terraform
// generated files
type Condition interface {
	Monitor
}

// ServicesCondition handles appending templating for the services run condition
// This is the default run condition
type ServicesCondition struct {
	ServicesMonitor
	SourceIncludesVar bool
}

// SourceIncludesVariable returns true if the variables are to be included
// and false otherwise
func (c ServicesCondition) SourceIncludesVariable() bool {
	return c.SourceIncludesVar
}

func (c ServicesCondition) appendTemplate(w io.Writer) error {
	var err error
	if c.SourceIncludesVariable() {
		err = c.ServicesMonitor.appendTemplate(w)
	} else {
		if c.Regexp == "" {
			return nil
		}
		q := c.hcatQuery()
		_, err = fmt.Fprintf(w, servicesRegexTmpl, q)
	}

	if err != nil {
		logging.Global().Named(logSystemName).Named(tftmplSubsystemName).Error(
			"unable to write service condition template", "error", err)
		return err
	}

	return nil
}
