package driver

import (
	"context"
	"errors"
	"testing"

	mocks "github.com/hashicorp/consul-nia/mocks/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestWorkerInit(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		initErr error
	}{
		{
			"happy path",
			nil,
		},
		{
			"error",
			errors.New("error"),
		},
	}

	for _, tc := range cases {
		ctx := context.Background()
		t.Run(tc.name, func(t *testing.T) {
			c := new(mocks.Client)
			c.On("Init", mock.Anything).Return(tc.initErr)
			w := worker{client: c}

			err := w.init(ctx)
			if tc.initErr != nil {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestWorkerApply(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		applyErr error
	}{
		{
			"happy path",
			nil,
		},
		{
			"error",
			errors.New("error"),
		},
	}

	for _, tc := range cases {
		ctx := context.Background()
		t.Run(tc.name, func(t *testing.T) {
			c := new(mocks.Client)
			c.On("Apply", mock.Anything).Return(tc.applyErr)
			w := worker{client: c}

			err := w.apply(ctx)
			if tc.applyErr != nil {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestWorkString(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		nilWork bool
		task    Task
		expect  string
	}{
		{
			"nil",
			true,
			Task{Name: "first"},
			"nil",
		},
		{
			"name, no provider",
			false,
			Task{Name: "test_task"},
			"TaskName: 'test_task', TaskProviders: ''",
		},
		{
			"name with single provider",
			false,
			Task{
				Name: "test_task",
				Providers: []map[string]interface{}{
					map[string]interface{}{"Provider1": true},
				},
			},
			"TaskName: 'test_task', TaskProviders: 'Provider1'",
		},
		{
			"name with multiple providers",
			false,
			Task{
				Name: "test_task",
				Providers: []map[string]interface{}{
					map[string]interface{}{"Provider1": true},
					map[string]interface{}{"Provider2": true},
					map[string]interface{}{"Provider3": true},
				},
			},
			"TaskName: 'test_task', TaskProviders: 'Provider1, Provider2, Provider3'",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var w *work
			if !tc.nilWork {
				w = &work{task: tc.task}
			}
			actual := w.String()
			assert.Equal(t, tc.expect, actual)

			if tc.nilWork {
				// no need to continue testing
				return
			}

			// confirm desc field is set
			assert.Equal(t, tc.expect, w.desc)

			// rerun for coverage on retrieving from desc
			rerun := w.String()
			assert.Equal(t, tc.expect, rerun)
		})
	}
}
