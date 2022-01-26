package tftmpl

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/hashicorp/consul-terraform-sync/templates/hcltmpl"
	goVersion "github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
)

// tfVersionSensitive is the first version to support the sensitive argument
// for variables
var tfVersionSensitive = goVersion.Must(goVersion.NewSemver("0.14.0"))

// VariableServices is versioned to track compatibility with the generated
// root module with modules.
var VariableServices = []byte(`
# Service definition protocol v0
variable "services" {
  description = "Consul services monitored by Consul Terraform Sync"
  type = map(
    object({
      id        = string
      name      = string
      kind      = string
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

// newVariablesTF writes variable definitions to a file. This includes the
// required services variable and generated provider variables based on CTS
// user configuration for the task.
func newVariablesTF(w io.Writer, filename string, input *RootModuleInputData) error {
	err := writePreamble(w, input.Task, filename)
	if err != nil {
		return err
	}

	// service variable is required to append
	if _, err = w.Write(VariableServices); err != nil {
		return err
	}

	// append a variable for each template unless template's variable is
	// a services variable. services variable already appended above.
	// note: assumes templates' variables are unique type. otherwise would
	// need to check to avoid appending duplicate variables
	for _, template := range input.Templates {
		if template.RendersVar() {
			if template.IsServicesVar() {
				// services variable is already appended earlier. skip
				continue
			}

			// append variable for non-service objects
			if err = template.appendVariable(w); err != nil {
				return err
			}
		}
	}

	hclFile := hclwrite.NewEmptyFile()
	rootBody := hclFile.Body()
	for _, p := range input.Providers {
		rootBody.AppendNewline()
		appendNamedBlockVariable(rootBody, p, input.TerraformVersion, true)
	}

	// Format the file before writing
	content := hclFile.Bytes()
	content = hclwrite.Format(content)
	_, err = w.Write(content)
	return err
}

// appendNamedBlockVariable creates an HCL file object that contains the variable
// blocks used by the root module.
func appendNamedBlockVariable(body *hclwrite.Body, block hcltmpl.NamedBlock,
	tfVersion *goVersion.Version, sensitive bool) {
	pBody := body.AppendNewBlock("variable", []string{block.Name}).Body()
	pBody.SetAttributeValue("default", cty.NullVal(*block.ObjectType()))
	pBody.SetAttributeValue("description", cty.StringVal(fmt.Sprintf(
		"Configuration object for %s", block.Name)))

	if sensitive {
		if tfVersion != nil && tfVersionSensitive.LessThanOrEqual(tfVersion) {
			pBody.SetAttributeValue("sensitive", cty.BoolVal(true))
		}
	}

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
