// Copyright IBM Corp. 2020, 2025
// SPDX-License-Identifier: MPL-2.0

//go:build integration
// +build integration

package client

import (
	"context"
	"os"
	"runtime"
	"testing"

	goVersion "github.com/hashicorp/go-version"
	hcinstall "github.com/hashicorp/hc-install"
	"github.com/hashicorp/hc-install/product"
	"github.com/hashicorp/hc-install/releases"
	"github.com/hashicorp/hc-install/src"
	"github.com/hashicorp/terraform-exec/tfexec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test_TerraformCLI_Leak checks for an issue with leaked goroutines in
// terraform-exec <v0.17
func Test_TerraformExec_Leak(t *testing.T) {
	// Do not run in parallel with other tasks because this test checks the
	// number of goroutines

	v, err := goVersion.NewVersion("1.1.8")
	require.NoError(t, err)

	// Download Terraform binary
	ctx := context.Background()
	installer := hcinstall.NewInstaller()
	_, err = installer.Ensure(ctx, []src.Source{
		&releases.ExactVersion{
			Product:    product.Terraform,
			Version:    v,
			InstallDir: ".",
		},
	})
	require.NoError(t, err)

	// Cleanup Terraform binary
	defer func() {
		_ = os.Remove("terraform")
	}()

	tf, err := tfexec.NewTerraform(".", ".")

	before := runtime.NumGoroutine()

	// Mimics CTS calling terraform
	tf.Init(ctx)
	tf.Apply(ctx)

	after := runtime.NumGoroutine()
	assert.Equal(t, before, after, "the number of goroutines after the terraform "+
		"requests should be the same as before")
}
