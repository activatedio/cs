package sources

import "github.com/activatedio/cs"

// FromValue returns the given value for a cs key
func FromValue(key string, val any) cs.Source {
	return func() (string, any, error) {
		return key, val, nil
	}
}
