package api

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/driver"
	"github.com/hashicorp/consul-terraform-sync/event"
	mocks "github.com/hashicorp/consul-terraform-sync/mocks/driver"
	"github.com/hashicorp/consul-terraform-sync/testutils"
	"github.com/hashicorp/go-rootcerts"
)

func TestServe(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		path       string
		method     string
		body       string
		statusCode int
	}{
		{
			"overall status",
			"status",
			http.MethodGet,
			"",
			http.StatusOK,
		},
		{
			"task status: all",
			"status/tasks",
			http.MethodGet,
			"",
			http.StatusOK,
		},
		{
			"task status: single",
			"status/tasks/task_b",
			http.MethodGet,
			"",
			http.StatusOK,
		},
		{
			"update task (patch)",
			"tasks/task_b",
			http.MethodPatch,
			`{"enabled": true}`,
			http.StatusOK,
		},
		{
			"delete task",
			"tasks/task_b",
			http.MethodDelete,
			"",
			http.StatusOK,
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	port := testutils.FreePort(t)

	task, err := driver.NewTask(driver.TaskConfig{Enabled: true})
	require.NoError(t, err)

	drivers := driver.NewDrivers()
	d := new(mocks.Driver)
	d.On("UpdateTask", mock.Anything, mock.Anything).
		Return(driver.InspectPlan{}, nil).Once()
	d.On("Task").Return(task)
	drivers.Add("task_b", d)

	api, err := NewAPI(APIConfig{
		Drivers: drivers,
		Port:    port,
	})
	require.NoError(t, err)

	go api.Serve(ctx)
	time.Sleep(3 * time.Second)

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			u := fmt.Sprintf("http://localhost:%d/%s/%s",
				port, defaultAPIVersion, tc.path)

			resp := testutils.RequestHTTP(t, tc.method, u, tc.body)
			defer resp.Body.Close()
			assert.Equal(t, tc.statusCode, resp.StatusCode)
		})
	}
}

func TestServe_context_cancel(t *testing.T) {
	t.Parallel()

	port := testutils.FreePort(t)
	api, err := NewAPI(APIConfig{Port: port})
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error)
	go func() {
		err := api.Serve(ctx)
		if err != nil {
			errCh <- err
		}
	}()
	cancel()

	select {
	case err := <-errCh:
		if err != context.Canceled {
			t.Error("wanted 'context canceled', got:", err)
		}
	case <-time.After(time.Second * 5):
		t.Fatal("Run did not exit properly from cancelling context")
	}
}

func TestServeWithTLS(t *testing.T) {
	t.Parallel()
	rootCert := "../testutils/certs/localhost_cert.pem"
	rootKey := "../testutils/certs/localhost_key.pem"

	leafCert := "../testutils/certs/localhost_leaf_cert.pem"
	leafKey := "../testutils/certs/localhost_leaf_key.pem"

	cases := []struct {
		name         string
		valid        bool
		serverCert   string
		serverKey    string
		clientCACert string
	}{
		{
			"self_signed_trusted",
			true,
			rootCert,
			rootKey,
			rootCert,
		},
		{
			"leaf_cert_trusted",
			true,
			leafCert,
			leafKey,
			rootCert,
		},
		{
			// client does not trust the CTS certificate
			"ca_untrusted",
			false,
			rootCert,
			rootKey,
			"../testutils/certs/localhost_cert2.pem",
		},
		{
			// client uses the default global CA, but server cert is
			// self-signed, so would not be trusted
			"ca_default",
			false,
			rootCert,
			rootKey,
			"",
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	task, err := driver.NewTask(driver.TaskConfig{Enabled: true})
	require.NoError(t, err)

	drivers := driver.NewDrivers()
	d := new(mocks.Driver)
	d.On("UpdateTask", mock.Anything, mock.Anything).
		Return(driver.InspectPlan{}, nil).Once()
	d.On("Task").Return(task)
	drivers.Add("task_b", d)

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Serve CTS API with TLS enabled
			port := testutils.FreePort(t)
			tlsConfig := &config.CTSTLSConfig{
				Enabled: config.Bool(true),
				Cert:    config.String(tc.serverCert),
				Key:     config.String(tc.serverKey),
			}
			api, err := NewAPI(APIConfig{
				Drivers: drivers,
				Port:    port,
				TLS:     tlsConfig,
			})
			require.NoError(t, err)
			go api.Serve(ctx)
			time.Sleep(3 * time.Second)

			// Set up a client with the CA for the test case
			tlsConf := &tls.Config{}
			if tc.clientCACert != "" {
				caCertPool := x509.NewCertPool()
				caCert, err := ioutil.ReadFile(tc.clientCACert)
				require.NoError(t, err)
				caCertPool.AppendCertsFromPEM(caCert)
				tlsConf = &tls.Config{
					RootCAs: caCertPool,
				}
			}
			client := &http.Client{Transport: &http.Transport{
				TLSClientConfig: tlsConf,
			}}

			// Make request to HTTPS endpoint
			u := fmt.Sprintf("https://localhost:%d/%s/status",
				port, defaultAPIVersion)
			resp, err := client.Get(u)
			if tc.valid {
				require.NoError(t, err)
				assert.Equal(t, http.StatusOK, resp.StatusCode)
			} else {
				require.Error(t, err)
			}

			// Make request to HTTP endpoint, expect a 400 Bad Request
			u = fmt.Sprintf("http://localhost:%d/%s/status",
				port, defaultAPIVersion)
			resp, err = client.Get(u)
			require.NoError(t, err)
			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		})
	}
}

