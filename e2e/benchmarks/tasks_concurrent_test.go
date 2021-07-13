// +build e2e

// BenchmarkTasksConcurrent executes the ReadWrite controller directly to
// benchmark tasks running concurrently. This emulates CTS in daemon mode.
// $ go test ./e2e/benchmarks -bench=BenchmarkTasksConcurrent_ -tags e2e
package benchmarks

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/consul-terraform-sync/controller"
	"github.com/hashicorp/consul-terraform-sync/event"
	"github.com/hashicorp/consul-terraform-sync/testutils"
	"github.com/hashicorp/consul/sdk/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func BenchmarkTasksConcurrent_t01_s50(b *testing.B) {
	benchmarkTasksConcurrent(b, 1, 50)
}

func BenchmarkTasksConcurrent_t02_s50(b *testing.B) {
	benchmarkTasksConcurrent(b, 2, 50)
}

func BenchmarkTasksConcurrent_t10_s50(b *testing.B) {
	benchmarkTasksConcurrent(b, 10, 50)
}

func BenchmarkTasksConcurrent_t50_s50(b *testing.B) {
	benchmarkTasksConcurrent(b, 50, 50)
}

func benchmarkTasksConcurrent(b *testing.B, numTasks, numServices int) {
	// Benchmarks Run for the ReadWrite controller
	//
	// ReadWriteController.Run involves executing Terraform apply concurrently
	srv := testutils.NewTestConsulServer(b, testutils.TestConsulServerConfig{
		HTTPSRelPath: "../../testutils",
	})

	defer srv.Stop()

	tempDir := b.Name()
	cleanup := testutils.MakeTempDir(b, tempDir)
	defer cleanup()

	ctx, ctxCancel := context.WithCancel(context.Background())
	conf := generateConf(benchmarkConfig{
		consul:      srv,
		tempDir:     tempDir,
		numTasks:    numTasks,
		numServices: numServices,
	})

	ctrl, err := controller.NewReadWrite(conf, event.NewStore())
	require.NoError(b, err)
	rwCtrl := ctrl.(*controller.ReadWrite)

	b.Run("task setup", func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			err = rwCtrl.Init(ctx)
			require.NoError(b, err)
		}
	})

	b.Run("once mode", func(b *testing.B) {
		// Run through once to initialize Terraform workspaces to isolate the
		// concurrent execution to only Terraform applies.
		for n := 0; n < b.N; n++ {
			err = rwCtrl.Once(ctx)
			require.NoError(b, err)
		}
	})

	b.Run("task concurrent execution", func(b *testing.B) {
		// This is the crux of the benchmark which evaluates the performance of
		// tasks triggered and executing concurrently.
		for n := 0; n < b.N; n++ {
			ctrlStopped := make(chan error)
			rwCtrl.EnableTestMode()
			completedTasksCh, err := rwCtrl.TaskNotifyChannel()
			require.NoError(b, err)

			go func() {
				err = rwCtrl.Run(ctx)
				ctrlStopped <- err
			}()

			// Benchmark setup is done, reset the timer
			b.ResetTimer()

			// Register service instance to Consul catalog. This triggers task execution
			// for all tasks watching service_000
			service := testutil.TestService{
				ID:      fmt.Sprintf("service_000_%s_%d", b.Name(), n),
				Name:    "service_000",
				Address: "5.6.7.8",
				Port:    8080,
			}
			testutils.RegisterConsulService(b, srv, service, testutil.HealthPassing, 0)

			ctxTimeout, _ := context.WithTimeout(context.Background(), 30*time.Second)
			completedTasks := make(map[string]bool, len(*conf.Tasks))
			for {
				select {
				case taskName := <-completedTasksCh:
					completedTasks[taskName] = true
					b.Logf("%s completed (%d/%d)", taskName, len(completedTasks), numTasks)
					if len(completedTasks) == numTasks {
						ctxCancel() // Benchmark completed before timing out, stop the controller
					}

				case <-ctxTimeout.Done():
					ctxCancel() // Benchmark timed out, stop the controller

				case err := <-ctrlStopped:
					assert.Equal(b, err, context.Canceled)
					assert.Len(b, completedTasks, numTasks, "%s timed out before tasks were triggered and had executed", b.Name())
					return
				}
			}
		}
	})
}
