//go:build e2e
// +build e2e

// $ go test -run TestCompatibility_Consul ./e2e/compatibility -tags=e2e -v
package compatibility

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/consul-terraform-sync/api"
	"github.com/hashicorp/consul-terraform-sync/templates/tftmpl"
	"github.com/hashicorp/consul-terraform-sync/testutils"
	capi "github.com/hashicorp/consul/api"
	"github.com/hashicorp/go-getter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	tempDirPrefix = "tmp_"
	resourcesDir  = "resources"
	configFile    = "config.hcl"

	nullTaskName = "null_task"

	defaultWaitForAPI   = 30 * time.Second
	defaultWaitForEvent = 8 * time.Second
)

// TestCompatibility_Compile confirms that the compatibility test(s) are
// compilable. Compatibility tests are only run weekly. This test is intended
// to run with each change (vs. weekly) to do a basic check that the tests are
// still in a compilable state.
func TestCompatibility_Compile(t *testing.T) {
	// no-op
}

func TestCompatibility_Consul(t *testing.T) {
	// Tested only OSS GA releases for the highest patch version given a
	// major minor version. v1.4.5 starts losing compatibility, details in
	// comments. Theoretical compatible versions 0.1.0 GA:
	consulVersions := []string{
		"1.11.0-beta1",
		"1.10.3",
		"1.9.10",
		"1.8.16",
		"1.7.14",
		"1.6.10",
		"1.5.3",
	}

	cases := []struct {
		name              string
		testCompatibility func(t *testing.T, tempDir string, port int)
	}{
		{
			// consulKV terraform backend
			"compat_consul_backend",
			testConsulBackendCompatibility,
		},
		{
			// adding and removing service instances
			"compat_service_instances",
			testServiceInstanceCompatibility,
		},
		{
			// changing service values
			"compat_service_values",
			testServiceValuesCompatibility,
		},
		{
			// filtering health service api by tag
			"compat_tag_api_query",
			testTagQueryCompatibility,
		},
		{
			// changing node values
			"compat_node_values",
			testNodeValuesCompatibility,
		},
	}

	wd, err := os.Getwd()
	require.NoError(t, err)

	for i := range consulVersions {
		cv := consulVersions[i]
		t.Run(cv, func(t *testing.T) {
			t.Parallel()

			tempDir := filepath.Join(wd, fmt.Sprint(tempDirPrefix, strings.ReplaceAll(t.Name(), "/", "_")))
			cleanup := testutils.MakeTempDir(t, tempDir)
			execPath := downloadConsul(t, tempDir, cv)

			// Output the Consul version
			consulVersion, err := exec.Command(execPath, "version").Output()
			require.NoError(t, err)
			t.Logf("%s\n%s", t.Name(), consulVersion)

			for _, tc := range cases {
				t.Run(tc.name, func(t *testing.T) {
					port := testutils.FreePort(t)

					stop := runConsul(t, execPath, port)
					defer stop()

					testTempDir := filepath.Join(tempDir, tc.name)
					testutils.MakeTempDir(t, testTempDir)
					tc.testCompatibility(t, testTempDir, port)
				})
			}

			err = cleanup()
			require.NoError(t, err)
		})
	}
}

// testConsulBackendCompatibility tests the compatibility of all the Consul
// features needed to use ConsulKV as the Terraform backend. ConsulKV is the
// default backend of CTS, so we need to ensure backwards compatibility
//
// From research, the following Consul features are used:
//  - Consul KV API (GET, PUT, DELETE)
//  - Consul KV API query parameters (cas, consistent, wait, acquire, key,
//	  separator, flags)
//  - Session API (Destroy, Create)
func testConsulBackendCompatibility(t *testing.T, tempDir string, port int) {
	config := baseConfig(tempDir, port) + nullTask()
	configPath := filepath.Join(tempDir, configFile)
	testutils.WriteFile(t, configPath, config)

	cts, stop := api.StartCTS(t, configPath)
	defer stop(t)
	err := cts.WaitForAPI(defaultWaitForAPI)
	require.NoError(t, err)

	// Test: ConsulKV backend
	// Register a service and confirm that TF state file is stored in ConsulKV
	registerService(t, &capi.AgentServiceRegistration{ID: "db1", Name: "db"}, port)
	addr := fmt.Sprintf("localhost:%d", port)
	testutils.CheckStateFile(t, addr, nullTaskName)
}

