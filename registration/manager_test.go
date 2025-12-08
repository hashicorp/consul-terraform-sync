// Copyright IBM Corp. 2020, 2025
// SPDX-License-Identifier: MPL-2.0

package registration

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/consul-terraform-sync/client"
	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/logging"
	mocks "github.com/hashicorp/consul-terraform-sync/mocks/client"
	consulapi "github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestNewServiceRegistrationManager(t *testing.T) {
	t.Parallel()
	id := "cts-123"
	port := 123
	serviceName := "cts"
	serviceAddress := "172.1.2.3"
	ns := "ns-1"
	testcases := []struct {
		name            string
		conf            *ServiceRegistrationManagerConfig
		expectedService *service
	}{
		{
			"defaults",
			&ServiceRegistrationManagerConfig{
				ID:                  id,
				Port:                port,
				TLSEnabled:          false,
				ServiceRegistration: config.DefaultServiceRegistrationConfig(),
			},
			&service{
				name:      config.DefaultServiceName,
				tags:      defaultServiceTags,
				id:        id,
				address:   "",
				port:      port,
				namespace: "",
			},
		},
		{
			"configured",
			&ServiceRegistrationManagerConfig{
				ID:         id,
				Port:       port,
				TLSEnabled: false,
				ServiceRegistration: &config.ServiceRegistrationConfig{
					ServiceName: &serviceName,
					Address:     &serviceAddress,
					Namespace:   &ns,
					DefaultCheck: &config.DefaultCheckConfig{
						Enabled: config.Bool(true),
					},
				},
			},
			&service{
				name:      serviceName,
				tags:      defaultServiceTags,
				id:        id,
				address:   serviceAddress,
				port:      port,
				namespace: ns,
			},
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			c := new(mocks.ConsulClientInterface)
			m := NewServiceRegistrationManager(tc.conf, c)

			// Verify general attributes
			assert.NotNil(t, m)
			assert.Equal(t, m.client, c)
			assert.NotNil(t, m.logger)

			// Verify service attributes and health check attributes
			assert.NotNil(t, m.service)
			if tc.expectedService.checks == nil {
				tc.expectedService.checks = []*consulapi.AgentServiceCheck{defaultHTTPCheck(tc.conf)}
			}
			assert.Equal(t, tc.expectedService, m.service)
		})
	}
}

