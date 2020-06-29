package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/consul/lib/decode"
	"github.com/hashicorp/hcl"
	"github.com/mitchellh/mapstructure"
)

const (
	// DefaultLogLevel is the default logging level.
	DefaultLogLevel = "WARN"
)

// Config is used to configure Consul NIA
type Config struct {
	LogLevel    *string `mapstructure:"log_level"`
	InspectMode *bool   `mapstructure:"inspect_mode"`

	Syslog *SyslogConfig `mapstructure:"syslog"`
	Consul *ConsulConfig `mapstructure:"consul"`
	Driver *DriverConfig `mapstructure:"driver"`
}

// BuildConfig builds a new Config object from the default configuration and
// the list of config files given and returns it after validation.
func BuildConfig(paths []string) (*Config, error) {
	config := DefaultConfig()
	for _, path := range paths {
		c, err := fromPath(path)
		if err != nil {
			return nil, err
		}

		config = config.Merge(c)
	}

	return config, nil
}

// DefaultConfig returns the default configuration struct
func DefaultConfig() *Config {
	consul := DefaultConsulConfig()
	return &Config{
		LogLevel:    String(DefaultLogLevel),
		InspectMode: Bool(false),
		Syslog:      DefaultSyslogConfig(),
		Consul:      consul,
		Driver:      DefaultDriverConfig(consul),
	}
}

// Copy returns a deep copy of the current configuration. This is useful because
// the nested data structures may be shared.
func (c *Config) Copy() *Config {
	if c == nil {
		return nil
	}

	return &Config{
		LogLevel:    StringCopy(c.LogLevel),
		InspectMode: BoolCopy(c.InspectMode),
		Syslog:      c.Syslog.Copy(),
		Consul:      c.Consul.Copy(),
		Driver:      c.Driver.Copy(),
	}
}

// Merge merges the values in config into this config object. Values in the
// config object overwrite the values in c.
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

	if o.InspectMode != nil {
		r.InspectMode = BoolCopy(o.InspectMode)
	}

	if o.Syslog != nil {
		r.Syslog = r.Syslog.Merge(o.Syslog)
	}

	if o.Consul != nil {
		r.Consul = r.Consul.Merge(o.Consul)
	}

	if o.Driver != nil {
		r.Driver = r.Driver.Merge(o.Driver)
	}

	return r
}

func (c *Config) Finalize() {
	if c == nil {
		return
	}

	if c.Syslog == nil {
		c.Syslog = DefaultSyslogConfig()
	}
	c.Syslog.Finalize()

	if c.Consul == nil {
		c.Consul = DefaultConsulConfig()
	}
	c.Consul.Finalize()

	if c.Driver == nil {
		c.Driver = DefaultDriverConfig(c.Consul)
	}
	c.Driver.Finalize()
}

func (c *Config) Validate() error {
	if c == nil {
		return nil
	}

	if err := c.Driver.Validate(); err != nil {
		return err
	}

	return nil
}

func (c *Config) GoString() string {
	if c == nil {
		return "(*Config)(nil)"
	}

	return fmt.Sprintf("&Config{"+
		"LogLevel:%s, "+
		"InspectMode:%#v, "+
		"Syslog:%s, "+
		"Consul:%s, "+
		"Driver:%s"+
		"}",
		StringVal(c.LogLevel),
		BoolVal(c.InspectMode),
		c.Syslog.GoString(),
		c.Consul.GoString(),
		c.Driver.GoString(),
	)
}

// decodeConfig attempts to decode bytes based on the provided format and
// returns the resulting Config struct.
func decodeConfig(content []byte, format string) (*Config, error) {
	var raw map[string]interface{}
	var decodeHook mapstructure.DecodeHookFunc
	var err error

	switch format {
	case "json":
		err = json.Unmarshal(content, &raw)
		decodeHook = mapstructure.ComposeDecodeHookFunc(decode.HookTranslateKeys)
	case "hcl":
		err = hcl.Decode(&raw, string(content))
		decodeHook = mapstructure.ComposeDecodeHookFunc(decode.HookWeakDecodeFromSlice,
			decode.HookTranslateKeys)
	default:
		return nil, fmt.Errorf("invalid format: %s", format)
	}
	if err != nil {
		log.Printf("[DEBUG] (config) failed to decode %s", format)
		return nil, err
	}

	var config Config
	var md mapstructure.Metadata
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		DecodeHook:  decodeHook,
		ErrorUnused: true,
		Metadata:    &md,
		Result:      &config,
	})
	if err != nil {
		log.Println("[DEBUG] (config) mapstructure decoder creation failed")
		return nil, err
	}

	if err := decoder.Decode(raw); err != nil {
		log.Println("[DEBUG] (config) mapstructure decode failed")
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

	content, err := ioutil.ReadFile(path)
	if err != nil {
		log.Printf("[DEBUG] (config) failed reading config file from disk: %s\n", path)
		return nil, err
	}

	config, err := decodeConfig(content, format)
	if err != nil {
		log.Printf("[DEBUG] (config) failed decoding content from file: %s\n", path)
		return nil, err
	}

	return config, nil
}

// fromPath iterates and merges all configuration files in a given directory,
// returning the resulting config.
func fromPath(path string) (*Config, error) {
	// Ensure the given filepath exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		log.Printf("[DEBUG] (config) missing file/folder: %s\n", path)
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
		log.Printf("[DEBUG] (config) failed listing directory: %s\n", path)
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
