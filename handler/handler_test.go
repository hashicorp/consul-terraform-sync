package handler

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTerraformProviderHandler(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name         string
		expectError  bool
		nilProvider  bool
		providerName string
		config       interface{}
	}{
		{
			"provider without handler",
			false,
			true,
			"no-handler-provider",
			map[string]interface{}{
				"token": "abcd",
			},
		},
		{
			"provider with malformed config",
			true,
			true,
			"some-provider",
			map[string]string{},
		},
		{
			"panos provider",
			false,
			false,
			TerraformProviderPanos,
			map[string]interface{}{
				"hostname": "10.10.10.10",
				"username": "user",
				"password": "pw123",
			},
		},
		{
			"fake provider",
			false,
			false,
			TerraformProviderFake,
			map[string]interface{}{
				"name": "1",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h, err := TerraformProviderHandler(tc.providerName, tc.config)
			if tc.expectError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)

			if tc.nilProvider {
				assert.Nil(t, h)
				return
			}
			assert.NotNil(t, h)
		})
	}
}

func ExampleTerraformProviderHandler() {
	providers := make([]map[string]interface{}, 0)

	third := make(map[string]interface{})
	third[TerraformProviderFake] = map[string]interface{}{
		"name": "3",
	}
	providers = append(providers, third)

	second := make(map[string]interface{})
	second[TerraformProviderFake] = map[string]interface{}{
		"name": "2",
	}
	providers = append(providers, second)

	first := make(map[string]interface{})
	first[TerraformProviderFake] = map[string]interface{}{
		"name": "1",
	}
	providers = append(providers, first)

	var next Handler = nil
	for _, p := range providers {
		for k, v := range p {
			h, err := TerraformProviderHandler(k, v)
			if err != nil {
				fmt.Println(err)
				return
			}
			if h != nil {
				h.SetNext(next)
				next = h
			}
		}
	}
	next.Do()
	// Output:
	// FakeHandler: '1'
	// FakeHandler: '2'
	// FakeHandler: '3'
}
