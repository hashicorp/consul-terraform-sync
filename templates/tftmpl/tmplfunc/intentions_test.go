package tmplfunc

import (
	//"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIntentionsQuery(t *testing.T) {
	//t.parrallel()

	cases := []struct {
		name string
		opts []string
		exp  *intentionsQuery
		err  bool
	}{
		{
			"no opts",
			[]string{},
			&intentionsQuery{},
			false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			act, err := newIntentionsQuery(tc.opts)
			if tc.err {
				assert.Error(t, err)
				return
			}

			if act != nil {
				act.stopCh = nil
			}

			assert.NoError(t, err, err)
			assert.Equal(t, tc.exp, act)
		})
	}
}

func TestIntentionsQuery_String(t *testing.T) {
	//t.parrallel()

	cases := []struct {
		name string
		i    []string
		exp  string
	}{
		{
			"empty",
			[]string{},
			"intentions.services",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d, err := newIntentionsQuery(tc.i)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, tc.exp, d.String())
		})
	}
}

func TestNewIntentionsQuery(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		opts []string
		exp  *intentionsQuery
		err  bool
	}{
		{
			"no opts",
			[]string{},
			&intentionsQuery{},
			false,
		},
		// {
		// 	"regexp",
		// 	[]string{"regexp=.*"},
		// 	&intentionsQuery{
		// 		regexp: regexp.MustCompile(".*"),
		// 	},
		// 	false,
		// },
		// {
		// 	"dc",
		// 	[]string{"dc=dc1"},
		// 	&intentionsQuery{
		// 		dc: "dc1",
		// 	},
		// 	false,
		// },
		// {
		// 	"ns",
		// 	[]string{"ns=namespace"},
		// 	&intentionsQuery{
		// 		ns: "namespace",
		// 	},
		// 	false,
		// },
		{
			"node-meta",
			[]string{"node-meta=k:v", "node-meta=foo:bar"},
			&intentionsQuery{
				nodeMeta: map[string]string{"k": "v", "foo": "bar"},
			},
			false,
		},
		// {
		// 	"multiple",
		// 	[]string{"node-meta=k:v", "ns=namespace", "dc=dc1", "regexp=.*"},
		// 	&intentionsQuery{
		// 		regexp:   regexp.MustCompile(".*"),
		// 		dc:       "dc1",
		// 		ns:       "namespace",
		// 		nodeMeta: map[string]string{"k": "v"},
		// 	},
		// 	false,
		// },
		{
			"invalid query",
			[]string{"invalid=true"},
			nil,
			true,
		},
		{
			"invalid query format",
			[]string{"dc1"},
			nil,
			true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			act, err := newIntentionsQuery(tc.opts)
			if tc.err {
				assert.Error(t, err)
				return
			}

			if act != nil {
				act.stopCh = nil
			}

			assert.NoError(t, err, err)
			assert.Equal(t, tc.exp, act)
		})
	}
}
