package testutils

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/hashicorp/consul/sdk/testutil"
	"github.com/stretchr/testify/require"
)

// defaultTFBackendKVPath is the same as config package. Duplicating to avoid
// import cycles
const defaultTFBackendKVPath = "consul-terraform-sync/terraform"

// TestConsulServerConfig configures a test Consul server
type TestConsulServerConfig struct {
	HTTPSRelPath string
	PortHTTPS    int // random port will be generated if unset
}

// NewTestConsulServer starts a test Consul server as configured
func NewTestConsulServer(tb testing.TB, config TestConsulServerConfig) *testutil.TestServer {
	log.SetOutput(ioutil.Discard)

	var certFile string
	var keyFile string
	if config.HTTPSRelPath != "" {
		path, err := filepath.Abs(config.HTTPSRelPath)
		require.NoError(tb, err, "unable to get absolute path of test certs")
		certFile = filepath.Join(path, "cert.pem")
		keyFile = filepath.Join(path, "key.pem")
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

			if config.PortHTTPS != 0 {
				c.Ports.HTTPS = config.PortHTTPS
			}
		})
	require.NoError(tb, err, "unable to start Consul server")

	return srv
}

// Bulk add test data for seeding consul
func AddServices(t testing.TB, srv *testutil.TestServer, svcs []testutil.TestService) {
	for _, s := range svcs {
		RegisterConsulService(t, srv, s, testutil.HealthPassing)
	}
}

// Generate service TestService entries.
// Services with different IDs and Names.
func TestServices(n int) []testutil.TestService {
	return generateServices(n,
		func(i int) string {
			return fmt.Sprintf("svc_name_%d", i)
		},
		func(i int) string {
			return fmt.Sprintf("svc_id_%d", i)
		})
}

// Generate service instance TestService entries.
// Services with different IDs but with the same name.
func TestInstances(n int) []testutil.TestService {
	return generateServices(n,
		func(i int) string {
			return fmt.Sprintf("svc_name_common")
		},
		func(i int) string {
			return fmt.Sprintf("svc_id_%d", i)
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
			Port:    int(baseport + i),
			Tags:    []string{},
		}
	}
	return services
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

// CheckStateFile checks statefile in the default Terraform backend ConsulKV.
func CheckStateFile(t *testing.T, consulAddr, taskname string) {
	u := fmt.Sprintf("http://%s/v1/kv/%s-env:%s", consulAddr,
		defaultTFBackendKVPath, taskname)
	resp := RequestHTTP(t, http.MethodGet, u, "")
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode, "Unable to find statefile"+
		" in Consul KV")
}
