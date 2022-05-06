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

func TestNewSelfRegistrationManager(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		name            string
		conf            *SelfRegistrationManagerConfig
		expectedService *service
	}{
		{
			"defaults",
			&SelfRegistrationManagerConfig{
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
			&SelfRegistrationManagerConfig{
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
			c := new(mocks.ConsulClientInterface)
			m := NewSelfRegistrationManager(tc.conf, c)

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

func TestSelfRegistrationManager_defaultHTTPCheck(t *testing.T) {
	t.Parallel()
	id := "cts-123"
	port := 8558
	httpAddress := fmt.Sprintf("http://localhost:%d/v1/health", port)
	httpsAddress := fmt.Sprintf("https://localhost:%d/v1/health", port)
	checkID := fmt.Sprintf("%s-health", id)

	testcases := []struct {
		name     string
		conf     *SelfRegistrationManagerConfig
		expected *consulapi.AgentServiceCheck
	}{
		{
			"tls_disabled",
			&SelfRegistrationManagerConfig{
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
			&SelfRegistrationManagerConfig{
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

func TestSelfRegistrationManager_Start(t *testing.T) {
	t.Parallel()
	id := "cts-123"
	manager := &SelfRegistrationManager{
		service: &service{
			name: defaultServiceName,
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
				Name: defaultServiceName,
			},
		).Return(nil)
		mockClient.On("DeregisterService", mock.Anything, id).Return(nil)
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
		mockClient.On("DeregisterService", mock.Anything, mock.Anything).
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

func TestSelfRegistrationManager_deregister(t *testing.T) {
	t.Parallel()
	id := "cts-123"
	ctx := context.Background()
	manager := &SelfRegistrationManager{
		service: &service{
			name: defaultServiceName,
			id:   id,
		},
		logger: logging.NewNullLogger(),
		// mock client will be set per test case
	}

	t.Run("success", func(t *testing.T) {
		mockClient := new(mocks.ConsulClientInterface)
		mockClient.On("DeregisterService", ctx, id).Return(nil)
		manager.client = mockClient

		err := manager.deregister(ctx)
		assert.NoError(t, err)
		mockClient.AssertExpectations(t)
	})

	t.Run("error on deregister", func(t *testing.T) {
		mockClient := new(mocks.ConsulClientInterface)
		mockClient.On("DeregisterService", mock.Anything, mock.Anything).
			Return(errors.New("mock deregister error"))
		manager.client = mockClient

		err := manager.deregister(ctx)
		assert.Error(t, err)
		mockClient.AssertExpectations(t)
	})

	t.Run("ignore error on deregister missing ACL", func(t *testing.T) {
		mockClient := new(mocks.ConsulClientInterface)
		mockClient.On("DeregisterService", mock.Anything, mock.Anything).
			Return(&client.MissingConsulACLError{Err: errors.New("mock deregister error")})
		manager.client = mockClient

		err := manager.deregister(ctx)
		assert.NoError(t, err)
		mockClient.AssertExpectations(t)
	})
}

func TestSelfRegistrationManager_register(t *testing.T) {
	t.Parallel()
	id := "cts-123"
	port := 8558
	ns := "ns-1"
	check := defaultHTTPCheck(&SelfRegistrationManagerConfig{
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
					// expect these values for the service registration request
					&consulapi.AgentServiceRegistration{
						ID:        id,
						Name:      defaultServiceName,
						Port:      port,
						Namespace: ns,
						Tags:      defaultServiceTags,
						Checks:    consulapi.AgentServiceChecks{check},
					},
				).Return(nil)
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
		{
			"ignore_error_register_missing_ACL",
			func(cMock *mocks.ConsulClientInterface) {
				cMock.On("RegisterService", mock.Anything, mock.Anything).Return(&client.MissingConsulACLError{Err: errors.New("mock error")})
			},
			false,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			mockClient := new(mocks.ConsulClientInterface)
			tc.setup(mockClient)

			m := &SelfRegistrationManager{
				client:  mockClient,
				service: service,
				logger:  logging.NewNullLogger(),
			}

			err := m.register(context.Background())

			if !tc.expectErr {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
			mockClient.AssertExpectations(t)
		})
	}
}
