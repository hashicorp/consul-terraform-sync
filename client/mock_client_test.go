package client

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMockClientInit(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		expectError bool
		errs        []error
	}{
		{
			"default, no error",
			false,
			nil,
		},
		{
			"channel input: mixed errors",
			false,
			[]error{
				errors.New("1 Error"),
				errors.New("2 Error"),
				nil,
				errors.New("4 Error"),
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := NewMockClient()

			// setup Init() return values
			if len(tc.errs) > 0 {
				errChan := make(chan error, len(tc.errs))
				for _, err := range tc.errs {
					errChan <- err
				}

				c.InitFunc = func() error {
					return <-errChan
				}
			}

			ctx := context.Background()
			for _, expectedErr := range tc.errs {
				actualErr := c.Init(ctx)

				if expectedErr == nil {
					assert.NoError(t, actualErr)
					continue
				}

				assert.Error(t, actualErr)
				assert.Equal(t, expectedErr, actualErr)
			}

			// make sure to check default test where no expected errors
			if len(tc.errs) == 0 {
				actualErr := c.Init(ctx)
				assert.NoError(t, actualErr)
			}
		})
	}
}

func TestMockClientApply(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		expectError bool
		errs        []error
	}{
		{
			"default, no error",
			false,
			nil,
		},
		{
			"channel input: mixed errors",
			false,
			[]error{
				errors.New("1 Error"),
				errors.New("2 Error"),
				errors.New("3 Error"),
				nil,
				nil,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := NewMockClient()

			// setup Apply() return values
			if len(tc.errs) > 0 {
				errChan := make(chan error, len(tc.errs))
				for _, err := range tc.errs {
					errChan <- err
				}

				c.ApplyFunc = func() error {
					return <-errChan
				}
			}

			ctx := context.Background()
			for _, expectedErr := range tc.errs {
				actualErr := c.Apply(ctx)

				if expectedErr == nil {
					assert.NoError(t, actualErr)
					continue
				}

				assert.Error(t, actualErr)
				assert.Equal(t, expectedErr, actualErr)
			}

			// make sure to check default test where no expected errors
			if len(tc.errs) == 0 {
				actualErr := c.Apply(ctx)
				assert.NoError(t, actualErr)
			}
		})
	}
}

func TestMockClientPlan(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		expectError bool
		errs        []error
	}{
		{
			"default, no error",
			false,
			nil,
		},
		{
			"channel input: mixed errors",
			false,
			[]error{
				nil,
				nil,
				errors.New("3 Error"),
				errors.New("4 Error"),
				nil,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := NewMockClient()

			// setup Plan() return values
			if len(tc.errs) > 0 {
				errChan := make(chan error, len(tc.errs))
				for _, err := range tc.errs {
					errChan <- err
				}

				c.PlanFunc = func() error {
					return <-errChan
				}
			}

			ctx := context.Background()
			for _, expectedErr := range tc.errs {
				actualErr := c.Plan(ctx)

				if expectedErr == nil {
					assert.NoError(t, actualErr)
					continue
				}

				assert.Error(t, actualErr)
				assert.Equal(t, expectedErr, actualErr)
			}

			// make sure to check default test where no expected errors
			if len(tc.errs) == 0 {
				actualErr := c.Plan(ctx)
				assert.NoError(t, actualErr)
			}
		})
	}
}

func TestMockClientGoString(t *testing.T) {
	cases := []struct {
		name string
		mock *MockClient
	}{
		{
			"nil mock client",
			nil,
		},
		{
			"happy path",
			&MockClient{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.mock == nil {
				assert.Contains(t, tc.mock.GoString(), "nil")
				return
			}

			assert.Contains(t, tc.mock.GoString(), "&MockClient")
		})
	}
}
