package client

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/hashicorp/consul-terraform-sync/api"
	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/logging"
	"github.com/hashicorp/consul-terraform-sync/retry"
	consulapi "github.com/hashicorp/consul/api"
	"github.com/hashicorp/hcat"
)

const (
	ConsulDefaultMaxRetry = 8 // to be consistent with hcat retries
	consulSubsystemName   = "consul"
)

var regexUnexpectedResponseCode = regexp.MustCompile("Unexpected response code: ([0-9]{3})")

//go:generate mockery --name=ConsulClientInterface --filename=consul.go --output=../mocks/client --tags=enterprise

// NonEnterpriseConsulError represents an error returned
// if expected enterprise Consul, but enterprise Consul was not found
type NonEnterpriseConsulError struct {
	Err error
}

// Error returns an error string
func (e *NonEnterpriseConsulError) Error() string {
	return fmt.Sprintf("consul is not consul enterprise: %v", e.Err)
}

// Unwrap returns the underlying error
func (e *NonEnterpriseConsulError) Unwrap() error {
	return e.Err
}

// MissingConsulACLError represents an error returned
// if the error was due to not having the correct ACL for
// accessing a Consul resource
type MissingConsulACLError struct {
	Err error
}

// Error returns an error string
func (e *MissingConsulACLError) Error() string {
	return fmt.Sprintf("missing required Consul ACL: %v", e.Err)
}

// Unwrap returns the underlying error
func (e *MissingConsulACLError) Unwrap() error {
	return e.Err
}

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
	RegisterService(ctx context.Context, s *consulapi.AgentServiceRegistration) error
	DeregisterService(ctx context.Context, serviceID string) error
}

// ConsulClient is a client to the Consul API
type ConsulClient struct {
	*consulapi.Client
	retry  retry.Retry
	logger logging.Logger
}

// ConsulAgentConfig represents the responseCode body from Consul /v1/agent/self API endpoint.
// The response contains configuration and member information of the requested agent.
// Care must always be taken to do type checks when casting, as structure could
// potentially change over time.
type ConsulAgentConfig = map[string]map[string]interface{}

// NewConsulClient constructs a consul api client
func NewConsulClient(conf *config.ConsulConfig, maxRetry int) (*ConsulClient, error) {
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
// GetLicense is a Consul Enterprise only endpoint, a 404 returned assumes we are connected to OSS Consul
// GetLicense does not require any ACLs
func (c *ConsulClient) GetLicense(ctx context.Context, q *consulapi.QueryOptions) (string, error) {
	c.logger.Debug("getting license")

	desc := "consul client get license"
	var err error
	var license string

	f := func(context.Context) error {
		var err error
		license, err = c.Operator().LicenseGetSigned(q)

		// Process the error by wrapping it in the correct error types
		if err != nil {
			statusCode := getResponseCodeFromError(ctx, err)

			// If we get a StatusNotFound assume that this is because CTS
			// is connected to OSS Consul where this endpoint isn't available
			// wrap in the appropriate error
			if statusCode == http.StatusNotFound {
				err = &NonEnterpriseConsulError{Err: err}
			}

			// non-retryable errors allows for termination of retries
			if !isResponseCodeRetryable(statusCode) {
				err = &retry.NonRetryableError{Err: err}
			}

			return err
		}
		return nil
	}

	err = c.retry.Do(ctx, f, desc)
	if err != nil {
		return "", err
	}

	return license, nil
}

// RegisterService registers a service through the Consul agent.
func (c *ConsulClient) RegisterService(ctx context.Context, r *consulapi.AgentServiceRegistration) error {
	logger := c.logger
	logger.Debug("registering service")

	desc := "AgentServiceRegister"

	if r != nil {
		logger = logger.With("service_name", r.Name, "service_id", r.ID)
	}

	f := func(context.Context) error {
		err := c.Agent().ServiceRegister(r)

		// Process the error by wrapping it in the correct error types
		if err != nil {
			statusCode := getResponseCodeFromError(ctx, err)

			// If we get a StatusForbidden assume that this is because CTS
			// does not have the correct ACLs to access this resource in Consul
			// and wrap in the appropriate error
			if statusCode == http.StatusForbidden {
				err = &MissingConsulACLError{Err: err}
			}

			// non-retryable errors allows for termination of retries
			if !isResponseCodeRetryable(statusCode) {
				err = &retry.NonRetryableError{Err: err}
			}

			return err
		}
		return nil
	}

	err := c.retry.Do(ctx, f, desc)
	if err != nil {
		return err
	}

	return nil
}

// DeregisterService removes a service through the Consul agent.
func (c *ConsulClient) DeregisterService(ctx context.Context, serviceID string) error {
	c.logger.Debug("deregistering service")
	desc := "AgentServiceDeregister"

	f := func(context.Context) error {
		err := c.Agent().ServiceDeregister(serviceID)
		if err != nil {
			statusCode := getResponseCodeFromError(ctx, err)

			// If we get a StatusForbidden assume that this is because CTS
			// does not have the correct ACLs to access this resource in Consul
			// and wrap in the appropriate error
			if statusCode == http.StatusForbidden {
				err = &MissingConsulACLError{Err: err}
			}

			// non-retryable errors allows for termination of retries
			if !isResponseCodeRetryable(statusCode) {
				err = &retry.NonRetryableError{Err: err}
			}

			return err
		}
		return nil
	}

	err := c.retry.Do(ctx, f, desc)
	if err != nil {
		return err
	}

	return nil
}

func getResponseCodeFromError(ctx context.Context, err error) int {
	// Extract the unexpected response substring
	s := regexUnexpectedResponseCode.FindString(err.Error())
	if s == "" {
		return 0
	}

	// Extract the response code substring from the unexpected response substring
	s = s[len(s)-3:]

	// Convert the response code to an integer
	i, err := strconv.Atoi(s)
	if err != nil {
		logging.FromContext(ctx).Debug("unable to convert string to integer", "error", err)
		return 0
	}

	return i
}

func isResponseCodeRetryable(statusCode int) bool {
	// 400 response codes are not useful to retry
	// with exception to 429, `too many requests` which may be useful for retries
	if api.CheckStatusCodeCategory(api.ClientErrorResponseCategory, statusCode) && statusCode != http.StatusTooManyRequests {
		return false
	}

	// Default to retry
	return true
}
