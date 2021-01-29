package handler

import (
	"context"
	"errors"
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

	fourth := make(map[string]interface{})
	fourth[TerraformProviderFake] = map[string]interface{}{
		"name": "4",
		"err":  true,
	}
	providers = append(providers, fourth)

	third := make(map[string]interface{})
	third[TerraformProviderFake] = map[string]interface{}{
		"name": "3",
	}
	providers = append(providers, third)

	second := make(map[string]interface{})
	second[TerraformProviderFake] = map[string]interface{}{
		"name": "2",
		"err":  true,
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
	fmt.Println("Handler Errors:", next.Do(context.Background(), nil))
	// Output:
	// FakeHandler: '1'
	// FakeHandler: '2'
	// FakeHandler: '3'
	// FakeHandler: '4'
	// Handler Errors: error 4: error 2
}

func TestCallNext(t *testing.T) {
	cases := []struct {
		name    string
		nextH   Handler
		nextErr bool
	}{
		{
			"happy path - no next handler",
			nil,
			false,
		},
		{
			"happy path - with next handler",
			&Fake{err: false},
			false,
		},
		{
			"error in next handler",
			&Fake{err: true},
			true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fmt.Println("fake: ", tc.nextH, " equal? ", tc.nextH == nil)
			err := callNext(context.Background(), tc.nextH, nil, nil)
			if tc.nextErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNextError(t *testing.T) {
	cases := []struct {
		name       string
		prevErr    error
		currentErr error
		nextErr    error
	}{
		{
			"no errors",
			nil,
			nil,
			nil,
		},
		{
			"no error, then error",
			nil,
			errors.New("current"),
			errors.New("current"),
		},
		{
			"error, then no error",
			errors.New("previous"),
			nil,
			errors.New("previous"),
		},
		{
			"both errors",
			errors.New("previous"),
			errors.New("current"),
			errors.New("current: previous"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual := nextError(tc.prevErr, tc.currentErr)
			assert.Equal(t, fmt.Sprintf("%s", tc.nextErr), fmt.Sprintf("%s", actual))
		})
	}
}
