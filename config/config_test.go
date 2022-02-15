package config

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	jsonConfig = []byte(`{
	"log_level": "ERR"
}`)

	hclConfig = []byte(`
	log_level = "ERR"
`)

	testConfig = Config{
		LogLevel: String("ERR"),
	}

	longConfig = Config{
		LogLevel:   String("ERR"),
		Port:       Int(8502),
		WorkingDir: String("working"),
		Syslog: &SyslogConfig{
			Enabled: Bool(true),
			Name:    String("syslog"),
		},
		Consul: &ConsulConfig{
			Address: String("consul-example.com"),
			Auth: &AuthConfig{
				Enabled:  Bool(true),
				Username: String("username"),
				Password: String("password"),
			},
			KVPath: String("kv_path"),
			TLS: &TLSConfig{
				CACert:     String("ca_cert"),
				CAPath:     String("ca_path"),
				Enabled:    Bool(true),
				Key:        String("key"),
				ServerName: String("server_name"),
				Verify:     Bool(false),
			},
			Token: String("token"),
			Transport: &TransportConfig{
				DialKeepAlive:       TimeDuration(5 * time.Second),
				DialTimeout:         TimeDuration(10 * time.Second),
				DisableKeepAlives:   Bool(false),
				IdleConnTimeout:     TimeDuration(1 * time.Minute),
				MaxIdleConnsPerHost: Int(100),
				TLSHandshakeTimeout: TimeDuration(10 * time.Second),
			},
		},
		TLS: &CTSTLSConfig{
			Enabled:        Bool(true),
			Cert:           String("../testutils/certs/consul_cert.pem"),
			Key:            String("../testutils/certs/consul_key.pem"),
			VerifyIncoming: Bool(true),
			CACert:         String("../testutils/certs/consul_cert.pem"),
		},
		Driver: &DriverConfig{
			Terraform: &TerraformConfig{
				Log:  Bool(true),
				Path: String("path"),
				Backend: map[string]interface{}{
					"consul": map[string]interface{}{
						"address": "consul-example.com",
						"path":    "kv-path/terraform",
						"gzip":    true,
					},
				},
				RequiredProviders: map[string]interface{}{
					"pName1": "v0.0.0",
					"pName2": map[string]interface{}{
						"version": "v0.0.1",
						"source":  "namespace/pName2",
					},
				},
			},
		},
		DeprecatedServices: &ServiceConfigs{
			{
				Name:        String("serviceA"),
				Description: String("descriptionA"),
			}, {
				Name:        String("serviceB"),
				Namespace:   String("teamB"),
				Description: String("descriptionB"),
			},
		},
		Tasks: &TaskConfigs{
			{
				Description:        String("automate services for X to do Y"),
				Name:               String("task"),
				DeprecatedServices: []string{"serviceA", "serviceB", "serviceC"},
				Providers:          []string{"X"},
				Module:             String("Y"),
				Condition: &CatalogServicesConditionConfig{
					CatalogServicesMonitorConfig{
						Regexp:           String(".*"),
						UseAsModuleInput: Bool(true),
						Datacenter:       String("dc2"),
						Namespace:        String("ns2"),
						NodeMeta: map[string]string{
							"key1": "value1",
							"key2": "value2",
						},
					},
				},
				ModuleInputs: &ModuleInputConfigs{
					&ConsulKVModuleInputConfig{
						ConsulKVMonitorConfig{
							Path:       String("key-path"),
							Recurse:    Bool(true),
							Datacenter: String("dc2"),
							Namespace:  String("ns2"),
						},
					},
				},
			},
		},
		TerraformProviders: &TerraformProviderConfigs{{
			"X": map[string]interface{}{},
		}},
		BufferPeriod: &BufferPeriodConfig{
			Min: TimeDuration(20 * time.Second),
			Max: TimeDuration(60 * time.Second),
		},
	}
)

