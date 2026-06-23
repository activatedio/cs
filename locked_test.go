package cs_test

import (
	"bytes"
	"testing"

	"github.com/activatedio/cs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// LockConfig mirrors the kind of struct kit locks: a security-critical bool, a
// mode string, and a tunable companion field that must stay overridable.
type LockConfig struct {
	DisableSecure bool
	Unsigned      bool
	Mode          string
	SameSiteMode  string
}

func TestConfig_LockedSource(t *testing.T) {

	a := assert.New(t)
	r := require.New(t)

	const (
		prefix           = "session"
		keyDisableSecure = "session.disableSecure"
	)

	type s struct {
		arrange func(c cs.Config)
		assert  func(c cs.Config)
	}

	cases := map[string]s{
		"locked leaf beats a config-file source": {
			arrange: func(c cs.Config) {
				// Operator config tries to turn security off.
				c.AddSource(func() (string, any, error) {
					return prefix, &LockConfig{DisableSecure: true, SameSiteMode: "lax"}, nil
				})
				// Profile locks it back on.
				c.AddLockedSource(func() (string, any, error) {
					return keyDisableSecure, false, nil
				})
			},
			assert: func(c cs.Config) {
				got := cs.MustGet[LockConfig](c, prefix)
				a.False(got.DisableSecure, "locked leaf must win over the config source")
				// A non-locked sibling stays operator-tunable.
				a.Equal("lax", got.SameSiteMode)
			},
		},
		"locked leaf beats a late-binding (env) source": {
			arrange: func(c cs.Config) {
				c.AddSource(func() (string, any, error) {
					return prefix, &LockConfig{DisableSecure: true}, nil
				})
				c.AddLockedSource(func() (string, any, error) {
					return keyDisableSecure, false, nil
				})
				// Simulate SESSION_DISABLE_SECURE=true in the environment.
				c.AddLateBindingSource(func(key string) (any, error) {
					if key == keyDisableSecure {
						return true, nil
					}
					return nil, nil
				})
			},
			assert: func(c cs.Config) {
				got := cs.MustGet[LockConfig](c, prefix)
				a.False(got.DisableSecure, "locked leaf must win over env late binding")
			},
		},
		"late binding still wins for a non-locked sibling": {
			arrange: func(c cs.Config) {
				c.AddSource(func() (string, any, error) {
					return prefix, &LockConfig{DisableSecure: true, SameSiteMode: "lax"}, nil
				})
				c.AddLockedSource(func() (string, any, error) {
					return keyDisableSecure, false, nil
				})
				c.AddLateBindingSource(func(key string) (any, error) {
					if key == "session.sameSiteMode" {
						return "strict", nil
					}
					return nil, nil
				})
			},
			assert: func(c cs.Config) {
				got := cs.MustGet[LockConfig](c, prefix)
				a.False(got.DisableSecure)
				a.Equal("strict", got.SameSiteMode, "non-locked field must still take the env value")
			},
		},
		"lock to a zero value is honored": {
			arrange: func(c cs.Config) {
				// Config sets a non-zero mode; lock forces it to empty string.
				c.AddSource(func() (string, any, error) {
					return prefix, &LockConfig{Mode: "ObjectStorage"}, nil
				})
				c.AddLockedSource(func() (string, any, error) {
					return "session.mode", "", nil
				})
			},
			assert: func(c cs.Config) {
				got := cs.MustGet[LockConfig](c, prefix)
				a.Empty(got.Mode, "a locked zero value must not be dropped by the merge")
			},
		},
		"subtree lock covers descendants and beats env": {
			arrange: func(c cs.Config) {
				c.AddSource(func() (string, any, error) {
					return prefix, &LockConfig{DisableSecure: true, Unsigned: true, Mode: "x"}, nil
				})
				// Lock the whole session subtree.
				c.AddLockedSource(func() (string, any, error) {
					return prefix, map[string]any{
						"disableSecure": false,
						"unsigned":      false,
						"mode":          "FileSystem",
					}, nil
				})
				c.AddLateBindingSource(func(key string) (any, error) {
					switch key {
					case "session.disableSecure":
						return true, nil
					case "session.mode":
						return "Env", nil
					}
					return nil, nil
				})
			},
			assert: func(c cs.Config) {
				got := cs.MustGet[LockConfig](c, prefix)
				a.False(got.DisableSecure)
				a.False(got.Unsigned)
				a.Equal("FileSystem", got.Mode, "subtree lock must beat env for every descendant")
			},
		},
		"locked value present with no other source": {
			arrange: func(c cs.Config) {
				c.AddLockedSource(func() (string, any, error) {
					return "session.mode", "FileSystem", nil
				})
				c.AddLateBindingSource(func(key string) (any, error) {
					if key == "session.mode" {
						return "Env", nil
					}
					return nil, nil
				})
			},
			assert: func(c cs.Config) {
				var got string
				c.MustRead("session.mode", &got)
				a.Equal("FileSystem", got)
			},
		},
		"normalizes key case so PascalCase lock matches": {
			arrange: func(c cs.Config) {
				c.AddSource(func() (string, any, error) {
					return prefix, &LockConfig{DisableSecure: true}, nil
				})
				// Lock authored with a leading-cap leaf segment.
				c.AddLockedSource(func() (string, any, error) {
					return "session.DisableSecure", false, nil
				})
				c.AddLateBindingSource(func(key string) (any, error) {
					if key == keyDisableSecure {
						return true, nil
					}
					return nil, nil
				})
			},
			assert: func(c cs.Config) {
				got := cs.MustGet[LockConfig](c, prefix)
				a.False(got.DisableSecure, "lock key must match regardless of segment case")
			},
		},
	}

	for k, v := range cases {
		t.Run(k, func(_ *testing.T) {
			unit := cs.New()
			v.arrange(unit)
			// Run assert twice to exercise the read cache.
			v.assert(unit)
			v.assert(unit)
		})
	}

	// Dump renders a [locked] marker for locked leaves and leaves non-locked
	// values unmarked.
	t.Run("dump marks locked leaves", func(_ *testing.T) {
		unit := cs.New()
		unit.AddSource(func() (string, any, error) {
			return prefix, &LockConfig{DisableSecure: true, SameSiteMode: "lax"}, nil
		})
		unit.AddLockedSource(func() (string, any, error) {
			return keyDisableSecure, false, nil
		})

		buf := &bytes.Buffer{}
		r.NoError(unit.Dump(cs.WithDumpOut(buf)))
		out := buf.String()

		a.Contains(out, "disableSecure: [locked] (false)")
		// A non-locked sibling carries no marker.
		a.Contains(out, "sameSiteMode: (lax)")
		a.NotContains(out, "sameSiteMode: [locked]")
	})
}
