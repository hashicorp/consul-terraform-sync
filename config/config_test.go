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
		LogLevel: String("ERR"),
		Port:     Int(8502),
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
		Driver: &DriverConfig{
			Terraform: &TerraformConfig{
				Log:        Bool(true),
				Path:       String("path"),
				WorkingDir: String("working"),
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
		Services: &ServiceConfigs{
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
				Description: String("automate services for X to do Y"),
				Name:        String("task"),
				Services:    []string{"serviceA", "serviceB", "serviceC"},
				Providers:   []string{"X"},
				Source:      String("Y"),
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
		format   string
		content  []byte
		expected *Config
	}{
		{
			"hcl",
			"hcl",
			hclConfig,
			&testConfig,
		}, {
			"json",
			"json",
			jsonConfig,
			&testConfig,
		}, {
			"unsupported format",
			"txt",
			hclConfig,
			nil,
		}, {
			"hcl invalid",
			"hcl",
			[]byte(`log_level: "ERR"`),
			nil,
		}, {
			"hcl unexpected key",
			"hcl",
			[]byte(`key = "does_not_exist"`),
			nil,
		}, {
			"json invalid",
			"json",
			[]byte(`{"log_level" = "ERR"}`),
			nil,
		}, {
			"json unexpected key",
			"json",
			[]byte(`{"key": "does_not_exist"}`),
			nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			c, err := decodeConfig(tc.content, tc.format)
			if tc.expected == nil {
				assert.Error(t, err)
				return
			}

			require.NotNil(t, c)
			assert.Equal(t, *tc.expected, *c)
		})
	}
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
			"load dir merges tasks and services",
			"testdata/merge",
			&Config{
				Services: &ServiceConfigs{
					{
						Name:        String("serviceA"),
						Description: String("descriptionA"),
					}, {
						Name:        String("serviceB"),
						Namespace:   String("teamB"),
						Description: String("descriptionB"),
					}, {
						Name:        String("serviceC"),
						Description: String("descriptionC"),
					},
				},
				Tasks: &TaskConfigs{
					{
						Name:     String("taskA"),
						Services: []string{"serviceA", "serviceB"},
					}, {
						Name:     String("taskB"),
						Services: []string{"serviceC", "serviceD"},
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
	expected.Syslog.Facility = String("LOCAL0")
	expected.BufferPeriod.Enabled = Bool(true)
	expected.Consul.KVNamespace = String("")
	expected.Consul.TLS.Cert = String("")
	expected.Consul.Transport.MaxIdleConns = Int(0)
	expected.Vault = DefaultVaultConfig()
	expected.Vault.Finalize()
	expected.Driver.consul = expected.Consul
	expected.Driver.Terraform.Version = String("")
	expected.Driver.Terraform.PersistLog = Bool(false)
	backend := expected.Driver.Terraform.Backend["consul"].(map[string]interface{})
	backend["scheme"] = "https"
	backend["ca_file"] = "ca_cert"
	backend["key_file"] = "key"
	(*expected.Tasks)[0].VarFiles = []string{}
	(*expected.Tasks)[0].Version = String("")
	(*expected.Tasks)[0].BufferPeriod = DefaultTaskBufferPeriodConfig()
	(*expected.Services)[0].ID = String("serviceA")
	(*expected.Services)[0].Namespace = String("")
	(*expected.Services)[0].Datacenter = String("")
	(*expected.Services)[0].Tag = String("")
	(*expected.Services)[0].CTSUserDefinedMeta = map[string]string{}
	(*expected.Services)[1].ID = String("serviceB")
	(*expected.Services)[1].Datacenter = String("")
	(*expected.Services)[1].Tag = String("")
	(*expected.Services)[1].CTSUserDefinedMeta = map[string]string{}

	c := longConfig.Copy()
	c.Finalize()
	assert.Equal(t, expected, c)
}

func TestConfig_Validate(t *testing.T) {
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
			longConfig.Copy(),
			true,
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