func TestDecodeConfig(t *testing.T) {
	testCases := []struct {
		name     string
		file     string
		content  []byte
		expected *Config
	}{
		{
			"hcl",
			"config.hcl",
			hclConfig,
			&testConfig,
		}, {
			"json",
			"config.json",
			jsonConfig,
			&testConfig,
		}, {
			"unsupported format",
			"config.txt",
			hclConfig,
			nil,
		}, {
			"hcl invalid",
			"config.hcl",
			[]byte(`log_level: "ERR"`),
			nil,
		}, {
			"hcl unexpected key",
			"config.hcl",
			[]byte(`key = "does_not_exist"`),
			nil,
		}, {
			"json invalid",
			"config.json",
			[]byte(`{"log_level" = "ERR"}`),
			nil,
		}, {
			"json unexpected key",
			"config.json",
			[]byte(`{"key": "does_not_exist"}`),
			nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			c, err := decodeConfig(tc.content, tc.file)
			if tc.expected == nil {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			require.NotNil(t, c)
			assert.Equal(t, *tc.expected, *c)
		})
	}

	t.Run("invalid provider block", func(t *testing.T) {
		content := []byte(`provider "local" {}`)
		c, err := decodeConfig(content, "config.hcl")
		assert.Nil(t, c)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "terraform_provider")
	})
}

func TestFromPath(t *testing.T) {
	testCases := []struct {
		name     string
		path     string
		expected *Config
	}{
		{
			"load file",
			"testdata/simple.hcl",
			&Config{
				LogLevel: String("ERR"),
			},
		}, {
			"load dir merge",
			"testdata/simple",
			&Config{
				LogLevel: String("ERR"),
				Port:     Int(8503),
				BufferPeriod: &BufferPeriodConfig{
					Enabled: Bool(true),
					Min:     TimeDuration(time.Duration(10 * time.Second)),
					Max:     TimeDuration(time.Duration(30 * time.Second)),
				},
			},
		}, {
			"load dir merges tasks and provider",
			"testdata/merge",
			&Config{
				TerraformProviders: &TerraformProviderConfigs{
					&TerraformProviderConfig{
						"tf_providerA": map[string]interface{}{},
					},
					&TerraformProviderConfig{
						"tf_providerB": map[string]interface{}{},
					},
					&TerraformProviderConfig{
						"tf_providerC": map[string]interface{}{},
					},
				},
				Tasks: &TaskConfigs{
					{
						Name: String("taskA"),
						Condition: &ServicesConditionConfig{
							ServicesMonitorConfig: ServicesMonitorConfig{
								Names: []string{"serviceA", "serviceB"},
							},
						},
					}, {
						Name: String("taskB"),
						Condition: &ServicesConditionConfig{
							ServicesMonitorConfig: ServicesMonitorConfig{
								Names: []string{"serviceC", "serviceD"},
							},
						},
					},
				},
			},
		}, {
			"load dir override sorted by filename",
			"testdata/override",
			&Config{
				LogLevel: String("DEBUG"),
				Port:     Int(8505),
			},
		}, {
			"file DNE",
			"testdata/dne.hcl",
			nil,
		}, {
			"dir DNE",
			"testdata/dne",
			nil,
		}, {
			"load long HCL file",
			"testdata/long.hcl",
			&longConfig,
		}, {
			"load long JSON file",
			"testdata/long.json",
			&longConfig,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			c, err := fromPath(tc.path)
			if tc.expected == nil {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, c)
			assert.Equal(t, *tc.expected, *c)
		})
	}
}

