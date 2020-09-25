package driver

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"reflect"
	"testing"
	"time"

	"github.com/hashicorp/consul-nia/client"
	mocks "github.com/hashicorp/consul-nia/mocks/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestNewWorker(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		expectError bool
		clientType  string
	}{
		{
			"happy path (mock client)",
			false,
			testClient,
		},
		{
			"error (default tf client)",
			true,
			"",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := newWorker(&workerConfig{
				task:       Task{},
				clientType: tc.clientType,
			})
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestInitClient(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		clientType  string
		expectError bool
		expect      client.Client
	}{
		{
			"happy path with development client",
			developmentClient,
			false,
			&client.Printer{},
		},
		{
			"happy path with mock client",
			testClient,
			false,
			&mocks.Client{},
		},
		{
			"error when creating Terraform CLI client",
			"",
			true,
			&client.TerraformCLI{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual, err := initClient(&workerConfig{
				task:       Task{},
				clientType: tc.clientType,
			})
			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, reflect.TypeOf(tc.expect), reflect.TypeOf(actual))
			}
		})
	}
}

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
			w := worker{
				client: c,
				random: rand.New(rand.NewSource(1)),
			}

			err := w.init(ctx)
			if tc.initErr != nil {
				assert.Error(t, err)
				assert.False(t, w.inited)
				return
			}
			assert.NoError(t, err)
			assert.True(t, w.inited)
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
			w := worker{
				client: c,
				random: rand.New(rand.NewSource(1)),
			}

			err := w.apply(ctx)
			if tc.applyErr != nil {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestWithRetry_context_cancel(t *testing.T) {
	t.Parallel()

	r := retry{
		desc:   "fake fxn that never succeeds",
		retry:  5,
		random: rand.New(rand.NewSource(1)),
		fxn: func() error {
			return errors.New("test error")
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error)
	go func() {
		err := r.do(ctx)
		if err != nil {
			errCh <- err
		}
	}()
	cancel()

	select {
	case err := <-errCh:
		if err != context.Canceled {
			t.Error("wanted 'context canceled', got:", err)
		}
	case <-time.After(time.Second * 5):
		t.Fatal("Run did not exit properly from cancelling context")
	}
}

func TestWithRetry(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		retry     int
		successOn int
		expected  error
	}{
		{
			"happy path: retry once, success on retry (2)",
			1,
			2,
			nil,
		},
		{
			"no success on retries: retry once",
			1,
			100,
			errors.New("attempt #2 failed 'error on 2': attempt #1 failed 'error on 1'"),
		},
		{
			"happy path: no retry",
			0,
			1,
			nil,
		},
		{
			"no retry, no success",
			0,
			100,
			errors.New("attempt #1 failed 'error on 1'"),
		},
	}

	ctx := context.Background()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// set up fake function
			count := 0
			fxn := func() error {
				count++
				if count == tc.successOn {
					return nil
				}
				return fmt.Errorf("error on %d", count)
			}

			r := retry{
				desc:   "test fxn",
				retry:  tc.retry,
				random: rand.New(rand.NewSource(1)),
				fxn:    fxn,
			}
			err := r.do(ctx)
			if tc.expected == nil {
				assert.NoError(t, err)
			} else {
				assert.Equal(t, tc.expected.Error(), err.Error())
			}
		})
	}
}

func TestWaitTime(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		attempt   int
		minReturn float64
		maxReturn float64
	}{
		{
			"first attempt",
			0,
			0,
			0.5,
		},
		{
			"second attempt",
			1,
			1,
			2.5,
		},
		{
			"third attempt",
			2,
			4,
			6.5,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := retry{
				random: rand.New(rand.NewSource(1)),
			}
			a := r.waitTime(tc.attempt)

			actual := float64(a) / float64(time.Second)
			assert.GreaterOrEqual(t, actual, tc.minReturn)
			assert.LessOrEqual(t, actual, tc.maxReturn)
		})
	}
}
