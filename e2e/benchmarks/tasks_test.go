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
	"testing"

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
	conf := generateConf(benchmarkConfig{
		consul:      srv,
		tempDir:     tempDir,
		numTasks:    numTasks,
		numServices: numServices,
	})

	b.Run("ReadOnlyCtrl", func(b *testing.B) {
		ctrl, err := controller.NewReadOnly(conf)
		require.NoError(b, err)

		b.Run("task setup", func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				_, err = ctrl.Init(ctx)
				require.NoError(b, err)
			}
		})

		b.Run("task execution", func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				err = ctrl.Run(ctx)
				require.NoError(b, err)
			}
		})
	})
}

type benchmarkConfig struct {
	consul      *testutil.TestServer
	tempDir     string
	numTasks    int
	numServices int
}

func generateConf(bConf benchmarkConfig) *config.Config {
	serviceNames := make([]string, bConf.numServices)
	for i := 0; i < bConf.numServices; i++ {
		serviceNames[i] = fmt.Sprintf("service_%03d", i)
	}

	taskConfigs := make(config.TaskConfigs, bConf.numTasks)
	for i := 0; i < bConf.numTasks; i++ {
		taskConfigs[i] = &config.TaskConfig{
			Name:     config.String(fmt.Sprintf("task_%03d", i)),
			Source:   config.String("../test_modules/local_file"),
			Services: serviceNames,
		}
	}

	conf := config.DefaultConfig()
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
	conf.Finalize()
	conf.Driver.Terraform.WorkingDir = config.String(bConf.tempDir)
	conf.Driver.Terraform.Path = config.String("../../../")
	return conf
}
