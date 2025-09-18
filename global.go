package cs

import "sync"

// cs is the global cs object
var (
	global     Config
	globalLock sync.RWMutex
)

// Global gets the global config instance
func Global() Config {
	globalLock.RLock()

	if global != nil {
		defer globalLock.RUnlock()
		return global
	}
	globalLock.RUnlock()
	globalLock.Lock()
	defer globalLock.Unlock()
	global = New()
	return global
}

// AddDefaultSource adds a source of default configuration values to the global configuration instance.
func AddDefaultSource(src Source) {
	Global().AddDefaultSource(src)
}

// AddSource adds a source to build the root cs object. Sources are invoked in the order they are added.
// Sources added later take precedent over sources added earlier
func AddSource(src Source) {
	Global().AddSource(src)
}

// AddLateBindingSource adds a source which is consulted at read time, meaning each property present on the
// underlying results are looked up again with provided keys
func AddLateBindingSource(src LateBindingSource) {
	Global().AddLateBindingSource(src)
}

// Read reads value from the key and assigns it to the provided object, which must be a pointer to a supported value
// supported values are all primitives and a map
func Read(key string, into any) error {
	return Global().Read(key, into)
}

// MustRead reads and panics on error
func MustRead(key string, into any) {
	Global().MustRead(key, into)
}
