package registration

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"path"

	"github.com/hashicorp/consul-terraform-sync/client"
	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/logging"
	consulapi "github.com/hashicorp/consul/api"
)

const (
	// Service defaults
	defaultNamespace = ""

	// Check defaults
	defaultCheckName                      = "CTS Health Status"
	defaultCheckNotes                     = "Check created by Consul-Terraform-Sync"
	defaultDeregisterCriticalServiceAfter = "30m"
	defaultCheckStatus                    = consulapi.HealthCritical

	// HTTP-specific check defaults
	defaultHealthEndpoint = "/v1/health"
	defaultMethod         = "GET"
	defaultInterval       = "10s"
	defaultTimeout        = "2s"
	defaultTLSSkipVerify  = true

	logSystemName = "registration"
)

var defaultServiceTags = []string{"cts"}

// ServiceRegistrationManager handles the registration of Consul-Terraform-Sync as a service to Consul.
type ServiceRegistrationManager struct {
	client  client.ConsulClientInterface
	service *service

	logger logging.Logger
}

// ServiceRegistrationManagerConfig defines the configurations needed to create a new ServiceRegistrationManager.
type ServiceRegistrationManagerConfig struct {
	ID                  string
	Port                int
	TLSEnabled          bool
	ServiceRegistration *config.ServiceRegistrationConfig
}

type service struct {
	name      string
	id        string
	tags      []string
	port      int
	namespace string

	checks []*consulapi.AgentServiceCheck
}

// NewServiceRegistrationManager creates a new ServiceRegistrationManager object with the given configuration
// and Consul client. It sets default values where relevant, including a default HTTP check.
func NewServiceRegistrationManager(conf *ServiceRegistrationManagerConfig, client client.ConsulClientInterface) *ServiceRegistrationManager {
	logger := logging.Global().Named(logSystemName)

	name := config.DefaultServiceName
	if conf.ServiceRegistration.ServiceName != nil {
		name = *conf.ServiceRegistration.ServiceName
	}

	ns := defaultNamespace
	if conf.ServiceRegistration.Namespace != nil {
		ns = *conf.ServiceRegistration.Namespace
	}

	var checks []*consulapi.AgentServiceCheck
	if *conf.ServiceRegistration.DefaultCheck.Enabled {
		checks = append(checks, defaultHTTPCheck(conf))
	}
	return &ServiceRegistrationManager{
		client: client,
		logger: logger,
		service: &service{
			name:      name,
			id:        conf.ID,
			tags:      defaultServiceTags,
			port:      conf.Port,
			namespace: ns,
			checks:    checks,
		},
	}
}

// Start starts the service registration manager, which will register CTS as a service
// with Consul and deregister it if CTS is stopped.
func (m *ServiceRegistrationManager) Start(ctx context.Context) error {
	// Register CTS with Consul
	err := m.register(ctx)
	if err != nil {
		return err
	}

	// Wait until the context is cancelled, initiate deregistration
	<-ctx.Done()
	err = m.deregister(ctx)
	if err != nil {
		return err
	}
	return ctx.Err()
}

// register registers Consul-Terraform-Sync with Consul
func (m *ServiceRegistrationManager) register(ctx context.Context) error {
	s := m.service
	logger := m.logger.With("service_name", m.service.name, "id", m.service.id)
	r := &consulapi.AgentServiceRegistration{
		ID:        s.id,
		Name:      s.name,
		Tags:      s.tags,
		Port:      s.port,
		Checks:    s.checks,
		Namespace: s.namespace,
	}

	logger.Info("registering Consul-Terraform-Sync as a service with Consul")

	// Ignore error and continue if due to a missing ACL
	var missingConsulACLError *client.MissingConsulACLError
	err := m.client.RegisterService(ctx, r)
	if err != nil {
		baseErrMsg := "error registering Consul-Terraform-Sync as a service with Consul"
		if errors.As(err, &missingConsulACLError) {
			logger.Error(fmt.Sprintf("%s: "+
				"configure CTS with an ACL including `service:write` or "+
				"disable registration in configuration", baseErrMsg), "error", err)
		} else {
			logger.Error(baseErrMsg)
		}

		return err
	}

	logger.Info("Consul-Terraform-Sync registered as a service with Consul")
	return nil
}

// deregister deregisters Consul-Terraform-Sync from Consul
func (m *ServiceRegistrationManager) deregister(ctx context.Context) error {
	logger := m.logger.With("service_name", m.service.name, "id", m.service.id)
	logger.Info("deregistering Consul-Terraform-Sync from Consul")

	// Ignore error and continue if due to a missing ACL
	var missingConsulACLError *client.MissingConsulACLError
	err := m.client.DeregisterService(ctx, m.service.id)
	if err != nil {
		baseErrMsg := "error deregistering Consul-Terraform-Sync from Consul"
		if errors.As(err, &missingConsulACLError) {
			logger.Error(fmt.Sprintf("%s: "+
				"configure CTS with an ACL including `service:write` or "+
				"disable registration in configuration", baseErrMsg), "error", err)
		} else {
			logger.Error(baseErrMsg, "error", err)
		}

		return err
	}

	logger.Info("Consul-Terraform-Sync deregistered from Consul")
	return nil
}

func defaultHTTPCheck(conf *ServiceRegistrationManagerConfig) *consulapi.AgentServiceCheck {
	logger := logging.Global().Named(logSystemName)

	// Determine base address for HTTP check
	var address string
	if conf.ServiceRegistration.DefaultCheck.Address != nil && *conf.ServiceRegistration.DefaultCheck.Address != "" {
		address = *conf.ServiceRegistration.DefaultCheck.Address
	} else {
		var protocol string
		if conf.TLSEnabled {
			protocol = "https"
		} else {
			protocol = "http"
		}
		address = fmt.Sprintf("%s://localhost:%d", protocol, conf.Port)
	}

	// Append path to health API endpoint
	u, err := url.ParseRequestURI(address)
	if err != nil {
		// this should not fail since the address configuration should have
		// been previously validated with this same ParseRequestURI method
		return nil
	}
	u.Path = path.Join(u.Path, defaultHealthEndpoint)
	http := u.String()
	logger.Debug("creating default HTTP health check", "url", http)
	return &consulapi.AgentServiceCheck{
		Name:                           defaultCheckName,
		CheckID:                        fmt.Sprintf("%s-health", conf.ID),
		Notes:                          defaultCheckNotes,
		DeregisterCriticalServiceAfter: defaultDeregisterCriticalServiceAfter,
		Status:                         defaultCheckStatus,
		HTTP:                           http,
		Method:                         defaultMethod,
		Interval:                       defaultInterval,
		Timeout:                        defaultTimeout,
		TLSSkipVerify:                  defaultTLSSkipVerify,
	}
}
