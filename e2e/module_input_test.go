//go:build e2e
// +build e2e

package e2e

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/hashicorp/consul-terraform-sync/api"
	"github.com/hashicorp/consul-terraform-sync/testutils"
	"github.com/hashicorp/consul/sdk/testutil"
)

const (
	// config snippets for test cases: module and module_input fields
	servicesInput = `
	module = "./test_modules/local_instances_file"
	module_input "services"{
		// insert names or regexp field below
		%s
		filter = "Service.Tags not contains \"filtered\""
		cts_user_defined_meta {
			meta_key = "meta_value"
		}
	}`
	consulKVInput = `
	module = "./test_modules/consul_kv_file"
	module_input "consul-kv" {
		path = "key"
		recurse = %t
	}`
)

// TestModuleInput_Basic_CatalogServicesCondition exercises:
// 1. basics of module input
// 2. simultaneously module input with catalog-services condition
//
// Module input basic testing exercises:
// - each type of module_input
// - module_input configuration field details
// - multiple module_inputs
func TestModuleInput_Basic_CatalogServicesCondition(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name        string
		tempDir     string
		inputConfig string
		validation  func(string, string)
	}{
		{
			name:        "services regexp",
			tempDir:     "mi_services_regexp",
			inputConfig: fmt.Sprintf(servicesInput, `regexp = "api"`),
			validation: func(workingDir, resourcesPath string) {
				validateServices(t, true, []string{"api"}, resourcesPath)
				validateServices(t, false, []string{"api-filtered"}, resourcesPath)
				validateVariable(t, true, workingDir, "services", "meta_value")
			},
		},
		{
			name:        "services names",
			tempDir:     "mi_services_names",
			inputConfig: fmt.Sprintf(servicesInput, `names = ["api"]`),
			validation: func(workingDir, resourcesPath string) {
				validateServices(t, true, []string{"api"}, resourcesPath)
				validateServices(t, false, []string{"api-filtered"}, resourcesPath)
				validateVariable(t, true, workingDir, "services", "meta_value")
			},
		},
		{
			name:        "consul-kv recurse true",
			tempDir:     "mi_consul_kv_recurse_true",
			inputConfig: fmt.Sprintf(consulKVInput, true),
			validation: func(workingDir, resourcesPath string) {
				validateModuleFile(t, true, true, resourcesPath, "key", "value")
				validateModuleFile(t, true, true, resourcesPath, "key/recurse",
					"value-recurse")
			},
		},
		{
			name:        "consul-kv recurse false",
			tempDir:     "mi_consul_kv_recurse_false",
			inputConfig: fmt.Sprintf(consulKVInput, false),
			validation: func(workingDir, resourcesPath string) {
				validateModuleFile(t, true, true, resourcesPath, "key", "value")
				validateModuleFile(t, true, false, resourcesPath, "key/recurse",
					"value-recurse")
			},
		},
		{
			name:    "multiple",
			tempDir: "mi_multiple",
			inputConfig: fmt.Sprintf(servicesInput+consulKVInput,
				`names = ["api"]`, false),
			validation: func(workingDir, resourcesPath string) {
				// services validation
				validateServices(t, true, []string{"api"}, resourcesPath)
				validateServices(t, false, []string{"api-filtered"}, resourcesPath)
				validateVariable(t, true, workingDir, "services", "meta_value")
				// consul-kv validation
				validateModuleFile(t, true, true, resourcesPath, "key", "value")
				validateModuleFile(t, true, false, resourcesPath, "key/recurse",
					"value-recurse")
			},
		},
	}

	for _, tc := range testcases {
		tc := tc // rebind tc into this lexical scope for parallel use
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel() // Run table tests in parallel as they can take a lot of time

			// set up Consul
			srv := testutils.NewTestConsulServer(t, testutils.TestConsulServerConfig{
				HTTPSRelPath: "../testutils",
			})
			defer srv.Stop()

			// set up task config with module & module_input
			taskName := "module_input_task"
			config := fmt.Sprintf(`task {
				name = "%s"
				condition "catalog-services" {
					regexp = ["web"]
					use_as_module_input = false
				}
				%s
			}`, taskName, tc.inputConfig)

			tempDir := fmt.Sprintf("%s%s", tempDirPrefix, tc.tempDir)
			cts := ctsSetup(t, srv, tempDir, config)

			// Test module_inputs basic behavior
			// 0. Confirm baseline: no resources have been created
			// 1. Confirm module_input don't trigger tasks: register a change
			//    only affecting module inputs. Confirm no resources are created.
			// 2. Confirm module_input templating & resources: register a change
			//    to trigger conditions. Confirm changes for module_input
			//    changes in step 1

			// 0. Confirm no resources have been created
			workingDir := fmt.Sprintf("%s/%s", tempDir, taskName)
			resourcesPath := filepath.Join(workingDir, resourcesDir)
			testutils.CheckDir(t, false, resourcesPath)

			// 1. Register stuff that the module_input monitors (for all test
			// cases). Confirm no resources created.
			service := testutil.TestService{ID: "api", Name: "api"}
			testutils.RegisterConsulService(t, srv, service, defaultWaitForRegistration)
			service = testutil.TestService{ID: "api-filtered", Name: "api", Tags: []string{"filtered"}}
			testutils.RegisterConsulService(t, srv, service, defaultWaitForRegistration)

			srv.SetKVString(t, "key", "value")
			srv.SetKVString(t, "key/recurse", "value-recurse")

			time.Sleep(defaultWaitForNoEvent)
			testutils.CheckDir(t, false, resourcesPath)

			// 2. Register "web" service to trigger condition. Confirm that
			// module_input's monitored stuff from step 1 happened
			now := time.Now()
			service = testutil.TestService{ID: "web", Name: "web"}
			testutils.RegisterConsulService(t, srv, service, defaultWaitForRegistration)
			api.WaitForEvent(t, cts, taskName, now, defaultWaitForEvent)

			tc.validation(workingDir, resourcesPath)
		})
	}
}

