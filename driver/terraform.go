package driver

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/hashicorp/consul-terraform-sync/client"
	"github.com/hashicorp/consul-terraform-sync/handler"
	"github.com/hashicorp/consul-terraform-sync/templates"
	"github.com/hashicorp/consul-terraform-sync/templates/hcltmpl"
	"github.com/hashicorp/consul-terraform-sync/templates/tftmpl"
	"github.com/hashicorp/hcat"
	"github.com/pkg/errors"
)

const (
	// Types of clients that are alternatives to the default Terraform CLI client
	developmentClient = "development"
	testClient        = "test"

	// Permissions for created directories and files
	workingDirPerms = os.FileMode(0750) // drwxr-x---
	filePerms       = os.FileMode(0640) // -rw-r-----

	errSuggestion = "remove Terraform from the configured path or specify a new path to safely install a compatible version."
)

var (
	_ Driver = (*Terraform)(nil)

	errUnsupportedTerraformVersion = fmt.Errorf("unsupported Terraform version: %s", errSuggestion)
	errIncompatibleTerraformBinary = fmt.Errorf("incompatible Terraform binary: %s", errSuggestion)
)

// Terraform is an Sync driver that uses the Terraform CLI to interface with
// low-level network infrastructure.
type Terraform struct {
	mu *sync.RWMutex

	task              Task
	backend           map[string]interface{}
	requiredProviders map[string]interface{}

	resolver   templates.Resolver
	template   templates.Template
	watcher    templates.Watcher
	fileReader func(string) ([]byte, error)

	workingDir string
	client     client.Client
	logClient  bool
	postApply  handler.Handler

	inited bool
}

// TerraformConfig configures the Terraform driver
type TerraformConfig struct {
	Task              Task
	Log               bool
	PersistLog        bool
	Path              string
	WorkingDir        string
	Backend           map[string]interface{}
	RequiredProviders map[string]interface{}
	Watcher           templates.Watcher
	// empty/unknown string will default to TerraformCLI client
	ClientType string
}

// NewTerraform configures and initializes a new Terraform driver for a task.
// The underlying Terraform CLI client and out-of-band handlers are prepared.
func NewTerraform(config *TerraformConfig) (*Terraform, error) {
	if _, err := os.Stat(config.WorkingDir); os.IsNotExist(err) {
		if err := os.Mkdir(config.WorkingDir, workingDirPerms); err != nil {
			log.Printf("[ERR] (driver.terraform) error creating task work directory: %s", err)
			return nil, err
		}
	}

	tfClient, err := newClient(&clientConfig{
		task:       config.Task,
		clientType: config.ClientType,
		log:        config.Log,
		persistLog: config.PersistLog,
		path:       config.Path,
		workingDir: config.WorkingDir,
	})
	if err != nil {
		log.Printf("[ERR] (driver.terraform) init client type '%s' error: %s", config.ClientType, err)
		return nil, err
	}

	if taskEnv := config.Task.Env; len(taskEnv) > 0 {
		// Terraform init requires discovering git in the PATH env.
		//
		// The terraform-exec package disables inheriting from the os environment
		// when using tfexec.SetEnv(). So for CTS purposes, we'll force inheritance
		// to allow Terraform commands to use the os environment as necessary.
		env := envMap(os.Environ())
		for k, v := range taskEnv {
			env[k] = v
		}
		tfClient.SetEnv(env)
	}

	handler, err := getTerraformHandlers(config.Task)
	if err != nil {
		return nil, err
	}

	return &Terraform{
		mu:                &sync.RWMutex{},
		task:              config.Task,
		backend:           config.Backend,
		requiredProviders: config.RequiredProviders,
		workingDir:        config.WorkingDir,
		client:            tfClient,
		logClient:         config.Log,
		postApply:         handler,
		resolver:          hcat.NewResolver(),
		watcher:           config.Watcher,
		fileReader:        ioutil.ReadFile,
	}, nil
}

// Version returns the Terraform CLI version for the Terraform driver.
func (tf *Terraform) Version() string {
	return TerraformVersion.String()
}

// Task returns the task config info
func (tf *Terraform) Task() Task {
	return tf.task
}

