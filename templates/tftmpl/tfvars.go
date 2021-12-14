package tftmpl

import (
	"fmt"
	"io"
	"sort"

	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

// newTFVarsTmpl writes the hcat services template to a .tfvars file. This is
// used by hcat for and for monitoring service changes from Consul evaluating a
// run condition to trigger a condition.
func newTFVarsTmpl(w io.Writer, filename string, input *RootModuleInputData) error {
	if err := writePreamble(w, input.Task, filename); err != nil {
		return err
	}

	// append templates for Template structs
	servicesAppended := false
	for _, template := range input.Templates {
		if err := template.appendTemplate(w); err != nil {
			return err
		}
		if template.isServicesVar() {
			servicesAppended = true
		}
	}

	// TODO: remove this in pr to convert services list into a type of template
	// separately handle appending templating for services list
	hclFile := hclwrite.NewEmptyFile()
	body := hclFile.Body()
	if len(input.Services) > 0 {
		appendRawServiceTemplateValues(body, input.Services)
		servicesAppended = true
	}

	// services var is required (see newVariablesTF). in variables.tf, services
	// var is always appended. ensure that corresponding var value is appended
	// to terraform.tfvars
	if !servicesAppended {
		// append empty services var value
		body.AppendNewline()
		body.SetAttributeRaw("services", hclwrite.Tokens{
			{Type: hclsyntax.TokenOBrace, Bytes: []byte("{\n}")}})
	}

	_, err := hclFile.WriteTo(w)
	return err
}

// newProvidersTFVars writes input variables for configured Terraform providers.
func newProvidersTFVars(w io.Writer, filename string, input *RootModuleInputData) error {
	err := writePreamble(w, input.Task, filename)
	if err != nil {
		return err
	}

	hclFile := hclwrite.NewEmptyFile()
	body := hclFile.Body()
	body.AppendNewline()

	lastIdx := len(input.Providers) - 1
	for i, p := range input.Providers {
		obj := p.ObjectVal()
		body.SetAttributeValue(p.Name, *obj)
		if i != lastIdx {
			body.AppendNewline()
		}
	}

	_, err = hclFile.WriteTo(w)
	return err
}

// newVariablesTFVars writes input variables for configured variables.
func newVariablesTFVars(w io.Writer, filename string, input *RootModuleInputData) error {
	err := writePreamble(w, input.Task, filename)
	if err != nil {
		return err
	}

	hclFile := hclwrite.NewEmptyFile()
	body := hclFile.Body()
	body.AppendNewline()

	// Order the keys so that we are always guaranteed to generate the same file given
	// the same variables
	keys := make([]string, 0, len(input.Variables))
	for k := range input.Variables {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		body.SetAttributeValue(k, input.Variables[k])
	}

	body.AppendNewline()

	_, err = hclFile.WriteTo(w)
	return err
}

// appendRawServiceTemplateValues appends raw lines representing blocks that
// assign value to the services variable `VariableServices` with `hcat` template
// syntax for dynamic rendering of Consul dependency values.
//
// services = {
//   <service>: {
//	   {{ <template syntax> }}
//   },
// }
func appendRawServiceTemplateValues(body *hclwrite.Body, services []Service) {
	tokens := make([]*hclwrite.Token, 0, len(services)+2)
	tokens = append(tokens, &hclwrite.Token{
		Type:  hclsyntax.TokenOBrace,
		Bytes: []byte("{"),
	})

	for _, s := range services {
		rawService := fmt.Sprintf(serviceBaseTmpl, s.hcatQuery())
		token := hclwrite.Token{
			Type:  hclsyntax.TokenNil,
			Bytes: []byte(rawService),
		}
		tokens = append(tokens, &token)
	}

	tokens = append(tokens, &hclwrite.Token{
		Bytes: []byte("\n}"),
	})
	body.AppendNewline()
	body.SetAttributeRaw("services", tokens)
}

// serviceBaseTmpl is the raw template following hcat syntax for addresses of
// Consul services.
const serviceBaseTmpl = `
{{- with $srv := service %s }}
  {{- range $s := $srv}}
  "{{ joinStrings "." .ID .Node .Namespace .NodeDatacenter }}" = {
{{ HCLService $s | indent 4 }}
  },
  {{- end}}
{{- end}}`
