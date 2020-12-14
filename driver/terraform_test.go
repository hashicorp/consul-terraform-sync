package driver

import (
	"context"
	"errors"
	"testing"

	"github.com/hashicorp/consul-terraform-sync/handler"
	mocks "github.com/hashicorp/consul-terraform-sync/mocks/client"
	"github.com/hashicorp/consul-terraform-sync/templates/hcltmpl"
	"github.com/stretchr/testify/assert"
)

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
				task:      Task{Name: "ApplyTaskTest"},
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

// testHandler returns a fake handler that can return an error or not on Do()
func testHandler(err bool) handler.Handler {
	config := map[string]interface{}{
		"name": "1",
		"err":  err,
	}

	h, _ := handler.NewFake(config)
	return h
}
