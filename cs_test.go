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

type ComplexConfig struct {
	cs.Entry `description:"Top ComplexConfig"`
	Value0   string          `description:"value0 desc"`
	Value1   []string        `description:"value1 desc"`
	Value2   []int           `description:"value2 desc"`
	Value3   []bool          `description:"value3 desc"`
	Value4   *ComplexConfig  `description:"value4 desc"`
	Value5   []ComplexConfig `description:"value5 desc"`
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

var (
	key1    = "key1"
	key2    = "key2"
	value1  = "value1"
	value2  = "value2"
	simple1 = &SimpleConfig{
		Value1: "a",
		Value2: 2,
		Value3: true,
	}
	simple2 = &SimpleConfig{
		Value1: "d",
		Value2: 3,
		Value3: false,
	}
	complex1 = &ComplexConfig{
		Value0: "0",
		Value1: []string{"a", "b", "c"},
		Value2: []int{2, 3, 4},
		Value3: []bool{true, false, true},
		Value4: &ComplexConfig{
			Value1: []string{"x", "y", "z"},
			Value2: []int{99, 100, 101},
			Value4: &ComplexConfig{},
		},
		Value5: []ComplexConfig{
			{
				Value1: []string{"d", "e", "f"},
			},
			{
				Value1: []string{"g", "h", "i"},
			},
		},
	}
)

func TestConfig_WriteRead(t *testing.T) {

	a := assert.New(t)
	r := require.New(t)

	type s struct {
		arrange func(c cs.Config)
		assert  func(c cs.Config)
	}

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
				got1 = *cs.MustGet[string](c, key1)
				got2 = *cs.MustGet[string](c, key2)
				a.Equal(value1, got1)
				a.Equal(value2, got2)
				got1 = ""
				var err error
				var tmp *string
				tmp, err = cs.Get[string](c, key1)
				got1 = *tmp
				r.NoError(err)
				a.Equal(value1, got1)
			},
		},
		"simple structs": {
			arrange: func(c cs.Config) {
				c.AddSource(func() (string, any, error) {
					return key1, simple1, nil
				})
				c.AddSource(func() (string, any, error) {
					return key2, simple2, nil
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

				got1 = *cs.MustGet[string](c, "key1.value1")
				got2 = *cs.MustGet[string](c, "key2.value1")
				a.Equal("a", got1)
				a.Equal("d", got2)

				got3val := cs.MustGet[SimpleConfig](c, key1)
				got3 = got3val
				a.Equal(&SimpleConfig{
					Value1: "a",
					Value2: 2,
					Value3: true,
				}, got3)

			},
		},
		"complex structs": {
			arrange: func(c cs.Config) {
				c.AddSource(func() (string, any, error) {
					return key1, complex1, nil
				})
			},
			assert: func(c cs.Config) {
				var got1 []string

				// We can read individual strings
				c.MustRead("key1.value1", &got1)
				a.Equal([]string{"a", "b", "c"}, got1)

				got2 := &ComplexConfig{}
				c.MustRead(key1, got2)
				a.Equal(&ComplexConfig{
					Value0: "0",
					Value1: []string{"a", "b", "c"},
					Value2: []int{2, 3, 4},
					Value3: []bool{true, false, true},
					Value4: &ComplexConfig{
						Value1: []string{"x", "y", "z"},
						Value2: []int{99, 100, 101},
						Value4: &ComplexConfig{},
					},
					Value5: []ComplexConfig{
						{
							Value1: []string{"d", "e", "f"},
						},
						{
							Value1: []string{"g", "h", "i"},
						},
					},
				}, got2)

				var got3 string

				// We can read individual strings
				c.MustRead("key1.value1[1]", &got3)
				a.Equal("b", got3)
				c.MustRead("key1.value5[1].value1[1]", &got3)
				a.Equal("h", got3)

				// Check the root map
				mapRef1 := map[string]any{
					"key1": map[string]any{
						"value0": "0",
						"value1": []string{"a", "b", "c"},
						"value2": []int{2, 3, 4},
						"value3": []bool{true, false, true},
						"value4": map[string]any{
							"value0": "",
							"value1": []string{"x", "y", "z"},
							"value2": []int{99, 100, 101},
							"value3": []bool(nil),
							"value4": map[string]any{
								"value0": "",
								"value1": []string(nil),
								"value2": []int(nil),
								"value3": []bool(nil),
								"value5": []map[string]any(nil),
							},
							"value5": []map[string]any(nil),
						},
						"value5": []map[string]any{
							{
								"value0": "",
								"value1": []string{"d", "e", "f"},
								"value2": []int(nil),
								"value3": []bool(nil),
								"value5": []map[string]any(nil),
							},
							{
								"value0": "",
								"value1": []string{"g", "h", "i"},
								"value2": []int(nil),
								"value3": []bool(nil),
								"value5": []map[string]any(nil),
							},
						},
					},
				}
				mapRes1 := map[string]any{}

				c.MustRead("", &mapRes1)
				a.Equal(mapRef1, mapRes1)

				mapRes1 = *cs.MustGet[map[string]any](c, "")
				a.Equal(mapRef1, mapRes1)

			},
		},
		"default source with override - simple structs": {
			arrange: func(c cs.Config) {
				c.AddSource(func() (string, any, error) {
					return key1, &SimpleConfig{
						Value2: 2,
						Value3: true,
					}, nil
				})
				c.AddDefaultSource(func() (string, any, error) {
					return key1, &SimpleConfig{
						Value1: "a",
						Value2: 1,
						Value3: false,
					}, nil
				})
			},
			assert: func(c cs.Config) {
				a.Equal(&SimpleConfig{
					Value1: "a",
					Value2: 2,
					Value3: true,
				}, cs.MustGet[SimpleConfig](c, key1))
			},
		},
		"default source with override - complex structs": {
			arrange: func(c cs.Config) {
				c.AddSource(func() (string, any, error) {
					return key1, &ComplexConfig{
						Value1: []string{"j", "k", "l"},
					}, nil
				})
				c.AddDefaultSource(func() (string, any, error) {
					return key1, complex1, nil
				})
			},
			assert: func(c cs.Config) {
				a.Equal(&ComplexConfig{
					Value0: "0",
					Value1: []string{"j", "k", "l"},
					Value2: []int{2, 3, 4},
					Value3: []bool{true, false, true},
					Value4: &ComplexConfig{
						Value1: []string{"x", "y", "z"},
						Value2: []int{99, 100, 101},
						Value4: &ComplexConfig{},
					},
					Value5: []ComplexConfig{
						{
							Value1: []string{"d", "e", "f"},
						},
						{
							Value1: []string{"g", "h", "i"},
						},
					},
				}, cs.MustGet[ComplexConfig](c, key1))
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
