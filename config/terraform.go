package config

import (
	"fmt"
	"log"
	"os"
	"path"
)

const (
	// DefaultTFLogLevel is the default log level for the local Terraform process.
	DefaultTFLogLevel = "info"

	// DefaultTFBackendKVPath is the default KV path used for configuring the
	// default backend to use Consul KV.
	DefaultTFBackendKVPath = "consul-nia/terraform"

	// DefaultTFDataDir is the default location where Terraform keeps its working
	// directory.
	DefaultTFDataDir = ".terraform"

	// DefaultTFWorkingDir is the default location where NIA will use as the
	// working directory to manage infrastructure.
	DefaultTFWorkingDir = ".terraform-workspace"
)

// TerraformConfig is the configuration for the Terraform driver.
type TerraformConfig struct {
	LogLevel   *string                `mapstructure:"log_level"`
	Path       *string                `mapstructure:"path"`
	DataDir    *string                `mapstructure:"data_dir"`
	WorkingDir *string                `mapstructure:"working_dir"`
	SkipVerify *bool                  `mapstructure:"skip_verify"`
	Backend    map[string]interface{} `mapstructure:"backend"`
}

// DefaultTerraformConfig returns the default configuration struct.
func DefaultTerraformConfig(consul *ConsulConfig) *TerraformConfig {
	backend, err := DefaultTerraformBackend(consul)
	if err != nil {
		log.Panic(err)
	}

	wd, err := os.Getwd()
	if err != nil {
		log.Println("[ERR] unable to retrieve current working directory to setup " +
			"default configuration for the Terraform driver")
		log.Panic(err)
	}

	return &TerraformConfig{
		LogLevel:   String(DefaultTFLogLevel),
		Path:       String(wd),
		DataDir:    String(path.Join(wd, DefaultTFDataDir)),
		WorkingDir: String(path.Join(wd, DefaultTFWorkingDir)),
		SkipVerify: Bool(false),
		Backend:    backend,
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

	if c.LogLevel != nil {
		o.LogLevel = StringCopy(c.LogLevel)
	}

	if c.Path != nil {
		o.Path = StringCopy(c.Path)
	}

	if c.DataDir != nil {
		o.DataDir = StringCopy(c.DataDir)
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

	if o.LogLevel != nil {
		r.LogLevel = StringCopy(o.LogLevel)
	}

	if o.Path != nil {
		r.Path = StringCopy(o.Path)
	}

	if o.DataDir != nil {
		r.DataDir = StringCopy(o.DataDir)
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

	if c.LogLevel == nil {
		c.LogLevel = String(DefaultTFLogLevel)
	}

	if c.Path == nil {
		c.Path = String(wd)
	}

	if c.DataDir == nil {
		c.DataDir = String(path.Join(wd, DefaultTFDataDir))
	}

	if c.WorkingDir == nil {
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
		"LogLevel:%s, "+
		"Path:%s, "+
		"DataDir:%s, "+
		"WorkingDir:%s, "+
		"SkipVerify:%v, "+
		"Backend:%+v"+
		"}",
		StringVal(c.LogLevel),
		StringVal(c.Path),
		StringVal(c.DataDir),
		StringVal(c.WorkingDir),
		BoolVal(c.SkipVerify),
		c.Backend,
	)
}
