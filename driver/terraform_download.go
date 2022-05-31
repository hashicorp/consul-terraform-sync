package driver

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/logging"
	ctsVersion "github.com/hashicorp/consul-terraform-sync/version"
	"github.com/hashicorp/go-checkpoint"
	goVersion "github.com/hashicorp/go-version"
	hcinstall "github.com/hashicorp/hc-install"
	"github.com/hashicorp/hc-install/fs"
	"github.com/hashicorp/hc-install/product"
	"github.com/hashicorp/hc-install/releases"
	"github.com/hashicorp/hc-install/src"
	"github.com/hashicorp/terraform-exec/tfexec"
)

const (
	logSystemName          = "driver"
	terraformSubsystemName = "terraform"
)

const fallbackTFVersion = "1.1.8"

// TerraformVersion is the version of Terraform CLI for the Terraform driver.
var TerraformVersion *goVersion.Version

// InstallTerraform installs the Terraform binary to the configured path.
// If an existing Terraform exists in the path, it is checked for compatibility.
func InstallTerraform(ctx context.Context, conf *config.TerraformConfig) error {
	path := *conf.Path

	logger := logging.Global().Named(logSystemName).Named(terraformSubsystemName)
	if isTFInstalled(path) {
		tfVersion, compatible, err := verifyInstalledTF(ctx, conf)
		if err != nil {
			if strings.Contains(err.Error(), "exec format error") {
				return errIncompatibleTerraformBinary
			}
			return err
		}

		// Set the global variable to the installed version
		TerraformVersion = tfVersion
		if !compatible {
			return errUnsupportedTerraformVersion
		}
		logger.Info("skipping install, terraform already exists",
			"tf_version", tfVersion.String(), "install_path", path)

		return nil
	}

	logger.Info("install terraform", "install_path", path)
	tfVersion, err := installTerraform(ctx, conf)
	if err != nil {
		logger.Error("error installing terraform", "error", err)
		return err
	}
	logger.Info("successfully installed terraform")

	// Set the global variable to the installed version
	TerraformVersion = tfVersion
	return nil
}

// isTFInstalled checks to see if terraform already exists at path.
func isTFInstalled(tfPath string) bool {
	tfPath = filepath.Join(tfPath, "terraform")

	// Check if terraform exists in target path
	if _, err := os.Stat(tfPath); err == nil {
		return true
	}

	// Check if terraform exists in $PATH to notify users about the new
	// installation for CTS
	path, err := exec.LookPath("terraform")
	if err != nil {
		return false
	}

	// have terraform at a different path
	logger := logging.Global().Named(logSystemName).Named(terraformSubsystemName)
	logger.Debug("an existing Terraform was found in another path", "install_path", path)
	logger.Debug("continuing with new installation")
	return false
}

// verifyInstalledTF checks if the installed Terraform is compatible with the
// current architecture and is valid within CTS version constraints.
func verifyInstalledTF(ctx context.Context, conf *config.TerraformConfig) (*goVersion.Version, bool, error) {
	tfPath := *conf.Path

	// NewTerraform requires an existing directory. This tfexec client is only
	// used for validation, so we don't need to use the actual working dir for the task
	wd, err := os.Getwd()
	if err != nil {
		return nil, false, err
	}

	// Verify version for existing terraform
	logger := logging.Global().Named(logSystemName).Named(terraformSubsystemName)
	tf, err := tfexec.NewTerraform(wd, filepath.Join(tfPath, "terraform"))
	if err != nil {
		logger.Error("unable to setup Terraform client", "terraform_path", tfPath, "error", err)
		return nil, false, err
	}

	tfVersion, _, err := tf.Version(ctx, true)
	if err != nil {
		logger.Error("unable to verify Terraform version", "terraform_path", tfPath, "error", err)
		return nil, false, err
	}

	if !ctsVersion.TerraformConstraint.Check(tfVersion) {
		logger.Error("found Terraform version does not satisfy the version constraint",
			"terraform_path", tfPath, "version", tfVersion.String(),
			"compatible_version_constraint", ctsVersion.CompatibleTerraformVersionConstraint)
		return tfVersion, false, nil
	}

	if err := isTFCompatible(conf, tfVersion); err != nil {
		return tfVersion, false, err
	}

	if *conf.Version != "" && *conf.Version != tfVersion.String() {
		logger.Warn("another terraform version was found that does not match the configured version",
			"terraform_path", tfPath, "terraform_version", tfVersion.String(),
			"configured_version", *conf.Version)
	}

	return tfVersion, true, nil
}

// isTFCompatible checks compatibility of the version of Terraform with the
// features of Consul-Terraform-Sync
func isTFCompatible(conf *config.TerraformConfig, version *goVersion.Version) error {
	// https://github.com/hashicorp/terraform/issues/23121
	if _, ok := conf.Backend["pg"]; ok {
		pgBackendConstraint, err := goVersion.NewConstraint(">= 0.14")
		if err != nil {
			return err
		}

		if !pgBackendConstraint.Check(version) {
			return fmt.Errorf("Consul-Terraform-Sync does not support pg "+
				"backend in automation with Terraform <= 0.13: %s", version.String())
		}
	}

	return nil
}

// installTerraform attempts to install the latest version of Terraform into
// the path. If the latest version is outside of the known supported range for
// CTS, the fall back version 0.13.5 is downloaded.
func installTerraform(ctx context.Context, conf *config.TerraformConfig) (*goVersion.Version, error) {
	var tfVersion *goVersion.Version
	logger := logging.Global().Named(logSystemName).Named(terraformSubsystemName)
	if conf.Version != nil && *conf.Version != "" {
		tfVersion = goVersion.Must(goVersion.NewVersion(*conf.Version))
	} else {
		// Fetch the latest
		resp, err := checkpoint.Check(&checkpoint.CheckParams{Product: "terraform"})
		if err != nil {
			logger.Error("error fetching Terraform versions from Checkpoint", "error", err)
		} else if resp.CurrentVersion != "" {
			tfVersion = goVersion.Must(goVersion.NewVersion(resp.CurrentVersion))
		}
	}

	if tfVersion == nil || !ctsVersion.TerraformConstraint.Check(tfVersion) {
		// Configured version shouldn't be invalid our outside of the constraint at
		// this point if the configuration was validated.
		//
		// At this point we cannot guarantee compatibility of the latest Terraform
		// version, so we will move forward with a safe fallback version.
		logger.Warn("could not determine latest version of terraform using Checkpoint, fallback to fallback version",
			"fallback_version", fallbackTFVersion)
		tfVersion = goVersion.Must(goVersion.NewVersion(fallbackTFVersion))
	}

	if err := isTFCompatible(conf, tfVersion); err != nil {
		return nil, err
	}

	// Create path if one doesn't already exist
	os.MkdirAll(*conf.Path, os.ModePerm)

	installer := hcinstall.NewInstaller()
	installedPath, err := installer.Ensure(ctx, []src.Source{
		&fs.ExactVersion{
			Product:    product.Terraform,
			Version:    tfVersion,
			ExtraPaths: []string{*conf.Path},
		},
		&releases.ExactVersion{
			Product:    product.Terraform,
			Version:    tfVersion,
			InstallDir: *conf.Path,
		},
	})

	if err != nil {
		return nil, err
	}

	logger.Debug("successfully installed terraform", "version", tfVersion.String(), "install_path", installedPath)
	return tfVersion, nil
}
