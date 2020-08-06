package tftmpl

import (
	"fmt"
	"io"
	"log"
	"sort"
	"strings"

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
      port    = number
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

	// Format the file before writing
	content := hclFile.Bytes()
	content = hclwrite.Format(content)
	_, err = w.Write(content)
	return err
}

// appendNamedBlockVariable creates an HCL file object that contains the variable
// blocks used by the root module.
func appendNamedBlockVariable(body *hclwrite.Body, block *namedBlock) {
	pBody := body.AppendNewBlock("variable", []string{block.Name}).Body()
	pBody.SetAttributeValue("default", cty.NullVal(*block.ObjectType()))
	pBody.SetAttributeValue("description", cty.StringVal(fmt.Sprintf(
		"Configuration object for %s", block.Name)))

	v := block.ObjectVal()
	rawTypeAttr := fmt.Sprintf("type = %s", variableTypeString(*v, v.Type()))
	pBody.AppendUnstructuredTokens(hclwrite.Tokens{{
		Type:  hclsyntax.TokenNil,
		Bytes: []byte(rawTypeAttr),
	}})
	pBody.AppendNewline()
}

// variableTypeString generates the raw Terraform type strings for supported
// variable types. Collection types are generic and accepts any element type.
// Structural types recursively calls this function to generate the nested
// types. If a type is not supported, "unknown" is returned.
func variableTypeString(val cty.Value, valType cty.Type) string {
	switch {
	case valType.IsPrimitiveType():
		return valType.FriendlyName()

	case valType.IsSetType(), valType.IsListType():
		return "list(any)"

	case valType.IsMapType():
		return "map(any)"

	case valType.IsObjectType():
		// object({ <key> = <type>, <key> = <type>, ... })
		m := val.AsValueMap()
		keys := make([]string, 0, len(m))
		keyTypePairs := make([]string, 0, len(m))

		for k := range m {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			v := m[k]
			keyType := fmt.Sprintf("%s = %s", k, variableTypeString(v, v.Type()))
			keyTypePairs = append(keyTypePairs, keyType)
		}
		return fmt.Sprintf("object({\n%s\n})", strings.Join(keyTypePairs, "\n"))

	case valType.IsTupleType():
		// tuple([<type>, <type>, ...])
		types := valType.TupleElementTypes()
		typeStrings := make([]string, len(types))
		for i, t := range types {
			typeStrings[i] = variableTypeString(cty.NullVal(t), t)
		}
		return fmt.Sprintf("tuple([%s])", strings.Join(typeStrings, ", "))
	}

	return "unknown"
}
