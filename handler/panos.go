package handler

import (
	"errors"
	"log"
	"os"
	"strings"

	"github.com/PaloAltoNetworks/pango"
	"github.com/PaloAltoNetworks/pango/commit"
	"github.com/mitchellh/mapstructure"
)

const (
	// TerraformProviderPanos is the name of a Palo Alto PANOS Terraform provider.
	TerraformProviderPanos = "panos"

	// Users with custom roles currently return an error for an empty commit with
	// this server response prefix. See GH-73 for more details.
	emptyCommitServerRespPrefix = `<response status="success" code="13">`
)

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
	adminUser    string
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

	var configPath string
	if val, ok := c["json_config_file"]; ok {
		if v, ok := val.(string); ok {
			configPath = v
		}
	}

	// Username is required to limit commiting changes to the admin user instead
	// of all queued changes by all users.
	var username string
	if val, ok := c["username"]; ok {
		if v, ok := val.(string); ok {
			username = v
		}
	} else {
		username, _ = os.LookupEnv("PANOS_USERNAME")
	}
	if username == "" {
		return nil, errors.New("detected panos provider with missing username. " +
			"Username of the admin the API key is associated with is required for " +
			"partial commits by Consul-Terraform-Sync to limit the changes " +
			"auto-committed to the admin user. Configure the admin username for " +
			"the panos provider or set the PANOS_USERNAME environment variable.")
	}

	fw := &pango.Firewall{
		Client: conf,
	}

	return &Panos{
		next:         nil,
		client:       fw,
		providerConf: conf,
		adminUser:    username,
		configPath:   configPath,
	}, nil
}

// Do executes panos' out-of-band Commit API and calls next handler while passing
// on relevant errors
func (h *Panos) Do(prevErr error) error {
	log.Printf("[INFO] (handler.panos) commit. host '%s'", h.providerConf.Hostname)
	err := h.commit()
	return callNext(h.next, prevErr, err)
}

// commit calls panos' InitializeUsing & Commit SDK
func (h *Panos) commit() error {
	if err := h.client.InitializeUsing(h.configPath, true); err != nil {
		// potential optimizations to call Initialize() once / less frequently
		log.Printf("[ERR] (handler.panos) error initializing panos client: %s", err)
		return err
	}
	log.Printf("[TRACE] (handler.panos) client config after init: %s", h.client.String())

	c := commit.FirewallCommit{
		Admins:      []string{h.adminUser},
		Description: "Consul Terraform Sync Commit",
	}
	job, resp, err := h.client.Commit(c.Element(), "", nil)
	if emptyCommit(job, resp, err) {
		return nil
	}
	if err != nil {
		log.Printf("[ERR] (handler.panos) error committing: %s. Server response: '%s'", err, resp)
		return err
	}

	if err := h.client.WaitForJob(job, nil); err != nil {
		log.Printf("[ERR] (handler.panos) error waiting for panos commit to finish: %s", err)
		return err
	}

	log.Printf("[DEBUG] (handler.panos) commit successful")
	return nil
}

// SetNext sets the next handler that should be called.
func (h *Panos) SetNext(next Handler) {
	h.next = next
}

// emptyCommit consumes the commit API return data to determine if commit was
// empty i.e. there were no resource to commit
func emptyCommit(job uint, resp []byte, err error) bool {
	if err == nil && job == 0 {
		log.Printf("[DEBUG] (handler.panos) superadmin commit not needed")
		return true
	}

	if err != nil && strings.HasPrefix(string(resp), emptyCommitServerRespPrefix) {
		log.Printf("[DEBUG] (handler.panos) custom-role commit not needed")
		log.Printf("[TRACE] (handler.panos) custom-role empty commit err: '%s'."+
			" server response: '%s'", err, resp)
		return true
	}

	return false
}
