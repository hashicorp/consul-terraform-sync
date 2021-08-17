package config

import "fmt"

const nodesConditionTypes = "nodes"

type NodesConditionConfig struct {
	Datacenter *string `mapstructure:"datacenter"`

	services []string
}

func (n *NodesConditionConfig) Copy() ConditionConfig {
	if n == nil {
		return nil
	}

	var o NodesConditionConfig
	o.Datacenter = n.Datacenter
	copy(o.services, n.services)

	return &o
}

func (n *NodesConditionConfig) Merge(o ConditionConfig) ConditionConfig {
	if n == nil {
		if isConditionNil(o) {
			return nil
		}
		return o.Copy()
	}

	if isConditionNil(o) {
		return n.Copy()
	}

	r := n.Copy()
	o2, ok := o.(*NodesConditionConfig)
	if !ok {
		return r
	}

	r2 := r.(*NodesConditionConfig)

	if o2.Datacenter != nil {
		r2.Datacenter = StringCopy(o2.Datacenter)
	}

	r2.services = append(r2.services, o2.services...)

	return r2
}

func (n *NodesConditionConfig) Finalize(services []string) {
	if n == nil {
		return
	}

	if n.Datacenter == nil {
		n.Datacenter = String("")
	}

	n.services = services
}

func (n *NodesConditionConfig) Validate() error {
	if n == nil {
		return nil
	}

	if len(n.services) != 0 {
		return fmt.Errorf("task.services cannot be set when using condition %q", nodesConditionTypes)
	}

	return nil
}

func (n *NodesConditionConfig) GoString() string {
	if n == nil {
		return "(*NodesConditionConfig)(nil)"
	}

	return fmt.Sprintf("&NodesConditionConfig{"+
		"Datacenter:%s, "+
		"}",
		StringVal(n.Datacenter),
	)
}
