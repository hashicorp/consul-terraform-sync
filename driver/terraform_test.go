package driver

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/consul-terraform-sync/handler"
	mocks "github.com/hashicorp/consul-terraform-sync/mocks/client"
	mocksTmpl "github.com/hashicorp/consul-terraform-sync/mocks/templates"
	"github.com/hashicorp/consul-terraform-sync/templates/hcltmpl"
	"github.com/hashicorp/consul-terraform-sync/testutils"
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
				mu:       &sync.RWMutex{},
				task:     &Task{name: "RenderTemplateTest", enabled: true},
				resolver: r,
				template: tmpl,
				watcher:  new(mocksTmpl.Watcher),
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
				mu:        &sync.RWMutex{},
				task:      &Task{name: "ApplyTaskTest", enabled: true},
				client:    c,
				postApply: tc.postApply,
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
			delete := testutils.MakeTempDir(t, tc.dirName)
			defer delete()

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
			tf := &Terraform{
				mu:       &sync.RWMutex{},
				task:     &Task{name: "test_task", enabled: tc.orig.Enabled, workingDir: tc.dirName},
				client:   c,
				resolver: r,
				watcher:  w,
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
			delete := testutils.MakeTempDir(t, tc.dirName)
			defer delete()

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

			tf := &Terraform{
				mu:       &sync.RWMutex{},
				task:     &Task{name: "test_task", enabled: false, workingDir: tc.dirName},
				client:   c,
				resolver: r,
				watcher:  w,
				fileReader: func(string) ([]byte, error) {
					return []byte{}, tc.fileReaderErr
				},
			}

			_, err := tf.UpdateTask(ctx, tc.patch)
			require.Error(t, err)
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
			},
		},
		{
			"disabled task",
			false,
			&Task{
				enabled: false,
				name:    "disabled_task",
			},
		},
		{
			"no buffer period",
			true,
			&Task{
				enabled: true,
				name:    "no_buf_period_task",
			},
		},
		{
			"no template",
			false,
			&Task{
				enabled: true,
				name:    "task_a",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tf := &Terraform{
				mu:   &sync.RWMutex{},
				task: tc.task,
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
			tf := &Terraform{
				fileReader: tc.fileReader,
				task:       &Task{name: "test", enabled: true},
				watcher:    w,
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
		// tests that disabled tasks don't require mocking any calls and does
		// not throw any errors

		tf := &Terraform{
			mu:      &sync.RWMutex{},
			task:    &Task{name: "disabled_task", enabled: false},
			watcher: new(mocksTmpl.Watcher),
		}

		ctx := context.Background()

		err := tf.InitTask(ctx)
		assert.NoError(t, err)

		tf.SetBufferPeriod()

		actual, err := tf.RenderTemplate(ctx)
		assert.NoError(t, err)
		assert.True(t, actual)

		err = tf.InspectTask(ctx)
		assert.NoError(t, err)

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
	}{
		{
			"happy path",
			false,
			nil,
			nil,
		},
		{
			"init workspace error",
			true,
			errors.New("error on init()"),
			nil,
		},
		{
			"validate error",
			true,
			nil,
			errors.New("error on validate()"),
		},
	}

	ctx := context.Background()
	for _, tc := range mockCases {
		t.Run(tc.name, func(t *testing.T) {
			dirName := "init-task-test"
			delete := testutils.MakeTempDir(t, dirName)
			defer delete()

			c := new(mocks.Client)
			c.On("Init", ctx).Return(tc.initErr).Once()
			c.On("Validate", ctx).Return(tc.validateErr)

			w := new(mocksTmpl.Watcher)
			w.On("Register", mock.Anything).Return(nil).Once()

			tf := &Terraform{
				mu:         &sync.RWMutex{},
				task:       &Task{name: "InitTaskTest", enabled: true, workingDir: dirName},
				client:     c,
				fileReader: func(string) ([]byte, error) { return []byte{}, nil },
				watcher:    w,
			}

			err := tf.initTask(ctx)
			if !tc.expectError {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
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
