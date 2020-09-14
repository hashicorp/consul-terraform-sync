package handler

import (
	"errors"
	"testing"

	"github.com/hashicorp/consul-nia/config"
	mocks "github.com/hashicorp/consul-nia/mocks/handler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestNewPanos(t *testing.T) {
	cases := []struct {
		name        string
		expectError bool
		config      map[string]interface{}
		expected    panosConfig
	}{
		{
			"happy path",
			false,
			map[string]interface{}{
				"hostname":           "10.10.10.10",
				"username":           "user",
				"password":           "pw123",
				"api_key":            "abcd",
				"protocol":           "http",
				"port":               8080,
				"timeout":            5,
				"verify_certificate": true,
			},
			panosConfig{
				Hostname:          config.String("10.10.10.10"),
				Username:          config.String("user"),
				Password:          config.String("pw123"),
				APIKey:            config.String("abcd"),
				Protocol:          config.String("http"),
				Port:              config.Int(8080),
				Timeout:           config.Int(5),
				VerifyCertificate: config.Bool(true),
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h, err := NewPanos(tc.config)
			if tc.expectError {
				assert.Error(t, err)
				assert.Nil(t, h)
				return
			}
			require.NoError(t, err)
			assertEqualCred(t, tc.expected, h.providerConf)
		})
	}
}

func TestPanosDo(t *testing.T) {
	cases := []struct {
		name       string
		initReturn error
		commitJob  uint
		commitResp []byte
		commitErr  error
		waitReturn error
		next       bool
	}{
		{
			"happy path - with next handler",
			nil,
			100,
			[]byte("ok"),
			nil,
			nil,
			true,
		},
		{
			"happy path - no next handler",
			nil,
			100,
			[]byte("ok"),
			nil,
			nil,
			false,
		},
		{
			"error on initialize",
			errors.New("initialize error"),
			100,
			[]byte("ok"),
			nil,
			nil,
			false,
		},
		{
			"error on commit",
			nil,
			100,
			[]byte("failure"),
			errors.New("commit error"),
			nil,
			false,
		},
		{
			"commit job 0",
			nil,
			0,
			[]byte("no commit needed"),
			nil,
			nil,
			false,
		},
		{
			"error on return",
			nil,
			10,
			[]byte("ok"),
			nil,
			errors.New("wait error"),
			false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := new(mocks.PanosClient)
			m.On("InitializeUsing", mock.Anything, mock.Anything, mock.Anything).
				Return(tc.initReturn).Once()
			m.On("Commit", mock.Anything, mock.Anything, mock.Anything).
				Return(tc.commitJob, tc.commitResp, tc.commitErr).Once()
			m.On("WaitForJob", mock.Anything, mock.Anything).
				Return(tc.waitReturn).Once()
			m.On("String").Return("client string").Once()

			h := &Panos{client: m}
			if tc.next {
				m := new(mocks.PanosClient)
				m.On("InitializeUsing", mock.Anything, mock.Anything, mock.Anything).
					Return(errors.New("stop")).Once()
				next := &Panos{client: m}
				h.SetNext(next)
			}

			h.Do()
			// nothing to assert. confirming successful run + expected coverage
		})
	}
}

func TestPanosSetNext(t *testing.T) {
	cases := []struct {
		name string
	}{
		{
			"happy path",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := &Panos{}
			h.SetNext(&Panos{})
			assert.NotNil(t, h.next)
		})
	}
}

func TestPanosConfigGoString(t *testing.T) {
	cases := []struct {
		name     string
		config   *panosConfig
		expected string
	}{
		{
			"happy path",
			&panosConfig{
				Hostname:          config.String("10.10.10.10"),
				Username:          config.String("user"),
				Password:          config.String("pw123"),
				APIKey:            config.String("abcd"),
				Protocol:          config.String("http"),
				Port:              config.Int(8080),
				Timeout:           config.Int(5),
				Logging:           []string{"action", "uid"},
				VerifyCertificate: config.Bool(true),
				JSONConfigFile:    config.String("config.json"),
			},
			`&panosConfig{Hostname:10.10.10.10, Username:user, ` +
				`Password:<password-redacted>, APIKey:<api-key-redacted>, ` +
				`Protocol:http, Port:8080, Timeout:5, Logging:[action uid], ` +
				`VerifyCertificate:true, JSONConfigFile:config.json}`,
		},
		{
			"nil",
			nil,
			`(*panosConfig)(nil)`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual := tc.config.GoString()
			assert.Equal(t, tc.expected, actual)
		})
	}
}

func assertEqualCred(t *testing.T, exp, act panosConfig) {
	assert.Equal(t, config.StringVal(exp.Hostname), config.StringVal(act.Hostname))
	assert.Equal(t, config.StringVal(exp.Username), config.StringVal(act.Username))
	assert.Equal(t, config.StringVal(exp.Password), config.StringVal(act.Password))
	assert.Equal(t, config.StringVal(exp.APIKey), config.StringVal(act.APIKey))
	assert.Equal(t, config.StringVal(exp.Protocol), config.StringVal(act.Protocol))
	assert.Equal(t, config.IntVal(exp.Port), config.IntVal(act.Port))
	assert.Equal(t, config.IntVal(exp.Timeout), config.IntVal(act.Timeout))
	assert.Equal(t, config.BoolVal(exp.VerifyCertificate), config.BoolVal(act.VerifyCertificate))
	assert.Equal(t, config.StringVal(exp.JSONConfigFile), config.StringVal(act.JSONConfigFile))
}
