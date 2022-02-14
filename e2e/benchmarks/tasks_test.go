//go:build e2e
// +build e2e

// BenchmarkTasks benchmarks CTS in inspect mode.
//
// Runs the ReadOnly controller directly and benchmarks Init and Run.
//
// ReadOnlyController.Init involves creating auto-generated Terraform files
// and the hcat template file for each task.
//
// ReadOnlyController.Run involves rendering the template file and executing
// Terraform init and Terraform plan serially across all tasks.
//
// $ go test ./e2e/benchmarks -bench=BenchmarkTasks_ -tags e2e
package benchmarks

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/controller"
	"github.com/hashicorp/consul-terraform-sync/testutils"
	"github.com/hashicorp/consul/sdk/testutil"
	"github.com/stretchr/testify/require"
)

func BenchmarkTasks_t01_s01(b *testing.B) {
	benchmarkTasks(b, 1, 1)
}

func BenchmarkTasks_t01_s50(b *testing.B) {
	benchmarkTasks(b, 1, 50)
}

func BenchmarkTasks_t10_s01(b *testing.B) {
	benchmarkTasks(b, 10, 1)
}

func BenchmarkTasks_t10_s50(b *testing.B) {
	benchmarkTasks(b, 10, 50)
}

func BenchmarkTasks_t50_s01(b *testing.B) {
	benchmarkTasks(b, 50, 1)
}

func BenchmarkTasks_t50_s50(b *testing.B) {
	benchmarkTasks(b, 50, 50)
}

func benchmarkTasks(b *testing.B, numTasks int, numServices int) {
	srv := testutils.NewTestConsulServer(b, testutils.TestConsulServerConfig{
		HTTPSRelPath: "../../testutils",
	})
	defer srv.Stop()

	tempDir := b.Name()
	cleanup := testutils.MakeTempDir(b, tempDir)
	defer cleanup()

	ctx := context.Background()
	conf := generateConf(b, benchmarkConfig{
		consul:      srv,
		tempDir:     tempDir,
		numTasks:    numTasks,
		numServices: numServices,
	})

	// Override Terraform output logging for benchmark readability
	// when running the ReadOnlyController
	controller.MuteReadOnlyController = true

	ctrl, err := controller.NewReadOnly(conf)
	require.NoError(b, err)

	err = ctrl.Init(ctx)
	require.NoError(b, err)

	// Make an initial dependency change as setup, reset timer
	random := rand.New(rand.NewSource(time.Now().UnixNano()))
	service := testutil.TestService{
		ID:      fmt.Sprintf("service-000-%d", random.Intn(99999)),
		Name:    "service-000",
		Address: "5.6.7.8",
		Port:    8080,
	}
	testutils.RegisterConsulServiceHealth(b, srv, service, 0, testutil.HealthPassing)
	b.ResetTimer()

	// Run task execution
	err = ctrl.Run(ctx)
	require.NoError(b, err)
}

type benchmarkConfig struct {
	consul      *testutil.TestServer
	driver      *config.DriverConfig
	tempDir     string
	numTasks    int
	numServices int
	timeout     time.Duration
}

func benchmarkTaskName(tb testing.TB, i int) string {
	return fmt.Sprintf("%s_task_%03d", strings.ReplaceAll(tb.Name(), "/", "-"), i)
}

func generateConf(tb testing.TB, bConf benchmarkConfig) *config.Config {
	serviceNames := make([]string, bConf.numServices)
	for i := 0; i < bConf.numServices; i++ {
		serviceNames[i] = fmt.Sprintf("service-%03d", i)
	}

	taskConfigs := make(config.TaskConfigs, bConf.numTasks)
	for i := 0; i < bConf.numTasks; i++ {
		taskConfigs[i] = &config.TaskConfig{
			Name:   config.String(benchmarkTaskName(tb, i)),
			Module: config.String("../test_modules/local_file"),
			Condition: &config.ServicesConditionConfig{
				ServicesMonitorConfig: config.ServicesMonitorConfig{
					Names: serviceNames,
				},
			},
		}
	}

	conf := config.DefaultConfig()
	conf.WorkingDir = &bConf.tempDir
	conf.BufferPeriod.Enabled = config.Bool(false)
	conf.Tasks = &taskConfigs
	conf.Consul.Address = config.String(bConf.consul.HTTPSAddr)
	conf.Consul.TLS = &config.TLSConfig{
		Enabled: config.Bool(true),
		Verify:  config.Bool(false),

		// This is needed for Terraform Consul backend when CTS is
		// connecting over HTTP/2 using TLS.
		CACert: config.String(bConf.consul.Config.CertFile),
	}
	if bConf.driver != nil {
		conf.Driver = bConf.driver
	}

	conf.Finalize()
	if conf.Driver.Terraform != nil {
		conf.Driver.Terraform.Path = config.String("../../../")
	}
	return conf
}
