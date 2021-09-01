package tmplfunc

import (
	"strings"

	"github.com/hashicorp/hcat/dep"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

// hclServiceFunc is a wrapper of the template function to marshal Consul
// service information into HCL. The function accepts a map representing
// metadata for services in scope of a task.
func hclServiceFunc(meta ServicesMeta) func(sDep *dep.HealthService) string {
	return func(sDep *dep.HealthService) string {
		if sDep == nil {
			return ""
		}

		// Find metdata based Consul service name scoped to a task to append to
		// that service within var.services
		var serviceMeta map[string]string
		if meta != nil {
			serviceMeta = meta[sDep.Name]
		}

		// Convert the hcat type to an HCL marshal-able object
		s := newHealthService(sDep, serviceMeta)

		f := hclwrite.NewEmptyFile()
		gohcl.EncodeIntoBody(s, f.Body())
		return strings.TrimSpace(string(f.Bytes()))
	}
}

type healthService struct {
	// Consul service information
	ID        string            `hcl:"id"`
	Name      string            `hcl:"name"`
	Kind      string            `hcl:"kind"`
	Address   string            `hcl:"address"`
	Port      int               `hcl:"port"`
	Meta      map[string]string `hcl:"meta"`
	Tags      []string          `hcl:"tags"`
	Namespace string            `hcl:"namespace"`
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
		Namespace: s.Namespace,
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

func nonNullMap(m map[string]string) map[string]string {
	if m == nil {
		return map[string]string{}
	}

	return m
}

func hclNodeFunc(nDep *dep.Node) string {
	if nDep == nil {
		return ""
	}

	n := node{
		ID:              nDep.ID,
		Node:            nDep.Node,
		Address:         nDep.Address,
		Datacenter:      nDep.Datacenter,
		TaggedAddresses: nDep.TaggedAddresses,
		Meta:            nDep.Meta,
	}

	f := hclwrite.NewEmptyFile()
	gohcl.EncodeIntoBody(n, f.Body())
	return strings.TrimSpace(string(f.Bytes()))
}

type node struct {
	ID              string            `hcl:"id"`
	Node            string            `hcl:"node"`
	Address         string            `hcl:"address"`
	Datacenter      string            `hcl:"datacenter"`
	TaggedAddresses map[string]string `hcl:"tagged_addresses"`
	Meta            map[string]string `hcl:"meta"`
}
