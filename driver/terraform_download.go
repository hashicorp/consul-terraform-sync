package driver

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/hashicorp/consul-terraform-sync/config"
	ctsVersion "github.com/hashicorp/consul-terraform-sync/version"
	"github.com/hashicorp/go-checkpoint"
	goVersion "github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-exec/tfexec"
	"github.com/hashicorp/terraform-exec/tfinstall"
)

const fallbackTFVersion = "0.13.5"

// TerraformVersion is the version of Terraform CLI for the Terraform driver.
var TerraformVersion string

// InstallTerraform installs the Terraform binary to the configured path.
// If an existing Terraform exists in the path, it is checked for compatibility.
func InstallTerraform(ctx context.Context, conf *config.TerraformConfig) error {
	workingDir := *conf.WorkingDir
	path := *conf.Path

	if _, err := os.Stat(workingDir); os.IsNotExist(err) {
		if err := os.MkdirAll(workingDir, workingDirPerms); err != nil {
			log.Printf("[ERR] (driver.terraform) error creating base work directory: %s", err)
			return err
		}
	}

	if isTFInstalled(path) {
		tfVersion, compatible, err := verifyInstalledTF(ctx, conf)
		if err != nil {
			if strings.Contains(err.Error(), "exec format error") {
				return errIncompatibleTerraformBinary
			}
			return err
		}

		// Set the global variable to the installed version
		version := tfVersion.String()
		TerraformVersion = version
		if !compatible {
			return errUnsupportedTerraformVersion
		}

		log.Printf("[INFO] (driver.terraform) skipping install, terraform %s "+
			"already exists at path %s/terraform", version, path)
		return nil
	}

	log.Printf("[INFO] (driver.terraform) installing terraform to path '%s'", path)
	tfVersion, err := installTerraform(ctx, conf)
	if err != nil {
		log.Printf("[ERR] (driver.terraform) error installing terraform: %s", err)
		return err
	}
	log.Printf("[INFO] (driver.terraform) successfully installed terraform")

	// Set the global variable to the installed version
	TerraformVersion = tfVersion.String()
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
	// installation for Sync
	path, err := exec.LookPath("terraform")
	if err != nil {
		return false
	}

	// have terraform at a different path
	log.Printf("[DEBUG] (driver.terraform) an existing Terraform was found in "+
		"another path: %s", path)
	log.Printf("[DEBUG] (driver.terraform) continuing with new installation")
	return false
}

// verifyInstalledTF checks if the installed Terraform is compatible with the
// current architecture and is valid within Sync version constraints.
func verifyInstalledTF(ctx context.Context, conf *config.TerraformConfig) (*goVersion.Version, bool, error) {
	workingDir := *conf.WorkingDir
	tfPath := *conf.Path

	// Verify version for existing terraform
	tf, err := tfexec.NewTerraform(workingDir, filepath.Join(tfPath, "terraform"))
	if err != nil {
		log.Printf("[ERR] (driver.terraform) unable to setup Terraform client "+
			"from path %s: %s", tfPath, err)
		return nil, false, err
	}

	tfVersion, _, err := tf.Version(ctx, true)
	if err != nil {
		log.Printf("[ERR] (driver.terraform) unable to verify Terraform version "+
			"from path %s: %s", tfPath, err)
		return nil, false, err
	}

	if !ctsVersion.TerraformConstraint.Check(tfVersion) {
		log.Printf("[ERR] (driver.terraform) Terraform found in path %s is "+
			"version %q and does not satisfy the constraint %q.",
			tfPath, tfVersion.String(), ctsVersion.CompatibleTerraformVersionConstraint)
		return tfVersion, false, nil
	}

	if err := isTFCompatible(conf, tfVersion); err != nil {
		return tfVersion, false, err
	}

	if *conf.Version != "" && *conf.Version != tfVersion.String() {
		log.Printf("[WARN] (driver.terraform) Terraform found in path %s is "+
			"version %q and does not match the configured version %q. Remove the "+
			"existing Terraform to install the selected version.",
			tfPath, tfVersion.String(), *conf.Version)
	}

	return tfVersion, true, nil
}

// isTFCompatible checks compatibility of the version of Terraform with the
// features of Consul Terraform Sync
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
//  Sync, the fall back version 0.13.5 is downloaded.
func installTerraform(ctx context.Context, conf *config.TerraformConfig) (*goVersion.Version, error) {
	var v *goVersion.Version
	if conf.Version != nil && *conf.Version != "" {
		v = goVersion.Must(goVersion.NewVersion(*conf.Version))
	} else {
		// Fetch the latest
		resp, err := checkpoint.Check(&checkpoint.CheckParams{Product: "terraform"})
		if err != nil {
			log.Printf("[ERR] (driver.terraform) error fetching Terraform versions "+
				"from Checkpoint: %s", err)
		} else if resp.CurrentVersion != "" {
			v = goVersion.Must(goVersion.NewVersion(resp.CurrentVersion))
		}
	}
	if v == nil || !ctsVersion.TerraformConstraint.Check(v) {
		// Configured version shouldn't be invalid our outside of the constraint at
		// this point if the configuration was validated.
		//
		// At this point we cannot guarantee compatibility of the latest Terraform
		// version, so we will move forward with a safe fallback version.
		log.Printf("[WARN] (driver.terrform) could not determine latest version "+
			"of terraform using checkpoint, fallback to version %s", fallbackTFVersion)
		v = goVersion.Must(goVersion.NewVersion(fallbackTFVersion))
	}

	if err := isTFCompatible(conf, v); err != nil {
		return nil, err
	}

	// Create path if doesn't already exist
	os.MkdirAll(*conf.Path, os.ModePerm)

	installedPath, err := tfinstall.ExactVersion(v.String(), *conf.Path).ExecPath(ctx)
	if err != nil {
		return nil, err
	}

	log.Printf("[DEBUG] (driver.terraform) successfully installed terraform %s: %s", v.String(), installedPath)
	return v, nil
}
