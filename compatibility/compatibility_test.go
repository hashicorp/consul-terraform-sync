package compatibility

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/consul-terraform-sync/config"
	"github.com/hashicorp/consul-terraform-sync/logging"
	"github.com/stretchr/testify/assert"
)

type TestConfig struct {
	S  *string `cts-ha:"unshared"`
	I  *int
	Ii int `cts-ha:"unshared"`
	C  *TestSubConfig
	C2 *TestSubConfig `cts-ha:"unshared"`
	C3 TestSubConfig
}

type TestSubConfig struct {
	S *string
	I *int `cts-ha:"unshared"`
}

func TestIsCompatibleConfig(t *testing.T) {
	ctx := context.Background()
	logger := logging.NewNullLogger()
	ctx = logging.WithContext(ctx, logger)

	isCompatible := IsCompatibleConfig(ctx, &config.Config{}, &config.Config{})

	assert.True(t, isCompatible)
}

func TestIsCompatible(t *testing.T) {
	t.Parallel()

	s := "blah"
	i := 123

	testCases := []struct {
		name                 string
		baseConf             TestConfig
		conf                 TestConfig
		expectedIsCompatible bool
		expectedNumLogs      int
	}{
		{
			name: "compatible same",
			baseConf: TestConfig{
				S:  &s, // unshared
				I:  &i,
				Ii: 456, // unshared
				C: &TestSubConfig{
					S: &s,
					I: &i, // unshared
				},
				C2: &TestSubConfig{ // unshared
					S: &s,
					I: &i, // unshared
				},
				C3: TestSubConfig{
					S: &s,
					I: &i, // unshared
				},
			},
			conf: TestConfig{
				S:  &s,
				I:  &i,
				Ii: 456,
				C: &TestSubConfig{
					S: &s,
					I: &i,
				},
				C2: &TestSubConfig{
					S: &s,
					I: &i,
				},
				C3: TestSubConfig{
					S: &s,
					I: &i,
				},
			},
			expectedNumLogs:      0,
			expectedIsCompatible: true,
		},
		{
			name: "compatible different unshared only",
			baseConf: TestConfig{
				S:  &s, // unshared
				I:  &i,
				Ii: 456, // unshared
				C: &TestSubConfig{
					S: &s,
					I: &i, // unshared
				},
				C2: &TestSubConfig{ // unshared
					S: &s,
					I: &i, // unshared
				},
				C3: TestSubConfig{
					S: &s,
					I: &i, // unshared
				},
			},
			conf: TestConfig{
				S:  nil, // unshared
				I:  &i,
				Ii: 0, // unshared
				C: &TestSubConfig{
					S: &s,
					I: nil, // unshared
				},
				C2: &TestSubConfig{ // unshared
					S: &s,
					I: &i, // unshared
				},
				C3: TestSubConfig{
					S: &s,
					I: nil, // unshared
				},
			},
			expectedNumLogs:      0,
			expectedIsCompatible: true,
		},
		{
			name: "incompatible different with nested structs",
			baseConf: TestConfig{
				S:  &s, // unshared
				I:  &i,
				Ii: 456, // unshared
				C: &TestSubConfig{
					S: &s,
					I: &i, // unshared
				},
				C2: &TestSubConfig{ // unshared
					S: &s,
					I: &i,
				},
			},
			conf: TestConfig{
				S:  nil,
				I:  nil,
				Ii: 0,
				C: &TestSubConfig{
					S: nil,
					I: nil,
				},
				C2: &TestSubConfig{
					S: nil,
					I: nil,
				},
			},
			expectedNumLogs:      2,
			expectedIsCompatible: false,
		},
		{
			name: "incompatible different with same nested struct",
			baseConf: TestConfig{
				S:  &s, // unshared
				I:  &i,
				Ii: 456, // unshared
				C: &TestSubConfig{
					S: &s,
					I: &i, // unshared
				},
			},
			conf: TestConfig{
				S:  &s, // unshared
				I:  nil,
				Ii: 456, // unshared
				C: &TestSubConfig{
					S: &s,
					I: &i, // unshared
				},
			},
			expectedNumLogs:      1,
			expectedIsCompatible: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			tw := newTestWriter()
			logger := logging.NewTestLogger("INFO", tw)
			ctx = logging.WithContext(ctx, logger)

			actualIsCompatible := IsCompatible(ctx, reflect.ValueOf(&tc.baseConf), reflect.ValueOf(&tc.conf))

			assert.Equal(t, tc.expectedIsCompatible, actualIsCompatible)
			assert.Equal(t, tc.expectedNumLogs, len(tw.Mem))
		})
	}
}

func TestCompatibility_Panics(t *testing.T) {
	ctx := context.Background()
	logger := logging.NewNullLogger()
	logging.WithContext(ctx, logger)

	t.Parallel()

	testCases := []struct {
		name string
		val1 reflect.Value
		val2 reflect.Value
	}{
		{
			name: "different structs",
			val1: reflect.ValueOf(&TestConfig{}),
			val2: reflect.ValueOf(&config.Config{}),
		},
		{
			name: "non pointers",
			val1: reflect.ValueOf(TestConfig{}),
			val2: reflect.ValueOf(TestConfig{}),
		},
		{
			name: "non nil pointers",
			val1: reflect.ValueOf(nil),
			val2: reflect.ValueOf(nil),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Panics(t, func() { IsCompatible(ctx, tc.val1, tc.val2) })
		})
	}
}

type testWriter struct {
	Mem []string
}

func newTestWriter() *testWriter {
	return &testWriter{
		Mem: make([]string, 0),
	}
}

func (tw *testWriter) Write(p []byte) (n int, err error) {
	s := string(p[:])
	tw.Mem = append(tw.Mem, s)
	return 0, nil
}