func TestConfig_Finalize(t *testing.T) {
	// Finalize tests top level config calls nested finalize
	// Backfill expected values
	expected := longConfig.Copy()
	expected.ClientType = String("")
	expected.Port = Int(8502)
	expected.WorkingDir = String("working")
	expected.Syslog.Facility = String("LOCAL0")
	expected.BufferPeriod.Enabled = Bool(true)
	expected.Consul.KVNamespace = String("")
	expected.Consul.TLS.Cert = String("")
	expected.Consul.Transport.MaxIdleConns = Int(0)
	expected.Vault = DefaultVaultConfig()
	expected.Vault.Finalize()
	expected.TLS.Cert = String("../testutils/certs/consul_cert.pem")
	expected.TLS.Key = String("../testutils/certs/consul_key.pem")
	expected.TLS.VerifyIncoming = Bool(true)
	expected.TLS.CACert = String("../testutils/certs/consul_cert.pem")
	expected.TLS.Finalize()
	expected.Driver.consul = expected.Consul
	expected.Driver.Terraform.Version = String("")
	expected.Driver.Terraform.PersistLog = Bool(false)
	backend := expected.Driver.Terraform.Backend["consul"].(map[string]interface{})
	backend["scheme"] = "https"
	backend["ca_file"] = "ca_cert"
	backend["key_file"] = "key"
	(*expected.Tasks)[0].Enabled = Bool(true)
	(*expected.Tasks)[0].TFVersion = String("")
	(*expected.Tasks)[0].VarFiles = []string{}
	(*expected.Tasks)[0].Version = String("")
	(*expected.Tasks)[0].BufferPeriod = &BufferPeriodConfig{}
	(*expected.Tasks)[0].BufferPeriod.Enabled = Bool(true)
	(*expected.Tasks)[0].BufferPeriod.Min = TimeDuration(20 * time.Second)
	(*expected.Tasks)[0].BufferPeriod.Max = TimeDuration(60 * time.Second)
	(*expected.Tasks)[0].Variables = map[string]string{}
	(*expected.Tasks)[0].WorkingDir = String("working/task")
	(*expected.DeprecatedServices)[0].ID = String("serviceA")
	(*expected.DeprecatedServices)[0].Namespace = String("")
	(*expected.DeprecatedServices)[0].Datacenter = String("")
	(*expected.DeprecatedServices)[0].Filter = String("")
	(*expected.DeprecatedServices)[0].CTSUserDefinedMeta = map[string]string{}
	(*expected.DeprecatedServices)[1].ID = String("serviceB")
	(*expected.DeprecatedServices)[1].Datacenter = String("")
	(*expected.DeprecatedServices)[1].Filter = String("")
	(*expected.DeprecatedServices)[1].CTSUserDefinedMeta = map[string]string{}

	c := longConfig.Copy()
	c.Finalize()
	assert.Equal(t, expected, c)
}

