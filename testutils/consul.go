// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package testutils

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path/filepath"
	"testing"
	"time"

	"github.com/hashicorp/consul/sdk/testutil"
	"github.com/stretchr/testify/require"
)

// defaultTFBackendKVPath is the same as config package. Duplicating to avoid
// import cycles
const defaultTFBackendKVPath = "consul-terraform-sync/terraform"

// TestConsulServerConfig configures a test Consul server
type TestConsulServerConfig struct {
	HTTPSRelPath string
	PortHTTP     int // random port will be generated if unset
	PortHTTPS    int // random port will be generated if unset
}

// NewTestConsulServer starts a test Consul server as configured
func NewTestConsulServer(tb testing.TB, config TestConsulServerConfig) *testutil.TestServer {
	var certFile string
	var keyFile string
	if config.HTTPSRelPath != "" {
		path, err := filepath.Abs(config.HTTPSRelPath)
		require.NoError(tb, err, "unable to get absolute path of test certs")
		certFile = filepath.Join(path, "../testutils/certs/consul_cert.pem")
		keyFile = filepath.Join(path, "../testutils/certs/consul_key.pem")
	}

	srv, err := testutil.NewTestServerConfigT(tb,
		func(c *testutil.TestServerConfig) {
			c.LogLevel = "warn"
			c.Stdout = ioutil.Discard
			c.Stderr = ioutil.Discard

			// Support CTS connecting over HTTP2
			if config.HTTPSRelPath != "" {
				c.VerifyIncomingHTTPS = false
				c.CertFile = certFile
				c.KeyFile = keyFile
			}

			if config.PortHTTP != 0 {
				c.Ports.HTTP = config.PortHTTP
			}

			if config.PortHTTPS != 0 {
				c.Ports.HTTPS = config.PortHTTPS
			}
		})
	require.NoError(tb, err, "unable to start Consul server")

	return srv
}

// RegisterConsulService regsiters a service to the Consul Catalog. The Consul
// sdk/testutil package currently does not support a method to register multiple
// service instances, distinguished by their IDs.
func RegisterConsulService(tb testing.TB, srv *testutil.TestServer,
	s testutil.TestService, wait time.Duration) {
	registerConsulService(tb, srv, s, wait, nil)
}

// RegisterConsulServiceHealth is similar to RegisterConsulService and also
// sets the health status of the service.
func RegisterConsulServiceHealth(tb testing.TB, srv *testutil.TestServer,
	s testutil.TestService, wait time.Duration, health string) {
	registerConsulService(tb, srv, s, wait, &health)
}

func registerConsulService(tb testing.TB, srv *testutil.TestServer,
	s testutil.TestService, wait time.Duration, health *string) {

	var body bytes.Buffer
	enc := json.NewEncoder(&body)
	require.NoError(tb, enc.Encode(&s))

	u := fmt.Sprintf("http://%s/v1/agent/service/register", srv.HTTPAddr)
	resp := RequestHTTP(tb, http.MethodPut, u, body.String())
	defer resp.Body.Close()

	if health != nil {
		srv.AddCheck(tb, s.ID, s.ID, *health)
	}

	if wait.Seconds() == 0 {
		return
	}

	WaitForConsulServiceRegistered(tb, srv, s.ID, wait)
}

// WaitForConsulServiceRegistered polls Consul until the given service is registered or the timeout is reached.
func WaitForConsulServiceRegistered(tb testing.TB, srv *testutil.TestServer, serviceID string, wait time.Duration) {
	waitForConsulService(tb, srv, serviceID, true, wait)
}

// WaitForConsulServiceDeregistered polls Consul until the given service is deregistered or the timeout is reached.
func WaitForConsulServiceDeregistered(tb testing.TB, srv *testutil.TestServer, serviceID string, wait time.Duration) {
	waitForConsulService(tb, srv, serviceID, false, wait)
}

