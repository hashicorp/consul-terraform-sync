// +build e2e

// BenchmarkTasksConcurrent executes the ReadWrite controller directly to
// benchmark tasks running concurrently. This emulates CTS in daemon mode.
// $ go test ./e2e/benchmarks -bench=BenchmarkTasksConcurrent_ -tags e2e
package benchmarks

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/hashicorp/consul-terraform-sync/controller"
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

	conf := generateConf(benchmarkConfig{
		consul:      srv,
		tempDir:     tempDir,
		numTasks:    numTasks,
		numServices: numServices,
	})

	ctrl, err := controller.NewReadWrite(conf)
	require.NoError(b, err)
	rwCtrl := ctrl.(*controller.ReadWrite)

	err = rwCtrl.Init(context.Background())
	require.NoError(b, err)

	err = rwCtrl.Once(context.Background())
	require.NoError(b, err)

	// This is the crux of the benchmark which evaluates the performance of
	// tasks triggered and executing concurrently.
	ctx, ctxCancel := context.WithCancel(context.Background())
	defer ctxCancel()
	ctrlStopped := make(chan error)
	completedTasksCh := rwCtrl.EnableTestMode()

	go func() {
		ctrlStopped <- rwCtrl.Run(ctx)
	}()

	// Benchmark setup is done, reset the timer
	b.ResetTimer()

	// Register service instance to Consul catalog. This triggers task execution
	// for all tasks watching service-000
	random := rand.New(rand.NewSource(time.Now().UnixNano()))
	service := testutil.TestService{
		ID:      fmt.Sprintf("service-000-%s-%d", b.Name(), random.Intn(99999)),
		Name:    "service-000",
		Address: "5.6.7.8",
		Port:    8080,
	}
	testutils.RegisterConsulServiceHealth(b, srv, service, 0, testutil.HealthPassing)

	ctxTimeout, _ := context.WithTimeout(context.Background(), 30*time.Second)
	completedTasks := make(map[string]bool, len(*conf.Tasks))
RunLoop:
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
			break RunLoop
		}
	}
}