// testServiceInstanceCompatibility tests the compatibility of Consul's Health
// Service API response's service instances. To test service instances, add and
// remove service instances and confirm that CTS task execution and resource
// creation is successful.
func testServiceInstanceCompatibility(t *testing.T, tempDir string, port int) {
	config := baseConfig(tempDir, port) + basicTask("db_task", "db", "api") +
		basicTask("web_task", "api", "web")
	configPath := filepath.Join(tempDir, configFile)
	testutils.WriteFile(t, configPath, config)

	cts, stop := api.StartCTS(t, configPath)
	defer stop(t)
	err := cts.WaitForAPI(defaultWaitForAPI)
	require.NoError(t, err)

	// Test adding and removing service instances
	// 0. Confirm no resources created yet
	// 1. Register db service instances. Confirm _only_ db_task resources
	//    created i.e. no web_task resources created.
	// 2. Register a web service instance. Confirm web_task resource created and
	//    db_task resources unchanged
	// 3. Deregister one db service instance. Confirm db_task resource now one and
	//    web_task resources unchanged

	// 0. no resources created yet
	dbResourcesPath := filepath.Join(tempDir, "db_task", resourcesDir)
	webResourcesPath := filepath.Join(tempDir, "web_task", resourcesDir)
	testutils.CheckDir(t, false, dbResourcesPath)
	testutils.CheckDir(t, false, webResourcesPath)

	// 1. register db service instances
	beforeRegister := time.Now()
	registerService(t, &capi.AgentServiceRegistration{ID: "db1", Name: "db"}, port)
	api.WaitForEvent(t, cts, "db_task", beforeRegister, defaultWaitForEvent)
	testutils.CheckFile(t, true, dbResourcesPath, "db1.txt")
	beforeRegister = time.Now()
	registerService(t, &capi.AgentServiceRegistration{ID: "db2", Name: "db"}, port)
	api.WaitForEvent(t, cts, "db_task", beforeRegister, defaultWaitForEvent)
	testutils.CheckFile(t, true, dbResourcesPath, "db2.txt")
	files := testutils.CheckDir(t, true, dbResourcesPath)
	require.Equal(t, 2, len(files))
	testutils.CheckDir(t, false, webResourcesPath)

	// 2. register a web service instance
	beforeRegister = time.Now()
	registerService(t, &capi.AgentServiceRegistration{ID: "web1", Name: "web"}, port)
	api.WaitForEvent(t, cts, "web_task", beforeRegister, defaultWaitForEvent)
	testutils.CheckFile(t, true, webResourcesPath, "web1.txt")
	testutils.CheckFile(t, true, dbResourcesPath, "db1.txt")
	testutils.CheckFile(t, true, dbResourcesPath, "db2.txt")

	//3. deregister one db service instance
	beforeDeregister := time.Now()
	deregisterService(t, "db1", port)
	api.WaitForEvent(t, cts, "db_task", beforeDeregister, defaultWaitForEvent)
	testutils.CheckFile(t, false, dbResourcesPath, "db1.txt")
	testutils.CheckFile(t, true, dbResourcesPath, "db2.txt")
	testutils.CheckFile(t, true, webResourcesPath, "web1.txt")

}

