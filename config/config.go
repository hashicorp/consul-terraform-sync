package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/hashicorp/consul-terraform-sync/internal/decode"
	"github.com/hashicorp/consul-terraform-sync/logging"
	"github.com/hashicorp/consul-terraform-sync/templates/hcltmpl"
	"github.com/hashicorp/hcl"
	"github.com/mitchellh/mapstructure"
)

const (
	// DefaultLogLevel is the default logging level.
	DefaultLogLevel = "INFO"

	// DefaultPort is the default port to use for api server.
	DefaultPort = 8558

	// DefaultWorkingDir is the default location where CTS will manage
	// artifacts generated for each task. By default, a child directory is
	// created for each task with its task name.
	DefaultWorkingDir = "sync-tasks"

	filePathLogKey = "file_path"
)

// Config is used to configure Sync
type Config struct {
	LogLevel   *string `mapstructure:"log_level"`
	ClientType *string `mapstructure:"client_type"`
	Port       *int    `mapstructure:"port"`
	WorkingDir *string `mapstructure:"working_dir"`

	Syslog             *SyslogConfig             `mapstructure:"syslog"`
	Consul             *ConsulConfig             `mapstructure:"consul"`
	Vault              *VaultConfig              `mapstructure:"vault"`
	Driver             *DriverConfig             `mapstructure:"driver"`
	Tasks              *TaskConfigs              `mapstructure:"task"`
	DeprecatedServices *ServiceConfigs           `mapstructure:"service"`
	TerraformProviders *TerraformProviderConfigs `mapstructure:"terraform_provider"`
	BufferPeriod       *BufferPeriodConfig       `mapstructure:"buffer_period"`
	TLS                *CTSTLSConfig             `mapstructure:"tls"`
}

// BuildConfig builds a new Config object from the default configuration and
// the list of config files given and returns it after validation.
func BuildConfig(paths []string) (*Config, error) {
	var configCount int
	config := DefaultConfig()
	for _, path := range paths {
		c, err := fromPath(path)
		if err != nil {
			return nil, err
		}

		if c != nil {
			config = config.Merge(c)
			configCount++
		}
	}

	if configCount == 0 {
		return nil, fmt.Errorf("no configuration files found")
	}

	return config, nil
}

// DefaultConfig returns the default configuration struct
func DefaultConfig() *Config {
	consul := DefaultConsulConfig()
	return &Config{
		LogLevel:           String(DefaultLogLevel),
		Syslog:             DefaultSyslogConfig(),
		Port:               Int(DefaultPort),
		Consul:             consul,
		Driver:             DefaultDriverConfig(),
		Tasks:              DefaultTaskConfigs(),
		DeprecatedServices: DefaultServiceConfigs(),
		TerraformProviders: DefaultTerraformProviderConfigs(),
		BufferPeriod:       DefaultBufferPeriodConfig(),
		TLS:                DefaultCTSTLSConfig(),
	}
}

// Copy returns a deep copy of the current configuration. This is useful because
// the nested data structures may be shared.
func (c *Config) Copy() *Config {
	if c == nil {
		return nil
	}

	return &Config{
		LogLevel:           StringCopy(c.LogLevel),
		Syslog:             c.Syslog.Copy(),
		Port:               IntCopy(c.Port),
		WorkingDir:         StringCopy(c.WorkingDir),
		Consul:             c.Consul.Copy(),
		Vault:              c.Vault.Copy(),
		Driver:             c.Driver.Copy(),
		Tasks:              c.Tasks.Copy(),
		DeprecatedServices: c.DeprecatedServices.Copy(),
		TerraformProviders: c.TerraformProviders.Copy(),
		BufferPeriod:       c.BufferPeriod.Copy(),
		TLS:                c.TLS.Copy(),
	}
}

