package driver

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/handler"
	"github.com/hashicorp/consul-terraform-sync/logging"
	mocks "github.com/hashicorp/consul-terraform-sync/mocks/client"
	mocksNoti "github.com/hashicorp/consul-terraform-sync/mocks/notifier"
	mocksTmpl "github.com/hashicorp/consul-terraform-sync/mocks/templates"
	"github.com/hashicorp/consul-terraform-sync/templates"
	"github.com/hashicorp/consul-terraform-sync/templates/hcltmpl"
	"github.com/hashicorp/consul-terraform-sync/templates/tftmpl/notifier"
	"github.com/hashicorp/consul-terraform-sync/templates/tftmpl/tmplfunc"
	"github.com/hashicorp/consul-terraform-sync/testutils"
	"github.com/hashicorp/go-uuid"
	goVersion "github.com/hashicorp/go-version"
	"github.com/hashicorp/hcat"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestRenderTemplate(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name           string
		expectError    bool
		expectRendered bool
		renderErr      error
		runErr         error
		runResult      hcat.ResolveEvent
	}{
		{
			"happy path: changes",
			false,
			true,
			nil,
			nil,
			hcat.ResolveEvent{Complete: true},
		},
		{
			"happy path: no changes",
			false,
			false,
			nil,
			nil,
			hcat.ResolveEvent{NoChange: true},
		},
		{
			"data not completely fetched",
			false,
			false,
			nil,
			nil,
			hcat.ResolveEvent{Complete: false},
		},
		{
			"error on resolver.Run()",
			true,
			false,
			nil,
			errors.New("error on resolver.Run()"),
			hcat.ResolveEvent{},
		},
		{
			"error on template.Render()",
			true,
			false,
			errors.New("error on template.Render()"),
			nil,
			hcat.ResolveEvent{Complete: true},
		},
	}
	ctx := context.Background()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {

			r := new(mocksTmpl.Resolver)
			r.On("Run", mock.Anything, mock.Anything).
				Return(tc.runResult, tc.runErr).Once()

			tmpl := new(mocksTmpl.Template)
			tmpl.On("Render", mock.Anything).Return(hcat.RenderResult{}, tc.renderErr).Once()

			tf := &Terraform{
				task:     &Task{name: "RenderTemplateTest", enabled: true, logger: logging.NewNullLogger()},
				resolver: r,
				template: tmpl,
				watcher:  new(mocksTmpl.Watcher),
				logger:   logging.NewNullLogger(),
			}

			actual, err := tf.RenderTemplate(ctx)
			assert.Equal(t, tc.expectRendered, actual)

			if tc.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.name)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTerraform_Version(t *testing.T) {
	var err error
	TerraformVersion, err = goVersion.NewVersion("1.2")
	require.NoError(t, err)
	var tf Terraform
	s := tf.Version()
	assert.Equal(t, "1.2.0", s)
}

func TestInspectTask(t *testing.T) {
	// Task Disabled
	t.Run("task disabled", func(t *testing.T) {
		tf := Terraform{
			task:   &Task{},
			logger: logging.NewNullLogger(),
		}
		ctx := context.Background()
		plan, err := tf.InspectTask(ctx)
		assert.NoError(t, err)
		require.Contains(t, plan.Plan, "Task is disabled, inspection was skipped.")
	})

	// Task Enabled
	t.Run("task enabled", func(t *testing.T) {
		// Task Enabled
		var w mocksTmpl.Watcher
		var c mocks.Client
		tf := Terraform{
			task: &Task{
				enabled: true,
			},
			logger:  logging.NewNullLogger(),
			watcher: &w,
			client:  &c,
		}

		ctx := context.Background()
		w.On("Deregister", mock.Anything).Return()
		c.On("Plan", ctx).Return(true, nil).Once()
		c.On("SetStdout", mock.Anything).Twice()

		ctx = context.Background()
		plan, err := tf.InspectTask(ctx)
		assert.NoError(t, err)
		require.Equal(t, "", plan.Plan)
	})
}