// InitTask initializes the task by creating the Terraform root module and related
// files to execute on.
func (tf *Terraform) InitTask(force bool) error {
	tf.mu.Lock()
	defer tf.mu.Unlock()

	if !tf.task.Enabled {
		log.Printf("[TRACE] (driver.terraform) task '%s' disabled. skip"+
			"initializing", tf.task.Name)
		return nil
	}

	return tf.initTask(force)
}

// SetBufferPeriod sets the buffer period for the task. Do not set this when
// task needs to immediately render a template and run.
func (tf *Terraform) SetBufferPeriod() {
	tf.mu.Lock()
	defer tf.mu.Unlock()

	if !tf.task.Enabled {
		log.Printf("[TRACE] (driver.terraform) task '%s' disabled. skip"+
			"setting buffer period", tf.task.Name)
		return
	}

	taskName := tf.task.Name

	if tf.template == nil {
		log.Printf("[WARN] (driver.terraform) attempted to set buffer for "+
			"'%s' which does not have a template", taskName)
		return
	}

	bp := tf.task.BufferPeriod
	if bp == nil {
		log.Printf("[TRACE] (driver.terraform) no buffer period for '%s'", taskName)
		return
	}

	log.Printf("[TRACE] (driver.terraform) set buffer period for '%s': %+v",
		taskName, bp)
	tf.watcher.SetBufferPeriod(bp.Min, bp.Max, tf.template.ID())
}

// RenderTemplate fetches data for the template. If the data is complete fetched,
// renders the template. Rendering a template for the first time may take several
// cycles to load all the dependencies asynchronously. Returns a boolean whether
// the template was rendered
func (tf *Terraform) RenderTemplate(ctx context.Context) (bool, error) {
	tf.mu.Lock()
	defer tf.mu.Unlock()

	if !tf.task.Enabled {
		log.Printf("[TRACE] (driver.terraform) task '%s' disabled. skip"+
			"rendering template", tf.task.Name)
		return true, nil
	}

	return tf.renderTemplate(ctx)
}

// InspectTask inspects for any differences pertaining to the task between
// the state of Consul and network infrastructure using the Terraform plan command
func (tf *Terraform) InspectTask(ctx context.Context) error {
	tf.mu.Lock()
	defer tf.mu.Unlock()

	if !tf.task.Enabled {
		log.Printf("[TRACE] (driver.terraform) task '%s' disabled. skip"+
			"inspecting", tf.task.Name)
		return nil
	}

	_, err := tf.inspectTask(ctx, false)
	return err
}

// ApplyTask applies the task changes.
func (tf *Terraform) ApplyTask(ctx context.Context) error {
	tf.mu.Lock()
	defer tf.mu.Unlock()

	if !tf.task.Enabled {
		log.Printf("[TRACE] (driver.terraform) task '%s' disabled. skip"+
			"applying", tf.task.Name)
		return nil
	}

	return tf.applyTask(ctx)
}

// InspectPlan stores return the information about what
type InspectPlan struct {
	ChangesPresent bool   `json:"changes_present"`
	Plan           string `json:"plan"`
}

// UpdateTask updates the task on the driver. Makes any calls to re-init
// depending on the fields updated. If update task is requested with the inspect
// run option, then returns the inspected plan
func (tf *Terraform) UpdateTask(ctx context.Context, patch PatchTask) (InspectPlan, error) {
	switch patch.RunOption {
	case "", RunOptionInspect, RunOptionNow:
		// valid options
	default:
		return InspectPlan{}, fmt.Errorf("Invalid run option '%s'. Please select a valid "+
			"option", patch.RunOption)
	}

	tf.mu.Lock()
	defer tf.mu.Unlock()

	reinit := false

	if tf.task.Enabled != patch.Enabled {
		tf.task.Enabled = patch.Enabled
		if tf.task.Enabled {
			reinit = true
		}
	}

	// identify cases where resources are not impacted and we can return early
	switch {
	case patch.Enabled == false:
		return InspectPlan{}, nil
	default:
		// onward!
	}

	if reinit {
		if err := tf.initTask(true); err != nil {
			return InspectPlan{}, fmt.Errorf("Error updating task '%s'. Unable to init "+
				"task: %s", tf.task.Name, err)
		}

		for i := int64(0); ; i++ {
			rendered, err := tf.renderTemplate(ctx)
			if err != nil {
				return InspectPlan{}, fmt.Errorf("Error updating task '%s'. Unable to "+
					"render template for task: %s", tf.task.Name, err)
			}
			if rendered {
				break
			}
		}
	}

	if patch.RunOption == RunOptionInspect {
		log.Printf("[TRACE] (driver.terraform) update task '%s'. inspect run "+
			"option", tf.task.Name)
		plan, err := tf.inspectTask(ctx, true)
		if err != nil {
			return InspectPlan{}, fmt.Errorf("Error updating task '%s'. Unable to inspect "+
				"task: %s", tf.task.Name, err)
		}
		return plan, nil
	}

	if patch.RunOption == RunOptionNow {
		log.Printf("[TRACE] (driver.terraform) update task '%s'. run now "+
			"option", tf.task.Name)
		return InspectPlan{}, tf.applyTask(ctx)
	}

	// allow the task to update naturally!
	return InspectPlan{}, nil
}

