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
	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/handler"
	"github.com/hashicorp/consul-terraform-sync/logging"
	"github.com/hashicorp/consul-terraform-sync/templates"
	"github.com/hashicorp/consul-terraform-sync/templates/tftmpl"
	"github.com/hashicorp/consul-terraform-sync/templates/tftmpl/notifier"
	"github.com/hashicorp/consul-terraform-sync/templates/tftmpl/tmplfunc"
	"github.com/hashicorp/hcat"
	"github.com/hashicorp/hcat/dep"
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

	taskNameLogKey = "task_name"
)

var (
	_ Driver = (*Terraform)(nil)

	errUnsupportedTerraformVersion = fmt.Errorf("unsupported Terraform version: %s", errSuggestion)
	errIncompatibleTerraformBinary = fmt.Errorf("incompatible Terraform binary: %s", errSuggestion)
)

// Terraform is a Sync driver that uses the Terraform CLI to interface with
// low-level network infrastructure.
type Terraform struct {
	mu sync.RWMutex

	task              *Task
	backend           map[string]interface{}
	requiredProviders map[string]interface{}

	resolver   templates.Resolver
	template   templates.Template
	watcher    templates.Watcher
	fileReader func(string) ([]byte, error)

	client    client.Client
	logClient bool
	postApply handler.Handler

	inited       bool
	renderedOnce bool

	logger logging.Logger
}

// TerraformConfig configures the Terraform driver
type TerraformConfig struct {
	Task              *Task
	Log               bool
	PersistLog        bool
	Path              string
	Backend           map[string]interface{}
	RequiredProviders map[string]interface{}
	Watcher           templates.Watcher
	// empty/unknown string will default to TerraformCLI client
	ClientType string
}

// NewTerraform configures and initializes a new Terraform driver for a task.
// The underlying Terraform CLI client and out-of-band handlers are prepared.
func NewTerraform(config *TerraformConfig) (*Terraform, error) {
	task := config.Task
	taskName := task.Name()
	wd := task.WorkingDir()
	logger := logging.Global().Named(logSystemName).Named(terraformSubsystemName)
	if _, err := os.Stat(wd); os.IsNotExist(err) {
		if err := os.MkdirAll(wd, workingDirPerms); err != nil {
			logger.Error("error creating task work directory", "error", err)
			return nil, err
		}
	}

	tfClient, err := newClient(&clientConfig{
		clientType: config.ClientType,
		log:        config.Log,
		taskName:   taskName,
		persistLog: config.PersistLog,
		path:       config.Path,
		workingDir: wd,
	})
	if err != nil {
		logger.Error("init client type error", "client_type", config.ClientType, "error", err)
		return nil, err
	}

	if taskEnv := task.Env(); len(taskEnv) > 0 {
		// Terraform init requires discovering git in the PATH env.
		//
		// The terraform-exec package disables inheriting from the os environment
		// when using tfexec.SetEnv(). So for CTS purposes, we'll force inheritance
		// to allow Terraform commands to use the os environment as necessary.
		env := envMap(os.Environ())
		for k, v := range taskEnv {
			env[k] = v
		}
		err = tfClient.SetEnv(env)
		if err != nil {
			logger.Error("error setting the environment for the client",
				"client_type", config.ClientType, "error", err)
			return nil, err
		}
	}

	h, err := getTerraformHandlers(taskName, task.Providers())
	if err != nil {
		return nil, err
	}

	return &Terraform{
		task:              config.Task,
		backend:           config.Backend,
		requiredProviders: config.RequiredProviders,
		client:            tfClient,
		logClient:         config.Log,
		postApply:         h,
		resolver:          hcat.NewResolver(),
		watcher:           config.Watcher,
		fileReader:        ioutil.ReadFile,
		logger:            logger,
	}, nil
}

// Version returns the Terraform CLI version for the Terraform driver.
func (tf *Terraform) Version() string {
	return TerraformVersion.String()
}

