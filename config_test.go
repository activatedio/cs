package config_test

import (
	"testing"

	"github.com/activatedio/config"
	"github.com/stretchr/testify/assert"
)

type SimpleConfig struct {
	Value1 string
	Value2 int
	Value3 bool
}

func TestConfig(t *testing.T) {

	a := assert.New(t)

	type s struct {
		arrange func(c config.Config)
		assert  func(c config.Config)
	}

	const key1 = "key1"
	const key2 = "key2"
	const value1 = "value1"
	const value2 = "value2"
	cases := map[string]s{
		"empty": {
			arrange: func(_ config.Config) {
			},
			assert: func(c config.Config) {
				var got1 string
				var got2 string
				c.MustRead(key1, &got1)
				c.MustRead(key2, &got2)
				a.Empty(got1)
				a.Empty(got2)
			},
		},
		"simple strings": {
			arrange: func(c config.Config) {
				c.AddSource(func() (string, any, error) {
					return key1, value1, nil
				})
				c.AddSource(func() (string, any, error) {
					return key2, value2, nil
				})
			},
			assert: func(c config.Config) {
				var got1 string
				var got2 string
				c.MustRead(key1, &got1)
				c.MustRead(key2, &got2)
				a.Equal(value1, got1)
				a.Equal(value2, got2)
			},
		},
		"simple structs": {
			arrange: func(c config.Config) {
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
			assert: func(c config.Config) {
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

			},
		},
		"simple maps": {
			arrange: func(c config.Config) {
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
			assert: func(c config.Config) {
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
			arrange: func(c config.Config) {
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
			assert: func(c config.Config) {
				var got1 int
				// We can read individual strings
				c.MustRead("key1.value2", &got1)
				a.Equal(3, got1)
			},
		},
		"late bindings overrides": {
			arrange: func(c config.Config) {
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
			assert: func(c config.Config) {
				var got1 int
				// We can read individual strings
				c.MustRead("key1.value2", &got1)
				a.Equal(3, got1)
			},
		},
	}

	for k, v := range cases {
		t.Run(k, func(_ *testing.T) {
			unit := config.NewConfig()
			v.arrange(unit)
			// Run assert twice to check for caching
			v.assert(unit)
			v.assert(unit)
		})
	}
}