// init initializes the Terraform workspace if needed
func (tf *Terraform) init(ctx context.Context) error {
	taskName := tf.task.Name

	if tf.inited {
		log.Printf("[TRACE] (driver.terraform) workspace for task already "+
			"initialized, skipping for '%s'", taskName)
		return nil
	}

	log.Printf("[TRACE] (driver.terraform) initializing workspace '%s'", taskName)
	if err := tf.client.Init(ctx); err != nil {
		return errors.Wrap(err, fmt.Sprintf("error tf-init for '%s'", taskName))
	}
	tf.inited = true
	return nil
}

// initTask initializes the task
func (tf *Terraform) initTask(force bool) error {
	task := tf.task

	services := make([]tftmpl.Service, len(task.Services))
	for i, s := range task.Services {
		services[i] = tftmpl.Service{
			Datacenter:  s.Datacenter,
			Description: s.Description,
			Name:        s.Name,
			Namespace:   s.Namespace,
			Tag:         s.Tag,
		}
	}

	var vars hcltmpl.Variables
	for _, vf := range task.VarFiles {
		tfvars, err := tftmpl.LoadModuleVariables(vf)
		if err != nil {
			return err
		}

		if len(vars) == 0 {
			vars = tfvars
			continue
		}

		for k, v := range tfvars {
			vars[k] = v
		}
	}

	input := tftmpl.RootModuleInputData{
		TerraformVersion: TerraformVersion,
		Backend:          tf.backend,
		Providers:        task.Providers.ProviderBlocks(),
		ProviderInfo:     task.ProviderInfo,
		Services:         services,
		Task: tftmpl.Task{
			Description: task.Description,
			Name:        task.Name,
			Source:      task.Source,
			Version:     task.Version,
		},
		Variables: vars,
	}
	input.Init()

	if err := tftmpl.InitRootModule(&input, tf.workingDir, filePerms, force); err != nil {
		return err
	}

	if err := tf.initTaskTemplate(); err != nil {
		return err
	}

	// initTask() can be called more than once. It's very likely initializing a
	// task will require re-initializing terraform. Reset to false so terraform
	// will reinit
	tf.inited = false

	return nil
}

// renderTemplate attempts to render the hashicat template
func (tf *Terraform) renderTemplate(ctx context.Context) (bool, error) {
	taskName := tf.task.Name
	log.Printf("[TRACE] (driver.terraform) checking dependency changes for task %s", taskName)

	var err error
	var result hcat.ResolveEvent
	if result, err = tf.resolver.Run(tf.template, tf.watcher); err != nil {
		log.Printf("[ERROR] (driver.terraform) checking dependency changes "+
			"for '%s': %s", taskName, err)

		return false, fmt.Errorf("error fetching template dependencies for task %s: %s",
			tf.task.Name, err)
	}

	// result.Complete is only `true` if the template has new data that has been
	// completely fetched.
	if result.Complete {
		log.Printf("[DEBUG] (driver.terraform) change detected for task %s", taskName)

		var rendered hcat.RenderResult
		if rendered, err = tf.template.Render(result.Contents); err != nil {
			log.Printf("[ERR] (driver.terraform) rendering template for '%s': %s",
				taskName, err)

			return false, err
		}
		log.Printf("[TRACE] (driver.terraform) template for task %q rendered: %+v",
			taskName, rendered)
	}

	return result.Complete, nil
}

