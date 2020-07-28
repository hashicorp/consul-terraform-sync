package tftmpl

import (
	"fmt"
	"io"
	"log"
	"sort"

	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
)

// VariableServices is versioned to track compatibility with the generated
// root module with modules.
var VariableServices = []byte(
	`# Service definition protocol v0
variable "services" {
	description = "Consul services monitored by Consul NIA (protocol v0)"
	type = map(object({
		# Name of the service
		name = string
		# Description of the service
		description = string
		# List of addresses for instances of the service by IP and port
		addresses = list(object({
			address = string
			port = number
		}))
	}))
}
`)

// NewVariablesTF writes content used for variables.tf of a Terraform root
// module.
func NewVariablesTF(w io.Writer, input *RootModuleInputData) error {
	_, err := w.Write(RootPreamble)
	if err != nil {
		// This isn't required for TF config files to be usable. So we'll just log
		// the error and continue.
		log.Printf("[WARN] (templates.tftmpl) unable to write preamble warning to %q",
			RootFilename)
	}

	_, err = w.Write(VariableServices)
	if err != nil {
		return err
	}

	hclFile := hclwrite.NewEmptyFile()
	rootBody := hclFile.Body()
	rootBody.AppendNewline()
	lastIdx := len(input.providers) - 1
	for i, p := range input.providers {
		appendNamedBlockVariable(rootBody, p)
		if i != lastIdx {
			rootBody.AppendNewline()
		}
	}

	_, err = hclFile.WriteTo(w)
	return err
}

// appendNamedBlockVariable creates an HCL file object that contains the variable
// blocks used by the root module.
func appendNamedBlockVariable(body *hclwrite.Body, block *namedBlock) {
	pBody := body.AppendNewBlock("variable", []string{block.Name}).Body()
	pBody.SetAttributeValue("default", cty.NullVal(*block.ObjectType()))
	pBody.SetAttributeValue("description", cty.StringVal(fmt.Sprintf(
		"Configuration object for %s", block.Name)))
	appendRawObjType(pBody, "type", block.ObjectVal())
}

// appendRawObjType appends a raw line containing a Terraform configuration
// variable type for an object with dynamic attributes.
//
// type = object({ <attr> = <type> })
func appendRawObjType(b *hclwrite.Body, attr string, val *cty.Value) {
	if !val.Type().IsObjectType() {
		return
	}

	line := fmt.Sprintf("\t%s = object({\n", attr)
	m := val.AsValueMap()
	attrs := make([]string, 0, len(m))
	for attr := range m {
		attrs = append(attrs, attr)
	}
	sort.Strings(attrs)
	for _, attr := range attrs {
		v := m[attr]
		line += fmt.Sprintf("\t\t%s = %s\n", attr, v.Type().FriendlyName())
	}
	line += "\t})\n"

	b.AppendUnstructuredTokens(hclwrite.Tokens{
		{
			Type:  hclsyntax.TokenNil,
			Bytes: []byte(line),
		},
	})
}
