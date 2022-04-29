//go:build e2e
// +build e2e

// BenchmarkTasksConcurrent executes the Daemon controller directly to
// benchmark tasks running concurrently with the default TF driver. This
// emulates CTS in daemon mode.
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
	benchmarkTasksConcurrent(b, benchmarkConfig{
		numTasks:    1,
		numServices: 50,
		timeout:     30 * time.Second,
	})
}

func BenchmarkTasksConcurrent_t02_s50(b *testing.B) {
	benchmarkTasksConcurrent(b, benchmarkConfig{
		numTasks:    2,
		numServices: 50,
		timeout:     30 * time.Second,
	})
}

func BenchmarkTasksConcurrent_t10_s50(b *testing.B) {
	benchmarkTasksConcurrent(b, benchmarkConfig{
		numTasks:    10,
		numServices: 50,
		timeout:     30 * time.Second,
	})
}

func BenchmarkTasksConcurrent_t50_s50(b *testing.B) {
	benchmarkTasksConcurrent(b, benchmarkConfig{
		numTasks:    50,
		numServices: 50,
		timeout:     30 * time.Second,
	})
}

func benchmarkTasksConcurrent(b *testing.B, bConf benchmarkConfig) {
	// Benchmarks Run for the Daemon controller
	//
	// Daemon.Run involves executing Terraform apply concurrently
	srv := bConf.consul
	if srv == nil {
		srv = testutils.NewTestConsulServer(b, testutils.TestConsulServerConfig{
			HTTPSRelPath: "../../testutils",
		})
		defer srv.Stop()
		bConf.consul = srv
	}

	if bConf.tempDir == "" {
		bConf.tempDir = b.Name()
		cleanup := testutils.MakeTempDir(b, bConf.tempDir)
		defer cleanup()
	}

	conf := generateConf(b, bConf)

	ctrl, err := controller.NewDaemon(conf)
	require.NoError(b, err)

	err = ctrl.Init(context.Background())
	require.NoError(b, err)

	err = ctrl.Once(context.Background())
	require.NoError(b, err)

	// This is the crux of the benchmark which evaluates the performance of
	// tasks triggered and executing concurrently.
	ctx, ctxCancel := context.WithCancel(context.Background())
	defer ctxCancel()
	ctrlStopped := make(chan error)
	completedTasksCh := ctrl.EnableTestMode()

	go func() {
		ctrlStopped <- ctrl.Run(ctx)
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

	ctxTimeout, ctxTimeoutCancel := context.WithTimeout(context.Background(), bConf.timeout)
	defer ctxTimeoutCancel()

	completedTasks := make(map[string]bool, len(*conf.Tasks))
RunLoop:
	for {
		select {
		case taskName := <-completedTasksCh:
			completedTasks[taskName] = true
			b.Logf("%s completed (%d/%d)", taskName, len(completedTasks), bConf.numTasks)
			if len(completedTasks) == bConf.numTasks {
				ctxCancel() // Benchmark completed before timing out, stop the controller
			}

		case <-ctxTimeout.Done():
			ctxCancel() // Benchmark timed out, stop the controller

		case err := <-ctrlStopped:
			assert.Equal(b, err, context.Canceled)
			assert.Len(b, completedTasks, bConf.numTasks, "%s timed out before tasks were triggered and had executed", b.Name())
			break RunLoop
		}
	}
}