// inspectTask inspects the task changes. Option to return inspection plan
// details rather than logging out
func (tf *Terraform) inspectTask(ctx context.Context, returnPlan bool) (InspectPlan, error) {
	taskName := tf.task.Name

	if err := tf.init(ctx); err != nil {
		log.Printf("[ERR] (driver.terraform) error initializing workspace, "+
			"skipping plan for '%s'", taskName)
		return InspectPlan{}, err
	}

	var buf bytes.Buffer
	if returnPlan {
		tf.client.SetStdout(&buf)

		var logger *log.Logger
		if tf.logClient {
			logger = log.New(log.Writer(), "", log.Flags())
		} else {
			logger = log.New(ioutil.Discard, "", 0)
		}
		defer tf.client.SetStdout(logger.Writer())
	}

	log.Printf("[TRACE] (driver.terraform) plan '%s'", taskName)
	c, err := tf.client.Plan(ctx)
	if err != nil {
		return InspectPlan{}, errors.Wrap(err,
			fmt.Sprintf("error tf-plan for '%s'", taskName))
	}

	return InspectPlan{
		ChangesPresent: c,
		Plan:           buf.String(),
	}, nil
}

// applyTask applies the task changes.
func (tf *Terraform) applyTask(ctx context.Context) error {
	taskName := tf.task.Name

	if err := tf.init(ctx); err != nil {
		log.Printf("[ERR] (driver.terraform) error initializing workspace, "+
			"skipping apply for '%s'", taskName)
		return err
	}

	log.Printf("[TRACE] (driver.terraform) apply '%s'", taskName)
	if err := tf.client.Apply(ctx); err != nil {
		return errors.Wrap(err, fmt.Sprintf("error tf-apply for '%s'", taskName))
	}

	if tf.postApply != nil {
		log.Printf("[TRACE] (driver.terraform) post-apply out-of-band actions "+
			"for '%s'", taskName)
		if err := tf.postApply.Do(ctx, nil); err != nil {
			return err
		}
	}

	return nil
}

// initTaskTemplate creates templates to be monitored and rendered.
func (tf *Terraform) initTaskTemplate() error {
	tmplFullpath := filepath.Join(tf.workingDir, tftmpl.TFVarsTmplFilename)
	tfvarsFilepath := filepath.Join(tf.workingDir, tftmpl.TFVarsFilename)

	content, err := tf.fileReader(tmplFullpath)
	if err != nil {
		log.Printf("[ERR] (driver.terraform) unable to read file for '%s': %s",
			tf.task.Name, err)
		return err
	}

	renderer := hcat.NewFileRenderer(hcat.FileRendererInput{
		Path:  tfvarsFilepath,
		Perms: filePerms,
	})

	tf.template = hcat.NewTemplate(hcat.TemplateInput{
		Contents:     string(content),
		Renderer:     renderer,
		FuncMapMerge: tftmpl.HCLTmplFuncMap(tf.task.UserDefinedMeta),
	})

	return nil
}

// getTerraformHandlers returns the first handler in a chain of handlers
// for a Terraform driver.
//
// Returned handler may be nil even if returned err is nil. This happens when
// no providers have a handler.
func getTerraformHandlers(task Task) (handler.Handler, error) {
	counter := 0
	var next handler.Handler
	for _, p := range task.Providers {
		h, err := handler.TerraformProviderHandler(p.Name(), p.ProviderBlock().RawConfig())
		if err != nil {
			log.Printf(
				"[ERR] (driver.terraform) could not initialize handler for "+
					"provider '%s': %s", p.Name(), err)
			return nil, err
		}
		if h != nil {
			counter++
			log.Printf(
				"[INFO] (driver.terraform) retrieved handler for provider '%s'", p.Name())
			h.SetNext(next)
			next = h
		}
	}
	log.Printf("[INFO] (driver.terraform) retrieved %d Terraform handlers for task '%s'",
		counter, task.Name)
	return next, nil
}

func envMap(environ []string) map[string]string {
	env := map[string]string{}
	for _, ev := range environ {
		parts := strings.SplitN(ev, "=", 2)
		if len(parts) == 0 {
			continue
		}
		k := parts[0]
		v := ""
		if len(parts) == 2 {
			v = parts[1]
		}
		env[k] = v
	}
	return env
}
