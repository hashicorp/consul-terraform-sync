package tftmpl

import (
	"fmt"
	"io"
	"log"

	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

// NewTFVarsTmpl writes content to assign values to the root module's variables
// that is commonly placed in a .tfvars file.
func NewTFVarsTmpl(w io.Writer, input *RootModuleInputData) error {
	_, err := w.Write(RootPreamble)
	if err != nil {
		// This isn't required for TF config files to be usable. So we'll just log
		// the error and continue.
		log.Printf("[WARN] (templates.tftmpl) unable to write preamble warning to %q",
			TFVarsTmplFilename)
	}

	hclFile := hclwrite.NewEmptyFile()
	body := hclFile.Body()
	appendNamedBlockValues(body, input.providers)
	body.AppendNewline()
	appendRawServiceTemplateValues(body, input.services)

	_, err = hclFile.WriteTo(w)
	return err
}

// appendNamedBlockValues appends blocks that assign value to the named
// variable blocks genernated by `appendNamedBlockVariable`
func appendNamedBlockValues(body *hclwrite.Body, blocks []*namedBlock) {
	lastIdx := len(blocks) - 1
	for i, b := range blocks {
		obj := b.ObjectVal()
		body.SetAttributeValue(b.Name, *obj)
		if i != lastIdx {
			body.AppendNewline()
		}
	}
}

// appendRawServiceTemplateValues appends raw lines representing blocks that
// assign value to the services variable `VariableServices` with `hcat` template
// syntax for dynamic rendering of Consul dependency values.
//
// services = {
//   <service>: {
//	   <attr> = <value>
//     <attr> = {{ <template syntax> }}
//   }
// }
func appendRawServiceTemplateValues(body *hclwrite.Body, services []*Service) {
	if len(services) == 0 {
		return
	}

	tokens := make([]*hclwrite.Token, 0, len(services)+2)
	tokens = append(tokens, &hclwrite.Token{
		Type:  hclsyntax.TokenOBrace,
		Bytes: []byte("{"),
	})
	lastIdx := len(services) - 1
	for i, s := range services {
		rawService := fmt.Sprintf(baseAddressStr, s.TemplateServiceID())

		if i == lastIdx {
			rawService += "\n}"
		} else {
			rawService += ","
		}

		token := hclwrite.Token{
			Type:  hclsyntax.TokenNil,
			Bytes: []byte(rawService),
		}
		tokens = append(tokens, &token)
	}
	body.SetAttributeRaw("services", tokens)
}

// baseAddressStr is the raw template following hcat syntax for addresses of
// Consul services.
const baseAddressStr = `
{{- with $srv := service "%s"}}
  {{- $last := len $srv | subtract 1}}
  {{- range $i, $s := $srv}}
  "{{.ID}}" : {
    id              = "{{.ID}}"
    name            = "{{.Name}}"
    address         = "{{.Address}}"
    port            = {{.Port}}
    meta            = {{hclStringMap .ServiceMeta 3}}
    tags            = {{hclStringList .Tags}}
    namespace       = {{hclString .Namespace}}
    status          = "{{.Status}}"
    node            = "{{.Node}}"
    node_id         = "{{.NodeID}}"
    node_address    = "{{.NodeAddress}}"
    node_datacenter = "{{.NodeDatacenter}}"
    node_tagged_addresses = {{hclStringMap .NodeTaggedAddresses 3}}
    node_meta = {{hclStringMap .NodeMeta 3}}
  }{{if (ne $i $last)}},{{- end}}
  {{- end}}
{{- end}}`
