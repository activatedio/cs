package config

// NewConfig returns a new config object
func NewConfig() Config {
	return newCachedConfig()
}

// config is the global config object
var global = newCachedConfig()

// AddSource adds a source to build the root config object. Sources are invoked in the order they are added.
// Sources added later take predecent over sources added earlier
func AddSource(src Source) {
	global.AddSource(src)
}

// AddLateBindingSource adds a source which is consulted at read time, meaning each property present on the
// underlying results are looked up again with provided keys
func AddLateBindingSource(src LateBindingSource) {
	global.AddLateBindingSource(src)
}

// Read reads value from the key and assigns it to the provided object, which must be a pointer to a supported value
// supported values are all primitives and a map
func Read(key string, into any) error {
	return global.Read(key, into)
}

// MustRead reads and panics on error
func MustRead(key string, into any) {
	global.MustRead(key, into)
}