// TestModuleInput_ServicesCondition generally exercises different variations
// of services condition block with different types of module_input blocks
func TestModuleInput_ServicesCondition(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name        string
		tempDir     string
		condField   string
		inputConfig string
		validate    func(string)
	}{
		{
			name:        "regex cond consul-kv module_input",
			tempDir:     "services_regex_mi_consul_kv",
			condField:   `regexp = "api"`,
			inputConfig: fmt.Sprintf(consulKVInput, true),
			validate: func(resourcesPath string) {
				validateModuleFile(t, true, true, resourcesPath, "key", "value")
				validateModuleFile(t, true, true, resourcesPath, "key/recurse",
					"value-recurse")
			},
		},
		{
			name:        "names cond consul-kv module_input",
			tempDir:     "services_names_mi_consul_k",
			condField:   `names = ["api"]`,
			inputConfig: fmt.Sprintf(consulKVInput, true),
			validate: func(resourcesPath string) {
				validateModuleFile(t, true, true, resourcesPath, "key", "value")
				validateModuleFile(t, true, true, resourcesPath, "key/recurse",
					"value-recurse")
			},
		},
	}

	for _, tc := range testcases {
		tc := tc // rebind tc into this lexical scope for parallel use
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel() // Run table tests in parallel as they can take a lot of time

			// set up Consul
			srv := testutils.NewTestConsulServer(t, testutils.TestConsulServerConfig{
				HTTPSRelPath: "../testutils",
			})
			defer srv.Stop()

			// set up task config with condition field, module, & module_input
			taskName := "module_input_task"
			config := fmt.Sprintf(`task {
				name = "%s"
				condition "services" {
					%s
					use_as_module_input = true
				}
				%s
			}`, taskName, tc.condField, tc.inputConfig)

			tempDir := fmt.Sprintf("%s%s", tempDirPrefix, tc.tempDir)
			cts := ctsSetup(t, srv, tempDir, config)

			// Test module_input behavior with Services Cond
			// 0. Confirm baseline: no resources have been created
			// 1. Confirm module_input don't trigger tasks: register a change
			//    only affecting module inputs. Confirm no resources are created.
			// 2. Confirm module_input templating & resources: register a change
			//    to trigger conditions. Confirm changes for module_input
			//    changes in step 1

			// 0. Confirm no resources have been created
			workingDir := fmt.Sprintf("%s/%s", tempDir, taskName)
			resourcesPath := filepath.Join(workingDir, resourcesDir)
			testutils.CheckDir(t, false, resourcesPath)

			// 1. Register stuff that the module_input monitors (for all test
			// cases). Confirm no resources created.
			srv.SetKVString(t, "key", "value")
			srv.SetKVString(t, "key/recurse", "value-recurse")

			time.Sleep(defaultWaitForNoEvent)
			testutils.CheckDir(t, false, resourcesPath)

			// 2. Register "api" service to trigger condition. Confirm that
			// module_input's monitored stuff from step 1 happened
			now := time.Now()
			service := testutil.TestService{ID: "api", Name: "api"}
			testutils.RegisterConsulService(t, srv, service, defaultWaitForRegistration)
			api.WaitForEvent(t, cts, taskName, now, defaultWaitForEvent)

			tc.validate(resourcesPath)
		})
	}
}

