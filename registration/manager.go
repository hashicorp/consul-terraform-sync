package registration

import (
	"context"
	"fmt"

	"github.com/hashicorp/consul-terraform-sync/client"
	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/logging"
	consulapi "github.com/hashicorp/consul/api"
)

const (
	// Service defaults
	defaultServiceName = "Consul-Terraform-Sync"
	defaultNamespace   = ""

	// Check defaults
	defaultCheckName                      = "CTS Health Status"
	defaultCheckNotes                     = "Check created by Consul-Terraform-Sync"
	defaultDeregisterCriticalServiceAfter = "30m"
	defaultCheckStatus                    = consulapi.HealthCritical

	// HTTP-specific check defaults
	defaultEndpoint      = "/v1/health"
	defaultMethod        = "GET"
	defaultInterval      = "10s"
	defaultTimeout       = "2s"
	defaultTLSSkipVerify = true

	logSystemName = "registration"
)

var defaultServiceTags = []string{"cts"}

// SelfRegistrationManager handles the registration of Consul-Terraform-Sync as a service to Consul.
type SelfRegistrationManager struct {
	client  client.ConsulClientInterface
	service *service

	logger logging.Logger
}

// SelfRegistrationManagerConfig defines the configurations needed to create a new SelfRegistrationManager.
type SelfRegistrationManagerConfig struct {
	ID               string
	Port             int
	TLSEnabled       bool
	SelfRegistration *config.SelfRegistrationConfig
}

type service struct {
	name      string
	id        string
	tags      []string
	port      int
	namespace string

	checks []*consulapi.AgentServiceCheck
}

// NewSelfRegistrationManager creates a new SelfRegistrationManager object with the given configuration
// and Consul client. It sets default values where relevant, including a default HTTP check.
func NewSelfRegistrationManager(conf *SelfRegistrationManagerConfig, client client.ConsulClientInterface) *SelfRegistrationManager {
	logger := logging.Global().Named(logSystemName)

	name := defaultServiceName

	ns := defaultNamespace
	if conf.SelfRegistration.Namespace != nil {
		ns = *conf.SelfRegistration.Namespace
	}

	var checks []*consulapi.AgentServiceCheck
	checks = append(checks, defaultHTTPCheck(conf))

	return &SelfRegistrationManager{
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

// SelfRegisterService registers Consul-Terraform-Sync with Consul
func (m *SelfRegistrationManager) SelfRegisterService(ctx context.Context) error {
	s := m.service
	r := &consulapi.AgentServiceRegistration{
		ID:        s.id,
		Name:      s.name,
		Tags:      s.tags,
		Port:      s.port,
		Checks:    s.checks,
		Namespace: s.namespace,
	}

	m.logger.Debug("self-registering Consul-Terraform-Sync as a service with Consul", "name", s.name, "id", s.id)
	err := m.client.RegisterService(ctx, r)
	if err != nil {
		m.logger.Error("error self-registering Consul-Terraform-Sync as a service with Consul", "name", s.name, "id", s.id)
		return err
	}
	return nil
}

func defaultHTTPCheck(conf *SelfRegistrationManagerConfig) *consulapi.AgentServiceCheck {
	var protocol string
	if conf.TLSEnabled {
		protocol = "https"
	} else {
		protocol = "http"
	}
	address := fmt.Sprintf("%s://localhost:%d%s", protocol, conf.Port, defaultEndpoint)
	return &consulapi.AgentServiceCheck{
		Name:                           defaultCheckName,
		CheckID:                        fmt.Sprintf("%s-health", conf.ID),
		Notes:                          defaultCheckNotes,
		DeregisterCriticalServiceAfter: defaultDeregisterCriticalServiceAfter,
		Status:                         defaultCheckStatus,
		HTTP:                           address,
		Method:                         defaultMethod,
		Interval:                       defaultInterval,
		Timeout:                        defaultTimeout,
		TLSSkipVerify:                  defaultTLSSkipVerify,
	}
}
