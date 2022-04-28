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
	defaultEndpoint      = "/v1/status" // TODO: temporary until /v1/health is implemented
	defaultMethod        = "GET"
	defaultInterval      = "10s"
	defaultTimeout       = "2s"
	defaultTLSSkipVerify = true

	logSystemName = "registration"
)

var defaultServiceTags = []string{"cts"}

// RegistrationManager handles the registration of a service and the health checks for that service
// to Consul.
type RegistrationManager struct {
	client  client.ConsulClientInterface
	service *service

	logger logging.Logger
}

type RegistrationManagerConfig struct {
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

func NewRegistrationManager(conf *RegistrationManagerConfig, client client.ConsulClientInterface) *RegistrationManager {
	logger := logging.Global().Named(logSystemName)

	name := defaultServiceName

	ns := defaultNamespace
	if conf.SelfRegistration.Namespace != nil {
		ns = *conf.SelfRegistration.Namespace
	}

	var checks []*consulapi.AgentServiceCheck
	checks = append(checks, defaultHTTPCheck(conf))

	return &RegistrationManager{
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

func (m *RegistrationManager) RegisterService(ctx context.Context) error {
	s := m.service
	r := &consulapi.AgentServiceRegistration{
		ID:        s.id,
		Name:      s.name,
		Tags:      s.tags,
		Port:      s.port,
		Checks:    s.checks,
		Namespace: s.namespace,
	}

	m.logger.Debug("registering service with Consul", "name", s.name, "id", s.id)
	err := m.client.RegisterService(ctx, r)
	if err != nil {
		m.logger.Error("error registering service with Consul", "name", s.name, "id", s.id)
		return err
	}
	return nil
}

func defaultHTTPCheck(conf *RegistrationManagerConfig) *consulapi.AgentServiceCheck {
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
