package config

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/hashicorp/consul-terraform-sync/logging"
	ctsVersion "github.com/hashicorp/consul-terraform-sync/version"
	goVersion "github.com/hashicorp/go-version"
)

// TerraformCloudWorkspaceConfig is an enterprise-only configuration that controls
// workspace attributes that are specific to a task.
type TerraformCloudWorkspaceConfig struct {
	ExecutionMode    *string `mapstructure:"execution_mode"`
	AgentPoolID      *string `mapstructure:"agent_pool_id"`
	AgentPoolName    *string `mapstructure:"agent_pool_name"`
	TerraformVersion *string `mapstructure:"terraform_version"`
}

func DefaultTerraformCloudWorkspaceConfig() *TerraformCloudWorkspaceConfig {
	return &TerraformCloudWorkspaceConfig{
		ExecutionMode:    String(""),
		AgentPoolID:      String(""),
		AgentPoolName:    String(""),
		TerraformVersion: String(""),
	}
}

func (c *TerraformCloudWorkspaceConfig) IsEmpty() bool {
	return (*c == TerraformCloudWorkspaceConfig{}) ||
		(reflect.DeepEqual(c, &TerraformCloudWorkspaceConfig{
			ExecutionMode:    String(""),
			AgentPoolID:      String(""),
			AgentPoolName:    String(""),
			TerraformVersion: String(""),
		}))
}

// Copy returns a deep copy of this configuration.
func (c *TerraformCloudWorkspaceConfig) Copy() *TerraformCloudWorkspaceConfig {
	if c == nil {
		return nil
	}

	var o TerraformCloudWorkspaceConfig
	o.ExecutionMode = StringCopy(c.ExecutionMode)
	o.AgentPoolID = StringCopy(c.AgentPoolID)
	o.AgentPoolName = StringCopy(c.AgentPoolName)
	o.TerraformVersion = StringCopy(c.TerraformVersion)
	return &o
}

// Merge combines all values in this configuration with the values in the other
// configuration, with values in the other configuration taking precedence.
func (c *TerraformCloudWorkspaceConfig) Merge(o *TerraformCloudWorkspaceConfig) *TerraformCloudWorkspaceConfig {
	if c == nil {
		if o == nil {
			return o
		}
		return o.Copy()
	}

	if o == nil {
		return c.Copy()
	}

	r := c.Copy()

	if o.ExecutionMode != nil {
		r.ExecutionMode = StringCopy(o.ExecutionMode)
	}

	if o.AgentPoolID != nil {
		r.AgentPoolID = StringCopy(o.AgentPoolID)
	}

	if o.AgentPoolName != nil {
		r.AgentPoolName = StringCopy(o.AgentPoolName)
	}

	if o.TerraformVersion != nil {
		r.TerraformVersion = StringCopy(o.TerraformVersion)
	}

	return r
}

// Finalize ensures that the receiver contains no nil pointers.
func (c *TerraformCloudWorkspaceConfig) Finalize() {
	if c == nil { // config is not required, return early
		return
	}

	if c.ExecutionMode == nil {
		c.ExecutionMode = String("")
	}

	if c.AgentPoolID == nil {
		c.AgentPoolID = String("")
	}

	if c.AgentPoolName == nil {
		c.AgentPoolName = String("")
	}

	if c.TerraformVersion == nil {
		c.TerraformVersion = String("")
	}
}

// Validate validates the values of the configuration struct
func (c *TerraformCloudWorkspaceConfig) Validate() error {
	if c == nil || c.IsEmpty() { // config not required, return early
		return nil
	}

	if StringVal(c.ExecutionMode) != "remote" && StringVal(c.ExecutionMode) != "agent" && StringVal(c.ExecutionMode) != "" {
		return fmt.Errorf("execution mode '%s' not supported for CTS, use 'remote' or 'agent' instead", *c.ExecutionMode)
	}

	if StringVal(c.ExecutionMode) == "agent" &&
		StringVal(c.AgentPoolID) == "" && StringVal(c.AgentPoolName) == "" {
		return errors.New("agent_pool_id or agent_pool_name is required if execution mode is 'agent'")
	}

	if StringVal(c.ExecutionMode) != "agent" &&
		(StringVal(c.AgentPoolID) != "" || StringVal(c.AgentPoolName) != "") {
		return errors.New("agent pool configured when execution mode is not 'agent'")
	}

	if StringVal(c.AgentPoolID) != "" && StringVal(c.AgentPoolName) != "" {
		logger := logging.Global().Named(logSystemName)
		logger.Warn("agent_pool_id and agent_pool_name are both configured, agent_pool_id will be used")
	}

	if c.TerraformVersion != nil && *c.TerraformVersion != "" {
		v, err := goVersion.NewSemver(*c.TerraformVersion)
		if err != nil {
			return err
		}

		if len(strings.Split(*c.TerraformVersion, ".")) < 3 {
			return fmt.Errorf("provide the exact Terraform version to install: %s", *c.TerraformVersion)
		}

		if !ctsVersion.TerraformConstraint.Check(v) {
			return fmt.Errorf("Terraform version is not supported by Consul-"+
				"Terraform-Sync, try updating to a different version (%s): %s",
				ctsVersion.CompatibleTerraformVersionConstraint, *c.TerraformVersion)
		}
	}

	return nil
}

// GoString defines the printable version of this struct.
func (c *TerraformCloudWorkspaceConfig) GoString() string {
	if c == nil {
		return "(*TerraformCloudWorkspaceConfig)(nil)"
	}

	return fmt.Sprintf("&TerraformCloudWorkspaceConfig{"+
		"AgentPoolID:%s, "+
		"AgentPoolName:%s, "+
		"ExecutionMode:%s, "+
		"TerraformVersion:%s"+
		"}",
		StringVal(c.AgentPoolID),
		StringVal(c.AgentPoolName),
		StringVal(c.ExecutionMode),
		StringVal(c.TerraformVersion),
	)
}
