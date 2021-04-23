package tftmpl

import (
	"fmt"
	"io"

	"github.com/hashicorp/hcat/dep"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
)

type healthService struct {
	// Consul service information
	ID        string            `hcl:"id"`
	Name      string            `hcl:"name"`
	Kind      string            `hcl:"kind"`
	Address   string            `hcl:"address"`
	Port      int               `hcl:"port"`
	Meta      map[string]string `hcl:"meta"`
	Tags      []string          `hcl:"tags"`
	Namespace cty.Value         `hcl:"namespace"`
	Status    string            `hcl:"status"`

	// Consul node information for a service
	Node                string            `hcl:"node"`
	NodeID              string            `hcl:"node_id"`
	NodeAddress         string            `hcl:"node_address"`
	NodeDatacenter      string            `hcl:"node_datacenter"`
	NodeTaggedAddresses map[string]string `hcl:"node_tagged_addresses"`
	NodeMeta            map[string]string `hcl:"node_meta"`

	// Added CTS information for a service
	CTSUserDefinedMeta map[string]string `hcl:"cts_user_defined_meta"`
}

func newHealthService(s *dep.HealthService, ctsUserDefinedMeta map[string]string) healthService {
	if s == nil {
		return healthService{}
	}

	// Namespace is null-able
	var namespace cty.Value
	if s.Namespace != "" {
		namespace = cty.StringVal(s.Namespace)
	} else {
		namespace = cty.NullVal(cty.String)
	}

	// Default to empty list instead of null
	tags := []string{}
	if s.Tags != nil {
		tags = s.Tags
	}

	return healthService{
		ID:        s.ID,
		Name:      s.Name,
		Kind:      s.Kind,
		Address:   s.Address,
		Port:      s.Port,
		Meta:      nonNullMap(s.ServiceMeta),
		Tags:      tags,
		Namespace: namespace,
		Status:    s.Status,

		Node:                s.Node,
		NodeID:              s.NodeID,
		NodeAddress:         s.NodeAddress,
		NodeDatacenter:      s.NodeDatacenter,
		NodeTaggedAddresses: nonNullMap(s.NodeTaggedAddresses),
		NodeMeta:            nonNullMap(s.NodeMeta),

		CTSUserDefinedMeta: nonNullMap(ctsUserDefinedMeta),
	}
}

// newTFVarsTmpl writes the hcat services template to a .tfvars file. This is used
// by hcat for monitoring service changes from Consul.
func newTFVarsTmpl(w io.Writer, filename string, input *RootModuleInputData) error {
	err := writePreamble(w, input.Task, filename)
	if err != nil {
		return err
	}

	hclFile := hclwrite.NewEmptyFile()
	body := hclFile.Body()
	appendRawServiceTemplateValues(body, input.Services)

	_, err = hclFile.WriteTo(w)
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
		rawService := fmt.Sprintf(serviceBaseTmpl, s.hcatQuery())

		if i == lastIdx {
			rawService += "\n}"
		}

		token := hclwrite.Token{
			Type:  hclsyntax.TokenNil,
			Bytes: []byte(rawService),
		}
		tokens = append(tokens, &token)
	}
	body.SetAttributeRaw("services", tokens)
}

func nonNullMap(m map[string]string) map[string]string {
	if m == nil {
		return map[string]string{}
	}

	return m
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