func waitForConsulService(tb testing.TB, srv *testutil.TestServer, serviceID string, expectRegister bool, wait time.Duration) {
	polling := make(chan struct{})
	stopPolling := make(chan struct{})
	go func() {
		for {
			select {
			case <-stopPolling:
				return
			default:
				ok := ConsulServiceRegistered(tb, srv, serviceID)
				if ok == expectRegister {
					polling <- struct{}{}
					return
				}
				time.Sleep(100 * time.Millisecond)
			}
		}
	}()

	select {
	case <-polling:
		return
	case <-time.After(wait):
		close(stopPolling)
		if expectRegister {
			tb.Fatalf("timed out after waiting for %v for service %q to register "+
				"with Consul", wait, serviceID)
		} else {
			tb.Fatalf("timed out after waiting for %v for service %q to deregister "+
				"with Consul", wait, serviceID)
		}
	}
}

// ListConsulServices returns a list of services registered with the Consul agent.
func ListConsulServices(tb testing.TB, srv *testutil.TestServer, filter string) map[string]ConsulService {
	u := fmt.Sprintf("http://%s/v1/agent/services", srv.HTTPAddr)
	if filter != "" {
		params := url.Values{}
		params.Add("filter", filter)
		u = fmt.Sprintf("%s?%s", u, params.Encode())
	}
	resp := RequestHTTP(tb, http.MethodGet, u, "")

	b, err := ioutil.ReadAll(resp.Body)
	require.NoError(tb, err)
	defer resp.Body.Close()

	require.Equal(tb, resp.StatusCode, 200, string(b))

	var services map[string]ConsulService
	err = json.Unmarshal(b, &services)
	require.NoError(tb, err)
	return services
}

