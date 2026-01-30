// Copyright IBM Corp. 2020, 2025
// SPDX-License-Identifier: MPL-2.0

package handler

import (
	"context"
	"errors"
	"testing"

	"github.com/PaloAltoNetworks/pango"
	"github.com/hashicorp/consul-terraform-sync/logging"
	mocks "github.com/hashicorp/consul-terraform-sync/mocks/handler"
	"github.com/hashicorp/consul-terraform-sync/retry"
	"github.com/hashicorp/consul-terraform-sync/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestNewPanos(t *testing.T) {
	cases := []struct {
		name               string
		expectError        bool
		config             map[string]interface{}
		expected           pango.Client
		expectedConfigPath string
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
				"logging":            []string{"action", "uid"},
				"verify_certificate": true,
				"json_config_file":   "/my/path/config.json",
			},
			pango.Client{
				Hostname:              "10.10.10.10",
				Username:              "user",
				Password:              "pw123",
				ApiKey:                "abcd",
				Protocol:              "http",
				Port:                  8080,
				Timeout:               5,
				LoggingFromInitialize: []string{"action", "uid"},
				VerifyCertificate:     true,
			},
			"/my/path/config.json",
		}, {
			"missing required username",
			true,
			map[string]interface{}{
				"hostname": "10.10.10.10",
				"api_key":  "abcd",
			},
			pango.Client{},
			"",
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
			assert.Equal(t, tc.expectedConfigPath, h.configPath)
		})
	}

	t.Run("username from env", func(t *testing.T) {
		adminUser := "admin"
		reset := testutils.Setenv("PANOS_USERNAME", adminUser)
		defer reset()

		config := map[string]interface{}{
			"hostname": "10.10.10.10",
			"api_key":  "abcd",
		}
		h, err := NewPanos(config)
		assert.NoError(t, err)
		assert.Equal(t, adminUser, h.adminUser)
	})
}

func TestPanosDo(t *testing.T) {
	cases := []struct {
		name string
		next bool
	}{
		{
			"happy path - with next handler",
			false,
		},
		{
			"happy path - no next handler",
			true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := new(mocks.PanosClient)
			m.On("InitializeUsing", mock.Anything, mock.Anything, mock.Anything).
				Return(nil)
			m.On("Commit", mock.Anything, mock.Anything, mock.Anything).
				Return(uint(1), []byte("message"), nil)
			m.On("WaitForJob", mock.Anything, mock.Anything, mock.Anything).
				Return(nil)
			m.On("String").Return("client string")

			h := &Panos{client: m, logger: logging.NewNullLogger()}
			if tc.next {
				next := &Panos{client: m, logger: logging.NewNullLogger()}
				h.SetNext(next)
			}

			assert.NoError(t, h.Do(context.Background(), nil))
		})
	}

	t.Run("autoCommit setting", func(t *testing.T) {
		m := new(mocks.PanosClient)
		m.On("InitializeUsing", mock.Anything, mock.Anything, mock.Anything).
			Return(nil).Once()
		m.On("Commit", mock.Anything, mock.Anything, mock.Anything).
			Return(uint(1), []byte("message"), nil).Once()
		m.On("WaitForJob", mock.Anything, mock.Anything, mock.Anything).
			Return(nil).Once()
		m.On("String").Return("client string").Once()

		h := &Panos{client: m, autoCommit: true, retry: retry.NewTestRetry(1), logger: logging.NewNullLogger()}
		assert.NoError(t, h.Do(context.Background(), nil))
		h.autoCommit = false
		assert.NoError(t, h.Do(context.Background(), nil))
	})
}

func TestPanosCommit(t *testing.T) {
	cases := []struct {
		name         string
		initReturn   error
		commitJob    uint
		commitResp   []byte
		commitReturn error
		waitReturn   error
		expectErr    bool
	}{
		{
			"happy path",
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
			true,
		},
		{
			"error on commit",
			nil,
			100,
			[]byte("failure"),
			errors.New("commit error"),
			nil,
			true,
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
			"error on wait",
			nil,
			10,
			[]byte("ok"),
			nil,
			errors.New("wait error"),
			true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := new(mocks.PanosClient)
			m.On("InitializeUsing", mock.Anything, mock.Anything, mock.Anything).
				Return(tc.initReturn).Once()
			m.On("Commit", mock.Anything, mock.Anything, mock.Anything).
				Return(tc.commitJob, tc.commitResp, tc.commitReturn)
			m.On("WaitForJob", mock.Anything, mock.Anything, mock.Anything).
				Return(tc.waitReturn)
			m.On("String").Return("client string").Once()

			h := &Panos{client: m, retry: retry.NewTestRetry(1), logger: logging.NewNullLogger()}
			err := h.commit(context.Background())
			if tc.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			expectedRetries := func() (commitTries int, waitTries int) {
				i, j := tc.initReturn == nil, tc.commitJob != 0
				c, w := tc.commitReturn == nil, tc.waitReturn == nil
				switch {
				case !i: // init fails, others not reached
					return 0, 0
				case !j: // job 0, commit called but not needed (skip waiting)
					return 1, 0
				case !c: // commit fails, wait never reached
					return 2, 0
				case c && w: // everyone happy
					return 1, 1
				case c && !w: // wait fails
					return 2, 2
				}
				return -1, -1 // never reached
			}
			commitTries, waitTries := expectedRetries()
			m.AssertNumberOfCalls(t, "Commit", commitTries)
			m.AssertNumberOfCalls(t, "WaitForJob", waitTries)
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
			h := &Panos{logger: logging.NewNullLogger()}
			h.SetNext(&Panos{logger: logging.NewNullLogger()})
			assert.NotNil(t, h.next)
		})
	}
}

func TestPanosEmptyCommit(t *testing.T) {
	cases := []struct {
		name  string
		job   uint
		resp  string
		err   error
		empty bool
	}{
		{
			"empty commit: super admin role",
			uint(0),
			`<response status="success" code="19"><msg>` +
				`There are no changes to commit.</msg></response>`,
			nil,
			true,
		},
		{
			"empty commit: custom role",
			uint(0),
			`<response status="success" code="13"><msg>` +
				`The result of this commit would be the same as the previous etc`,
			errors.New("The result of this commit would be the same as the previous etc"),
			true,
		},
		{
			"unknown commit: auth error",
			uint(0),
			`<response status = 'error' code = '403'><result><msg>` +
				`Type [commit] not authorized for user role etc`,
			errors.New("Type [commit] not authorized for user role"),
			false,
		},
		{
			"not empty commit: happy path",
			uint(17),
			`<response status="success" code="19"><result><msg><line>` +
				`Commit job enqueued with jobid 17</line></msg><job>17</job> etc`,
			nil,
			false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual := emptyCommit(tc.job, []byte(tc.resp), tc.err)
			assert.Equal(t, tc.empty, actual)
		})
	}
}

func assertEqualCred(t *testing.T, exp, act pango.Client) {
	assert.Equal(t, exp.Hostname, act.Hostname)
	assert.Equal(t, exp.Username, act.Username)
	assert.Equal(t, exp.Password, act.Password)
	assert.Equal(t, exp.ApiKey, act.ApiKey)
	assert.Equal(t, exp.Protocol, act.Protocol)
	assert.Equal(t, exp.Port, act.Port)
	assert.Equal(t, exp.Timeout, act.Timeout)
	assert.Equal(t, exp.LoggingFromInitialize, act.LoggingFromInitialize)
	assert.Equal(t, exp.VerifyCertificate, act.VerifyCertificate)
}
