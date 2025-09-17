package cs_test

import (
	"errors"
	"testing"

	"github.com/activatedio/cs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type SimpleConfig struct {
	Value1 string
	Value2 int
	Value3 bool
}

type Validating struct {
	Value1 string
}

func (v *Validating) Validate() error {
	if v.Value1 == "error" {
		return errors.New("validation failed")
	}
	return nil
}

func TestConfig(t *testing.T) {

	a := assert.New(t)
	r := require.New(t)

	type s struct {
		arrange func(c cs.Config)
		assert  func(c cs.Config)
	}

	const key1 = "key1"
	const key2 = "key2"
	const value1 = "value1"
	const value2 = "value2"
	cases := map[string]s{
		"empty": {
			arrange: func(_ cs.Config) {
			},
			assert: func(c cs.Config) {
				var got1 string
				var got2 string
				c.MustRead(key1, &got1)
				c.MustRead(key2, &got2)
				a.Empty(got1)
				a.Empty(got2)
			},
		},
		"simple strings": {
			arrange: func(c cs.Config) {
				c.AddSource(func() (string, any, error) {
					return key1, value1, nil
				})
				c.AddSource(func() (string, any, error) {
					return key2, value2, nil
				})
			},
			assert: func(c cs.Config) {
				var got1 string
				var got2 string
				c.MustRead(key1, &got1)
				c.MustRead(key2, &got2)
				a.Equal(value1, got1)
				a.Equal(value2, got2)

				// test other accessors
				got1 = ""
				got2 = ""
				got1 = cs.MustGet[string](c, key1)
				got2 = cs.MustGet[string](c, key2)
				a.Equal(value1, got1)
				a.Equal(value2, got2)
				got1 = ""
				var err error
				got1, err = cs.Get[string](c, key1)
				r.NoError(err)
				a.Equal(value1, got1)
			},
		},
		"simple structs": {
			arrange: func(c cs.Config) {
				c.AddSource(func() (string, any, error) {
					return key1, &SimpleConfig{
						Value1: "a",
						Value2: 2,
						Value3: true,
					}, nil
				})
				c.AddSource(func() (string, any, error) {
					return key2, &SimpleConfig{
						Value1: "d",
						Value2: 3,
						Value3: false,
					}, nil
				})
			},
			assert: func(c cs.Config) {
				var got1 string
				var got2 string

				// We can read individual strings
				c.MustRead("key1.value1", &got1)
				c.MustRead("key2.value1", &got2)
				a.Equal("a", got1)
				a.Equal("d", got2)

				got3 := &SimpleConfig{}
				c.MustRead(key1, got3)
				a.Equal(&SimpleConfig{
					Value1: "a",
					Value2: 2,
					Value3: true,
				}, got3)

				got1 = ""
				got2 = ""

				got1 = cs.MustGet[string](c, "key1.value1")
				got2 = cs.MustGet[string](c, "key2.value1")
				a.Equal("a", got1)
				a.Equal("d", got2)

				got3val := cs.MustGet[SimpleConfig](c, key1)
				got3 = &got3val
				a.Equal(&SimpleConfig{
					Value1: "a",
					Value2: 2,
					Value3: true,
				}, got3)

			},
		},
		"validating struct - pass": {
			arrange: func(c cs.Config) {
				c.AddSource(func() (string, any, error) {
					return key1, &Validating{
						Value1: "a",
					}, nil
				})
			},
			assert: func(c cs.Config) {
				got := &Validating{}
				err := c.Read(key1, got)
				r.NoError(err)
				a.Equal(&Validating{
					Value1: "a",
				}, got)
			},
		},
		"validating struct - error": {
			arrange: func(c cs.Config) {
				c.AddSource(func() (string, any, error) {
					return key1, &Validating{
						Value1: "error",
					}, nil
				})
			},
			assert: func(c cs.Config) {
				got := &Validating{}
				err := c.Read(key1, got)
				r.EqualError(err, "type: Validating key: key1 detail: validation failed")
			},
		},
		"validating struct - custom validation hook with error": {
			arrange: func(c cs.Config) {
				c.AddSource(func() (string, any, error) {
					return key1, &Validating{
						Value1: "a",
					}, nil
				})
				c.SetValidatingHook(func(in any) error {
					r.Equal(&Validating{
						Value1: "a",
					}, in)
					return errors.New("validation failed - custom")
				})
			},
			assert: func(c cs.Config) {
				got := &Validating{}
				err := c.Read(key1, got)
				r.EqualError(err, "type: Validating key: key1 detail: validation failed - custom")
			},
		},
		"simple maps": {
			arrange: func(c cs.Config) {
				c.AddSource(func() (string, any, error) {
					return key1, map[string]any{
						value1:   "a",
						value2:   2,
						"value3": true,
					}, nil
				})
				c.AddSource(func() (string, any, error) {
					return key2, map[string]any{
						value1:   "d",
						value2:   3,
						"value3": false,
					}, nil
				})
			},
			assert: func(c cs.Config) {
				var got1 string
				var got2 string
				// We can read individual strings
				c.MustRead("key1.value1", &got1)
				c.MustRead("key2.value1", &got2)
				a.Equal("a", got1)
				a.Equal("d", got2)

				got3 := map[string]any{}
				c.MustRead(key1, &got3)
				a.Equal(map[string]any{
					value1:   "a",
					value2:   2,
					"value3": true,
				}, got3)

			},
		},
		"simple overrides": {
			arrange: func(c cs.Config) {
				c.AddSource(func() (string, any, error) {
					return key1, &SimpleConfig{
						Value1: "a",
						Value2: 2,
						Value3: true,
					}, nil
				})
				c.AddSource(func() (string, any, error) {
					return "key1.value2", 3, nil
				})
			},
			assert: func(c cs.Config) {
				var got1 int
				// We can read individual strings
				c.MustRead("key1.value2", &got1)
				a.Equal(3, got1)
			},
		},
		"late bindings overrides": {
			arrange: func(c cs.Config) {
				c.AddSource(func() (string, any, error) {
					return key1, &SimpleConfig{
						Value1: "a",
						Value2: 2,
						Value3: true,
					}, nil
				})
				c.AddLateBindingSource(func(key string) (any, error) {
					if key == "key1.value2" {
						return 3, nil
					}
					return nil, nil
				})
			},
			assert: func(c cs.Config) {
				var got1 int
				// We can read individual strings
				c.MustRead("key1.value2", &got1)
				a.Equal(3, got1)
			},
		},
	}

	for k, v := range cases {
		t.Run(k, func(_ *testing.T) {
			unit := cs.New()
			v.arrange(unit)
			// Run assert twice to check for caching
			v.assert(unit)
			v.assert(unit)
		})
	}
}
