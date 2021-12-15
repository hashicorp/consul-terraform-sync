package tftmpl

import (
	"io"

	"github.com/hashicorp/hcl/v2/hclwrite"
)

const (
	logSystemName       = "templates"
	tftmplSubsystemName = "tftmpl"
)

// Template handles templates for different template functions to monitor
// different types of variables
type Template interface {
	// isServicesVar returns whether or not the template function returns a
	// variable of type services
	isServicesVar() bool

	// SourceIncludesVariable returns if the module source expects to
	// include the monitored variable.
	SourceIncludesVariable() bool

	// appendModuleAttribute writes to an HCL module body the monitored variable
	// as a module argument in main.tf file.
	// module "name" {
	//   catalog_services = var.catalog_services
	// }
	appendModuleAttribute(*hclwrite.Body)

	// appendTemplate writes the generated variable template to the
	// terrafort.tfvars.tmpl file based on whether the source includes the
	// monitored variable.
	appendTemplate(io.Writer) error

	// appendVariable writes the corresponding Terraform variable block to
	// the variables.tf file.
	appendVariable(io.Writer) error
}
