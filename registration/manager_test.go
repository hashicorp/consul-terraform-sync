package registration

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/logging"
	mocks "github.com/hashicorp/consul-terraform-sync/mocks/client"
	consulapi "github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestNewRegistrationManager(t *testing.T) {
	testcases := []struct {
		name            string
		conf            *RegistrationManagerConfig
		expectedService *service
	}{
		{
			"defaults",
			&RegistrationManagerConfig{
				ID:               "cts-123",
				Port:             123,
				TLSEnabled:       false,
				SelfRegistration: config.DefaultSelfRegistrationConfig(),
			},
			&service{
				name:      defaultServiceName,
				tags:      defaultServiceTags,
				id:        "cts-123",
				port:      123,
				namespace: "",
			},
		},
		{
			"namespace",
			&RegistrationManagerConfig{
				ID:         "cts-123",
				Port:       123,
				TLSEnabled: false,
				SelfRegistration: &config.SelfRegistrationConfig{
					Namespace: config.String("ns-1"),
				},
			},
			&service{
				name:      defaultServiceName,
				tags:      defaultServiceTags,
				id:        "cts-123",
				port:      123,
				namespace: "ns-1",
			},
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			client := new(mocks.ConsulClientInterface)
			m := NewRegistrationManager(tc.conf, client)

			// Verify general attributes
			assert.NotNil(t, m)
			assert.Equal(t, m.client, client)
			assert.NotNil(t, m.logger)

			// Verify service attributes
			assert.NotNil(t, m.service)
			assert.Equal(t, tc.expectedService.name, m.service.name)
			assert.Equal(t, tc.expectedService.id, m.service.id)
			assert.Equal(t, tc.expectedService.tags, m.service.tags)
			assert.Equal(t, tc.expectedService.port, m.service.port)
			assert.Equal(t, tc.expectedService.namespace, m.service.namespace)

			// Verify service has default health check
			assert.NotNil(t, m.service.checks)
			assert.Equal(t, 1, len(m.service.checks))
			check := m.service.checks[0]
			defaultCheck := defaultHTTPCheck(tc.conf)
			assert.Equal(t, defaultCheck, check)
		})
	}
}

func TestRegistrationManager_defaultHTTPCheck(t *testing.T) {
	id := "cts-123"
	port := 8558
	// TODO: update these addresses when /v1/health implemented
	httpAddress := fmt.Sprintf("http://localhost:%d/v1/status", port)
	httpsAddress := fmt.Sprintf("https://localhost:%d/v1/status", port)
	checkID := fmt.Sprintf("%s-health", id)

	testcases := []struct {
		name     string
		conf     *RegistrationManagerConfig
		expected *consulapi.AgentServiceCheck
	}{
		{
			"tls_disabled",
			&RegistrationManagerConfig{
				ID:               id,
				Port:             port,
				TLSEnabled:       false,
				SelfRegistration: config.DefaultSelfRegistrationConfig(),
			},
			&consulapi.AgentServiceCheck{
				HTTP:                           httpAddress,
				Name:                           defaultCheckName,
				CheckID:                        checkID,
				Notes:                          defaultCheckNotes,
				DeregisterCriticalServiceAfter: defaultDeregisterCriticalServiceAfter,
				Status:                         defaultCheckStatus,
				Method:                         defaultMethod,
				Interval:                       defaultInterval,
				Timeout:                        defaultTimeout,
				TLSSkipVerify:                  defaultTLSSkipVerify,
			},
		},
		{
			"tls_enabled",
			&RegistrationManagerConfig{
				ID:               id,
				Port:             port,
				TLSEnabled:       true,
				SelfRegistration: config.DefaultSelfRegistrationConfig(),
			},
			&consulapi.AgentServiceCheck{
				HTTP:                           httpsAddress,
				Name:                           defaultCheckName,
				CheckID:                        checkID,
				Notes:                          defaultCheckNotes,
				DeregisterCriticalServiceAfter: defaultDeregisterCriticalServiceAfter,
				Status:                         defaultCheckStatus,
				Method:                         defaultMethod,
				Interval:                       defaultInterval,
				Timeout:                        defaultTimeout,
				TLSSkipVerify:                  defaultTLSSkipVerify,
			},
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			c := defaultHTTPCheck(tc.conf)
			assert.Equal(t, tc.expected, c)
		})
	}
}

func TestRegistrationManager_RegisterService(t *testing.T) {
	id := "cts-123"
	port := 8558
	ns := "ns-1"
	check := defaultHTTPCheck(&RegistrationManagerConfig{
		ID:   id,
		Port: port,
		SelfRegistration: &config.SelfRegistrationConfig{
			Namespace: &ns,
		},
	})
	service := &service{
		name:      defaultServiceName,
		tags:      defaultServiceTags,
		port:      port,
		id:        id,
		namespace: ns,
		checks:    []*consulapi.AgentServiceCheck{check},
	}
	testcases := []struct {
		name      string
		setup     func(*mocks.ConsulClientInterface)
		expectErr bool
	}{
		{
			"success",
			func(cMock *mocks.ConsulClientInterface) {
				cMock.On("RegisterService", mock.Anything,
					mock.MatchedBy(func(r *consulapi.AgentServiceRegistration) bool {
						// expect these values as for the service registration request
						return r.ID == id &&
							r.Name == defaultServiceName &&
							r.Port == port &&
							r.Namespace == ns &&
							reflect.DeepEqual(r.Tags, defaultServiceTags) &&
							reflect.DeepEqual(r.Checks, consulapi.AgentServiceChecks{check})
					})).Return(nil)
			},
			false,
		},
		{
			"registration_errors",
			func(cMock *mocks.ConsulClientInterface) {
				cMock.On("RegisterService", mock.Anything, mock.Anything).Return(errors.New("mock error"))
			},
			true,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			mockClient := new(mocks.ConsulClientInterface)
			tc.setup(mockClient)

			m := &RegistrationManager{
				client:  mockClient,
				service: service,
				logger:  logging.NewNullLogger(),
			}

			err := m.RegisterService(context.Background())

			if !tc.expectErr {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
			mockClient.AssertExpectations(t)
		})
	}
}
