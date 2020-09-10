package version

import (
	"log"

	"github.com/hashicorp/go-version"
)

const CompatibleTerraformVersionConstraint = "~>0.13.0"

var TerraformConstraint version.Constraints

func init() {
	var err error
	TerraformConstraint, err = version.NewConstraint(CompatibleTerraformVersionConstraint)
	if err != nil {
		log.Panicf("error setting up Terraform version constraint %q: %s",
			CompatibleTerraformVersionConstraint, err)
	}
}
