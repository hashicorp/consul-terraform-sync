//go:build e2e
// +build e2e

package e2e

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/controller"
	"github.com/hashicorp/consul-terraform-sync/testutils"
	"github.com/stretchr/testify/require"
)

// TestServicesRenderRace tests that the template file terraform.tfvars.tmpl
// renders all services to terraform.tfvars before executing Terraform.
//
// issues GH-236 and GH-239
func TestServicesRenderRace(t *testing.T) {
	const NumberOfServices = 100

	srv := testutils.NewTestConsulServer(t,
		testutils.TestConsulServerConfig{HTTPSRelPath: "../testutils"})
	defer srv.Stop()

	tempDir := t.Name()
	cleanup := testutils.MakeTempDir(t, tempDir)
	//_ = cleanup
	defer cleanup()

	ctx, ctxCancel := context.WithCancel(context.Background())
	defer ctxCancel()

	// seed server with test data
	testServices := testutils.TestServices(NumberOfServices)
	testutils.AddServices(t, srv, testServices)

	// config
	conf := config.DefaultConfig()
	conf.BufferPeriod.Enabled = config.Bool(false)
	serviceNames := make([]string, NumberOfServices)
	for i, s := range testServices {
		serviceNames[i] = s.Name
	}
	conf.Tasks = &config.TaskConfigs{
		&config.TaskConfig{
			Name:   config.String("serv_rend_race_task"),
			Module: config.String("./test_modules/null_resource"),
			Condition: &config.ServicesConditionConfig{
				ServicesMonitorConfig: config.ServicesMonitorConfig{
					Names: serviceNames,
				},
			},
			WorkingDir: config.String(tempDir),
		}}
	conf.Consul.Address = config.String(srv.HTTPAddr)
	conf.Finalize()
	path, err := filepath.Abs("./")
	require.NoError(t, err)
	conf.Driver.Terraform.Path = config.String(path)

	// download terraform
	err = controller.InstallDriver(ctx, conf)
	require.NoError(t, err)

	// run controller
	ctrl, err := controller.NewReadWrite(conf)
	defer ctrl.Stop()
	require.NoError(t, err)
	err = ctrl.Init(ctx)
	require.NoError(t, err)
	err = ctrl.Once(ctx)
	require.NoError(t, err)

	// veryify results
	tfvarsFile := filepath.Join(tempDir, "terraform.tfvars")
	data, err := os.ReadFile(tfvarsFile)
	require.NoError(t, err)
	// 'svc-name-' is unique per service and much easier to count than
	// parsing the file and counting the real entries
	require.Equal(t, NumberOfServices, bytes.Count(data, []byte("svc-name-")))
}
