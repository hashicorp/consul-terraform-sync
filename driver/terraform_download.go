package driver

import (
	"archive/zip"
	"crypto/sha256"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/pkg/errors"
)

// terraformInstalled checks to see if terraform already exists at path.
// Note: at this point assuming if terraform already exists, that it is the
// correct version, os, arch. User may have previously installed a version
// that we don't support. May want to add handling in the future.
func terraformInstalled(tfPath string) bool {
	tfPath = filepath.Join(tfPath, "terraform")

	// Check if terraform exists in target path
	if _, err := os.Stat(tfPath); err == nil {
		return true
	}

	// Check if terraform exists in $PATH to notify users about the new
	// installation for NIA
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

// installTerraform installs terraform: download file, verifies download, and
// unzips binary into path, and cleans up downloaded artifacts
func (tf *Terraform) install() error {
	// system info needed to install compatible binary
	opsys := runtime.GOOS
	arch := runtime.GOARCH

	filename := fmt.Sprintf("terraform_%s_%s_%s.zip", tf.version, opsys, arch)
	fullFilePath := filepath.Join(tf.path, filename)
	url := fmt.Sprintf("%s/terraform/%s/%s", releasesURL, tf.version, filename)

	log.Printf("[DEBUG] (driver.terraform) downloading %s from %s\n", filename, url)
	if err := download(url, fullFilePath); err != nil {
		return errors.Wrap(err, "Unable to download zip")
	}
	log.Printf("[DEBUG] (driver.terraform) successfully downloaded terraform")

	if !tf.skipVerify {
		log.Printf("[DEBUG] (driver.terraform) verifying checksum and signature")
		if err := verifyTerraformDownload(tf.path, filename, tf.version); err != nil {
			return err
		}
		log.Printf("[DEBUG] (driver.terraform) checksum and signature verified")
	} else {
		log.Printf("[DEBUG] (driver.terraform) verifying download skipped")
	}

	log.Printf("[DEBUG] (driver.terraform) unziping %s to %s\n", filename, tf.path)
	if err := unzip(fullFilePath, tf.path); err != nil {
		return errors.Wrap(err, "Unable to unzip binary")
	}

	// Cleanup zip file after successful installation
	log.Printf("[DEBUG] (driver.terraform) removing %s", filename)
	os.Remove(fullFilePath)

	return nil
}

// download downloads a file from the provided URL and writes it locally to a
// file. Include the full path of the file to specificy the download location.
func download(url, filename string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err = io.Copy(out, resp.Body); err != nil {
		return err
	}
	return nil
}

func verifyTerraformDownload(path, filename, version string) error {
	// Download SHASUM file and signature
	shasumFilename := fmt.Sprintf("terraform_%s_SHA256SUMS", version)
	resp, err := http.Get(fmt.Sprintf("%s/terraform/%s/%s",
		releasesURL, version, shasumFilename))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Println("[ERR] (driver.terraform) unable to download SHASUM to verify download")
		return err
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("[ERR] (driver.terraform) error reading SHASUM file to verify download")
		return err
	}

	var expectedChecksum string
	for _, line := range strings.Split(string(respBody), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}
		checksum, f := fields[0], fields[1]
		if f == filename {
			expectedChecksum = checksum
		}
	}
	if expectedChecksum == "" {
		return fmt.Errorf("checksum for %s not found to verify download", filename)
	}

	f, err := os.Open(filepath.Join(path, filename))
	if err != nil {
		return errors.Wrap(err, "issue reading download from disk to calculate SHASUM")
	}
	defer f.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, f); err != nil {
		return errors.Wrap(err, "error fetching the SHASUM of the terraform archive")
	}

	actualChecksum := fmt.Sprintf("%x", hash.Sum(nil))
	if actualChecksum != expectedChecksum {
		log.Println("[ERR] (driver.terraform) terraform archive has incorrect SHA-256 "+
			"checksum %x (expected %x)", actualChecksum, expectedChecksum)
		return fmt.Errorf("SHASUM does not match the downloaded terraform archive")
	}

	return nil
}

// unzip unzips the terraform binary. Assumes only one file inside the zip file.
// Simplied version of https://stackoverflow.com/a/24792688 for the sake of the poc
func unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer rc.Close()

		path := filepath.Join(dest, f.Name)
		fc, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}
		defer fc.Close()

		_, err = io.Copy(fc, rc)
		if err != nil {
			return err
		}
	}

	return nil
}
