package version

import (
	"log"

	"github.com/hashicorp/go-version"
)

// CompatibleTerraformVersionConstraint is the version constraint
// imposed for running Terraform in automation with CTS. This is
// currently upward bounded to prevent using newer versions of
// Terraform that may introduce breaking changes CTS currently does not
// account for. The upper bound may be removed once CTS has protocols
// set in place for compatible modules and can handle Terraform syntax changes
// and enhancements between versions.
const CompatibleTerraformVersionConstraint = ">= 0.13.0, < 0.16"

// TerraformConstraint is the go-version constraint variable for
// CompatibleTerraformVersionConstraint
var TerraformConstraint version.Constraints

func init() {
	var err error
	TerraformConstraint, err = version.NewConstraint(CompatibleTerraformVersionConstraint)
	if err != nil {
		log.Panicf("error setting up Terraform version constraint %q: %s",
			CompatibleTerraformVersionConstraint, err)
	}
}