// testServiceValuesCompatibility tests the compatibility of Consul's Health
// Service API response's service-related values. To test, update service-related
// field's values and confirm that terraform.tfvars is updated with the
// recent values.
//
// Tested service-related values: kind, port, address, meta, tags, status
// Does not test: modifying the ID and Name field. Modifying ID results in
// registering a new service instance (tested elsewhere). Modifying Name results
// in registering a new service (unrelated scenario for this particular test).
func testServiceValuesCompatibility(t *testing.T, tempDir string, port int) {
	config := baseConfig(tempDir, port) + nullTask()
	configPath := filepath.Join(tempDir, configFile)
	testutils.WriteFile(t, configPath, config)

	cts, stop := api.StartCTS(t, configPath)
	defer stop(t)
	err := cts.WaitForAPI(defaultWaitForAPI)
	require.NoError(t, err)

	// Test updating service-related values
	// 0. Confirm no services exist in terraform.tfvars
	// 1. Register db service instances with all service-related values.
	// 2. Modify service-related values (kind, port, address, meta, tags, status).
	//    Confirm that new values is captured in terraform.tfvars

	// 0. confirm empty service block
	workingDir := filepath.Join(tempDir, nullTaskName)
	content := testutils.CheckFile(t, true, workingDir, tftmpl.TFVarsFilename)
	require.Contains(t, content, "services = {\n}")

	// 1. register service instance
	meta := make(map[string]string)
	meta["meta_key1"] = "meta_value1"
	meta["meta_key2"] = "meta_value2"
	tags := []string{"tag1", "tag2"}
	serviceInstance := &capi.AgentServiceRegistration{
		Kind:    capi.ServiceKind("kind"),
		ID:      "db1",
		Name:    "db",
		Port:    1,
		Address: "address",
		Meta:    meta,
		Tags:    tags,
	}
	registerService(t, serviceInstance, port)

	// 2. modify kind
	serviceInstance.Kind = "kind_update"
	registerService(t, serviceInstance, port)
	content = testutils.CheckFile(t, true, workingDir, tftmpl.TFVarsFilename)
	assert.Contains(t, content, "kind_update")

	// 2. modify port
	serviceInstance.Port = 123456
	registerService(t, serviceInstance, port)
	content = testutils.CheckFile(t, true, workingDir, tftmpl.TFVarsFilename)
	assert.Contains(t, content, "123456")

	// 2. modify address
	serviceInstance.Address = "address_update"
	registerService(t, serviceInstance, port)
	content = testutils.CheckFile(t, true, workingDir, tftmpl.TFVarsFilename)
	assert.Contains(t, content, "address_update")

	// 2. modify meta
	meta["meta_key3"] = "meta_value3"        // create
	meta["meta_key1"] = "meta_value1_update" // update
	delete(meta, "meta_key2")                // delete
	registerService(t, serviceInstance, port)
	content = testutils.CheckFile(t, true, workingDir, tftmpl.TFVarsFilename)
	assert.Contains(t, content, "meta_value1_update")
	assert.NotContains(t, content, "meta_key2")
	assert.Contains(t, content, "meta_value3")

	// 2. modify tags
	tags[0] = "tag1_update" // update
	tags[1] = "tag3"        // create & delete (replace)
	registerService(t, serviceInstance, port)
	content = testutils.CheckFile(t, true, workingDir, tftmpl.TFVarsFilename)
	assert.Contains(t, content, "tag1_update")
	assert.NotContains(t, content, "tag2")
	assert.Contains(t, content, "tag3")

	// 2. change health status. when no health check, 'passing' by default.
	// add a 'critical' health check. Critical services do not render by default.
	serviceInstance.Check = &capi.AgentServiceCheck{
		CheckID:  "db1_check",
		HTTP:     "fake_url",
		Status:   "critical",
		Interval: "10s",
	}
	registerService(t, serviceInstance, port)
	content = testutils.CheckFile(t, true, workingDir, tftmpl.TFVarsFilename)
	assert.NotContains(t, content, "critical")
}

// testTagQueryCompatibility tests the compatibility of Consul's Health Service
// API tag querying (and API name querying implicitly).
//
// Not tested: Namespace querying (Enterprise), Datacenter querying (manually
// tested since it requires setting up at least 2 datacenters)
func testTagQueryCompatibility(t *testing.T, tempDir string, port int) {
	config := baseConfig(tempDir, port) + basicTask("redis_task", "redis",
		"db", `filter = "\"v1\" in Service.Tags"`)
	configPath := filepath.Join(tempDir, configFile)
	testutils.WriteFile(t, configPath, config)

	cts, stop := api.StartCTS(t, configPath)
	defer stop(t)
	err := cts.WaitForAPI(defaultWaitForAPI)
	require.NoError(t, err)

	// Test that filtering by tags
	// 0. Confirm no resources created yet
	// 2. Register redis service instance with tag v2. Confirm no resources created
	// 3. Register redis service instance with tag v1. Confirm resource created

	// 0. no resources created yet
	resourcesPath := filepath.Join(tempDir, "redis_task", resourcesDir)
	testutils.CheckDir(t, false, resourcesPath)

	// 2. register filtered-in tag
	registerService(t, &capi.AgentServiceRegistration{ID: "redis_v1",
		Name: "redis", Tags: []string{"v1"}}, port)
	testutils.CheckFile(t, true, resourcesPath, "redis_v1.txt")

	// 3. register filtered-out tag. This fails in v1.4.5 (as expected) since
	// tag filtering was added in v1.5.0
	registerService(t, &capi.AgentServiceRegistration{ID: "redis_v2",
		Name: "redis", Tags: []string{"v2"}}, port)
	testutils.CheckFile(t, false, resourcesPath, "redis_v2.txt")
}