func TestServeWithMutualTLS(t *testing.T) {
	t.Parallel()

	caCert := "../testutils/certs/localhost_cert.pem"
	cases := []struct {
		name       string
		valid      bool
		clientCert string
		clientKey  string
	}{
		{
			"valid_client_cert",
			true,
			"../testutils/certs/localhost_leaf_cert.pem",
			"../testutils/certs/localhost_leaf_key.pem",
		},
		{
			"ca_cert_for_client_cert",
			true,
			"../testutils/certs/localhost_cert.pem",
			"../testutils/certs/localhost_key.pem",
		},
		{
			"no_client_cert",
			false,
			"",
			"",
		},
		{
			"untrusted_client_cert",
			false,
			"../testutils/certs/localhost_cert2.pem",
			"../testutils/certs/localhost_key2.pem",
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	port := testutils.FreePort(t)
	serverCert := "../testutils/certs/localhost_cert2.pem"
	serverKey := "../testutils/certs/localhost_key2.pem"
	tlsConfig := &config.CTSTLSConfig{
		Enabled:        config.Bool(true),
		Cert:           config.String(serverCert),
		Key:            config.String(serverKey),
		VerifyIncoming: config.Bool(true),
		CACert:         config.String(caCert),
	}
	api, err := NewAPI(APIConfig{
		Port: port,
		TLS:  tlsConfig,
	})
	require.NoError(t, err)
	go api.Serve(ctx)
	time.Sleep(3 * time.Second)

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Set up a client that trusts the server's self-signed certificate and
			// uses the client certificate of the test case
			clientTLS := &tls.Config{}
			rootcerts.ConfigureTLS(clientTLS, &rootcerts.Config{
				CAFile: serverCert,
			})
			if tc.clientCert != "" {
				clientCert, err := tls.LoadX509KeyPair(tc.clientCert, tc.clientKey)
				require.NoError(t, err)
				clientTLS.Certificates = []tls.Certificate{clientCert}
			}
			client := &http.Client{Transport: &http.Transport{
				TLSClientConfig: clientTLS,
			}}

			// Make request to HTTPS endpoint
			u := fmt.Sprintf("https://localhost:%d/%s/status",
				port, defaultAPIVersion)
			resp, err := client.Get(u)
			if tc.valid {
				require.NoError(t, err)
				assert.Equal(t, http.StatusOK, resp.StatusCode)
			} else {
				require.Error(t, err)
			}

			// Make request to HTTP endpoint, expect a 400 Bad Request
			u = fmt.Sprintf("http://localhost:%d/%s/status",
				port, defaultAPIVersion)
			resp, err = client.Get(u)
			require.NoError(t, err)
			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		})
	}
}

