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

	const (
		prefix           = "session"
		keyDisableSecure = "session.disableSecure"
		keyMode          = "session.mode"
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
					return keyMode, "", nil
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
					case keyMode:
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
					return keyMode, "FileSystem", nil
				})
				c.AddLateBindingSource(func(key string) (any, error) {
					if key == keyMode {
						return "Env", nil
					}
					return nil, nil
				})
			},
			assert: func(c cs.Config) {
				var got string
				c.MustRead(keyMode, &got)
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
		"root lock beats env for every descendant": {
			arrange: func(c cs.Config) {
				c.AddSource(func() (string, any, error) {
					return prefix, &LockConfig{Mode: "fromConfig"}, nil
				})
				// Lock the entire config root.
				c.AddLockedSource(func() (string, any, error) {
					return "", map[string]any{"session": map[string]any{"mode": "locked"}}, nil
				})
				c.AddLateBindingSource(func(key string) (any, error) {
					if key == keyMode {
						return "env", nil
					}
					return nil, nil
				})
			},
			assert: func(c cs.Config) {
				got := cs.MustGet[LockConfig](c, prefix)
				a.Equal("locked", got.Mode, "a root-level lock must cover all descendants")
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
}

// DescConfig carries a description tag so we can assert the Dump audit line
// keeps it after a lock replaces the value.
type DescConfig struct {
	Mode string `description:"the storage mode"`
}

// TestConfig_LockedSource_Dump asserts the [locked] marker in the Dump output
// covers exactly what isLocked enforces — including untouched siblings under a
// subtree lock — and that locking a field preserves its description.
func TestConfig_LockedSource_Dump(t *testing.T) {

	a := assert.New(t)
	r := require.New(t)

	cases := map[string]struct {
		arrange func(c cs.Config)
		want    []string
		notWant []string
	}{
		"subtree lock marks every descendant including untouched siblings": {
			arrange: func(c cs.Config) {
				c.AddSource(func() (string, any, error) {
					return "session", &LockConfig{DisableSecure: true, SameSiteMode: "lax"}, nil
				})
				// Partial map keyed at the parent locks the whole session subtree.
				c.AddLockedSource(func() (string, any, error) {
					return "session", map[string]any{"disableSecure": false}, nil
				})
			},
			// disableSecure is locked-and-set; sameSiteMode is untouched by the
			// lock map but still under the locked prefix — it must be marked.
			want: []string{"disableSecure: [locked] (false)", "sameSiteMode: [locked] (lax)"},
		},
		"leaf lock marks only the leaf": {
			arrange: func(c cs.Config) {
				c.AddSource(func() (string, any, error) {
					return "session", &LockConfig{DisableSecure: true, SameSiteMode: "lax"}, nil
				})
				c.AddLockedSource(func() (string, any, error) {
					return "session.disableSecure", false, nil
				})
			},
			want:    []string{"disableSecure: [locked] (false)", "sameSiteMode: (lax)"},
			notWant: []string{"sameSiteMode: [locked]"},
		},
		"locked field keeps its description": {
			arrange: func(c cs.Config) {
				c.AddSource(func() (string, any, error) {
					return "asset", &DescConfig{Mode: "fromConfig"}, nil
				})
				c.AddLockedSource(func() (string, any, error) {
					return "asset.mode", "locked", nil
				})
			},
			want: []string{"mode: the storage mode [locked] (locked)"},
		},
	}

	for k, v := range cases {
		t.Run(k, func(_ *testing.T) {
			unit := cs.New()
			v.arrange(unit)
			buf := &bytes.Buffer{}
			r.NoError(unit.Dump(cs.WithDumpOut(buf)))
			out := buf.String()
			for _, w := range v.want {
				a.Contains(out, w)
			}
			for _, nw := range v.notWant {
				a.NotContains(out, nw)
			}
		})
	}
}

// TestConfig_LockedSource_SliceIndexRejected asserts a lock keyed at a slice
// index fails loud rather than silently no-opping (the merge cannot descend
// into a slice by index — lock the parent key instead).
func TestConfig_LockedSource_SliceIndexRejected(t *testing.T) {
	r := require.New(t)

	unit := cs.New()
	unit.AddSource(func() (string, any, error) {
		return "top", &LockConfig{Mode: "x"}, nil
	})
	unit.AddLockedSource(func() (string, any, error) {
		return "top.servers[0].host", "y", nil
	})

	var got string
	err := unit.Read("top.mode", &got)
	r.Error(err)
	r.Contains(err.Error(), "slice index")
}

// TestConfig_LockedSource_CacheInvalidation asserts a lock registered after a
// prior read still takes effect — the cached read must be invalidated.
func TestConfig_LockedSource_CacheInvalidation(t *testing.T) {
	a := assert.New(t)

	unit := cs.New()
	unit.AddSource(func() (string, any, error) {
		return "session", &LockConfig{DisableSecure: true}, nil
	})

	// Read once to populate the read cache.
	var before bool
	unit.MustRead("session.disableSecure", &before)
	a.True(before)

	// Lock added AFTER the read.
	unit.AddLockedSource(func() (string, any, error) {
		return "session.disableSecure", false, nil
	})

	var after bool
	unit.MustRead("session.disableSecure", &after)
	a.False(after, "lock added after a read must still take effect")
}
