package config

import (
	"fmt"

	"github.com/hashicorp/vault/api"
	homedir "github.com/mitchellh/go-homedir"
)

const (
	// DefaultVaultRenewToken is the default value for if the Vault token should
	// be renewed.
	DefaultVaultRenewToken = true

	// DefaultVaultUnwrapToken is the default value for if the Vault token should
	// be unwrapped.
	DefaultVaultUnwrapToken = false
)

// VaultConfig is the configuration for connecting to a vault server.
type VaultConfig struct {
	// Address is the URI to the Vault server.
	Address *string `mapstructure:"address"`

	// Enabled controls whether the Vault integration is active.
	Enabled *bool `mapstructure:"enabled"`

	// Namespace is the Vault namespace to use for reading/writing secrets. This can
	// also be set via the VAULT_NAMESPACE environment variable.
	Namespace *string `mapstructure:"namespace"`

	// RenewToken renews the Vault token.
	RenewToken *bool `mapstructure:"renew_token"`

	// TLS indicates we should use a secure connection while talking to Vault.
	TLS *TLSConfig `mapstructure:"tls"`

	// Token is the Vault token to communicate with for requests. It may be
	// a wrapped token or a real token. This can also be set via the VAULT_TOKEN
	// environment variable, or via the VaultAgentTokenFile.
	Token *string `mapstructure:"token" json:"-"`

	// VaultAgentTokenFile is the path of file that contains a Vault Agent token.
	// If vault_agent_token_file is specified:
	//   - Consul-Terraform-Sync will not try to renew the Vault token.
	VaultAgentTokenFile *string `mapstructure:"vault_agent_token_file" json:"-"`

	// Transport configures the low-level network connection details.
	Transport *TransportConfig `mapstructure:"transport"`

	// UnwrapToken unwraps the provided Vault token as a wrapped token.
	UnwrapToken *bool `mapstructure:"unwrap_token"`

	// test will ignore the user's ~/.vault_token file for testing
	test bool
}

// DefaultVaultConfig returns a configuration that is populated with the
// default values.
func DefaultVaultConfig() *VaultConfig {
	v := &VaultConfig{
		TLS:       DefaultTLSConfig(),
		Transport: DefaultTransportConfig(),
	}

	// Force TLS when communicating with Vault since Vault assumes TLS by default
	v.TLS.Enabled = Bool(true)

	return v
}

// Copy returns a deep copy of this configuration.
func (c *VaultConfig) Copy() *VaultConfig {
	if c == nil {
		return nil
	}

	var o VaultConfig
	o.Address = StringCopy(c.Address)

	o.Enabled = BoolCopy(c.Enabled)

	o.Namespace = StringCopy(c.Namespace)

	o.RenewToken = BoolCopy(c.RenewToken)

	if c.TLS != nil {
		o.TLS = c.TLS.Copy()
	}

	o.Token = StringCopy(c.Token)

	o.VaultAgentTokenFile = StringCopy(c.VaultAgentTokenFile)

	if c.Transport != nil {
		o.Transport = c.Transport.Copy()
	}

	o.UnwrapToken = BoolCopy(c.UnwrapToken)

	return &o
}

// Merge combines all values in this configuration with the values in the other
// configuration, with values in the other configuration taking precedence.
// Maps and slices are merged, most other values are overwritten. Complex
// structs define their own merge functionality.
func (c *VaultConfig) Merge(o *VaultConfig) *VaultConfig {
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

	if o.Address != nil {
		r.Address = StringCopy(o.Address)
	}

	if o.Enabled != nil {
		r.Enabled = BoolCopy(o.Enabled)
	}

	if o.Namespace != nil {
		r.Namespace = StringCopy(o.Namespace)
	}

	if o.RenewToken != nil {
		r.RenewToken = BoolCopy(o.RenewToken)
	}

	if o.TLS != nil {
		r.TLS = r.TLS.Merge(o.TLS)
	}

	if o.Token != nil {
		r.Token = StringCopy(o.Token)
	}

	if o.VaultAgentTokenFile != nil {
		r.VaultAgentTokenFile = StringCopy(o.VaultAgentTokenFile)
	}

	if o.Transport != nil {
		r.Transport = r.Transport.Merge(o.Transport)
	}

	if o.UnwrapToken != nil {
		r.UnwrapToken = BoolCopy(o.UnwrapToken)
	}

	return r
}

// Finalize ensures there no nil pointers.
func (c *VaultConfig) Finalize() {
	if c.Address == nil {
		c.Address = stringFromEnv([]string{
			api.EnvVaultAddress,
		}, "")
	}

	if c.Namespace == nil {
		c.Namespace = stringFromEnv([]string{api.EnvVaultNamespace}, "")
	}

	// Vault has custom TLS settings
	if c.TLS == nil {
		c.TLS = DefaultTLSConfig()
	}
	c.TLS.FinalizeVault()

	// Order of precedence
	// 1. `vault_agent_token_file` configuration value
	// 2. `token` configuration value`
	// 3. `VAULT_TOKEN` environment variable
	if c.Token == nil {
		c.Token = stringFromEnv([]string{
			api.EnvVaultToken,
		}, "")
	}

	if c.VaultAgentTokenFile == nil {
		if StringVal(c.Token) == "" {
			// homePath is the location to the user's home directory.
			homePath, _ := homedir.Dir()
			if homePath != "" && !c.test {
				c.Token = stringFromFile([]string{
					homePath + "/.vault-token",
				}, "")
			}
		}
	} else {
		c.Token = stringFromFile([]string{*c.VaultAgentTokenFile}, "")
	}

	// must be after c.Token setting, as default depends on that.
	if c.RenewToken == nil {
		defaultRenew := DefaultVaultRenewToken
		if c.VaultAgentTokenFile != nil {
			defaultRenew = false
		} else if StringVal(c.Token) == "" {
			defaultRenew = false
		}
		c.RenewToken = boolFromEnv([]string{
			"VAULT_RENEW_TOKEN",
		}, defaultRenew)
	}

	if c.Transport == nil {
		c.Transport = DefaultTransportConfig()
	}
	c.Transport.Finalize()

	if c.UnwrapToken == nil {
		c.UnwrapToken = boolFromEnv([]string{
			api.PluginUnwrapTokenEnv,
		}, DefaultVaultUnwrapToken)
	}

	if c.Enabled == nil {
		c.Enabled = Bool(StringPresent(c.Address))
	}
}

// GoString defines the printable version of this struct.
func (c *VaultConfig) GoString() string {
	if c == nil {
		return "(*VaultConfig)(nil)"
	}

	return fmt.Sprintf("&VaultConfig{"+
		"Address:%s, "+
		"Enabled:%v, "+
		"Namespace:%s,"+
		"RenewToken:%v, "+
		"TLS:%#v, "+
		"Token:%t, "+
		"VaultAgentTokenFile:%t, "+
		"Transport:%#v, "+
		"UnwrapToken:%v"+
		"}",
		StringVal(c.Address),
		BoolVal(c.Enabled),
		StringVal(c.Namespace),
		BoolVal(c.RenewToken),
		c.TLS.GoString(),
		StringPresent(c.Token),
		StringPresent(c.VaultAgentTokenFile),
		c.Transport.GoString(),
		BoolVal(c.UnwrapToken),
	)
}
