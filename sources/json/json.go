// Package json supports cs sources from json files
package json

import (
	"bytes"
	"encoding/json"
	"io"
	"os"

	"github.com/activatedio/cs"
)

// FromPath creates a new source by parsing a json file at the given path
//
// A non-empty keyPrefix will prepend the prefix to stored keys, in format [keyPrefix].[key]
func FromPath(path, keyPrefix string) cs.Source {
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

// FromReader reads from the reader into a byte array
func FromReader(r io.Reader, keyPrefix string) cs.Source {

	bs, err := io.ReadAll(r)
	if err != nil {
		panic(err)
	}

	return func() (string, any, error) {

		res := map[string]any{}

		err = json.NewDecoder(bytes.NewBuffer(bs)).Decode(&res)

		if err != nil {
			return "", nil, err
		}

		return keyPrefix, res, nil
	}
}