func TestApplyTask(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		expectError bool
		applyReturn error
		postApply   handler.Handler
	}{
		{
			"happy path - no post-apply handler",
			false,
			nil,
			nil,
		},
		{
			"happy path - post-apply handler",
			false,
			nil,
			testHandler(false),
		},
		{
			"error on apply",
			true,
			errors.New("apply error"),
			nil,
		},
		{
			"error on post-apply handler",
			true,
			nil,
			testHandler(true),
		},
	}
	ctx := context.Background()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := new(mocks.Client)
			c.On("Apply", ctx).Return(tc.applyReturn).Once()

			tf := &Terraform{
				task:      &Task{name: "ApplyTaskTest", enabled: true, logger: logging.NewNullLogger()},
				client:    c,
				postApply: tc.postApply,
				logger:    logging.NewNullLogger(),
			}

			err := tf.ApplyTask(ctx)
			if !tc.expectError {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestUpdateTask(t *testing.T) {
	t.Parallel()

	// this set of cases tests that the mocks are appropriately called
	mockCases := []struct {
		name        string
		dirName     string
		orig        PatchTask
		patch       PatchTask
		callInit    bool
		callRender  bool
		callInspect bool
		callApply   bool
	}{
		{
			"disabling a task. no run ops",
			"disable-no-run-ops",
			PatchTask{Enabled: true},
			PatchTask{Enabled: false},
			false,
			false,
			false,
			false,
		},
		{
			"disabling a task. run=now",
			"disable-run-now",
			PatchTask{Enabled: true},
			PatchTask{Enabled: false, RunOption: RunOptionNow},
			false,
			false,
			false,
			false,
		},
		{
			"disabling a task. run=inspect",
			"disable-run-inspect",
			PatchTask{Enabled: true},
			PatchTask{Enabled: false, RunOption: RunOptionInspect},
			false,
			false,
			false,
			false,
		},
		{
			"enabling a task. no run ops",
			"enable-no-run-ops",
			PatchTask{Enabled: false},
			PatchTask{Enabled: true},
			true,
			true,
			false,
			false,
		},
		{
			"enabling a task. run=now",
			"enable-run-now",
			PatchTask{Enabled: false},
			PatchTask{Enabled: true, RunOption: RunOptionNow},
			true,
			true,
			false,
			true,
		},
		{
			"enabling a task. run=inspect",
			"enable-run-inspect",
			PatchTask{Enabled: false},
			PatchTask{Enabled: true, RunOption: RunOptionInspect},
			true,
			true,
			true,
			false,
		},
	}

	ctx := context.Background()
	for _, tc := range mockCases {
		t.Run(tc.name, func(t *testing.T) {
			deleteTemp := testutils.MakeTempDir(t, tc.dirName)
			defer deleteTemp()

			r := new(mocksTmpl.Resolver)
			if tc.callRender {
				r.On("Run", mock.Anything, mock.Anything).
					Return(hcat.ResolveEvent{Complete: true, NoChange: false}, nil).Once()
			}

			c := new(mocks.Client)
			if tc.callInspect {
				c.On("Plan", ctx).Return(true, nil).Once()
				c.On("SetStdout", mock.Anything).Twice()
			}
			if tc.callApply {
				c.On("Apply", ctx).Return(nil).Once()
			}

			w := new(mocksTmpl.Watcher)
			w.On("Register", mock.Anything).Return(nil).Once()
			w.On("Clients").Return(nil).Once()
			tf := &Terraform{
				task: &Task{name: "test_task", enabled: tc.orig.Enabled, workingDir: tc.dirName,
					logger: logging.NewNullLogger()},
				client:   c,
				resolver: r,
				watcher:  w,
				logger:   logging.NewNullLogger(),
			}
			if tc.callInspect {
				w.On("Deregister", mock.Anything).Return()
			}

			if tc.callInit {
				c.On("Init", ctx).Return(nil).Once()
				c.On("Validate", ctx).Return(nil).Once()
				tf.fileReader = func(string) ([]byte, error) { return []byte{}, nil }
			}

			_, err := tf.UpdateTask(ctx, tc.patch)
			require.NoError(t, err)

			// check that mocks were called as expected
			r.AssertExpectations(t)
			c.AssertExpectations(t)
		})
	}

	// this set of cases tests error handling
	errorCases := []struct {
		name          string
		dirName       string
		patch         PatchTask
		fileReaderErr error
		resolverErr   error
		planErr       error
		applyErr      error
		expectErr     bool
	}{
		{
			"invalid run option",
			"invalid-run-ops-err",
			PatchTask{Enabled: true, RunOption: "invalid run option"},
			nil,
			nil,
			nil,
			nil,
			true,
		},
		{
			"init task template error: file reader err",
			"init-task-err",
			PatchTask{Enabled: true},
			errors.New("filereader err"),
			nil,
			nil,
			nil,
			true,
		},
		{
			"render template error: resolver err",
			"render-templ-err",
			PatchTask{Enabled: true},
			nil,
			errors.New("resolver err"),
			nil,
			nil,
			true,
		},
		{
			"inspect task error: plan err",
			"inspect-task-err",
			PatchTask{Enabled: true, RunOption: RunOptionInspect},
			nil,
			nil,
			errors.New("plan err"),
			nil,
			true,
		},
		{
			"apply task error",
			"apply-task-err",
			PatchTask{Enabled: true, RunOption: RunOptionNow},
			nil,
			nil,
			nil,
			errors.New("apply err"),
			true,
		},
	}
	for _, tc := range errorCases {
		t.Run(tc.name, func(t *testing.T) {
			deleteTemp := testutils.MakeTempDir(t, tc.dirName)
			defer deleteTemp()

			r := new(mocksTmpl.Resolver)
			r.On("Run", mock.Anything, mock.Anything).
				Return(hcat.ResolveEvent{Complete: true}, tc.resolverErr).Once()

			c := new(mocks.Client)
			c.On("Init", ctx).Return(nil).Once()
			c.On("Validate", ctx).Return(nil).Once()
			c.On("Plan", ctx).Return(true, tc.planErr).Once()
			c.On("SetStdout", mock.Anything).Twice()
			c.On("Apply", ctx).Return(tc.applyErr).Once()

			w := new(mocksTmpl.Watcher)
			w.On("Register", mock.Anything).Return(nil).Once()
			w.On("Clients").Return(nil).Once()

			tf := &Terraform{
				task: &Task{name: "test_task", enabled: false, workingDir: tc.dirName,
					logger: logging.NewNullLogger()},
				client:   c,
				resolver: r,
				watcher:  w,
				fileReader: func(string) ([]byte, error) {
					return []byte{}, tc.fileReaderErr
				},
				logger: logging.NewNullLogger(),
			}

			_, err := tf.UpdateTask(ctx, tc.patch)
			require.Error(t, err)
		})
	}
}

func TestUpdateTask_Inspect(t *testing.T) {
	t.Parallel()
	// test cases confirm that updating a task with ?run=inspect runs task in
	// dry-run and does not change the task

	cases := []struct {
		name  string
		task  *Task
		patch PatchTask
	}{
		{
			"disable an enabled task",
			&Task{
				enabled: true,
			},
			PatchTask{
				RunOption: RunOptionInspect,
				Enabled:   false,
			},
		},
		{
			"enable a disabled task",
			&Task{
				enabled: false,
			},
			PatchTask{
				RunOption: RunOptionInspect,
				Enabled:   true,
			},
		},
	}

	ctx := context.Background()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			workingDir := strings.ReplaceAll(tc.name, " ", "_")
			deleteTemp := testutils.MakeTempDir(t, workingDir)
			defer deleteTemp()

			// set up mocks
			r := new(mocksTmpl.Resolver)
			r.On("Run", mock.Anything, mock.Anything).
				Return(hcat.ResolveEvent{Complete: true, NoChange: false}, nil)

			c := new(mocks.Client)
			c.On("Init", ctx).Return(nil).Once()
			c.On("Validate", ctx).Return(nil).Once()
			c.On("Plan", ctx).Return(true, nil)
			c.On("SetStdout", mock.Anything)

			w := new(mocksTmpl.Watcher)
			w.On("Register", mock.Anything).Return(nil)
			w.On("Clients").Return(nil).Once()
			w.On("Deregister", mock.Anything).Return()

			tf := &Terraform{
				task:     tc.task,
				client:   c,
				resolver: r,
				watcher:  w,
				logger:   logging.NewNullLogger(),
				fileReader: func(string) ([]byte, error) {
					return []byte{}, nil
				},
			}

			// set some fields on task
			tc.name = "task_a"
			tc.task.logger = logging.NewNullLogger()
			tc.task.workingDir = workingDir

			copyTask := &Task{
				name:       tc.task.Name(),
				enabled:    tc.task.IsEnabled(),
				logger:     tc.task.logger,
				workingDir: tc.task.workingDir,
			}

			_, err := tf.UpdateTask(ctx, tc.patch)
			require.NoError(t, err)

			// confirm that task is unchanged
			assert.Equal(t, copyTask, tf.Task())
		})
	}
}

