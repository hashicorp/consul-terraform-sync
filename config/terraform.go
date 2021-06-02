package config

import (
	"fmt"
	"log"
	"os"
	"path"
	"strings"

	ctsVersion "github.com/hashicorp/consul-terraform-sync/version"
	goVersion "github.com/hashicorp/go-version"
)

const (
	// DefaultTFBackendKVPath is the default KV path used for configuring the
	// default backend to use Consul KV.
	DefaultTFBackendKVPath = "consul-terraform-sync/terraform"

	// DefaultTFWorkingDir is the default location where Sync will use as the
	// working directory to manage infrastructure.
	DefaultTFWorkingDir = "sync-tasks"
)

// TerraformConfig is the configuration for the Terraform driver.
type TerraformConfig struct {
	Version           *string                `mapstructure:"version"`
	Log               *bool                  `mapstructure:"log"`
	PersistLog        *bool                  `mapstructure:"persist_log"`
	Path              *string                `mapstructure:"path"`
	WorkingDir        *string                `mapstructure:"working_dir"`
	Backend           map[string]interface{} `mapstructure:"backend"`
	RequiredProviders map[string]interface{} `mapstructure:"required_providers"`
}

// DefaultTerraformConfig returns the default configuration struct.
func DefaultTerraformConfig() *TerraformConfig {
	wd, err := os.Getwd()
	if err != nil {
		log.Println("[ERR] (config) unable to retrieve current working directory " +
			"to setup default configuration for the Terraform driver")
		log.Panic(err)
	}

	return &TerraformConfig{
		Log:               Bool(false),
		PersistLog:        Bool(false),
		Path:              String(wd),
		WorkingDir:        String(path.Join(wd, DefaultTFWorkingDir)),
		Backend:           make(map[string]interface{}),
		RequiredProviders: make(map[string]interface{}),
	}
}

// DefaultTerraformBackend returns the default configuration to Consul KV.
func DefaultTerraformBackend(consul *ConsulConfig) (map[string]interface{}, error) {
	if consul == nil {
		return nil, fmt.Errorf("Consul is not configured to set the default backend for Terraform")
	}

	kvPath := DefaultTFBackendKVPath // "consul-terraform-sync/terraform-env:<task-name>"
	if consul.KVPath != nil && *consul.KVPath != "" {
		// Terraform Consul backend will append "-env:workspace" to the configured
		// path. This modifies the KV path for CTS to have the same KV path structure for
		// configured paths: "<configured-path>/terraform-env:<task-name>"
		if !strings.HasSuffix(*consul.KVPath, "/terraform") {
			kvPath = path.Join(*consul.KVPath, "terraform")
		} else {
			kvPath = *consul.KVPath
		}
	}

	backend := map[string]interface{}{
		"address": *consul.Address,
		"path":    kvPath,
		"gzip":    true,
	}

	if consul.TLS != nil && *consul.TLS.Enabled {
		backend["scheme"] = "https"

		if caCert := consul.TLS.CACert; *caCert != "" {
			backend["ca_file"] = *caCert
		}

		if cert := consul.TLS.Cert; *cert != "" {
			backend["cert_file"] = *cert
		}

		if key := consul.TLS.Key; *key != "" {
			backend["key_file"] = *key
		}
	}

	return map[string]interface{}{"consul": backend}, nil
}

// Copy returns a deep copy of this configuration.
func (c *TerraformConfig) Copy() *TerraformConfig {
	if c == nil {
		return nil
	}

	var o TerraformConfig

	if c.Version != nil {
		o.Version = StringCopy(c.Version)
	}

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

	if o.Version != nil {
		r.Version = StringCopy(o.Version)
	}

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

	if o.Backend != nil {
		for k, v := range o.Backend {
			if r.Backend == nil {
				r.Backend = make(map[string]interface{})
				r.Backend[k] = v
			} else {
				r.Backend[k] = mergeMaps(r.Backend[k].(map[string]interface{}),
					v.(map[string]interface{}))
			}
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
		log.Println("[ERR] (config) unable to retrieve current working directory " +
			"to setup default configuration for the Terraform driver")
		log.Panic(err)
	}

	if c.Version == nil {
		c.Version = String("")
	}

	if c.Log == nil {
		c.Log = Bool(false)
	}

	if c.PersistLog == nil {
		c.PersistLog = Bool(false)
	}

	if c.Path == nil || *c.Path == "" {
		c.Path = String(wd)
	}

	if c.WorkingDir == nil || *c.WorkingDir == "" {
		c.WorkingDir = String(path.Join(wd, DefaultTFWorkingDir))
	}

	if c.Backend == nil {
		c.Backend = make(map[string]interface{})
	}

	if consul != nil {
		defaultBackend, _ := DefaultTerraformBackend(consul)
		if len(c.Backend) == 0 {
			c.Backend = defaultBackend
		} else if b, ok := c.Backend["consul"]; ok {
			c.Backend["consul"] = mergeMaps(defaultBackend["consul"].(map[string]interface{}), b.(map[string]interface{}))
		}
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

	if c.Version != nil && *c.Version != "" {
		v, err := goVersion.NewSemver(*c.Version)
		if err != nil {
			return err
		}

		if len(strings.Split(*c.Version, ".")) < 3 {
			return fmt.Errorf("provide the exact Terraform version to install: %s", *c.Version)
		}

		if !ctsVersion.TerraformConstraint.Check(v) {
			return fmt.Errorf("Terraform version is not supported by Consul "+
				"Terraform Sync, try updating to a different version (%s): %s",
				ctsVersion.CompatibleTerraformVersionConstraint, *c.Version)
		}
	}

	if c.Backend == nil {
		return fmt.Errorf("missing Terraform backend configuration")
	}

	// Backend is only validated for supported backend label. The backend
	// configuration options are verified at run time. The allowed backends
	// for state store have state locking and workspace suppport.
	for k := range c.Backend {
		switch k {
		case "azurerm",
			"consul",
			"cos",
			"gcs",
			"kubernetes",
			"local",
			"manta",
			"pg",
			"s3":
		default:
			return fmt.Errorf("unsupported Terraform backend by Sync %q", k)
		}
	}

	return nil
}

// GoString defines the printable version of this struct.
func (c *TerraformConfig) GoString() string {
	if c == nil {
		return "(*TerraformConfig)(nil)"
	}

	return fmt.Sprintf("&TerraformConfig{"+
		"Version:%s, "+
		"Log:%v, "+
		"PersistLog:%v, "+
		"Path:%s, "+
		"WorkingDir:%s, "+
		"Backend:%+v, "+
		"RequiredProviders:%+v"+
		"}",
		StringVal(c.Version),
		BoolVal(c.Log),
		BoolVal(c.PersistLog),
		StringVal(c.Path),
		StringVal(c.WorkingDir),
		c.Backend,
		c.RequiredProviders,
	)
}

// IsConsulBackend returns if the Terraform backend is using Consul KV for
// remote state store.
func (c *TerraformConfig) IsConsulBackend() bool {
	if c.Backend == nil {
		return false
	}

	_, ok := c.Backend["consul"]
	return ok
}

func mergeMaps(c, o map[string]interface{}) map[string]interface{} {
	r := make(map[string]interface{})
	for k, v := range c {
		r[k] = v
	}

	for k, v := range o {
		r[k] = v
	}

	return r
}