// TestModuleInput_ConsulKVCondition generally exercises the consul-kv condition
// block with different types of module_input blocks
func TestModuleInput_ConsulKVCondition(t *testing.T) {
	t.Parallel()

	testcases := []struct {
		name        string
		tempDir     string
		inputConfig string
		validate    func(string, string)
	}{
		{
			name:        "services",
			tempDir:     "consul_kv_mi_services",
			inputConfig: fmt.Sprintf(servicesInput, `regexp = "api"`),
			validate: func(workingDir, resourcesPath string) {
				validateServices(t, true, []string{"api"}, resourcesPath)
			},
		},
	}

	for _, tc := range testcases {
		tc := tc // rebind tc into this lexical scope for parallel use
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel() // Run table tests in parallel as they can take a lot of time

			// set up Consul
			srv := testutils.NewTestConsulServer(t, testutils.TestConsulServerConfig{
				HTTPSRelPath: "../testutils",
			})
			defer srv.Stop()

			// set up task config with module & module_input
			taskName := "module_input_task"
			config := fmt.Sprintf(`task {
				name = "%s"
				condition "consul-kv" {
					path = "key"
					use_as_module_input = false
				}
				%s
			}`, taskName, tc.inputConfig)

			tempDir := fmt.Sprintf("%s%s", tempDirPrefix, tc.tempDir)
			cts := ctsSetup(t, srv, tempDir, config)

			// Test module_input behavior with Consul KV Cond
			// 0. Confirm baseline: no resources have been created
			// 1. Confirm module_input don't trigger tasks: register a change
			//    only affecting module inputs. Confirm no resources are created.
			// 2. Confirm module_input templating & resources: register a change
			//    to trigger conditions. Confirm changes for module_input
			//    changes in step 1

			// 0. Confirm no resources have been created
			workingDir := fmt.Sprintf("%s/%s", tempDir, taskName)
			resourcesPath := filepath.Join(workingDir, resourcesDir)
			testutils.CheckDir(t, false, resourcesPath)

			// 1. Register stuff that the module_input monitors (for all test
			// cases). Confirm no resources created.
			service := testutil.TestService{ID: "api", Name: "api"}
			testutils.RegisterConsulService(t, srv, service, defaultWaitForRegistration)

			time.Sleep(defaultWaitForNoEvent)
			testutils.CheckDir(t, false, resourcesPath)

			// 2. Register kv pair to trigger condition. Confirm that
			// module_input's monitored stuff from step 1 happened
			now := time.Now()
			srv.SetKVString(t, "key", "value")
			api.WaitForEvent(t, cts, taskName, now, defaultWaitForEvent)

			tc.validate(workingDir, resourcesPath)
		})
	}
}
