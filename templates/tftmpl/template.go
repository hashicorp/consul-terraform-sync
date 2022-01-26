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
	// IsServicesVar returns whether or not the template function returns a
	// variable of type services
	IsServicesVar() bool

	// RendersVar returns whether or not the template renders the monitored
	// variable
	RendersVar() bool

	// appendModuleAttribute writes to an HCL module body the monitored variable
	// as a module argument in main.tf file.
	// module "name" {
	//   catalog_services = var.catalog_services
	// }
	appendModuleAttribute(*hclwrite.Body)

	// appendTemplate writes the generated variable template to the
	// terrafort.tfvars.tmpl file based on whether the template should render
	// the monitored variable.
	appendTemplate(io.Writer) error

	// appendVariable writes the corresponding Terraform variable block to
	// the variables.tf file.
	appendVariable(io.Writer) error
}