// TestServeWithMutualTLS_MultipleCA tests using ca_path to set multiple
// CAs when verify_incoming is enabled. It checks that only client certs
// issued by any of the CAs in the path are considered valid.
func TestServeWithMutualTLS_MultipleCA(t *testing.T) {
	t.Parallel()

	caFiles := []string{
		"../testutils/certs/localhost_cert.pem",
		"../testutils/certs/localhost_cert2.pem",
	}

	cases := []struct {
		name       string
		valid      bool
		clientCert string
		clientKey  string
	}{
		{
			// localhost_leaf_cert was issued by localhost_cert
			"client_uses_leaf_cert",
			true,
			"../testutils/certs/localhost_leaf_cert.pem",
			"../testutils/certs/localhost_leaf_key.pem",
		},
		{
			"client_uses_valid_ca_cert",
			true,
			"../testutils/certs/localhost_cert.pem",
			"../testutils/certs/localhost_key.pem",
		},
		{
			"client_uses_valid_ca_cert2",
			true,
			"../testutils/certs/localhost_cert2.pem",
			"../testutils/certs/localhost_key2.pem",
		},
		{
			"no_client_cert",
			false,
			"",
			"",
		},
		{
			"untrusted_client_cert",
			false,
			"../testutils/certs/localhost_cert3.pem",
			"../testutils/certs/localhost_key3.pem",
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	port := testutils.FreePort(t)
	serverCert := "../testutils/certs/localhost_cert.pem"
	serverKey := "../testutils/certs/localhost_key.pem"

	// Copy multiple CA files to a temporary directory
	tmpDir := "tmp_ca_path_mtls"
	cleanup := testutils.MakeTempDir(t, tmpDir)
	defer cleanup()
	for _, src := range caFiles {
		input, err := ioutil.ReadFile(src)
		file := filepath.Base(src)
		dest := filepath.Join(tmpDir, file)
		require.NoError(t, err)
		err = ioutil.WriteFile(dest, input, 0644)
		require.NoError(t, err)
	}

	// Configure and start CTS with mTLS enabled using ca_path
	tlsConfig := &config.CTSTLSConfig{
		Enabled:        config.Bool(true),
		Cert:           config.String(serverCert),
		Key:            config.String(serverKey),
		VerifyIncoming: config.Bool(true),
		CAPath:         config.String(tmpDir),
	}
	api, err := NewAPI(APIConfig{
		Port: port,
		TLS:  tlsConfig,
	})
	require.NoError(t, err)
	go api.Serve(ctx)
	time.Sleep(3 * time.Second)

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Set up a client that trusts the server's self-signed certificate and
			// uses the client certificate of the test case
			clientTLS := &tls.Config{}
			rootcerts.ConfigureTLS(clientTLS, &rootcerts.Config{
				CAFile: serverCert,
			})
			if tc.clientCert != "" {
				clientCert, err := tls.LoadX509KeyPair(tc.clientCert, tc.clientKey)
				require.NoError(t, err)
				clientTLS.Certificates = []tls.Certificate{clientCert}
			}
			client := &http.Client{Transport: &http.Transport{
				TLSClientConfig: clientTLS,
			}}

			// Make request to HTTPS endpoint
			u := fmt.Sprintf("https://localhost:%d/%s/status",
				port, defaultAPIVersion)
			resp, err := client.Get(u)

			if tc.valid {
				require.NoError(t, err)
				assert.Equal(t, http.StatusOK, resp.StatusCode)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestJsonResponse(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		code     int
		response interface{}
	}{
		{
			"task status: error",
			http.StatusBadRequest,
			map[string]string{
				"error": "bad request",
			},
		},
		{
			"task status: success",
			http.StatusOK,
			map[string]TaskStatus{
				"task_a": TaskStatus{
					TaskName:  "task_a",
					Status:    StatusErrored,
					Providers: []string{"local", "null", "f5"},
					Services:  []string{"api", "web", "db"},
					EventsURL: "/v1/status/tasks/test_task?include=events",
				},
				"task_b": TaskStatus{
					TaskName:  "task_b",
					Status:    StatusUnknown,
					Providers: []string{},
					Services:  []string{},
					EventsURL: "",
				},
			},
		},
		{
			"task status: success with events",
			http.StatusOK,
			map[string]TaskStatus{
				"task_a": TaskStatus{
					TaskName:  "task_a",
					Status:    StatusErrored,
					Providers: []string{"local", "null", "f5"},
					Services:  []string{"api", "web", "db"},
					EventsURL: "/v1/status/tasks/test_task?include=events",
					Events: []event.Event{
						event.Event{
							ID:        "123",
							TaskName:  "task_a",
							StartTime: time.Now(),
							EndTime:   time.Now(),
							Success:   true,
							Config: &event.Config{
								Providers: []string{"local", "null", "f5"},
								Services:  []string{"api", "web", "db"},
								Source:    "./test_modules/local_instances_file",
							},
						},
						event.Event{
							ID:        "456",
							TaskName:  "task_a",
							StartTime: time.Now(),
							EndTime:   time.Now(),
							Success:   false,
							EventError: &event.Error{
								Message: "there was an error :(",
							},
							Config: &event.Config{
								Providers: []string{"local", "null", "f5"},
								Services:  []string{"api", "web", "db"},
								Source:    "./test_modules/local_instances_file",
							},
						},
					},
				},
			},
		},
		{
			"overall status: success",
			http.StatusOK,
			OverallStatus{
				TaskSummary: TaskSummary{
					Status: StatusSummary{
						Successful: 1,
						Errored:    0,
						Critical:   1,
					},
					Enabled: EnabledSummary{
						True:  2,
						False: 5,
					},
				},
			},
		},
		{
			"update task inspect",
			http.StatusOK,
			UpdateTaskResponse{
				Inspect: &driver.InspectPlan{
					ChangesPresent: true,
					Plan:           "plan!",
				},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			err := jsonResponse(w, tc.code, tc.response)
			assert.NoError(t, err)
		})
	}
}

func TestGetTaskName(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		path      string
		expectErr bool
		expected  string
	}{
		{
			"all task statuses",
			"/v1/status/tasks",
			false,
			"",
		},
		{
			"task status for a specific task",
			"/v1/status/tasks/my_specific_task",
			false,
			"my_specific_task",
		},
		{
			"empty task name",
			"/v1/status/tasks/",
			false,
			"",
		},
		{
			"tasks task name",
			"/v1/status/tasks/tasks",
			false,
			"tasks",
		},
		{
			"invalid name",
			"/v1/status/tasks/mytask/stuff",
			true,
			"",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := getTaskName(tc.path, taskStatusPath, "v1")
			if tc.expectErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, actual)
		})
	}
}