// testNodeValuesCompatibility tests the compatibility of Consul's Health
// Service API response's node-related values. To test, update node-related
// fields' values and confirm that terraform.tfvars is updated with the
// recent values.
//
// Tested node-related values: Node name, node id, node address, tagged address,
// and node meta. Node datacenter not tested.
func testNodeValuesCompatibility(t *testing.T, tempDir string, port int) {
	config := baseConfig(tempDir, port) + nullTask()
	configPath := filepath.Join(tempDir, configFile)
	testutils.WriteFile(t, configPath, config)

	cts, stop := api.StartCTS(t, configPath)
	defer stop(t)
	err := cts.WaitForAPI(defaultWaitForAPI)
	require.NoError(t, err)

	// Test updating node-related values
	// 0. Confirm no services exist in terraform.tfvars
	// 1. Register service entity in catalog with all node-related values filled
	// 2. Modify node-related values (name, ID, Address, Datacenter,
	//    TaggedAddresses, Meta). Confirm that new values are in terraform.tfvars

	// 0. confirm empty service block
	workingDir := filepath.Join(tempDir, nullTaskName)
	content := testutils.CheckFile(t, true, workingDir, tftmpl.TFVarsFilename)
	require.Contains(t, content, "services = {\n}")

	// 1. register catalog entity
	meta := make(map[string]string)
	meta["meta_key1"] = "meta_value1"
	meta["meta_key2"] = "meta_value2"
	taggedAddr := make(map[string]string)
	taggedAddr["addr_key1"] = "addr_value1"
	taggedAddr["addr_key2"] = "addr_value2"
	entity := &capi.CatalogRegistration{
		ID:              "7b1f0351-2f7c-4088-8126-d3f7a636cbcd",
		Node:            "node_name",
		Address:         "node_address",
		TaggedAddresses: taggedAddr,
		NodeMeta:        meta,
		Service:         &capi.AgentService{ID: "api1", Service: "api"},
		Check: &capi.AgentCheck{
			Node:      "node_name",
			CheckID:   "1234",
			Status:    "passing",
			ServiceID: "api1",
			Definition: capi.HealthCheckDefinition{
				HTTP:     "http://www.consul.io",
				Method:   http.MethodGet,
				Interval: *capi.NewReadableDuration(1 * time.Second),
			},
		},
	}
	registerCatalog(t, entity, port)

	// 2. modify node name
	entity.Node = "node_name_update"
	entity.Check.Node = "node_name_update"
	registerCatalog(t, entity, port)
	content = testutils.CheckFile(t, true, workingDir, tftmpl.TFVarsFilename)
	assert.Contains(t, content, "node_name_update")

	// 2. modify node id. Registering a new node id fails in v1.4.5
	entity.ID = "8d5bf2e4-88f1-11eb-8dcd-0242ac130003"
	registerCatalog(t, entity, port)
	content = testutils.CheckFile(t, true, workingDir, tftmpl.TFVarsFilename)
	assert.Contains(t, content, "8d5bf2e4-88f1-11eb-8dcd-0242ac130003")

	// 2. modify address
	entity.Address = "node_address_update"
	registerCatalog(t, entity, port)
	content = testutils.CheckFile(t, true, workingDir, tftmpl.TFVarsFilename)
	assert.Contains(t, content, "node_address_update")

	// 2. modify tagged address
	taggedAddr["addr_key3"] = "addr_value3"        // create
	taggedAddr["addr_key1"] = "addr_value1_update" // update
	delete(taggedAddr, "addr_key2")                // delete
	registerCatalog(t, entity, port)
	content = testutils.CheckFile(t, true, workingDir, tftmpl.TFVarsFilename)
	assert.Contains(t, content, "addr_value1_update")
	assert.NotContains(t, content, "addr_value2")
	assert.Contains(t, content, "addr_value3")

	// 2. modify meta
	meta["meta_key3"] = "meta_value3"        // create
	meta["meta_key1"] = "meta_value1_update" // update
	delete(meta, "meta_key2")                // delete
	registerCatalog(t, entity, port)
	content = testutils.CheckFile(t, true, workingDir, tftmpl.TFVarsFilename)
	assert.Contains(t, content, "meta_value1_update")
	assert.NotContains(t, content, "meta_value2")
	assert.Contains(t, content, "meta_value3")
}