// Merge combines all values in this configuration with the values in the other
// configuration, with values in the other configuration taking precedence.
// Maps and slices are merged, most other values are overwritten. Complex
// structs define their own merge functionality.
func (c *Config) Merge(o *Config) *Config {
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

	if o.Port != nil {
		r.Port = IntCopy(o.Port)
	}

	if o.WorkingDir != nil {
		r.WorkingDir = StringCopy(o.WorkingDir)
	}

	if o.Syslog != nil {
		r.Syslog = r.Syslog.Merge(o.Syslog)
	}

	if o.Consul != nil {
		r.Consul = r.Consul.Merge(o.Consul)
	}

	if o.Vault != nil {
		r.Vault = r.Vault.Merge(o.Vault)
	}

	if o.Driver != nil {
		r.Driver = r.Driver.Merge(o.Driver)
	}

	if o.Tasks != nil {
		r.Tasks = r.Tasks.Merge(o.Tasks)
	}

	if o.DeprecatedServices != nil {
		r.DeprecatedServices = r.DeprecatedServices.Merge(o.DeprecatedServices)
	}

	if o.TerraformProviders != nil {
		r.TerraformProviders = r.TerraformProviders.Merge(o.TerraformProviders)
	}

	if o.BufferPeriod != nil {
		r.BufferPeriod = r.BufferPeriod.Merge(o.BufferPeriod)
	}

	if o.TLS != nil {
		r.TLS = r.TLS.Merge(o.TLS)
	}

	return r
}

// Finalize ensures there no nil pointers.
func (c *Config) Finalize() {
	if c == nil {
		return
	}

	if c.Port == nil {
		c.Port = Int(DefaultPort)
	}

	if c.ClientType == nil {
		c.ClientType = String("")
	}

	if c.Syslog == nil {
		c.Syslog = DefaultSyslogConfig()
	}
	c.Syslog.Finalize()

	if c.Consul == nil {
		c.Consul = DefaultConsulConfig()
	}
	c.Consul.Finalize()

	if c.Vault == nil {
		c.Vault = DefaultVaultConfig()
	}
	c.Vault.Finalize()

	// Finalize driver after Consul to configure the default driver if needed
	if c.Driver == nil {
		c.Driver = DefaultDriverConfig()
	}
	if c.Driver.consul == nil {
		c.Driver.consul = c.Consul
	}
	c.Driver.Finalize()

	// global working directory and buffer period must be finalized before
	// resolving task configs
	if c.WorkingDir == nil {
		c.WorkingDir = String(DefaultWorkingDir)
	}

	if c.BufferPeriod == nil {
		c.BufferPeriod = DefaultBufferPeriodConfig()
	}
	c.BufferPeriod.Finalize(DefaultBufferPeriodConfig())

	if c.Tasks == nil {
		c.Tasks = DefaultTaskConfigs()
	}
	c.Tasks.Finalize(c.BufferPeriod, *c.WorkingDir)

	if c.DeprecatedServices == nil {
		c.DeprecatedServices = DefaultServiceConfigs()
	}
	if len(*c.DeprecatedServices) > 0 {
		logger := logging.Global().Named(logSystemName).Named(taskSubsystemName)
		logger.Warn(serviceBlockLogMsg)
	}
	c.DeprecatedServices.Finalize()

	if c.TerraformProviders == nil {
		c.TerraformProviders = DefaultTerraformProviderConfigs()
	}
	c.TerraformProviders.Finalize()

	if c.TLS == nil {
		c.TLS = DefaultCTSTLSConfig()
	}
	c.TLS.Finalize()
}

// Validate validates the values and nested values of the configuration struct
func (c *Config) Validate() error {
	if c == nil {
		return fmt.Errorf("missing required configuration")
	}

	if err := c.Driver.Validate(); err != nil {
		return err
	}

	if err := c.Tasks.Validate(); err != nil {
		return err
	}

	if err := c.DeprecatedServices.Validate(); err != nil {
		return err
	}

	if err := c.TerraformProviders.Validate(); err != nil {
		return err
	}

	if err := c.BufferPeriod.Validate(); err != nil {
		return err
	}

	if err := c.validateTaskProvider(); err != nil {
		return err
	}

	if err := c.validateDynamicConfigs(); err != nil {
		return err
	}

	if err := c.TLS.Validate(); err != nil {
		return err
	}

	return nil
}

// validateTaskProvider checks that task <-> provider relations are good
func (c *Config) validateTaskProvider() error {
	// which providers have auto_commit enabled
	acProviders := make(map[string]bool, len(*c.TerraformProviders))
	for _, p := range *c.TerraformProviders {
		for name, values := range *p {
			if a, ok := values.(map[string]interface{})["alias"]; ok {
				name = fmt.Sprintf("%s.%s", name, a)
			}
			if v, ok := values.(map[string]interface{})["auto_commit"]; ok {
				if b, ok := v.(bool); ok {
					acProviders[name] = b
				}
			}
		}
	}

	autocomUsed := make(map[string]bool)
	for _, t := range *c.Tasks {
		for _, pname := range t.Providers {
			if autocom := acProviders[pname]; autocom {
				if autocomUsed[pname] {
					return fmt.Errorf("provider with autocommit (%s)"+
						" cannot be used by more than one task", pname)
				}
				autocomUsed[pname] = true
			}
		}
	}
	return nil
}

