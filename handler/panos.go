package handler

import (
	"fmt"
	"log"

	"github.com/PaloAltoNetworks/pango"
	"github.com/PaloAltoNetworks/pango/commit"
	"github.com/hashicorp/consul-nia/config"
	"github.com/mitchellh/mapstructure"
)

// TerraformProviderPanos is the name of a Palo Alto PANOS Terraform provider.
const TerraformProviderPanos = "panos"

//go:generate mockery --name=panosClient  --structname=PanosClient --output=../mocks/handler

var _ panosClient = (*pango.Firewall)(nil)

type panosClient interface {
	InitializeUsing(filename string, chkenv bool) error
	Commit(cmd interface{}, action string, extras interface{}) (uint, []byte, error)
	WaitForJob(id uint, resp interface{}) error
	String() string
}

var _ Handler = (*Panos)(nil)

// Panos is the post-apply handler for the panos Terraform Provider.
// It performs the out-of-band Commit API request needed after a Terraform apply.
//
// See https://registry.terraform.io/providers/PaloAltoNetworks/panos/latest/docs
// for details on Commit and panos provider (outdated use of SDK at the time).
// See https://github.com/PaloAltoNetworks/pango for latest version of SDK.
type Panos struct {
	next         Handler
	client       panosClient
	providerConf panosConfig
}

// panosConfig captures panos provider configuration
type panosConfig struct {
	Hostname          *string  `mapstructure:"hostname"`
	Username          *string  `mapstructure:"username"`
	Password          *string  `mapstructure:"password"`
	APIKey            *string  `mapstructure:"api_key" json:"api_key"`
	Protocol          *string  `mapstructure:"protocol"`
	Port              *int     `mapstructure:"port"`
	Timeout           *int     `mapstructure:"timeout"`
	Logging           []string `mapstructure:"logging"`
	VerifyCertificate *bool    `mapstructure:"verify_certificate" json:"verify_certificate"`
	JSONConfigFile    *string  `mapstructure:"json_config_file"`
}

// NewPanos configures and returns a new panos handler
func NewPanos(c map[string]interface{}) (*Panos, error) {
	var conf panosConfig
	if err := mapstructure.Decode(c, &conf); err != nil {
		return nil, err
	}

	log.Printf("[INFO] (handler.panos) creating handler with initial config: %s", conf.GoString())

	fw := &pango.Firewall{
		Client: pango.Client{
			Hostname:              config.StringVal(conf.Hostname),
			Username:              config.StringVal(conf.Username),
			Password:              config.StringVal(conf.Password),
			ApiKey:                config.StringVal(conf.APIKey),
			Protocol:              config.StringVal(conf.Protocol),
			Port:                  uint(config.IntVal(conf.Port)),
			Timeout:               config.IntVal(conf.Timeout),
			VerifyCertificate:     config.BoolVal(conf.VerifyCertificate),
			LoggingFromInitialize: conf.Logging,
		},
	}

	return &Panos{
		next:         nil,
		client:       fw,
		providerConf: conf,
	}, nil
}

// Do executes panos' out-of-band Commit API. Errors are logged.
func (h *Panos) Do() {
	log.Printf("[INFO] (handler.panos) do")
	defer func() {
		if h.next != nil {
			h.next.Do()
		}
	}()

	configPath := config.StringVal(h.providerConf.JSONConfigFile)
	if err := h.client.InitializeUsing(configPath, true); err != nil {
		// potential optimizations to call Initialize() once / less frequently
		log.Printf("[ERR] (handler.panos) error initializing panos client: %s", err)
		return
	}
	log.Printf("[TRACE] (handler.panos) client config after init: %s", h.client.String())

	c := commit.FirewallCommit{
		Description: "NIA Commit",
	}
	job, resp, err := h.client.Commit(c.Element(), "", nil)
	if err != nil {
		log.Printf("[ERR] (handler.panos) error committing: %s. Server response: '%s'", err, resp)
		return
	}
	if job == 0 {
		log.Printf("[DEBUG] (handler.panos) commit was not needed")
		return
	}

	if err := h.client.WaitForJob(job, nil); err != nil {
		log.Printf("[ERR] (handler.panos) error waiting for panos commit to finish: %s", err)
		return
	}

	log.Printf("[DEBUG] (handler.panos) commit successful")
}

// SetNext sets the next handler that should be called.
func (h *Panos) SetNext(next Handler) {
	h.next = next
}

// GoString defines the printable version of this struct.
func (c *panosConfig) GoString() string {
	if c == nil {
		return "(*panosConfig)(nil)"
	}
	return fmt.Sprintf("&panosConfig{"+
		"Hostname:%s, "+
		"Username:%s, "+
		"Password:%s, "+
		"APIKey:%s, "+
		"Protocol:%s, "+
		"Port:%d, "+
		"Timeout:%d, "+
		"Logging:%v, "+
		"VerifyCertificate:%t, "+
		"JSONConfigFile:%s"+
		"}",
		config.StringVal(c.Hostname),
		config.StringVal(c.Username),
		"<password-redacted>",
		"<api-key-redacted>",
		config.StringVal(c.Protocol),
		config.IntVal(c.Port),
		config.IntVal(c.Timeout),
		c.Logging,
		config.BoolVal(c.VerifyCertificate),
		config.StringVal(c.JSONConfigFile),
	)
}
