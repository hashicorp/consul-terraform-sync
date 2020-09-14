package handler

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewFake(t *testing.T) {
	cases := []struct {
		name        string
		expectError bool
		config      map[string]interface{}
	}{
		{
			"happy path",
			false,
			map[string]interface{}{
				"name": "1",
			},
		},
		{
			"missing configuration",
			true,
			map[string]interface{}{},
		},
		{
			"happy path + extra config",
			false,
			map[string]interface{}{
				"name":  "1",
				"err":   true,
				"count": 8,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h, err := NewFake(tc.config)
			if tc.expectError {
				assert.Error(t, err)
				assert.Nil(t, h)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestFakeDo(t *testing.T) {
	cases := []struct {
		name string
		next Handler
	}{
		{
			"existing next handler",
			&Fake{},
		},
		{
			"no next handler",
			nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := &Fake{}
			if tc.next != nil {
				h.SetNext(tc.next)
			}
			h.Do()
			// nothing to assert at this moment. confirming that it runs successfully
		})
	}
}

func TestFakeSetNext(t *testing.T) {
	cases := []struct {
		name string
	}{
		{
			"happy path",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := &Fake{}
			h.SetNext(&Fake{})
			assert.NotNil(t, h.next)
		})
	}
}