func TestSetBufferPeriod(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		setTmpl bool
		task    *Task
	}{
		{
			"happy path",
			true,
			&Task{
				enabled: true,
				name:    "happy_task",
				bufferPeriod: &BufferPeriod{
					Min: 2 * time.Second,
					Max: 5 * time.Second,
				},
				logger: logging.NewNullLogger(),
			},
		},
		{
			"disabled task",
			false,
			&Task{
				enabled: false,
				name:    "disabled_task",
				logger:  logging.NewNullLogger(),
			},
		},
		{
			"no buffer period",
			true,
			&Task{
				enabled: true,
				name:    "no_buf_period_task",
				logger:  logging.NewNullLogger(),
			},
		},
		{
			"no template",
			false,
			&Task{
				enabled: true,
				name:    "task_a",
				logger:  logging.NewNullLogger(),
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tf := &Terraform{
				task:   tc.task,
				logger: logging.NewNullLogger(),
			}

			if tc.setTmpl {
				tmpl := new(mocksTmpl.Template)
				tmpl.On("ID").Return("abcd").Once()
				tf.template = tmpl
			}

			if _, ok := tc.task.BufferPeriod(); ok {
				w := new(mocksTmpl.Watcher)
				w.On("SetBufferPeriod", mock.Anything, mock.Anything, mock.Anything)
				tf.watcher = w
			}

			tf.SetBufferPeriod()
			// no errors to check. just testing coverage and no failures
		})
	}
}