// ConsulServiceRegistered returns whether the Consul service with the given ID is registered or not.
func ConsulServiceRegistered(tb testing.TB, srv *testutil.TestServer, serviceID string) bool {
	u := fmt.Sprintf("http://%s/v1/agent/service/%s", srv.HTTPAddr, serviceID)
	resp := RequestHTTP(tb, http.MethodGet, u, "")
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// ConsulService represents a Consul service with a subset of its attributes that are verified by the tests.
type ConsulService struct {
	ID      string
	Service string
	Tags    []string
	Address string
	Port    int
}

// GetConsulService returns a service instance with the given ID from the Consul agent.
func GetConsulService(tb testing.TB, srv *testutil.TestServer, serviceID string) (ConsulService, error) {
	u := fmt.Sprintf("http://%s/v1/agent/service/%s", srv.HTTPAddr, serviceID)
	resp := RequestHTTP(tb, http.MethodGet, u, "")
	defer resp.Body.Close()

	b, err := ioutil.ReadAll(resp.Body)
	require.NoError(tb, err)

	if resp.StatusCode != 200 {
		return ConsulService{}, errors.New(string(b))
	}

	var service ConsulService
	err = json.Unmarshal(b, &service)
	require.NoError(tb, err)
	return service, nil
}

// Bulk add test data for seeding consul
func AddServices(t testing.TB, srv *testutil.TestServer, svcs []testutil.TestService) {
	for _, s := range svcs {
		RegisterConsulService(t, srv, s, 0)
	}
}

func DeregisterConsulService(tb testing.TB, srv *testutil.TestServer, id string) {
	u := fmt.Sprintf("http://%s/v1/agent/service/deregister/%s", srv.HTTPAddr, id)
	resp := RequestHTTP(tb, http.MethodPut, u, "")
	defer resp.Body.Close()
}

func DeleteKV(tb testing.TB, srv *testutil.TestServer, key string) {
	u := fmt.Sprintf("http://%s/v1/kv/%s", srv.HTTPAddr, key)
	resp := RequestHTTP(tb, http.MethodDelete, u, "")
	defer resp.Body.Close()
}

// ConsulCheck represents a Consul check with a subset of its attributes that are verified by the tests.
type ConsulCheck struct {
	CheckID     string
	Name        string
	Status      string
	Notes       string
	Output      string
	ServiceID   string
	ServiceName string
	ServiceTags []string
	Type        string
}

// ListConsulChecks returns a list of checks from a Consul agent.
func ListConsulChecks(tb testing.TB, srv *testutil.TestServer) map[string]ConsulCheck {
	u := fmt.Sprintf("http://%s/v1/agent/checks", srv.HTTPAddr)
	resp := RequestHTTP(tb, http.MethodGet, u, "")

	b, err := ioutil.ReadAll(resp.Body)
	require.NoError(tb, err)
	defer resp.Body.Close()

	require.Equal(tb, resp.StatusCode, 200, string(b))

	var checks map[string]ConsulCheck
	err = json.Unmarshal(b, &checks)
	require.NoError(tb, err)
	return checks
}

// WaitForConsulCheckStatus polls a Consul check until the check has the given status or the timeout is reached.
func WaitForConsulCheckStatus(tb testing.TB, srv *testutil.TestServer, checkID, status string, wait time.Duration) (ConsulCheck, error) {
	polling := make(chan ConsulCheck)
	stopPolling := make(chan struct{})
	go func() {
		for {
			select {
			case <-stopPolling:
				return
			default:
				checks := ListConsulChecks(tb, srv)
				c, ok := checks[checkID]
				if !ok {
					// check does not exist but may exist eventually
					time.Sleep(100 * time.Millisecond)
					continue
				}
				if c.Status == status {
					polling <- c
					return
				}
				time.Sleep(100 * time.Millisecond)
			}
		}
	}()

	select {
	case c := <-polling:
		return c, nil
	case <-time.After(wait):
		close(stopPolling)
		return ConsulCheck{}, fmt.Errorf("timed out after waiting for %v for check %q to become %s",
			wait, checkID, status)
	}
}

// Generate service TestService entries.
// Services with different IDs and Names.
func TestServices(n int) []testutil.TestService {
	return generateServices(n,
		func(i int) string {
			return fmt.Sprintf("svc-name-%d", i)
		},
		func(i int) string {
			return fmt.Sprintf("svc-id-%d", i)
		})
}

// shorter name for the formatting functions
type fmtFunc func(i int) string

// does the actual work of generating the TestService objects
func generateServices(n int, namefmt, idfmt fmtFunc) []testutil.TestService {
	baseport := 30000
	services := make([]testutil.TestService, n)
	for i := 0; i < n; i++ {
		services[i] = testutil.TestService{
			Name:    namefmt(i),
			ID:      idfmt(i),
			Address: "127.0.0.2",
			Port:    baseport + i,
			Tags:    []string{},
		}
	}
	return services
}

// CheckStateFile checks statefile in the default Terraform backend ConsulKV.
func CheckStateFile(t *testing.T, consulAddr, taskname string) {
	u := fmt.Sprintf("http://%s/v1/kv/%s-env:%s", consulAddr,
		defaultTFBackendKVPath, taskname)
	resp := RequestHTTP(t, http.MethodGet, u, "")
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode, "Unable to find statefile"+
		" in Consul KV")
}

// ---------------------------------------------------------------------------
// The below functions are not used in tests but are handy to keep around for
// when you need to check on Consul's data .

// All registered services
func ShowMeServices(t testing.TB, srv *testutil.TestServer) {
	u := fmt.Sprintf("http://%s/v1/agent/services", srv.HTTPAddr)
	resp := RequestHTTP(t, http.MethodGet, u, "")
	b, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	fmt.Println(string(b))
	defer resp.Body.Close()
}

// Health status for all services
func ShowMeHealth(t testing.TB, srv *testutil.TestServer, svcName string) {
	// get the node
	u := fmt.Sprintf("http://%s/v1/health/service/%s", srv.HTTPAddr, svcName)
	resp := RequestHTTP(t, http.MethodGet, u, "")
	b, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	resp.Body.Close()
	nodes := []struct{ Node struct{ Node string } }{}
	json.Unmarshal(b, &nodes)
	node := nodes[0].Node.Node

	// all services on that node
	u = fmt.Sprintf("http://%s/v1/health/node/%s", srv.HTTPAddr, node)
	resp = RequestHTTP(t, http.MethodGet, u, "")
	b, err = ioutil.ReadAll(resp.Body)
	require.NoError(t, err)
	resp.Body.Close()

	fmt.Println(string(b))
}
