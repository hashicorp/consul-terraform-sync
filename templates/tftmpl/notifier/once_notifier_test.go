// Copyright IBM Corp. 2020, 2025
// SPDX-License-Identifier: MPL-2.0

package notifier

import (
	"testing"

	mocks "github.com/hashicorp/consul-terraform-sync/mocks/templates"
	"github.com/hashicorp/hcat/dep"
	"github.com/stretchr/testify/assert"
)

func TestOnceNotifier(t *testing.T) {
	allFalse := func(interface{}) (bool, bool) { return false, false }
	allTrue := func(interface{}) (bool, bool) { return true, true }
	t.Run("before once done should always run", func(t *testing.T) {
		tmpl := &mocks.Template{}
		n := NewOnceNotifier(allFalse, tmpl)
		// Notify before once-mode is done should cause a render and a trigger.
		tmpl.EXPECT().Notify(nil).Return(false)
		assert.True(t, n.Notify(nil))
	})
	t.Run("after once done should not run if false func", func(t *testing.T) {
		tmpl := &mocks.Template{}
		n := NewOnceNotifier(allFalse, tmpl)
		n.SetOnceDone()
		// Do not set an EXPECT on the tmpl, because it should not be invoked.
		assert.False(t, n.Notify(nil))
	})
	t.Run("after once done should run if true func", func(t *testing.T) {
		tmpl := &mocks.Template{}
		n := NewOnceNotifier(allTrue, tmpl)
		n.SetOnceDone()
		tmpl.EXPECT().Notify(nil).Return(false)
		assert.True(t, n.Notify(nil))
	})
}

func TestTriggerCheckConsulKV(t *testing.T) {
	re, tr := TriggerCheckConsulKV((*dep.KeyPair)(nil))
	assert.True(t, re)
	assert.True(t, tr)
	re, tr = TriggerCheckConsulKV(([]*dep.KeyPair)(nil))
	assert.True(t, re)
	assert.True(t, tr)
	re, tr = TriggerCheckConsulKV(nil)
	assert.False(t, re)
	assert.False(t, tr)
}

func TestTriggerCheckService(t *testing.T) {
	re, tr := TriggerCheckService(([]*dep.HealthService)(nil))
	assert.True(t, re)
	assert.True(t, tr)
	re, tr = TriggerCheckService(nil)
	assert.False(t, re)
	assert.False(t, tr)
}

func TestMakeTriggerCheckCatalogService(t *testing.T) {
	t.Run("only trigger on snippets", func(t *testing.T) {
		check := MakeTriggerCheckCatalogService()
		re, tr := check(nil)
		assert.False(t, re)
		assert.False(t, tr)
	})
	t.Run("trigger when change detected", func(t *testing.T) {
		check := MakeTriggerCheckCatalogService()
		re, tr := check([]*dep.CatalogSnippet{
			{Name: "one"},
		})
		assert.True(t, re)
		assert.True(t, tr)
		re, tr = check([]*dep.CatalogSnippet{
			{Name: "one"},
			{Name: "two"},
		})
		assert.True(t, re)
		assert.True(t, tr)
		re, tr = check([]*dep.CatalogSnippet{
			{Name: "two"},
			{Name: "one"},
		})
		assert.False(t, re)
		assert.False(t, tr)
		re, tr = check([]*dep.CatalogSnippet{
			{Name: "two"},
		})
		assert.True(t, re)
		assert.True(t, tr)
	})
}