func TestInitTaskTemplates(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		expectError bool
		fileReader  func(string) ([]byte, error)
	}{
		{
			"error on reading file",
			true,
			func(string) ([]byte, error) {
				return []byte{}, errors.New("error on newTaskTemplates()")
			},
		},
		{
			"happy path",
			false,
			func(string) ([]byte, error) { return []byte{}, nil },
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := new(mocksTmpl.Watcher)
			w.On("Register", mock.Anything).Return(nil).Once()
			w.On("Clients").Return(nil).Once()
			tf := &Terraform{
				fileReader: tc.fileReader,
				task:       &Task{name: "test", enabled: true},
				watcher:    w,
				logger:     logging.NewNullLogger(),
			}
			err := tf.initTaskTemplate()
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetTerraformHandlers(t *testing.T) {
	cases := []struct {
		name        string
		expectError bool
		nilHandler  bool
		providers   TerraformProviderBlocks
	}{
		{
			"no provider",
			false,
			true,
			TerraformProviderBlocks{},
		},
		{
			"provider without handler (no error)",
			true,
			true,
			NewTerraformProviderBlocks([]hcltmpl.NamedBlock{
				hcltmpl.NewNamedBlock(map[string]interface{}{
					handler.TerraformProviderFake: map[string]interface{}{
						"required-config": "missing",
					},
				})},
			),
		},
		{
			"provider without handler (no error)",
			false,
			true,
			NewTerraformProviderBlocks([]hcltmpl.NamedBlock{
				hcltmpl.NewNamedBlock(map[string]interface{}{
					"provider-no-handler": map[string]interface{}{},
				})},
			),
		},
		{
			"happy path - provider with handler",
			false,
			false,
			NewTerraformProviderBlocks([]hcltmpl.NamedBlock{
				hcltmpl.NewNamedBlock(map[string]interface{}{
					handler.TerraformProviderFake: map[string]interface{}{
						"name": "happy-path",
					},
				})},
			),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h, err := getTerraformHandlers(tc.name, tc.providers)
			if tc.expectError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			if tc.nilHandler {
				assert.Nil(t, h)
				return
			}
			assert.NotNil(t, h)
		})
	}
}

func TestDisabledTask(t *testing.T) {
	t.Run("disabled-tasks", func(t *testing.T) {
		// tests that disabled tasks don't require mocking any calls other
		// than initialization and validation and that the calls do not
		// throw any errors

		ctx := context.Background()
		c := new(mocks.Client)
		c.On("Init", ctx).Return(nil).Once()
		c.On("Validate", ctx).Return(nil).Once()

		w := new(mocksTmpl.Watcher)
		w.On("Register", mock.Anything).Return(nil).Once()
		w.On("Clients").Return(nil).Once()

		dirName := "disabled-task-test"
		deleteTemp := testutils.MakeTempDir(t, dirName)
		defer deleteTemp()

		tf := &Terraform{
			task: &Task{name: "disabled_task", enabled: false,
				workingDir: dirName, logger: logging.NewNullLogger()},
			client:     c,
			fileReader: func(string) ([]byte, error) { return []byte{}, nil },
			watcher:    w,
			logger:     logging.NewNullLogger(),
		}

		err := tf.InitTask(ctx)
		assert.NoError(t, err)

		tf.SetBufferPeriod()

		actual, err := tf.RenderTemplate(ctx)
		assert.NoError(t, err)
		assert.True(t, actual)

		plan, err := tf.InspectTask(ctx)
		assert.NoError(t, err)
		assert.Equal(t, InspectPlan{
			Plan: "Task is disabled, inspection was skipped.",
		}, plan)

		err = tf.ApplyTask(ctx)
		assert.NoError(t, err)
	})
}

func TestInitTask(t *testing.T) {
	t.Parallel()

	mockCases := []struct {
		name        string
		expectError bool
		initErr     error
		validateErr error
		module      string
	}{
		{
			name: "happy path",
		},
		{
			name:        "init workspace error",
			expectError: true,
			initErr:     errors.New("error on init()"),
		},
		{
			name:        "validate error",
			expectError: true,
			validateErr: errors.New("error on validate()"),
		},
		{
			name:        "relative path module",
			expectError: true,
			validateErr: errors.New("error on validate()"),
			module:      "./",
		},
		{
			name:        "relative path module one back",
			expectError: true,
			validateErr: errors.New("error on validate()"),
			module:      "../",
		},
	}

	ctx := context.Background()
	for _, tc := range mockCases {
		t.Run(tc.name, func(t *testing.T) {
			dirName := "init-task-test"
			deleteTemp := testutils.MakeTempDir(t, dirName)
			defer deleteTemp()

			c := new(mocks.Client)
			c.On("Init", ctx).Return(tc.initErr).Once()
			c.On("Validate", ctx).Return(tc.validateErr)

			tmpl := new(mocksTmpl.Template)
			tmpl.On("ID").Return(uuid.GenerateUUID())

			w := new(mocksTmpl.Watcher)
			w.On("Clients").Return(nil).Once()
			w.On("MarkForSweep", tmpl).Return().Once()
			w.On("Sweep", tmpl).Return().Once()
			w.On("Register", mock.Anything).Return(nil).Once()

			tf := &Terraform{
				task:       &Task{name: "InitTaskTest", enabled: true, workingDir: dirName, logger: logging.NewNullLogger(), module: tc.module},
				client:     c,
				fileReader: func(string) ([]byte, error) { return []byte{}, nil },
				watcher:    w,
				logger:     logging.NewNullLogger(),
				template:   tmpl,
			}

			err := tf.initTask(ctx)
			if !tc.expectError {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}

			assert.NotEqual(t, tf.task.module, "./")
			assert.NotEqual(t, tf.task.module, "../")
		})
	}
}

func TestTerraform_DestroyTask(t *testing.T) {
	var w mocksTmpl.Watcher
	tf := Terraform{
		watcher: &w,
	}
	ctx := context.Background()

	w.AssertExpectations(t)
	w.On("Deregister", mock.Anything).Return().Once()
	tf.DestroyTask(ctx)
}

func TestTerraform_TemplateIDs(t *testing.T) {
	var tmpl mocksTmpl.Template
	tf := Terraform{
		template: &tmpl,
	}

	// Return string ID
	t.Run("return string", func(t *testing.T) {
		// Return a string
		ts := "as12!@#$%^&*()"
		tmpl.On("ID").Return(ts).Once()
		s := tf.TemplateIDs()
		require.Equal(t, s, []string{ts})
	})

	// Template is nil
	t.Run("nil template", func(t *testing.T) {
		tf.template = nil
		s := tf.TemplateIDs()
		require.Nil(t, s)
	})
}

func TestTerraform_OverrideNotifier(t *testing.T) {
	var o mocksNoti.Overrider
	tf := Terraform{
		overrider: &o,
	}

	// Call override method
	t.Run("call override method", func(t *testing.T) {
		o.On("Override").Once()
		tf.OverrideNotifier()
	})

	// Nil overrider
	t.Run("nil overrider", func(t *testing.T) {
		tf.overrider = nil
		tf.OverrideNotifier()
	})
}

func TestTerraform_countTmplFunc(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		expected int
		task     *Task
	}{
		{
			"services field",
			3,
			&Task{
				services: []Service{{Name: "api"}, {Name: "db"}, {Name: "web"}},
			},
		},
		{
			"condition: schedule",
			0, // on its own, schedule cond has no tmpl funcs
			&Task{
				condition: &config.ScheduleConditionConfig{},
			},
		},
		{
			"condition: catalog-services",
			1,
			&Task{
				condition: &config.CatalogServicesConditionConfig{},
			},
		},
		{
			"condition: consul-kv",
			1,
			&Task{
				condition: &config.ConsulKVConditionConfig{},
			},
		},
		{
			"condition: services-regex",
			1,
			&Task{
				condition: &config.ServicesConditionConfig{
					ServicesMonitorConfig: config.ServicesMonitorConfig{
						Regexp: config.String(".*"),
					},
				},
			},
		},
		{
			"condition: services-names",
			3,
			&Task{
				condition: &config.ServicesConditionConfig{
					ServicesMonitorConfig: config.ServicesMonitorConfig{
						Names: []string{"api", "db", "web"},
					},
				},
			},
		},
		{
			"module_input: consul-kv",
			1,
			&Task{
				moduleInputs: config.ModuleInputConfigs{
					&config.ConsulKVModuleInputConfig{},
				},
			},
		},
		{
			"module_input: services-regex",
			1,
			&Task{
				moduleInputs: config.ModuleInputConfigs{
					&config.ServicesModuleInputConfig{
						ServicesMonitorConfig: config.ServicesMonitorConfig{
							Regexp: config.String(".*"),
						},
					},
				},
			},
		},
		{
			"module_input: services-names",
			3,
			&Task{
				moduleInputs: config.ModuleInputConfigs{
					&config.ServicesModuleInputConfig{
						ServicesMonitorConfig: config.ServicesMonitorConfig{
							Names: []string{"api", "db", "web"},
						},
					},
				},
			},
		},
		{
			"combination",
			3,
			&Task{
				condition: &config.CatalogServicesConditionConfig{},
				moduleInputs: config.ModuleInputConfigs{
					&config.ConsulKVModuleInputConfig{},
					&config.ServicesModuleInputConfig{
						ServicesMonitorConfig: config.ServicesMonitorConfig{
							Regexp: config.String(".*"),
						},
					},
				},
			},
		},
		{
			"combination w services",
			4,
			&Task{
				services:  []Service{{Name: "api"}, {Name: "db"}, {Name: "web"}},
				condition: &config.ScheduleConditionConfig{},
				moduleInputs: config.ModuleInputConfigs{
					&config.ConsulKVModuleInputConfig{},
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tf := &Terraform{
				task: tc.task,
			}
			actual, err := tf.countTmplFunc()
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestTerraform_setNotifier(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name         string
		task         *Task
		expectedType templates.Template
	}{
		{
			"condition: none, default to services field",
			&Task{},
			&notifier.Services{},
		},
		{
			"condition: schedule",
			&Task{
				condition: &config.ScheduleConditionConfig{},
			},
			&notifier.SuppressNotification{},
		},
		{
			"condition: catalog-services",
			&Task{
				condition: &config.CatalogServicesConditionConfig{},
			},
			&notifier.CatalogServicesRegistration{},
		},
		{
			"condition: consul-kv",
			&Task{
				condition: &config.ConsulKVConditionConfig{},
			},
			&notifier.ConsulKV{},
		},
		{
			"condition: services",
			&Task{
				condition: &config.ServicesConditionConfig{},
			},
			&notifier.Services{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tf := &Terraform{
				task: tc.task,
			}
			err := tf.setNotifier(&hcat.Template{})
			assert.NoError(t, err)
			assert.IsType(t, tc.expectedType, tf.template)
		})
	}
}

func TestGetServicesMetaData(t *testing.T) {
	meta := map[string]string{
		"my_key": "my_value",
	}

	cases := []struct {
		name    string
		task    *Task
		expFunc func() *tmplfunc.ServicesMeta
	}{
		{
			"meta-data configured in condition",
			&Task{
				condition: &config.ServicesConditionConfig{
					ServicesMonitorConfig: config.ServicesMonitorConfig{
						CTSUserDefinedMeta: meta,
					},
				},
			},
			func() *tmplfunc.ServicesMeta {
				sm := &tmplfunc.ServicesMeta{}
				sm.SetMeta(meta)
				return sm
			},
		},
		{
			"meta-data configured in module_input",
			&Task{
				moduleInputs: config.ModuleInputConfigs{
					&config.ServicesModuleInputConfig{
						ServicesMonitorConfig: config.ServicesMonitorConfig{
							CTSUserDefinedMeta: meta,
						},
					},
				},
			},
			func() *tmplfunc.ServicesMeta {
				sm := &tmplfunc.ServicesMeta{}
				sm.SetMeta(meta)
				return sm
			},
		},
		{
			"meta-data configured in service block",
			&Task{
				services: []Service{
					{
						Name:            "api",
						UserDefinedMeta: meta,
					},
				},
			},
			func() *tmplfunc.ServicesMeta {
				sm := &tmplfunc.ServicesMeta{}
				metaMap := map[string]map[string]string{"api": meta}
				sm.SetMetaMap(metaMap)
				return sm
			},
		},
		{
			"no meta-data",
			&Task{
				condition: &config.ConsulKVConditionConfig{},
				moduleInputs: config.ModuleInputConfigs{
					&config.ConsulKVModuleInputConfig{},
				},
			},
			func() *tmplfunc.ServicesMeta {
				return &tmplfunc.ServicesMeta{}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := getServicesMetaData(logging.NewNullLogger(), tc.task)
			assert.NoError(t, err)
			assert.Equal(t, tc.expFunc(), actual)
		})
	}
}

// testHandler returns a fake handler that can return an error or not on Do()
func testHandler(err bool) handler.Handler {
	config := map[string]interface{}{
		"name": "1",
		"err":  err,
	}

	h, _ := handler.NewFake(config)
	return h
}