func TestConfig_Validate(t *testing.T) {
	valid := longConfig.Copy()

	// 2 tasks using same provider w/ auto_commit enabled (should err)
	autoCommit := valid.Copy()
	autoCommit.TerraformProviders = &TerraformProviderConfigs{
		&TerraformProviderConfig{"X": map[string]interface{}{"auto_commit": true}}}
	ts := *autoCommit.Tasks
	t2 := ts[0].Copy()
	*t2.Name = "task2"
	ts = append(ts, t2)
	autoCommit.Tasks = &ts

	// task configured with no providers configured (default provider)
	noProvider := *valid.Copy()
	noProvider.TerraformProviders = &TerraformProviderConfigs{}

	// valid case with multiple tasks w/ different providers
	validMultiTask := longConfig.Copy()
	*validMultiTask.Tasks = append(*validMultiTask.Tasks, &TaskConfig{
		Description:        String("test task1"),
		Name:               String("task1"),
		DeprecatedServices: []string{"serviceD"},
		Providers:          []string{"Y"},
		Module:             String("Z"),
		Condition:          EmptyConditionConfig(),
		ModuleInputs:       DefaultModuleInputConfigs(),
	})
	*validMultiTask.TerraformProviders = append(*validMultiTask.TerraformProviders,
		&TerraformProviderConfig{"Y": map[string]interface{}{}})

	cases := []struct {
		name    string
		i       *Config
		isValid bool
	}{
		{
			"nil",
			nil,
			false,
		}, {
			"empty",
			&Config{},
			false,
		}, {
			"valid long",
			valid.Copy(),
			true,
		}, {
			"multi-task valid",
			validMultiTask.Copy(),
			true,
		}, {
			"empty provider",
			noProvider.Copy(),
			true,
		}, {
			"autocommitting provider reuse error",
			autoCommit.Copy(),
			false,
		},
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, tc.name), func(t *testing.T) {
			err := tc.i.Validate()
			if tc.isValid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestConfig_validateDynamicConfig(t *testing.T) {
	testCases := []struct {
		name    string
		i       Config
		isValid bool
	}{
		{
			"no dynamic configs",
			Config{},
			true,
		}, {
			"provider with no dynamic configs",
			Config{
				TerraformProviders: &TerraformProviderConfigs{
					&TerraformProviderConfig{
						"foo": map[string]interface{}{
							"arg": "value",
						},
					},
				},
			},
			true,
		}, {
			"provider with dynamic configs",
			Config{
				TerraformProviders: &TerraformProviderConfigs{
					&TerraformProviderConfig{
						"foo": map[string]interface{}{
							"arg":     "value",
							"dynamic": "{{ key \"my/key\" }}",
						},
					},
				},
			},
			true,
		}, {
			"provider with dynamic configs with vault",
			Config{
				TerraformProviders: &TerraformProviderConfigs{
					&TerraformProviderConfig{
						"foo": map[string]interface{}{
							"arg":           "value",
							"dynamic_vault": "{{ with secret \"my/secret\" }}",
						},
					},
				},
				Vault: &VaultConfig{
					Address: String("vault.example.com"),
				},
			},
			true,
		}, {
			"provider with dynamic configs missing vault",
			Config{
				TerraformProviders: &TerraformProviderConfigs{
					&TerraformProviderConfig{
						"foo": map[string]interface{}{
							"arg":           "value",
							"dynamic_vault": "{{ with secret \"my/secret\" }}",
						},
					},
				},
			},
			false,
		}, {
			"dynamic configs unsupported outside of providers",
			Config{
				Tasks: &TaskConfigs{{
					Name: String("{{ env \"NOT_SUPPORTED\" }}"),
				}},
			},
			false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.i.Finalize()
			err := tc.i.validateDynamicConfigs()
			if tc.isValid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestConfig_BufferPeriod(t *testing.T) {
	// Tests that global-level and task-level buffer period config are
	// resolved as expected

	cases := []struct {
		name     string
		confBp   *BufferPeriodConfig
		taskBp   *BufferPeriodConfig
		expected *BufferPeriodConfig
	}{
		{
			"only global-level configured",
			&BufferPeriodConfig{
				Enabled: Bool(true),
				Min:     TimeDuration(1 * time.Second),
				Max:     TimeDuration(3 * time.Second),
			},
			nil,
			&BufferPeriodConfig{
				Enabled: Bool(true),
				Min:     TimeDuration(1 * time.Second),
				Max:     TimeDuration(3 * time.Second),
			},
		},
		{
			"only task-level configured",
			nil,
			&BufferPeriodConfig{
				Enabled: Bool(true),
				Min:     TimeDuration(1 * time.Second),
				Max:     TimeDuration(3 * time.Second),
			},
			&BufferPeriodConfig{
				Enabled: Bool(true),
				Min:     TimeDuration(1 * time.Second),
				Max:     TimeDuration(3 * time.Second),
			},
		},
		{
			"both configured",
			&BufferPeriodConfig{
				Enabled: Bool(true),
				Min:     TimeDuration(1 * time.Second),
				Max:     TimeDuration(3 * time.Second),
			},
			&BufferPeriodConfig{
				Enabled: Bool(true),
				Min:     TimeDuration(5 * time.Second),
				Max:     TimeDuration(7 * time.Second),
			},
			&BufferPeriodConfig{
				Enabled: Bool(true),
				Min:     TimeDuration(5 * time.Second),
				Max:     TimeDuration(7 * time.Second),
			},
		},
		{
			"neither configured",
			nil,
			nil,
			DefaultBufferPeriodConfig(),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// set-up config with global-level and task-level buf period
			config := &Config{
				BufferPeriod: tc.confBp,
				Tasks: &TaskConfigs{
					{
						Name: String("test_task"),
						Condition: &ServicesConditionConfig{
							ServicesMonitorConfig: ServicesMonitorConfig{
								Names: []string{"api"},
							},
						},
						Module:       String("/path"),
						BufferPeriod: tc.taskBp,
					},
				},
			}

			// replicate config processing in cts cli
			config.Finalize()
			err := config.Validate()
			require.NoError(t, err)

			// confirm task-level buf period as expected
			task := (*config.Tasks)[0]
			assert.Equal(t, tc.expected, task.BufferPeriod)
		})
	}
}
