package tftmpl

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/hashicorp/consul-terraform-sync/templates/hcltmpl"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
)

// VariableServices is versioned to track compatibility with the generated
// root module with modules.
var VariableServices = []byte(
	`# Service definition protocol v0
variable "services" {
  description = "Consul services monitored by Consul Terraform Sync"
  type = map(
    object({
      id        = string
      name      = string
      address   = string
      port      = number
      meta      = map(string)
      tags      = list(string)
      namespace = string
      status    = string

      node                  = string
      node_id               = string
      node_address          = string
      node_datacenter       = string
      node_tagged_addresses = map(string)
      node_meta             = map(string)

      cts_user_defined_meta = map(string)
    })
  )
}
`)

// newVariablesTF writes content used for variables.tf of a Terraform root
// module.
func newVariablesTF(w io.Writer, input *RootModuleInputData) error {
	err := writePreamble(w, input.Task, VarsFilename)
	if err != nil {
		return err
	}

	_, err = w.Write(VariableServices)
	if err != nil {
		return err
	}

	hclFile := hclwrite.NewEmptyFile()
	rootBody := hclFile.Body()
	rootBody.AppendNewline()
	lastIdx := len(input.Providers) - 1
	for i, p := range input.Providers {
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
func appendNamedBlockVariable(body *hclwrite.Body, block hcltmpl.NamedBlock) {
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

	case valType.IsListType():
		return "list(any)"

	case valType.IsSetType():
		return "set(any)"

	case valType.IsMapType():
		return "map(any)"

	case valType.IsTupleType():
		// tuple([<type>, <type>, ...])
		types := valType.TupleElementTypes()
		typeStrings := make([]string, len(types))
		for i, t := range types {
			typeStrings[i] = variableTypeString(cty.NullVal(t), t)
		}
		return fmt.Sprintf("tuple([%s])", strings.Join(typeStrings, ", "))

	case valType.IsObjectType():
		// object({ <key> = <type>, <key> = <type>, ... })

		if !val.Type().IsObjectType() {
			// This is an unlikely edge case where the value does not match the type
			return "object({})"
		}

		m := val.AsValueMap()
		if len(m) == 0 {
			return "object({})"
		}

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
	}

	return "unknown"
}
