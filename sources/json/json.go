// Package json supports config sources from json files
package json

import (
	"encoding/json"
	"os"

	"github.com/activatedio/config"
)

// NewSourceFromPath creates a new source by parsing a json file at the given path
//
// A non-empty keyPrefix will prepend the prefix to stored keys, in format [keyPrefix].[key]
func NewSourceFromPath(path, keyPrefix string) config.Source {
	return func() (string, any, error) {

		res := map[string]any{}

		f, err := os.Open(path) //nolint:gosec // users of this library should never use user input for this value

		if err != nil {
			return "", nil, err
		}

		defer f.Close()

		err = json.NewDecoder(f).Decode(&res)

		if err != nil {
			return "", nil, err
		}

		return keyPrefix, res, nil
	}
}
