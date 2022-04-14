package client

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/logging"
	"github.com/hashicorp/consul-terraform-sync/retry"
	consulapi "github.com/hashicorp/consul/api"
	"github.com/hashicorp/go-version"
	"github.com/hashicorp/hcat"
)

const (
	ConsulEnterpriseSignifier = "ent"
	ConsulDefaultMaxRetry     = 8 // to be consistent with hcat retries
	consulSubsystemName       = "consul"
)

//go:generate mockery --name=ConsulClientInterface --filename=consul_client.go --output=../mocks/client --tags=enterprise

var _ ConsulClientInterface = (*ConsulClient)(nil)

// ConsulClientInterface is an interface for a Consul Client
// If more consul client functionality is required, this interface should be extended with the following
// considerations:
// Each request to Consul is:
// - Retried
// - Logged at DEBUG-level
// - Easily mocked
type ConsulClientInterface interface {
	GetLicense(ctx context.Context, q *consulapi.QueryOptions) (string, error)
	IsEnterprise(ctx context.Context) (bool, error)
}

// ConsulClient is a client to the Consul API
type ConsulClient struct {
	*consulapi.Client
	retry  retry.Retry
	logger logging.Logger
}

// ConsulAgentConfig represents the response body from Consul /v1/agent/self API endpoint.
// The response contains configuration and member information of the requested agent.
// Care must always be taken to do type checks when casting, as structure could
// potentially change over time.
type ConsulAgentConfig = map[string]map[string]interface{}

// NewConsulClient constructs a consul api client
func NewConsulClient(conf *config.ConsulConfig, maxRetry int) (ConsulClientInterface, error) {
	t := hcat.TransportInput{
		SSLEnabled: *conf.TLS.Enabled,
		SSLVerify:  *conf.TLS.Verify,
		SSLCert:    *conf.TLS.Cert,
		SSLKey:     *conf.TLS.Key,
		SSLCACert:  *conf.TLS.CACert,
		SSLCAPath:  *conf.TLS.CAPath,
		ServerName: *conf.TLS.ServerName,

		DialKeepAlive:       *conf.Transport.DialKeepAlive,
		DialTimeout:         *conf.Transport.DialTimeout,
		DisableKeepAlives:   *conf.Transport.DisableKeepAlives,
		IdleConnTimeout:     *conf.Transport.IdleConnTimeout,
		MaxIdleConns:        *conf.Transport.MaxIdleConns,
		MaxIdleConnsPerHost: *conf.Transport.MaxIdleConnsPerHost,
		TLSHandshakeTimeout: *conf.Transport.TLSHandshakeTimeout,
	}

	ci := hcat.ConsulInput{
		Address:      *conf.Address,
		Token:        *conf.Token,
		AuthEnabled:  *conf.Auth.Enabled,
		AuthUsername: *conf.Auth.Username,
		AuthPassword: *conf.Auth.Password,
		Transport:    t,
	}

	clients := hcat.NewClientSet()
	if err := clients.AddConsul(ci); err != nil {
		return nil, err
	}

	logger := logging.Global().Named(loggingSystemName).Named(consulSubsystemName)

	r := retry.NewRetry(maxRetry, time.Now().UnixNano())
	c := &ConsulClient{
		Client: clients.Consul(),
		retry:  r,
		logger: logger,
	}

	return c, nil
}

// GetLicense queries Consul for a signed license, and returns it if available
func (c *ConsulClient) GetLicense(ctx context.Context, q *consulapi.QueryOptions) (string, error) {
	c.logger.Debug("getting license")

	desc := "consul client get license"
	var err error
	var license string

	f := func(context.Context) error {
		license, err = c.Operator().LicenseGetSigned(q)
		if err != nil {
			license = ""
			return err
		}
		return nil
	}

	err = c.retry.Do(ctx, f, desc)

	return license, err
}

// IsEnterprise queries Consul for information about itself, it then
// parses this information to determine whether the Consul being
// queried is Enterprise or OSS. Returns true if Consul is Enterprise.
func (c *ConsulClient) IsEnterprise(ctx context.Context) (bool, error) {
	c.logger.Debug("checking if connected to Consul Enterprise")

	desc := "consul client get is enterprise"
	var err error
	var info ConsulAgentConfig

	f := func(context.Context) error {
		info, err = c.Agent().Self()
		if err != nil {
			info = nil
			return err
		}
		return nil
	}

	err = c.retry.Do(ctx, f, desc)
	if err != nil {
		return false, err
	}

	ctx = logging.WithContext(ctx, c.logger)

	isEnterprise, err := isConsulEnterprise(ctx, info)
	if err != nil {
		return false, fmt.Errorf("unable to parse if Consul is enterprise: %v", err)
	}
	return isEnterprise, nil
}

func isConsulEnterprise(ctx context.Context, info ConsulAgentConfig) (bool, error) {
	logger := logging.FromContext(ctx)
	v, ok := info["Config"]["Version"]
	if !ok {
		logger.Debug("expected keys, map[Config][Version], to exist", "ConsulAgentConfig", info)
		return false, errors.New("unable to parse map[Config][Version], keys do not exist")
	}

	vs, ok := v.(string)
	if !ok {
		logger.Debug("expected keys, map[Config][Version], do not map to a string", "ConsulAgentConfig", info["Config"]["Version"])
		return false, errors.New("unable to parse map[Config][Version], keys do not map to string")
	}

	ver, err := version.NewVersion(vs)
	if err != nil {
		return false, err
	}

	if strings.Contains(ver.Metadata(), ConsulEnterpriseSignifier) {
		return true, nil
	}
	return false, nil
}
