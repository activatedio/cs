package config

// Source and LateBindingSource can return
// map[string]any
// struct
// any primitive type, expect byte, or uintptr
// slice of any of the above types

// Source returns a key, the config object, and an error
type Source func() (string, any, error)

// LateBindingSource source returns a config value for a given key at the time a configuration is read
type LateBindingSource func(key string) (any, error)

// Config is main interface for config data.  Keys are in dot format, `prefix.name`
type Config interface {

	// AddSource adds a source to build the root config object. Sources are invoked in the order they are added.
	// Sources added later take predecent over sources added earlier
	AddSource(src Source)

	// AddLateBindingSource adds a source which is consulted at read time, meaning each property present on the
	// underlying results are looked up again with provided keys
	AddLateBindingSource(src LateBindingSource)

	// Read reads value from the key and assigns it to the provided object, which must be a pointer to a supported value
	// supported values are all primitives and a map
	Read(key string, into any) error

	// MustRead reads and panics on error
	MustRead(key string, into any)
}