// GoString defines the printable version of this struct.
func (c *Config) GoString() string {
	if c == nil {
		return "(*Config)(nil)"
	}

	return fmt.Sprintf("&Config{"+
		"LogLevel:%s, "+
		"Port:%d, "+
		"WorkingDir:%s, "+
		"Syslog:%s, "+
		"Consul:%s, "+
		"Vault:%s, "+
		"Driver:%s, "+
		"Tasks:%s, "+
		"Services (deprecated):%s, "+
		"TerraformProviders:%s, "+
		"BufferPeriod:%s,"+
		"TLS:%s"+
		"}",
		StringVal(c.LogLevel),
		IntVal(c.Port),
		StringVal(c.WorkingDir),
		c.Syslog.GoString(),
		c.Consul.GoString(),
		c.Vault.GoString(),
		c.Driver.GoString(),
		c.Tasks.GoString(),
		c.DeprecatedServices.GoString(),
		c.TerraformProviders.GoString(),
		c.BufferPeriod.GoString(),
		c.TLS.GoString(),
	)
}

func (c *Config) validateDynamicConfigs() error {
	// If dynamic provider configs contain Vault dependency, verify that Vault is
	// configured.
	if c.Vault != nil && !*c.Vault.Enabled {
		for _, p := range *c.TerraformProviders {
			if hcltmpl.ContainsVaultSecret(fmt.Sprint(*p)) {
				return fmt.Errorf("detected dynamic configuration using Vault: missing Vault configuration")
			}
		}
	}

	// Dynamic configuration is only supported for terraform_provider blocks.
	// Provider blocks are redacted, so using the stringified version of the
	// config to check for templates used elsewhere.
	if hcltmpl.ContainsDynamicTemplate(c.GoString()) {
		return fmt.Errorf("dynamic configuration using template syntax is only supported " +
			"for terraform_provider blocks")
	}

	return nil
}

// decodeConfig attempts to decode bytes based on the provided format and
// returns the resulting Config struct.
func decodeConfig(content []byte, file string) (*Config, error) {
	var raw map[string]interface{}
	var decodeHook mapstructure.DecodeHookFunc
	var err error

	format := fileFormat(file)
	logger := logging.Global().Named(logSystemName)
	switch format {
	case "json":
		err = json.Unmarshal(content, &raw)
		decodeHook = mapstructure.ComposeDecodeHookFunc(
			conditionToTypeFunc(),
			moduleInputToTypeFunc(),
			mapstructure.StringToTimeDurationHookFunc(),
			decode.HookTranslateKeys,
		)
	case "hcl":
		err = hcl.Decode(&raw, string(content))
		decodeHook = mapstructure.ComposeDecodeHookFunc(
			conditionToTypeFunc(),
			moduleInputToTypeFunc(),
			decode.HookWeakDecodeFromSlice,
			mapstructure.StringToTimeDurationHookFunc(),
			decode.HookTranslateKeys)
	default:
		return nil, fmt.Errorf("invalid format: %s", format)
	}
	if err != nil {
		logger.Error("failed to decode config", "file_format", format, "error", err)
		return nil, err
	}

	var config Config
	var md mapstructure.Metadata
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		DecodeHook:       decodeHook,
		WeaklyTypedInput: true,
		ErrorUnused:      false,
		Metadata:         &md,
		Result:           &config,
	})
	if err != nil {
		logger.Debug("mapstructure decoder create failed")
		return nil, err
	}

	if err := decoder.Decode(raw); err != nil {
		logger.Debug("mapstructure decode failed")
		return nil, decodeError(err)
	}

	if err := processUnusedConfigKeys(md, file); err != nil {
		return nil, err
	}

	return &config, nil
}

