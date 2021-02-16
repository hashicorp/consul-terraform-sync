package driver

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/hashicorp/consul-terraform-sync/handler"
	mocks "github.com/hashicorp/consul-terraform-sync/mocks/client"
	mocksTmpl "github.com/hashicorp/consul-terraform-sync/mocks/templates"
	"github.com/hashicorp/consul-terraform-sync/templates/hcltmpl"
	"github.com/hashicorp/hcat"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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
			"happy path",
			false,
			true,
			nil,
			nil,
			hcat.ResolveEvent{Complete: true},
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
			hcat.ResolveEvent{Complete: true},
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
				task:     Task{Name: "RenderTemplateTest", Enabled: true},
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
		inited      bool
		initReturn  error
		applyReturn error
		postApply   handler.Handler
	}{
		{
			"happy path - no post-apply handler",
			false,
			false,
			nil,
			nil,
			nil,
		},
		{
			"happy path - post-apply handler",
			false,
			false,
			nil,
			nil,
			testHandler(false),
		},
		{
			"already inited",
			false,
			true,
			nil,
			nil,
			nil,
		},
		{
			"error on init",
			true,
			false,
			errors.New("init error"),
			nil,
			nil,
		},
		{
			"error on apply",
			true,
			false,
			nil,
			errors.New("apply error"),
			nil,
		},
		{
			"error on post-apply handler",
			true,
			false,
			nil,
			nil,
			testHandler(true),
		},
	}
	ctx := context.Background()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := new(mocks.Client)
			c.On("Init", ctx).Return(tc.initReturn).Once()
			c.On("Apply", ctx).Return(tc.applyReturn).Once()

			tf := &Terraform{
				mu:        &sync.RWMutex{},
				task:      Task{Name: "ApplyTaskTest", Enabled: true},
				client:    c,
				postApply: tc.postApply,
				inited:    tc.inited,
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
			tf := &Terraform{
				fileReader: tc.fileReader,
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
		task        Task
	}{
		{
			"no provider",
			false,
			true,
			Task{},
		},
		{
			"provider without handler (no error)",
			true,
			true,
			Task{
				Providers: NewTerraformProviderBlocks([]hcltmpl.NamedBlock{
					hcltmpl.NewNamedBlock(map[string]interface{}{
						handler.TerraformProviderFake: map[string]interface{}{
							"required-config": "missing",
						},
					})}),
			},
		},
		{
			"provider without handler (no error)",
			false,
			true,
			Task{
				Providers: NewTerraformProviderBlocks([]hcltmpl.NamedBlock{
					hcltmpl.NewNamedBlock(map[string]interface{}{
						"provider-no-handler": map[string]interface{}{},
					})}),
			},
		},
		{
			"happy path - provider with handler",
			false,
			false,
			Task{
				Providers: NewTerraformProviderBlocks([]hcltmpl.NamedBlock{
					hcltmpl.NewNamedBlock(map[string]interface{}{
						handler.TerraformProviderFake: map[string]interface{}{
							"name": "happy-path",
						},
					})}),
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h, err := getTerraformHandlers(tc.task)
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
			task:    Task{Name: "disabled_task", Enabled: false},
			watcher: new(mocksTmpl.Watcher),
		}

		ctx := context.Background()

		err := tf.InitTask(true)
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

// testHandler returns a fake handler that can return an error or not on Do()
func testHandler(err bool) handler.Handler {
	config := map[string]interface{}{
		"name": "1",
		"err":  err,
	}

	h, _ := handler.NewFake(config)
	return h
}