// Task returns the task config info
func (tf *Terraform) Task() *Task {
	return tf.task
}

// InitTask initializes the task by creating the Terraform root module and related
// files to execute on.
func (tf *Terraform) InitTask(ctx context.Context) error {
	tf.mu.Lock()
	defer tf.mu.Unlock()

	return tf.initTask(ctx)
}

// DestroyTask destroys task dependencies so that it is safe for deletion
func (tf *Terraform) DestroyTask(ctx context.Context) {
	tf.mu.Lock()
	defer tf.mu.Unlock()

	tf.deregisterTemplate(ctx)
}

// SetBufferPeriod sets the buffer period for the task. Do not set this when
// task needs to immediately render a template and run.
func (tf *Terraform) SetBufferPeriod() {
	tf.mu.Lock()
	defer tf.mu.Unlock()

	taskName := tf.task.Name()
	if !tf.task.IsEnabled() {
		tf.logger.Trace("task disabled. skip setting buffer period", taskNameLogKey, taskName)
		return
	}

	if tf.template == nil {
		tf.logger.Warn("attempted to set buffer for task which does not have a template", taskNameLogKey, taskName)
		return
	}

	bp, ok := tf.task.BufferPeriod()
	if !ok {
		tf.logger.Trace("no buffer period for task", taskNameLogKey, taskName)
		return
	}

	tf.logger.Trace("set buffer period for task", taskNameLogKey, taskName, "buffer_period", bp)
	tf.watcher.SetBufferPeriod(bp.Min, bp.Max, tf.template.ID())
}

func (tf *Terraform) TemplateIDs() []string {
	tf.mu.RLock()
	defer tf.mu.RUnlock()

	if tf.template == nil {
		return nil
	}
	return []string{tf.template.ID()}
}

// RenderTemplate fetches data for the template. If the data is complete fetched,
// renders the template. Rendering a template for the first time may take several
// cycles to load all the dependencies asynchronously. Returns a boolean whether
// the template was rendered
func (tf *Terraform) RenderTemplate(ctx context.Context) (bool, error) {
	tf.mu.Lock()
	defer tf.mu.Unlock()
	taskName := tf.task.Name()

	if !tf.task.IsEnabled() {
		tf.logger.Trace("task disabled. skip rendering template", taskNameLogKey, taskName)
		return true, nil
	}

	tf.logger.Trace("checking dependency changes for task", taskNameLogKey, taskName)
	re, err := tf.renderTemplate(ctx)
	return (re.Complete && !re.NoChange), err
}

// InspectTask inspects for any differences pertaining to the task between
// the state of Consul and network infrastructure using the Terraform plan command
func (tf *Terraform) InspectTask(ctx context.Context) (InspectPlan, error) {
	tf.mu.Lock()
	defer tf.mu.Unlock()

	if !tf.task.IsEnabled() {
		tf.logger.Trace(
			"task disabled. skip inspecting", taskNameLogKey, tf.task.Name())
		return InspectPlan{
			Plan: "Task is disabled, inspection was skipped.",
		}, nil
	}

	plan, err := tf.inspectTask(ctx, true)
	tf.deregisterTemplate(ctx)
	return plan, err
}

// ApplyTask applies the task changes.
func (tf *Terraform) ApplyTask(ctx context.Context) error {
	tf.mu.Lock()
	defer tf.mu.Unlock()

	if !tf.task.IsEnabled() {
		tf.logger.Trace(
			"task disabled. skip applying", taskNameLogKey, tf.task.Name())
		return nil
	}

	return tf.applyTask(ctx)
}

// InspectPlan stores return the information about what
type InspectPlan struct {
	ChangesPresent bool   `json:"changes_present"`
	Plan           string `json:"plan"`
	URL            string `json:"url,omitempty"`
}

