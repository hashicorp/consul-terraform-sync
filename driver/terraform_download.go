package driver

import (
	"context"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/hashicorp/consul-terraform-sync/version"
	"github.com/hashicorp/go-checkpoint"
	goVersion "github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-exec/tfexec"
	"github.com/hashicorp/terraform-exec/tfinstall"
)

const fallbackTFVersion = "0.13.2"

// TerraformVersion is the version of Terraform CLI for the Terraform driver.
var TerraformVersion string

// InstallTerraform installs the Terraform binary to the configured path.
// If an existing Terraform exists in the path, it is checked for compatibility.
func InstallTerraform(ctx context.Context, path, workingDir string) error {
	if _, err := os.Stat(workingDir); os.IsNotExist(err) {
		if err := os.MkdirAll(workingDir, workingDirPerms); err != nil {
			log.Printf("[ERR] (driver.terraform) error creating base work directory: %s", err)
			return err
		}
	}

	if isTFInstalled(path) {
		tfVersion, compatible, err := isTFCompatible(ctx, workingDir, path)
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
	version, err := installTerraform(ctx, path)
	if err != nil {
		log.Printf("[ERR] (driver.terraform) error installing terraform: %s", err)
		return err
	}
	log.Printf("[INFO] (driver.terraform) successfully installed terraform")

	// Set the global variable to the installed version
	TerraformVersion = version
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

// isTFCompatible checks if the installed Terraform is compatible with the
// current architecture and is valid within Sync version constraints.
func isTFCompatible(ctx context.Context, workingDir, tfPath string) (*goVersion.Version, bool, error) {
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

	if !version.TerraformConstraint.Check(tfVersion) {
		log.Printf("[ERR] (driver.terraform) Terraform found in path %s is "+
			"version %q and does not satisfy the constraint %q.",
			tfPath, tfVersion.String(), version.CompatibleTerraformVersionConstraint)
		return tfVersion, false, nil
	}

	return tfVersion, true, nil
}

// installTerraform attempts to install the latest version of Terraform into
// the path. If the latest version is outside of the known supported range for
//  Sync, the fall back version 0.13.2 is downloaded.
func installTerraform(ctx context.Context, path string) (string, error) {
	resp, err := checkpoint.Check(&checkpoint.CheckParams{Product: "terraform"})
	if err != nil {
		log.Printf("[ERR] (driver.terraform) error fetching Terraform versions "+
			"from Checkpoint: %s", err)
		return "", err
	}

	var v string
	if resp.CurrentVersion == "" {
		log.Printf("[WARN] (driver.terrform) could not determine latest version "+
			"of terraform using checkpoint, fallback to version %s", fallbackTFVersion)
		v = fallbackTFVersion
	} else {
		// Check if the latest version is within support range for Sync.
		latest := goVersion.Must(goVersion.NewVersion(resp.CurrentVersion))
		if version.TerraformConstraint.Check(latest) {
			v = resp.CurrentVersion
		} else {
			// At this point we cannot guarantee compatibility for the latest
			// Terraform version, so we will move forward with a safe fallback
			// version.
			v = fallbackTFVersion
		}
	}

	// Create path if doesn't already exist
	os.MkdirAll(path, os.ModePerm)

	installedPath, err := tfinstall.ExactVersion(v, path).ExecPath(ctx)
	if err != nil {
		log.Printf("[ERR] (driver.terraform) error installing terraform")
		return "", err
	}

	log.Printf("[DEBUG] (driver.terraform) successfully installed terraform %s: %s", v, installedPath)
	return v, nil
}
