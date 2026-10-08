package cs_test

import (
	"strings"
	"testing"

	"github.com/activatedio/cs"
	"github.com/activatedio/cs/sources"
	"github.com/activatedio/cs/sources/yaml"
	"github.com/stretchr/testify/assert"
)

// DefaultOnConfig has a bool whose default is true — the case a zero-skipping
// merge could never turn off — beside a string and a pointer bool.
type DefaultOnConfig struct {
	On   bool   `json:"on"`
	Host string `json:"host"`
	Ptr  *bool  `json:"ptr"`
}

func TestConfig_ExplicitFalse(t *testing.T) {

	const prefix = "x"

	f := false

	type s struct {
		arrange func(c cs.Config)
		assert  func(a *assert.Assertions, got DefaultOnConfig)
	}

	cases := map[string]s{
		"a file's false beats a default true": {
			arrange: func(c cs.Config) {
				c.AddSource(yaml.FromReader(strings.NewReader("x:\n  on: false\n"), ""))
			},
			assert: func(a *assert.Assertions, got DefaultOnConfig) {
				a.False(got.On)
			},
		},
		"a file's true keeps a default true": {
			arrange: func(c cs.Config) {
				c.AddSource(yaml.FromReader(strings.NewReader("x:\n  on: true\n"), ""))
			},
			assert: func(a *assert.Assertions, got DefaultOnConfig) {
				a.True(got.On)
			},
		},
		"a file that omits the key keeps a default true": {
			arrange: func(c cs.Config) {
				c.AddSource(yaml.FromReader(strings.NewReader("x:\n  host: h\n"), ""))
			},
			assert: func(a *assert.Assertions, got DefaultOnConfig) {
				a.True(got.On)
				a.Equal("h", got.Host)
			},
		},
		"a scalar preset false beats a lower default true": {
			arrange: func(c cs.Config) {
				c.AddDefaultSource(sources.FromValue("x.on", false))
			},
			assert: func(a *assert.Assertions, got DefaultOnConfig) {
				a.False(got.On)
			},
		},
		"a struct source's zero bool field is unset, not false": {
			arrange: func(c cs.Config) {
				c.AddSource(sources.FromValue(prefix, &DefaultOnConfig{Host: "h"}))
			},
			assert: func(a *assert.Assertions, got DefaultOnConfig) {
				a.True(got.On)
				a.Equal("h", got.Host)
			},
		},
		"a struct source's pointer false is stated": {
			arrange: func(c cs.Config) {
				c.AddSource(sources.FromValue(prefix, &DefaultOnConfig{Ptr: &f}))
			},
			assert: func(a *assert.Assertions, got DefaultOnConfig) {
				if a.NotNil(got.Ptr) {
					a.False(*got.Ptr)
				}
			},
		},
		"a file's empty string still reads as unset": {
			arrange: func(c cs.Config) {
				c.AddSource(yaml.FromReader(strings.NewReader("x:\n  host: \"\"\n"), ""))
			},
			assert: func(a *assert.Assertions, got DefaultOnConfig) {
				a.Equal("default-host", got.Host)
			},
		},
	}

	for k, v := range cases {
		t.Run(k, func(t *testing.T) {
			tr := true
			c := cs.New()
			// Arrange first: a default registered before this one is
			// prepended after it, so it outranks it — how a runner gives a
			// profile preset precedence over the code default.
			v.arrange(c)
			c.AddDefaultSource(sources.FromValue(prefix, &DefaultOnConfig{On: true, Host: "default-host", Ptr: &tr}))
			v.assert(assert.New(t), *cs.MustGet[DefaultOnConfig](c, prefix))
		})
	}
}
