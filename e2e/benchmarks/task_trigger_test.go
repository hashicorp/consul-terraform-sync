//go:build e2e
// +build e2e

// BenchmarkTaskTrigger benchmarks the time for a Consul catalog change to
// trigger and re-render templates used for tasks. This does not benchmark
// task execution time.
// $ go test ./e2e/benchmarks/ -bench BenchmarkTaskTrigger -benchtime=10s -tags e2e
package benchmarks

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hashicorp/consul-terraform-sync/controller"
	"github.com/hashicorp/consul-terraform-sync/testutils"
	"github.com/hashicorp/consul/sdk/testutil"
	"github.com/stretchr/testify/require"
)

func BenchmarkTaskTrigger(b *testing.B) {
	srv := testutils.NewTestConsulServer(b, testutils.TestConsulServerConfig{
		HTTPSRelPath: "../../testutils",
	})
	defer srv.Stop()

	tempDir := b.Name()
	cleanup := testutils.MakeTempDir(b, tempDir)
	defer cleanup()

	ctx, ctxCancel := context.WithCancel(context.Background())
	defer ctxCancel()
	conf := generateConf(b, benchmarkConfig{
		consul:      srv,
		tempDir:     tempDir,
		numTasks:    1,
		numServices: 25,
	})

	rwCtrl, err := controller.NewDaemon(conf)
	require.NoError(b, err)
	err = rwCtrl.Init(ctx)
	require.NoError(b, err)
	err = rwCtrl.Once(ctx)
	require.NoError(b, err)

	ctrlStopped := make(chan error)
	go func() {
		err := rwCtrl.Run(ctx)
		ctrlStopped <- err
	}()

	// Monitor templated file for updates
	tfvarsFile := filepath.Join(tempDir, benchmarkTaskName(b, 0), "terraform.tfvars")
	fileLastUpdated := getFileModTime(b, tfvarsFile)
	fileUpdated := make(chan struct{}, b.N)
	go func(lastUpdate *time.Time) {
		for n := 0; n < b.N; {
			modTime := getFileModTime(b, tfvarsFile)
			if modTime.After(*lastUpdate) {
				fileUpdated <- struct{}{}
				// b.Logf("terraform.tfvars modified: %s\t%s\t%s",
				// 	lastUpdate.String(), modTime.String(), modTime.Sub(*lastUpdate))
				*lastUpdate = modTime
				n++
			}
		}
		// b.Log("done monitoring file updates")
	}(&fileLastUpdated)

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		// Make a Consul catalog change for a monitored service
		go func() {
			service := testutil.TestService{
				ID:      fmt.Sprintf("service-000-%s%d", b.Name(), n),
				Name:    "service-000",
				Address: "5.6.7.8",
				Port:    8080,
			}
			testutils.RegisterConsulServiceHealth(b, srv, service, 0, testutil.HealthPassing)
			// b.Logf("service instance registered: %s", service.ID)
		}()

		// Detect when the template has re-rendered
		var updateDetected bool
		for !updateDetected {
			select {
			case <-fileUpdated:
				updateDetected = true
			case err := <-ctrlStopped:
				select {
				case <-ctx.Done():
					break
				default:
					b.Fatalf("controller stopped unexpectedly: %s", err)
				}
			}
		}
	}
}

func getFileModTime(tb testing.TB, file string) time.Time {
	fileInfo, err := os.Stat(file)
	require.NoError(tb, err)
	return fileInfo.ModTime()
}
