// Package yaml support cs sources from yaml files
package yaml

import (
	"os"

	"github.com/activatedio/cs"
	"gopkg.in/yaml.v3"
)

// NewSourceFromPath creates a new source by parsing a yaml file at the given path
//
// A non-empty keyPrefix will prepend the prefix to stored keys, in format [keyPrefix].[key]
func NewSourceFromPath(path, keyPrefix string) cs.Source {
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

		return keyPrefix, res, nil
	}
}
