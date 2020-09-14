package handler

import (
	"log"

	"github.com/PaloAltoNetworks/pango"
	"github.com/PaloAltoNetworks/pango/commit"
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
	providerConf pango.Client
	configPath   string
}

// NewPanos configures and returns a new panos handler
func NewPanos(c map[string]interface{}) (*Panos, error) {
	log.Printf("[INFO] (handler.panos) creating handler")

	var conf pango.Client
	decoderConf := &mapstructure.DecoderConfig{TagName: "json", Result: &conf}
	decoder, err := mapstructure.NewDecoder(decoderConf)
	if err != nil {
		return nil, err
	}
	if err := decoder.Decode(c); err != nil {
		return nil, err
	}

	configPath := ""
	for k, val := range c {
		if k == "json_config_file" {
			if v, ok := val.(string); ok {
				configPath = v
				break
			}
		}
	}

	fw := &pango.Firewall{
		Client: conf,
	}

	return &Panos{
		next:         nil,
		client:       fw,
		providerConf: conf,
		configPath:   configPath,
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

	if err := h.client.InitializeUsing(h.configPath, true); err != nil {
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
