package config

import (
	"fmt"
	"log"
	"os"
	"path"
)

const (
	// DefaultTFBackendKVPath is the default KV path used for configuring the
	// default backend to use Consul KV.
	DefaultTFBackendKVPath = "consul-nia/terraform"

	// DefaultTFWorkingDir is the default location where NIA will use as the
	// working directory to manage infrastructure.
	DefaultTFWorkingDir = "nia-tasks"
)

// TerraformConfig is the configuration for the Terraform driver.
type TerraformConfig struct {
	Log               *bool                  `mapstructure:"log"`
	PersistLog        *bool                  `mapstructure:"persist_log"`
	Path              *string                `mapstructure:"path"`
	WorkingDir        *string                `mapstructure:"working_dir"`
	SkipVerify        *bool                  `mapstructure:"skip_verify"`
	Backend           map[string]interface{} `mapstructure:"backend"`
	RequiredProviders map[string]interface{} `mapstructure:"required_providers"`
}

// DefaultTerraformConfig returns the default configuration struct.
func DefaultTerraformConfig() *TerraformConfig {
	wd, err := os.Getwd()
	if err != nil {
		log.Println("[ERR] unable to retrieve current working directory to setup " +
			"default configuration for the Terraform driver")
		log.Panic(err)
	}

	return &TerraformConfig{
		Log:               Bool(false),
		PersistLog:        Bool(false),
		Path:              String(wd),
		WorkingDir:        String(path.Join(wd, DefaultTFWorkingDir)),
		SkipVerify:        Bool(false),
		Backend:           make(map[string]interface{}),
		RequiredProviders: make(map[string]interface{}),
	}
}

// DefaultTerraformBackend returns the default configuration to Consul KV.
func DefaultTerraformBackend(consul *ConsulConfig) (map[string]interface{}, error) {
	if consul == nil {
		return nil, fmt.Errorf("Consul is not configured to set the default backend for Terraform")
	}

	kvPath := DefaultTFBackendKVPath
	if consul.KVPath != nil && *consul.KVPath != "" {
		kvPath = path.Join(*consul.KVPath, "terraform")
	}

	backend := map[string]interface{}{
		"address": *consul.Address,
		"path":    kvPath,
		"gzip":    true,
	}

	return map[string]interface{}{"consul": backend}, nil
}

// Copy returns a deep copy of this configuration.
func (c *TerraformConfig) Copy() *TerraformConfig {
	if c == nil {
		return nil
	}

	var o TerraformConfig

	if c.Log != nil {
		o.Log = BoolCopy(c.Log)
	}

	if c.PersistLog != nil {
		o.PersistLog = BoolCopy(c.PersistLog)
	}

	if c.Path != nil {
		o.Path = StringCopy(c.Path)
	}

	if c.WorkingDir != nil {
		o.WorkingDir = StringCopy(c.WorkingDir)
	}

	if c.SkipVerify != nil {
		o.SkipVerify = BoolCopy(c.SkipVerify)
	}

	if c.Backend != nil {
		o.Backend = make(map[string]interface{})
		for k, v := range c.Backend {
			o.Backend[k] = v
		}
	}

	if c.RequiredProviders != nil {
		o.RequiredProviders = make(map[string]interface{})
		for k, v := range c.RequiredProviders {
			o.RequiredProviders[k] = v
		}
	}

	return &o
}

// Merge combines all values in this configuration with the values in the other
// configuration, with values in the other configuration taking precedence.
// Maps and slices are merged, most other values are overwritten. Complex
// structs define their own merge functionality.
func (c *TerraformConfig) Merge(o *TerraformConfig) *TerraformConfig {
	if c == nil {
		if o == nil {
			return nil
		}
		return o.Copy()
	}

	if o == nil {
		return c.Copy()
	}

	r := c.Copy()

	if o.Log != nil {
		r.Log = BoolCopy(o.Log)
	}

	if o.PersistLog != nil {
		r.PersistLog = BoolCopy(o.PersistLog)
	}

	if o.Path != nil {
		r.Path = StringCopy(o.Path)
	}

	if o.WorkingDir != nil {
		r.WorkingDir = StringCopy(o.WorkingDir)
	}

	if o.SkipVerify != nil {
		r.SkipVerify = BoolCopy(o.SkipVerify)
	}

	if o.Backend != nil {
		for k, v := range o.Backend {
			if r.Backend == nil {
				r.Backend = make(map[string]interface{})
			}
			r.Backend[k] = v
		}
	}

	if o.RequiredProviders != nil {
		for k, v := range o.RequiredProviders {
			if r.RequiredProviders == nil {
				r.RequiredProviders = make(map[string]interface{})
			}
			r.RequiredProviders[k] = v
		}
	}

	return r
}

// Finalize ensures there no nil pointers.
func (c *TerraformConfig) Finalize(consul *ConsulConfig) {
	if c == nil {
		return
	}

	wd, err := os.Getwd()
	if err != nil {
		log.Println("[ERR] unable to retrieve current working directory to setup " +
			"default configuration for the Terraform driver")
		log.Panic(err)
	}

	if c.Log == nil {
		c.Log = Bool(false)
	}

	if c.PersistLog == nil {
		c.PersistLog = Bool(false)
	}

	if c.Path == nil {
		c.Path = String(wd)
	}

	if c.WorkingDir == nil || *c.WorkingDir == "" {
		c.WorkingDir = String(path.Join(wd, DefaultTFWorkingDir))
	}

	if c.SkipVerify == nil {
		c.SkipVerify = Bool(false)
	}

	if c.Backend == nil {
		c.Backend = make(map[string]interface{})
	}

	if len(c.Backend) == 0 && consul != nil {
		c.Backend, _ = DefaultTerraformBackend(consul)
	}

	if c.RequiredProviders == nil {
		c.RequiredProviders = make(map[string]interface{})
	}
}

// Validate validates the values and nested values of the configuration struct
func (c *TerraformConfig) Validate() error {
	if c == nil {
		return fmt.Errorf("missing Terraform driver configuration")
	}

	if c.Backend == nil {
		return fmt.Errorf("missing Terraform backend configuration")
	}

	for k := range c.Backend {
		if k != "consul" {
			return fmt.Errorf("unsupported Terraform backend by NIA %q", k)
		}
	}

	return nil
}

func (c *TerraformConfig) GoString() string {
	if c == nil {
		return "(*TerraformConfig)(nil)"
	}

	return fmt.Sprintf("&TerraformConfig{"+
		"Log:%v, "+
		"PersistLog:%v, "+
		"Path:%s, "+
		"WorkingDir:%s, "+
		"SkipVerify:%v, "+
		"Backend:%+v, "+
		"RequiredProviders:%+v"+
		"}",
		BoolVal(c.Log),
		BoolVal(c.PersistLog),
		StringVal(c.Path),
		StringVal(c.WorkingDir),
		BoolVal(c.SkipVerify),
		c.Backend,
		c.RequiredProviders,
	)
}
