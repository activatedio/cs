package cs

import (
	"errors"
	"reflect"
)

// New returns a new cs object
func New() Config {
	return newCachedConfig()
}

// Get retrieves the value of type T associated with the specified key from the provided Config. Returns an error if retrieval fails.
func Get[T any](c Config, key string) (*T, error) {

	var res *T

	rt := reflect.TypeFor[T]()

	switch rt.Kind() {
	case reflect.Ptr:
		return nil, errors.New("type parameter must not be a pointer")
	case reflect.Map:
		tmp := reflect.MakeMap(rt).Interface().(T)
		res = &tmp
	default:
		res = new(T)
	}

	err := c.Read(key, res)

	return res, err
}

// MustGet retrieves the value of type T associated with the specified key from the provided Config. It panics on errors.
func MustGet[T any](c Config, key string) *T {

	res, err := Get[T](c, key)
	if err != nil {
		panic(err)
	}
	return res
}