// fromFile reads the configuration file at the given path and returns a new
// Config struct with the data populated.
func fromFile(path string) (*Config, error) {
	format := fileFormat(path)
	if !supportedFormat(format) {
		return nil, fmt.Errorf("invalid file format: %s", format)
	}

	logger := logging.Global().Named(logSystemName)
	content, err := ioutil.ReadFile(path)
	if err != nil {
		logger.Error("failed reading config file from disk", filePathLogKey, path)
		return nil, err
	}

	config, err := decodeConfig(content, filepath.Base(path))
	if err != nil {
		logger.Error("failed decoding content from file", filePathLogKey, path)
		return nil, err
	}

	return config, nil
}

// fromPath iterates and merges all configuration files in a given directory,
// returning the resulting config.
func fromPath(path string) (*Config, error) {
	// Ensure the given filepath exists
	logger := logging.Global().Named(logSystemName)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		logger.Error("missing file/folder", filePathLogKey, path)
		return nil, err
	}

	// Check if a file was given or a path to a directory
	stat, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	if stat.Mode().IsRegular() {
		// Skip files when we can
		if stat.Size() == 0 || !supportedFormat(fileFormat(path)) {
			return nil, nil
		}
		return fromFile(path)
	}

	if !stat.Mode().IsDir() {
		return nil, fmt.Errorf("unknown filetype %q: %s", stat.Mode().String(), path)
	}

	// Ensure the given filepath has at least one config file
	files, err := ioutil.ReadDir(path)
	if err != nil {
		logger.Error("failed listing directory", filePathLogKey, path)
		return nil, err
	}

	// Create a blank config to merge off of
	var c *Config

	for _, fileInfo := range files {
		// Skip subdirectories
		if fileInfo.IsDir() {
			continue
		}

		// Skip file based on extension before processing
		if !supportedFormat(fileFormat(fileInfo.Name())) {
			continue
		}

		// Parse and merge the config
		newConfig, err := fromFile(filepath.Join(path, fileInfo.Name()))
		if err != nil {
			return nil, err
		}
		c = c.Merge(newConfig)
	}

	return c, nil
}

// fileFormat extracts the file format from the file extension
func fileFormat(path string) string {
	ext := filepath.Ext(path)
	return strings.TrimLeft(ext, ".")
}

// supportedFormat is a helper to determine if the file format is a supported
// configuration type
func supportedFormat(format string) bool {
	if format == "hcl" || format == "json" {
		return true
	}

	return false
}

func stringFromEnv(list []string, def string) *string {
	for _, s := range list {
		if v := os.Getenv(s); v != "" {
			return String(strings.TrimSpace(v))
		}
	}
	return String(def)
}

func stringFromFile(list []string, def string) *string {
	for _, s := range list {
		c, err := ioutil.ReadFile(s)
		if err == nil {
			return String(strings.TrimSpace(string(c)))
		}
	}
	return String(def)
}

func boolFromEnv(list []string, def bool) *bool {
	for _, s := range list {
		if v := os.Getenv(s); v != "" {
			b, err := strconv.ParseBool(v)
			if err == nil {
				return Bool(b)
			}
		}
	}
	return Bool(def)
}

func antiboolFromEnv(list []string, def bool) *bool {
	for _, s := range list {
		if v := os.Getenv(s); v != "" {
			b, err := strconv.ParseBool(v)
			if err == nil {
				return Bool(!b)
			}
		}
	}
	return Bool(def)
}

// serviceBlockLogMsg is the log message for deprecating the `service` block
const serviceBlockLogMsg = `the 'service' block is deprecated ` +
	`in v0.5.0 and will be removed in a future major version after v0.8.0.

` +
	`In order to replace the 'service' block, the associated 'services' field ` +
	`(deprecated) should first be upgraded to 'condition "services"' or ` +
	`'module_input "services"'. Then the configuration in the 'service' block ` +
	`can be set in the 'condition' or 'module_input' block.` +
	`

We will be releasing a tool to help upgrade your configuration for this deprecation.

Example of replacing service block information in condition block:
|  - service {
|  -   name       = "api"
|  -   datacenter = "dc2"
|  - }
|
|    task {
|      condition "services" {
|        names      = ["api"]
|  +     datacenter = "dc2"
|      }
|      ...
|    }

More complex cases with 'service' blocks can require splitting a task into multiple tasks.
For more details and additional examples, please see:
https://consul.io/docs/nia/release-notes/0-5-0#deprecate-service-block
`
