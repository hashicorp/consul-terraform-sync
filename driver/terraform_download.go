package driver

import (
	"archive/zip"
	"bytes"
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
	"golang.org/x/crypto/openpgp"
)

const hashicorpPublicKeyURL = "https://keybase.io/hashicorp/pgp_keys.asc"

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
		log.Printf("[DEBUG] (driver.terraform) skipping download verification")
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
	shasumURL := fmt.Sprintf("%s/terraform/%s/%s", releasesURL, version, shasumFilename)
	shasumContent, err := getFile(http.DefaultClient, shasumURL)
	if err != nil {
		return errors.Wrap(err, "unable to fetch SHASUM to verify download")
	}

	signatureURL := fmt.Sprintf("%s.sig", shasumURL)
	signatureContent, err := getFile(http.DefaultClient, signatureURL)
	if err != nil {
		return errors.Wrap(err, "unable to fetch signature to verify download")
	}

	hashicorpPublicKey, err := getFile(http.DefaultClient, hashicorpPublicKeyURL)
	if err != nil {
		return errors.Wrap(err, "unable to fetch HashiCorp public GPG key to verify download")
	}

	// Verify the signature using the HashiCorp public key
	err = verifySignature(hashicorpPublicKey, shasumContent, signatureContent)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("unable to verify HashiCorp signature "+
			"for terraform downloaded from %s", signatureURL))
	}

	// Verify that the SHA256 calculated for the file matches the SHASUM file
	err = verifyChecksum(path, filename, shasumContent)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("unable to verify checksum for terraform "+
			"downloaded from %s", shasumURL))
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

// getFile returns the content of a downloaded file.
func getFile(client *http.Client, url string) ([]byte, error) {
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s", resp.Status)
	}

	return ioutil.ReadAll(resp.Body)
}

// verifySignature verifies a PGP signature for a file with the provided key
func verifySignature(key, file, signature []byte) error {
	keyring, err := openpgp.ReadArmoredKeyRing(bytes.NewReader(key))
	if err != nil {
		return err
	}

	_, err = openpgp.CheckDetachedSignature(keyring,
		bytes.NewReader(file), bytes.NewReader(signature))
	return err
}

// verifyChecksum calculates and verifies the SHA256 of the file with the
// checksum file containing lines of checksums and filenames.
func verifyChecksum(path, filename string, shasumContent []byte) error {
	var expectedChecksum string
	for _, line := range strings.Split(string(shasumContent), "\n") {
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
		return err
	}
	defer f.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, f); err != nil {
		return err
	}

	actualChecksum := fmt.Sprintf("%x", hash.Sum(nil))
	if actualChecksum != expectedChecksum {
		return fmt.Errorf("%s has incorrect SHA-256 checksum %x (expected %x)",
			filename, actualChecksum, expectedChecksum)
	}

	return nil
}