// UpdateTask updates the task on the driver. Makes any calls to re-init
// depending on the fields updated. If update task is requested with the inspect
// run option, then dry run the updates by returning the inspected plan for the
// expected updates but do not update the task
func (tf *Terraform) UpdateTask(ctx context.Context, patch PatchTask) (InspectPlan, error) {
	taskName := tf.task.Name()
	switch patch.RunOption {
	case "", RunOptionInspect, RunOptionNow:
		// valid options
	default:
		return InspectPlan{}, fmt.Errorf("Invalid run option '%s'. Please select a valid "+
			"option", patch.RunOption)
	}

	tf.mu.Lock()
	defer tf.mu.Unlock()

	originalEnabled := tf.task.IsEnabled()

	// for inspect, dry-run the task with the planned change and then make sure
	// to reset the task back to the way it was
	if patch.RunOption == RunOptionInspect && originalEnabled != patch.Enabled {
		defer func() {
			if originalEnabled {
				tf.task.Enable()
			} else {
				tf.task.Disable()
			}
		}()
	}

	reinit := false

	if originalEnabled != patch.Enabled {
		if patch.Enabled {
			tf.task.Enable()
			reinit = true
		} else {
			tf.task.Disable()
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
		if err := tf.initTask(ctx); err != nil {
			return InspectPlan{}, fmt.Errorf("Error updating task '%s'. Unable to init "+
				"task: %s", taskName, err)
		}

		for {
			result, err := tf.renderTemplate(ctx)
			if err != nil {
				return InspectPlan{}, fmt.Errorf("Error updating task '%s'. Unable to "+
					"render template for task: %s", taskName, err)
			}
			if (result.Complete && !result.NoChange) || (result.Complete && result.NoChange && tf.renderedOnce) {
				// Continue if the template has completed or the template had already
				// completed prior to enabling the task and there is no change.
				break
			}
		}
	}

	if patch.RunOption == RunOptionInspect {
		tf.logger.Trace("update task. inspect run option", taskNameLogKey, taskName)
		plan, err := tf.inspectTask(ctx, true)
		if err != nil {
			return InspectPlan{}, fmt.Errorf("Error updating task '%s'. Unable to inspect "+
				"task: %s", taskName, err)
		}
		return plan, nil
	}

	if patch.RunOption == RunOptionNow {
		tf.logger.Trace("update task. run now option", taskNameLogKey, taskName)
		return InspectPlan{}, tf.applyTask(ctx)
	}

	// allow the task to update naturally!
	return InspectPlan{}, nil
}

// init initializes the Terraform workspace if needed
func (tf *Terraform) init(ctx context.Context) error {
	taskName := tf.task.Name()

	if tf.inited {
		tf.logger.Trace("workspace for task already initialized, skipping for task", taskNameLogKey, taskName)
		return nil
	}

	tf.logger.Trace("initializing workspace", taskNameLogKey, taskName)
	if err := tf.client.Init(ctx); err != nil {
		return errors.Wrap(err, fmt.Sprintf("error tf-init for '%s'", taskName))
	}
	tf.inited = true
	return nil
}

// initTask initializes the task
func (tf *Terraform) initTask(ctx context.Context) error {
	input := tftmpl.RootModuleInputData{
		TerraformVersion: TerraformVersion,
		Backend:          tf.backend,
		Path:             tf.task.WorkingDir(),
		FilePerms:        filePerms,
	}

	// convert relative paths to absolute paths for local modules
	module := tf.task.module
	if strings.HasPrefix(module, "./") || strings.HasPrefix(module, "../") {
		wd, err := os.Getwd()
		if err != nil {
			tf.logger.Error("unable to retrieve current working directory to determine path to local module",
				"error", err)
			return err
		}
		module = filepath.Join(wd, tf.task.module)
		tf.task.module = module
	}

	if err := tf.task.configureRootModuleInput(&input); err != nil {
		return err
	}

	if err := tftmpl.InitRootModule(&input); err != nil {
		return err
	}

	if err := tf.initTaskTemplate(); err != nil {
		return err
	}

	// initTask() can be called more than once. It's very likely initializing a
	// task will require re-initializing terraform. Reset to false so terraform
	// will reinit
	tf.inited = false

	// initialize workspace
	taskName := tf.task.Name()
	if err := tf.init(ctx); err != nil {
		tf.logger.Error("error initializing workspace for task", taskNameLogKey, taskName)
		return err
	}

	// validate workspace
	if err := tf.validateTask(ctx); err != nil {
		return err
	}

	return nil
}

// deregisterTemplate attempts to deregister the hashicat template
func (tf *Terraform) deregisterTemplate(ctx context.Context) {
	tf.watcher.Deregister(tf.template)
}

// renderTemplate attempts to render the hashicat template
func (tf *Terraform) renderTemplate(ctx context.Context) (hcat.ResolveEvent, error) {
	taskName := tf.task.Name()

	// log the task name with each log
	tnlog := tf.logger.With(taskNameLogKey, taskName)
	result, err := tf.resolver.Run(tf.template, tf.watcher)
	if err != nil {
		tnlog.Error("error checking dependency changes for task", "error", err)

		return hcat.ResolveEvent{}, fmt.Errorf("error fetching template dependencies for task %s: %s",
			taskName, err)
	}

	// result.NoChange can occur when template rendering is forced even though
	// there may be no dependency changes rather than naturally triggered
	// e.g. when a task is re-enabled
	if result.Complete && result.NoChange && tf.renderedOnce {
		tnlog.Trace("no changes detected for task")
		return result, nil
	}

	if result.Complete && !result.NoChange {
		tnlog.Debug("change detected for task")

		rendered, err := tf.template.Render(result.Contents)
		if err != nil {
			tnlog.Error("rendering template for task", "error", err)

			return hcat.ResolveEvent{}, err
		}
		tnlog.Trace("template for task rendered", "rendered_template", rendered)
		tf.renderedOnce = true
	}

	return result, nil
}

// inspectTask inspects the task changes. Option to return inspection plan
// details rather than logging out
func (tf *Terraform) inspectTask(ctx context.Context, returnPlan bool) (InspectPlan, error) {
	taskName := tf.task.Name()

	var buf bytes.Buffer
	if returnPlan {
		tf.client.SetStdout(&buf)

		var tfLogger *log.Logger
		if tf.logClient {
			tfLogger = log.New(log.Writer(), "", log.Flags())
		} else {
			tfLogger = log.New(ioutil.Discard, "", 0)
		}
		defer tf.client.SetStdout(tfLogger.Writer())
	}

	tf.logger.Trace("plan", taskNameLogKey, taskName)
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
	taskName := tf.task.Name()

	tf.logger.Trace("apply", taskNameLogKey, taskName)
	if err := tf.client.Apply(ctx); err != nil {
		return errors.Wrap(err, fmt.Sprintf("error tf-apply for '%s'", taskName))
	}

	if tf.postApply != nil {
		tf.logger.Trace("post-apply out-of-band actions for task", taskNameLogKey, taskName)
		if err := tf.postApply.Do(ctx, nil); err != nil {
			return err
		}
	}

	return nil
}

// initTaskTemplate creates templates to be monitored and rendered.
func (tf *Terraform) initTaskTemplate() error {
	wd := tf.task.WorkingDir()
	tmplFullpath := filepath.Join(wd, tftmpl.TFVarsTmplFilename)
	tfvarsFilepath := filepath.Join(wd, tftmpl.TFVarsFilename)
	logger := tf.logger.With(taskNameLogKey, tf.task.Name())

	content, err := tf.fileReader(tmplFullpath)
	if err != nil {
		logger.Error("unable to read for task", "error", err)
		return err
	}

	renderer := hcat.NewFileRenderer(hcat.FileRendererInput{
		Path:  tfvarsFilepath,
		Perms: filePerms,
	})

	servicesMeta, err := getServicesMetaData(tf.logger, tf.task)
	if err != nil {
		return err
	}

	tmpl := hcat.NewTemplate(hcat.TemplateInput{
		Contents:     string(content),
		Renderer:     renderer,
		FuncMapMerge: tmplfunc.HCLMap(servicesMeta),
	})

	if tf.template != nil {
		if tf.template.ID() == tmpl.ID() {
			// if the new template ID is the same as an existing one (e.g.
			// during a task update), then the template content is the same.
			// Template content must be unique.
			// See: https://github.com/hashicorp/consul-terraform-sync/pull/167

			// Reset the buffer period if applicable to avoid possible waiting after
			// re-init to render latest content
			tf.watcher.BufferReset(tf.template)
			return nil
		}

		// cleanup old template from watcher
		tf.watcher.MarkForSweep(tf.template)
		tf.watcher.Sweep(tf.template)
	}

	tf.setNotifier(tmpl)

	logger.Debug("validating template")
	err = validateTemplate(tmpl, tf.watcher.Clients())
	if err != nil {
		logger.Error("error validating template", "error", err)
		return errors.Wrap(err, "unable to retrieve data from Consul")
	}
	logger.Debug("template validation complete")

	err = tf.watcher.Register(tf.template)
	if err != nil && err != hcat.ErrRegistry {
		logger.Error("unable to register template", "error", err)
		return err
	}

	return nil
}

// validateTemplate verifies that executing the fetch requests of
// a template's dependencies does not error.
func validateTemplate(t *hcat.Template, clients hcat.Looker) error {
	var err error
	recaller := func(dep dep.Dependency) (interface{}, bool) {
		data, _, fetchErr := dep.Fetch(clients)
		if fetchErr != nil {
			err = fetchErr
			return nil, false
		}
		return data, true
	}
	t.Execute(recaller)
	// Mark that template needs to be re-run
	t.Notify(nil)
	return err
}

// setNotifier sets a notifier on the template to ensure only the condition's
// monitored changes (and not the module input's changes) trigger the task.
func (tf *Terraform) setNotifier(tmpl templates.Template) error {
	tmplFuncTotal, err := tf.countTmplFunc()
	if err != nil {
		return err
	}

	switch tf.task.Condition().(type) {
	case *config.ServicesConditionConfig:
		tf.template = notifier.NewServices(tmpl, tmplFuncTotal)
	case *config.CatalogServicesConditionConfig:
		tf.template = notifier.NewCatalogServicesRegistration(tmpl, tmplFuncTotal)
	case *config.ConsulKVConditionConfig:
		tf.template = notifier.NewConsulKV(tmpl, tmplFuncTotal)
	case *config.ScheduleConditionConfig:
		tf.template = notifier.NewSuppressNotification(tmpl, tmplFuncTotal)
	default:
		// services list
		tf.template = notifier.NewServices(tmpl, tmplFuncTotal)
	}
	return nil
}

// countTmplFunc counts the number of template functions (tmplfunc) that are
// added to the template file. Counts the tmplfunc needed by the task's
// services field, condition block, and module_input blocks.
func (tf *Terraform) countTmplFunc() (int, error) {
	// Count tmplfuncs for the service variable separately. Currently services
	// can only be configured in one of services field, condition "services",
	// and module_input "services". Enforced by config validation
	serviceCount := len(tf.task.Services())
	nonServiceCount := 0

	switch cond := tf.task.Condition().(type) {
	case *config.CatalogServicesConditionConfig:
		nonServiceCount++
	case *config.ServicesConditionConfig:
		if cond.Regexp != nil {
			serviceCount = 1
		} else {
			serviceCount = len(cond.Names)
		}
	case *config.ConsulKVConditionConfig:
		nonServiceCount++
	default:
		// no-op: condition block currently not required since services list
		// can be used alternatively. enforced by config validation
	}

	for _, moduleInput := range tf.task.ModuleInputs() {
		switch input := moduleInput.(type) {
		case *config.ServicesModuleInputConfig:
			// relies on config validation to restrict to one ServicesModuleInput
			if input.Regexp != nil {
				serviceCount = 1
			} else {
				serviceCount = len(input.Names)
			}
		case *config.ConsulKVModuleInputConfig:
			nonServiceCount++
		default:
			return 0, fmt.Errorf("task %q has unsupported type of module_input "+
				"block configuration %T", tf.task.name, input)
		}
	}

	return serviceCount + nonServiceCount, nil
}

func (tf *Terraform) validateTask(ctx context.Context) error {
	err := tf.client.Validate(ctx)
	if err != nil {
		return err
	}
	return nil
}

// getTerraformHandlers returns the first handler in a chain of handlers
// for a Terraform driver.
//
// Returned handler may be nil even if returned err is nil. This happens when
// no providers have a handler.
func getTerraformHandlers(taskName string, providers TerraformProviderBlocks) (handler.Handler, error) {
	counter := 0
	var next handler.Handler
	logger := logging.Global().Named(logSystemName).Named(terraformSubsystemName)
	for _, p := range providers {
		h, err := handler.TerraformProviderHandler(p.Name(), p.ProviderBlock().RawConfig())
		if err != nil {
			logger.Error("error, could not initialize handler for provider",
				"provider", p.Name, "error", err)
			return nil, err
		}
		if h != nil {
			counter++
			logger.Info("retrieved handler for provider", "provider", p.Name())
			h.SetNext(next)
			next = h
		}
	}
	logger.Info(fmt.Sprintf("retrieved %d Terraform handlers for task", counter), taskNameLogKey, taskName)
	return next, nil
}

// getServicesMetaData helps retrieve metadata which can come from a number of
// configuration sources: task.services' related service block, condition
// "service" block, module_input "service" block.
//
// Currently it is only possible for a task to be configured with one of these
// configuration sources, hence a task only has one metadata source. This is
// validated at the configuration-level. This method relies on this assumption.
func getServicesMetaData(logger logging.Logger, task *Task) (*tmplfunc.ServicesMeta, error) {
	servicesMeta := &tmplfunc.ServicesMeta{}

	// Introduced in 0.5. Metadata comes from condition "services"
	servicesCond, ok := task.Condition().(*config.ServicesConditionConfig)
	if ok {
		err := servicesMeta.SetMeta(servicesCond.CTSUserDefinedMeta)
		if err != nil {
			logger.Error("unable to to set metadata from services condition",
				taskNameLogKey, task.Name(), "error", err)
			return nil, err
		}

		return servicesMeta, nil
	}

	// Introduced in 0.5. Metadata comes from module_input "services"
	for _, moduleInput := range task.ModuleInputs() {
		servicesInput, ok := moduleInput.(*config.ServicesModuleInputConfig)
		if ok {
			err := servicesMeta.SetMeta(servicesInput.CTSUserDefinedMeta)
			if err != nil {
				logger.Error("unable to to set metadata from services module_input",
					taskNameLogKey, task.Name(), "error", err)
				return nil, err
			}

			// currently due to config validation, only on services module_input
			// can be configured
			return servicesMeta, nil
		}
	}

	// Deprecated in 0.5. Metadata comes from service block
	metaMap := make(map[string]map[string]string)
	services := task.Services()
	for _, s := range services {
		metaMap[s.Name] = s.UserDefinedMeta
	}

	if len(metaMap) == 0 {
		return servicesMeta, nil
	}

	if err := servicesMeta.SetMetaMap(metaMap); err != nil {
		logger.Error("unable to to set metadata from task service blocks",
			taskNameLogKey, task.Name(), "error", err)
		return nil, err
	}

	return servicesMeta, nil
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
