package cs_test

import (
	"bytes"
	"testing"

	"github.com/activatedio/cs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCs_Dump(t *testing.T) {

	r := require.New(t)
	a := assert.New(t)

	cases := []struct {
		name           string
		arrange        func() cs.Config
		expectedResult string
	}{
		{
			name:           "empty",
			arrange:        cs.New,
			expectedResult: ``,
		},
		{
			name: "simple",
			arrange: func() cs.Config {
				c := cs.New()
				c.AddSource(func() (string, any, error) {
					return key1, value1, nil
				})
				c.AddSource(func() (string, any, error) {
					return key2, value2, nil
				})
				return c
			},
			expectedResult: `key1: (value1)
key2: (value2)
`,
		},
		{
			name: "full",
			arrange: func() cs.Config {
				c := cs.New()
				c.AddSource(func() (string, any, error) {
					return key1, simple1, nil
				})
				c.AddSource(func() (string, any, error) {
					return key2, complex1, nil
				})
				return c
			},
			expectedResult: `key1:
  value1: (a)
  value2: (2)
  value3: (true)
key2: Top ComplexConfig
  value0: value0 desc (0)
  value1: value1 desc
    [0]: (a)
    [1]: (b)
    [2]: (c)
  value2: value2 desc
    [0]: (2)
    [1]: (3)
    [2]: (4)
  value3: value3 desc
    [0]: (true)
    [1]: (false)
    [2]: (true)
  value4: value4 desc
    value0: value0 desc ()
    value1: value1 desc
      [0]: (x)
      [1]: (y)
      [2]: (z)
    value2: value2 desc
      [0]: (99)
      [1]: (100)
      [2]: (101)
    value3: value3 desc
    value4: value4 desc
      value0: value0 desc ()
      value1: value1 desc
      value2: value2 desc
      value3: value3 desc
      value5: value5 desc
    value5: value5 desc
  value5: value5 desc
    [0]: Top ComplexConfig
      value0: value0 desc ()
      value1: value1 desc
        [0]: (d)
        [1]: (e)
        [2]: (f)
      value2: value2 desc
      value3: value3 desc
      value5: value5 desc
    [1]: Top ComplexConfig
      value0: value0 desc ()
      value1: value1 desc
        [0]: (g)
        [1]: (h)
        [2]: (i)
      value2: value2 desc
      value3: value3 desc
      value5: value5 desc
`,
		},
		{
			name: "full with late binding",
			arrange: func() cs.Config {
				c := cs.New()
				c.AddSource(func() (string, any, error) {
					return key1, simple1, nil
				})
				c.AddSource(func() (string, any, error) {
					return key2, complex1, nil
				})
				c.AddLateBindingSource(func(key string) (any, error) {

					switch key {
					case "key1.value1":
						return "late binding value 1", nil
					case "key2.value0":
						return "late binding value 0", nil
					case "key2.value2[2]":
						return "9999", nil
					default:
						return nil, nil
					}

				})
				return c
			},
			expectedResult: `key1:
  value1: (late binding value 1)
  value2: (2)
  value3: (true)
key2: Top ComplexConfig
  value0: value0 desc (late binding value 0)
  value1: value1 desc
    [0]: (a)
    [1]: (b)
    [2]: (c)
  value2: value2 desc
    [0]: (2)
    [1]: (3)
    [2]: (9999)
  value3: value3 desc
    [0]: (true)
    [1]: (false)
    [2]: (true)
  value4: value4 desc
    value0: value0 desc ()
    value1: value1 desc
      [0]: (x)
      [1]: (y)
      [2]: (z)
    value2: value2 desc
      [0]: (99)
      [1]: (100)
      [2]: (101)
    value3: value3 desc
    value4: value4 desc
      value0: value0 desc ()
      value1: value1 desc
      value2: value2 desc
      value3: value3 desc
      value5: value5 desc
    value5: value5 desc
  value5: value5 desc
    [0]: Top ComplexConfig
      value0: value0 desc ()
      value1: value1 desc
        [0]: (d)
        [1]: (e)
        [2]: (f)
      value2: value2 desc
      value3: value3 desc
      value5: value5 desc
    [1]: Top ComplexConfig
      value0: value0 desc ()
      value1: value1 desc
        [0]: (g)
        [1]: (h)
        [2]: (i)
      value2: value2 desc
      value3: value3 desc
      value5: value5 desc
`,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(_ *testing.T) {

			unit := tt.arrange()
			buf := &bytes.Buffer{}
			r.NoError(unit.Dump(cs.WithDumpOut(buf)))
			a.Equal(tt.expectedResult, buf.String())

		})
	}
}