func TestServiceRegistrationManager_defaultHTTPCheck(t *testing.T) {
	t.Parallel()
	id := "cts-123"
	port := 8558
	httpAddress := fmt.Sprintf("http://localhost:%d/v1/health", port)
	httpsAddress := fmt.Sprintf("https://localhost:%d/v1/health", port)
	checkID := fmt.Sprintf("%s-health", id)

	testcases := []struct {
		name     string
		conf     *ServiceRegistrationManagerConfig
		expected *consulapi.AgentServiceCheck
	}{
		{
			"tls_disabled",
			&ServiceRegistrationManagerConfig{
				ID:                  id,
				Port:                port,
				TLSEnabled:          false,
				ServiceRegistration: config.DefaultServiceRegistrationConfig(),
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
			&ServiceRegistrationManagerConfig{
				ID:                  id,
				Port:                port,
				TLSEnabled:          true,
				ServiceRegistration: config.DefaultServiceRegistrationConfig(),
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
		{
			"address_configured",
			&ServiceRegistrationManagerConfig{
				ID:         id,
				Port:       port,
				TLSEnabled: true,
				ServiceRegistration: &config.ServiceRegistrationConfig{
					DefaultCheck: &config.DefaultCheckConfig{
						Address: config.String("http://127.0.0.1:5885"),
					},
				},
			},
			&consulapi.AgentServiceCheck{
				HTTP:                           "http://127.0.0.1:5885" + defaultHealthEndpoint,
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

func TestServiceRegistrationManager_Start(t *testing.T) {
	t.Parallel()
	id := "cts-123"
	name := "cts-service"
	manager := &ServiceRegistrationManager{
		service: &service{
			name: name,
			id:   id,
		},
		logger: logging.NewNullLogger(),
		// mock client will be set per test case
	}

	t.Run("start registers, cancel deregisters", func(t *testing.T) {
		mockClient := new(mocks.ConsulClientInterface)
		mockClient.On("RegisterService", mock.Anything,
			&consulapi.AgentServiceRegistration{
				ID:   id,
				Name: name,
			},
		).Return(nil)
		mockClient.On("DeregisterService", mock.Anything, id, mock.Anything).Return(nil)
		manager.client = mockClient

		ctx, cancel := context.WithCancel(context.Background())
		errCh := make(chan error)
		go func() {
			if err := manager.Start(ctx); err != nil {
				errCh <- err
			}
		}()
		cancel()

		select {
		case err := <-errCh:
			// Confirm that exit is due to context cancel
			assert.Equal(t, err, context.Canceled)
		case <-time.After(time.Second * 5):
			t.Fatal("Start did not exit properly from cancelling context")
		}
		mockClient.AssertExpectations(t)
	})

	t.Run("error on register", func(t *testing.T) {
		mockClient := new(mocks.ConsulClientInterface)
		mockClient.On("RegisterService", mock.Anything, mock.Anything).
			Return(errors.New("mock register error"))
		manager.client = mockClient

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		errCh := make(chan error)
		go func() {
			if err := manager.Start(ctx); err != nil {
				errCh <- err
			}
		}()

		select {
		case err := <-errCh:
			// Confirm that exit is due to mock error
			assert.Contains(t, err.Error(), "mock register error")
		case <-time.After(time.Second * 5):
			t.Fatal("Start did not exit properly from cancelling context")
		}
		mockClient.AssertExpectations(t)
	})

	t.Run("error on deregister", func(t *testing.T) {
		mockClient := new(mocks.ConsulClientInterface)
		mockClient.On("RegisterService", mock.Anything, mock.Anything).
			Return(nil)
		mockClient.On("DeregisterService", mock.Anything, mock.Anything, mock.Anything).
			Return(errors.New("mock deregister error"))
		manager.client = mockClient

		ctx, cancel := context.WithCancel(context.Background())
		errCh := make(chan error)
		go func() {
			if err := manager.Start(ctx); err != nil {
				errCh <- err
			}
		}()
		cancel() // cancel to initiate deregister

		select {
		case err := <-errCh:
			// Confirm that exit is due to mock error
			assert.Contains(t, err.Error(), "mock deregister error")
		case <-time.After(time.Second * 5):
			t.Fatal("Run did not exit properly from cancelling context")
		}
		mockClient.AssertExpectations(t)
	})
}

func TestServiceRegistrationManager_deregister(t *testing.T) {
	t.Parallel()
	id := "cts-123"
	ctx := context.Background()
	manager := &ServiceRegistrationManager{
		service: &service{
			name: config.DefaultServiceName,
			id:   id,
		},
		logger: logging.NewNullLogger(),
		// mock client will be set per test case
	}

	t.Run("success", func(t *testing.T) {
		mockClient := new(mocks.ConsulClientInterface)
		mockClient.On("DeregisterService", ctx, id, &consulapi.QueryOptions{}).Return(nil)
		manager.client = mockClient

		err := manager.deregister(ctx)
		assert.NoError(t, err)
		mockClient.AssertExpectations(t)
	})

	t.Run("success with namespace", func(t *testing.T) {
		mockClient := new(mocks.ConsulClientInterface)
		mockClient.On("DeregisterService", ctx, id,
			mock.MatchedBy(func(r *consulapi.QueryOptions) bool {
				return r.Namespace == "test-ns"
			})).Return(nil)
		nsManager := &ServiceRegistrationManager{
			service: &service{
				name:      config.DefaultServiceName,
				id:        id,
				namespace: "test-ns",
			},
			logger: logging.NewNullLogger(),
			client: mockClient,
		}
		err := nsManager.deregister(ctx)
		assert.NoError(t, err)
		mockClient.AssertExpectations(t)
	})

	t.Run("error on deregister", func(t *testing.T) {
		mockClient := new(mocks.ConsulClientInterface)
		mockClient.On("DeregisterService", mock.Anything, mock.Anything, mock.Anything).
			Return(errors.New("mock deregister error"))
		manager.client = mockClient

		err := manager.deregister(ctx)
		assert.Error(t, err)
		mockClient.AssertExpectations(t)
	})

	t.Run("error on deregister missing ACL", func(t *testing.T) {
		mockClient := new(mocks.ConsulClientInterface)
		mockClient.On("DeregisterService", mock.Anything, mock.Anything, mock.Anything).
			Return(&client.MissingConsulACLError{Err: errors.New("mock deregister error")})
		manager.client = mockClient

		err := manager.deregister(ctx)
		assert.Error(t, err)
		var missingConsulACLError *client.MissingConsulACLError
		assert.ErrorAs(t, err, &missingConsulACLError)
		mockClient.AssertExpectations(t)
	})
}

func TestServiceRegistrationManager_register(t *testing.T) {
	t.Parallel()
	id := "cts-123"
	name := "cts-service"
	port := 8558
	ns := "ns-1"

	tags := make([]string, 0, 3)
	tags = append(tags, defaultServiceTags...)
	tags = append(tags, "one", "two")

	check := defaultHTTPCheck(&ServiceRegistrationManagerConfig{
		ID:   id,
		Port: port,
		Tags: []string{"one", "two"},
		ServiceRegistration: &config.ServiceRegistrationConfig{
			Namespace: &ns,
			DefaultCheck: &config.DefaultCheckConfig{
				Enabled: config.Bool(true),
			},
		},
	})
	service := &service{
		name:      name,
		tags:      tags,
		port:      port,
		id:        id,
		namespace: ns,
		checks:    []*consulapi.AgentServiceCheck{check},
	}
	testcases := []struct {
		name              string
		setup             func(*mocks.ConsulClientInterface)
		expectErr         bool
		isMissingACLError bool
	}{
		{
			name: "success",
			setup: func(cMock *mocks.ConsulClientInterface) {
				cMock.On("RegisterService", mock.Anything,
					// expect these values for the service registration request
					&consulapi.AgentServiceRegistration{
						ID:        id,
						Name:      name,
						Port:      port,
						Namespace: ns,
						Tags:      tags,
						Checks:    consulapi.AgentServiceChecks{check},
					},
				).Return(nil)
			},
		},
		{
			name: "registration_errors",
			setup: func(cMock *mocks.ConsulClientInterface) {
				cMock.On("RegisterService", mock.Anything, mock.Anything).Return(errors.New("mock error"))
			},
			expectErr: true,
		},
		{
			name: "ignore_error_register_missing_ACL",
			setup: func(cMock *mocks.ConsulClientInterface) {
				cMock.On("RegisterService", mock.Anything, mock.Anything).Return(&client.MissingConsulACLError{Err: errors.New("mock error")})
			},
			expectErr:         true,
			isMissingACLError: true,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			mockClient := new(mocks.ConsulClientInterface)
			tc.setup(mockClient)

			m := &ServiceRegistrationManager{
				client:  mockClient,
				service: service,
				logger:  logging.NewNullLogger(),
			}

			err := m.register(context.Background())

			if !tc.expectErr {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				var missingConsulACLError *client.MissingConsulACLError
				assert.Equal(t, tc.isMissingACLError, errors.As(err, &missingConsulACLError))
			}
			mockClient.AssertExpectations(t)
		})
	}
}
