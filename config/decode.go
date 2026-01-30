// Copyright IBM Corp. 2020, 2025
// SPDX-License-Identifier: MPL-2.0

package config

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/mitchellh/mapstructure"
)

var reRepeatedBlock = regexp.MustCompile(`'([^\']+)' expected a map, got 'slice'`)

func processUnusedConfigKeys(md mapstructure.Metadata, file string) error {
	if len(md.Unused) == 0 {
		return nil
	}

	sort.Strings(md.Unused)
	err := fmt.Errorf("'%s' has invalid keys: %s", file, strings.Join(md.Unused, ", "))

	for _, key := range md.Unused {
		switch key {
		case "provider":
			return fmt.Errorf(`%s

'provider' is an invalid key for Consul-Terraform-Sync (CTS) configuration,
try 'terraform_provider'. The terraform_provider configuration blocks are
similar to provider blocks in Terraform but have additional features
supported only by CTS.`, err)

			// Enterprise-specific configurations, will only error in OSS
		case "driver.terraform-cloud":
			return fmt.Errorf(`%s

Terraform Cloud is a Consul-Terraform-Sync (CTS) Enterprise feature.
Upgrade to Consul Enterprise to enable CTS Enterprise features.`, err)

		case "license_path":
			return fmt.Errorf(`%s

'license_path' is a Consul-Terraform-Sync (CTS) Enterprise configuration.`, err)

		case "license":
			return fmt.Errorf(`%s

'license' is a Consul-Terraform-Sync (CTS) Enterprise configuration.`, err)

		case "high_availability":
			return fmt.Errorf(`%s

High availability is a Consul-Terraform-Sync (CTS) Enterprise feature.
Upgrade to Consul Enterprise to enable CTS Enterprise features.`, err)

		}
	}
	return err
}

// decodeError is a middleware for mapstructure.Decoder.Decode() errors
// with decode hooks specifically for CTS.
func decodeError(err error) error {
	if err == nil {
		return nil
	}

	match := reRepeatedBlock.FindStringSubmatch(err.Error())
	if len(match) >= 2 {
		return fmt.Errorf("only one '%s' block can be configured", match[1])
	}
	return err
}
