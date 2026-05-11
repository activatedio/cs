// Package yaml support cs sources from yaml files
package yaml

import (
	"bytes"
	"io"
	"os"

	"github.com/activatedio/cs"
	"gopkg.in/yaml.v3"
)

// Option configures a yaml source.
type Option func(*options)

type options struct {
	expandEnv bool
}

// ExpandEnv enables ${VAR} and $VAR expansion on string leaves of the
// decoded yaml tree using os.ExpandEnv semantics (unset variables expand
// to an empty string). Expansion is applied recursively to values inside
// maps and slices.
func ExpandEnv() Option {
	return func(o *options) {
		o.expandEnv = true
	}
}

func buildOptions(opts []Option) options {
	var o options
	for _, opt := range opts {
		opt(&o)
	}
	return o
}

// FromPath creates a new source by parsing a yaml file at the given path
//
// A non-empty keyPrefix will prepend the prefix to stored keys, in format [keyPrefix].[key]
func FromPath(path, keyPrefix string, opts ...Option) cs.Source {
	o := buildOptions(opts)
	return func() (string, any, error) {

		res := map[string]any{}

		f, err := os.Open(path) //nolint:gosec // users of this library should never use user input for this value

		if err != nil {
			return "", nil, err
		}

		defer f.Close()

		err = yaml.NewDecoder(f).Decode(&res)

		if err != nil {
			return "", nil, err
		}

		if o.expandEnv {
			expandTree(res)
		}

		return keyPrefix, res, nil
	}
}

// FromReader reads from the reader into a byte array
func FromReader(r io.Reader, keyPrefix string, opts ...Option) cs.Source {

	bs, err := io.ReadAll(r)
	if err != nil {
		panic(err)
	}

	o := buildOptions(opts)

	return func() (string, any, error) {

		res := map[string]any{}

		err = yaml.NewDecoder(bytes.NewReader(bs)).Decode(&res)

		if err != nil {
			return "", nil, err
		}

		if o.expandEnv {
			expandTree(res)
		}

		return keyPrefix, res, nil
	}
}

// expandTree walks v in place and applies os.ExpandEnv to every string
// leaf reachable through map[string]any or []any containers. Other types
// are left untouched.
func expandTree(v any) {
	switch t := v.(type) {
	case map[string]any:
		for k, child := range t {
			if s, ok := child.(string); ok {
				t[k] = os.ExpandEnv(s)
				continue
			}
			expandTree(child)
		}
	case []any:
		for i, child := range t {
			if s, ok := child.(string); ok {
				t[i] = os.ExpandEnv(s)
				continue
			}
			expandTree(child)
		}
	}
}
