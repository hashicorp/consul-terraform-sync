package tftmpl

import (
	"fmt"
	"io"

	//"sort"
	"strings"

	//"github.com/hashicorp/consul-terraform-sync/logging"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

var (
	_ Template = (*IntentionsTemplate)(nil)
)

type intentionServices struct {
	Regexp string
	Names  []string
}

type IntentionsTemplate struct {
	Datacenter          string
	Namespace           string
	SourceServices      *intentionServices
	DestinationServices *intentionServices
}

// IsServicesVar returns false because the template returns an intentions
// variable, not a services variable
func (t IntentionsTemplate) IsServicesVar() bool {
	return false
}

func (t IntentionsTemplate) RendersVar() bool {
	return false
}

func (t IntentionsTemplate) appendModuleAttribute(body *hclwrite.Body) {
	body.SetAttributeTraversal("intentions", hcl.Traversal{
		hcl.TraverseRoot{Name: "var"},
		hcl.TraverseAttr{Name: "intentions"},
	})
}

func (t IntentionsTemplate) appendTemplate(w io.Writer) error {
	q := t.hcatQuery()

	if _, err := fmt.Fprintf(w, catalogServicesEmptyTmpl, q); err != nil {
		err = fmt.Errorf("unable to write intentions empty template, error %v", err)
		return err
	}
	return nil
}

func (t IntentionsTemplate) appendVariable(w io.Writer) error {
	return nil
}

// NEED TO DO 1
// sortIntentionsTemplates sorts the services by precedence and then alphabetically
// func (t ServicesTemplate) sortIntentionsTemplates() (string, error) {
// }

// NEED TO DO 2
func (t IntentionsTemplate) hcatQuery() string {
	var opts []string

	if t.Datacenter != "" {
		opts = append(opts, fmt.Sprintf("dc=%s", t.Datacenter))
	}

	if t.Namespace != "" {
		opts = append(opts, fmt.Sprintf("ns=%s", t.Namespace))
	}

	if t.SourceServices != nil {
		if len(t.SourceServices.Names) > 0 {
			ds := strings.Join(t.SourceServices.Names, " ")
			opts = append(opts, fmt.Sprintf("source services names=%s", ds))
		} else {
			opts = append(opts, fmt.Sprintf("source services regexp=%s", t.SourceServices.Regexp))
		}
	}

	if t.DestinationServices != nil {
		if len(t.DestinationServices.Names) > 0 {
			ds := strings.Join(t.DestinationServices.Names, " ")
			opts = append(opts, fmt.Sprintf("destination services names=%s", ds))
		} else {
			opts = append(opts, fmt.Sprintf("destination services regexp=%s", t.DestinationServices.Regexp))
		}
	}

	if len(opts) > 0 {
		return `"` + strings.Join(opts, `" "`) + `"` // add space at end ??
	}
	return ""
}

var intentionsSetVarTmpl = fmt.Sprintf(`
intentions = {%s}
`, intentionsBaseTmpl)

// NEED TO DO 3
const intentionsBaseTmpl = `
{{- with $intentions := intentions %s}}
  {{- range $cs := $intentions }}
  "{{ $cs.Name }}" = {{ HCLServiceTags $cs.Tags }}
{{- end}}{{- end}}
`

const intentionsEmptyTmpl = `
{{- with $intentions := intentions %s}}
  {{- range $cs := $intentions }}
    {{- /* Empty template. Detects changes in Intentions */ -}}
{{- end}}{{- end}}
`