// downloadConsul downloads Consul into the current directory. Returns a function
// to delete the downloaded Consul binary.
func downloadConsul(t *testing.T, dst string, version string) string {
	opsys := runtime.GOOS
	arch := runtime.GOARCH

	filename := fmt.Sprintf("consul_%s_%s_%s.zip", version, opsys, arch)
	url := fmt.Sprintf("https://releases.hashicorp.com/consul/%s/%s", version, filename)

	client := getter.Client{
		Getters: map[string]getter.Getter{
			"https": &getter.HttpGetter{},
		},
		Mode: getter.ClientModeDir,
		Src:  url,
		Dst:  dst,
	}
	err := client.Get()
	require.NoError(t, err)

	return filepath.Join(dst, "consul")
}

// runConsul starts running a Consul binary that is in the current directory.
// Returns a function that stops running Consul. Does not log to standard out.
func runConsul(t *testing.T, execPath string, port int) func() {
	cmd := exec.Command(execPath, "agent", "-dev",
		fmt.Sprintf("-http-port=%d", port),
		// Randomize ports to run multiple consul servers on the same node.
		// These ports are not used for CTS compatibility testing
		fmt.Sprintf("-server-port=%d", testutils.FreePort(t)),
		fmt.Sprintf("-serf-lan-port=%d", testutils.FreePort(t)),
		fmt.Sprintf("-serf-wan-port=%d", testutils.FreePort(t)),
		fmt.Sprintf("-dns-port=%d", testutils.FreePort(t)),
		fmt.Sprintf("-grpc-port=%d", testutils.FreePort(t)),
	)
	// uncomment to see logs
	// cmd.Stdout = os.Stdout
	// cmd.Stderr = os.Stderr

	err := cmd.Start()
	require.NoError(t, err)

	// wait a little for Consul to get fully started up
	time.Sleep(3 * time.Second)

	return func() {
		cmd := exec.Command(execPath, "leave",
			fmt.Sprintf("-http-addr=localhost:%d", port))
		err := cmd.Run() // Run() waits for `consul leave` to finish
		require.NoError(t, err)
	}
}

// registerService registers a Consul service to running Consul binary + wait time
func registerService(t *testing.T, service *capi.AgentServiceRegistration, port int) {
	conf := capi.DefaultConfig()
	conf.Address = fmt.Sprintf("localhost:%d", port)
	client, err := capi.NewClient(conf)
	require.NoError(t, err)

	err = client.Agent().ServiceRegister(service)
	require.NoError(t, err)

	// wait a little for CTS to respond
	time.Sleep(8 * time.Second)
}

// deregisterService registers a Consul service to running Consul binary + wait time
func deregisterService(t *testing.T, serviceID string, port int) {
	conf := capi.DefaultConfig()
	conf.Address = fmt.Sprintf("localhost:%d", port)
	client, err := capi.NewClient(conf)
	require.NoError(t, err)

	err = client.Agent().ServiceDeregister(serviceID)
	require.NoError(t, err)

	// wait a little for CTS to respond
	time.Sleep(8 * time.Second)
}

// registerCatalog registers an entity to running Consul binary + wait time
func registerCatalog(t *testing.T, entity *capi.CatalogRegistration, port int) {
	conf := capi.DefaultConfig()
	conf.Address = fmt.Sprintf("localhost:%d", port)
	client, err := capi.NewClient(conf)
	require.NoError(t, err)

	_, err = client.Catalog().Register(entity, nil)
	require.NoError(t, err)

	// wait a little for CTS to respond
	time.Sleep(8 * time.Second)
}

func baseConfig(dir string, port int) string {
	return fmt.Sprintf(`log_level = "INFO"

# port 0 will automatically select next free port
port = 0
working_dir = "%s"

driver "terraform" {
	log = true
	path = "%s"
}

consul {
	address = "localhost:%d"
}
`, dir, dir, port)
}

// nullTask returns config for a task with null resource module
func nullTask() string {
	return fmt.Sprintf(`
task {
	name = "%s"
	description = "null task for api & db"
	condition "services" {
		names = ["api", "db"]
	}
	providers = ["null"]
	module = "../test_modules/null_resource"
}
`, nullTaskName)
}

// basicTask returns config for a task with basic task module
func basicTask(taskName, service1, service2 string, conditionOpts ...string) string {
	var opts string
	if len(conditionOpts) > 0 {
		opts = strings.Join(conditionOpts, "\n")
	}

	return fmt.Sprintf(`
task {
	name = "%s"
	description = "basic task"
	condition "services" {
		names = ["%s", "%s"]
		%s
	}
	providers = ["local"]
	module = "../test_modules/local_instances_file"
}
`, taskName, service1, service2, opts)
}
