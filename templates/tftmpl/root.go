package tftmpl

import (
	"log"
	"os"
	"path"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/hashicorp/terraform/configs/hcl2shim"
	"github.com/zclconf/go-cty/cty"
)

const (
	TerraformRequiredVersion = ">=0.13"

	RootFilename = "main.tf"
)

type Task struct {
	Name    string
	Source  string
	Version string
}

type RootModuleInputData struct {
	Task    Task
	Backend map[string]interface{}
}

func NewRootModule(input RootModuleInputData) (*hclwrite.File, error) {
	f := hclwrite.NewEmptyFile()
	rootBody := f.Body()

	// Terraform block to declare the version constraint
	tfBlock := rootBody.AppendNewBlock("terraform", nil)
	tfBody := tfBlock.Body()
	tfBody.SetAttributeValue("required_version", cty.StringVal(TerraformRequiredVersion))

	// Configure the Terraform backend
	// Remove one layer of nesting to use for block label
	var backendLabel string
	var backendBlock map[string]interface{}
	for label, block := range input.Backend {
		backendLabel = label
		backendBlock = block.(map[string]interface{})
	}
	backendBody := tfBody.AppendNewBlock("backend", []string{backendLabel}).Body()
	for attr, val := range backendBlock {
		backendBody.SetAttributeValue(attr, hcl2shim.HCL2ValueFromConfigValue(val))
	}
	rootBody.AppendNewline()

	// Module block
	moduleBlock := rootBody.AppendNewBlock("module", []string{input.Task.Name})
	moduleBody := moduleBlock.Body()
	moduleBody.SetAttributeValue("source", cty.StringVal(input.Task.Source))
	if len(input.Task.Version) > 0 {
		moduleBody.SetAttributeValue("version", cty.StringVal(input.Task.Version))
	}
	rootBody.AppendNewline()
	moduleBody.SetAttributeTraversal("services", hcl.Traversal{
		hcl.TraverseRoot{
			Name: "var",
		},
		hcl.TraverseAttr{
			Name: "services",
		},
	})

	return f, nil
}

func InitRootModule(input RootModuleInputData, dir string, force bool) error {
	modulePath := path.Join(dir, input.Task.Name)
	fullPath := path.Join(modulePath, RootFilename)

	if fileExists(fullPath) {
		if !force {
			log.Printf("[DEBUG] (templates.terraform) root module for task %q "+
				"already exists, skipping file creation", input.Task.Name)
			return nil
		} else {
			log.Printf("[INFO] (templates.terraform) overwriting root module for "+
				"task %q", input.Task.Name)
		}
	}
	os.MkdirAll(modulePath, os.ModePerm)

	module, err := NewRootModule(input)
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] (templates.terraform) creating root module for task %q: %s",
		input.Task.Name, fullPath)

	f, err := os.Create(fullPath)
	if err != nil {
		return err
	}

	_, err = module.WriteTo(f)
	return err
}

func fileExists(name string) bool {
	if _, err := os.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}
