package cs

// Source and LateBindingSource can return
// map[string]any
// struct
// any primitive type, expect byte, or uintptr
// slice of any of the above types

// Source returns a key, the cs object, and an error
type Source func() (string, any, error)

// LateBindingSource source returns a cs value for a given key at the time a csuration is read
type LateBindingSource func(key string) (any, error)

// Config is main interface for cs data.  Keys are in dot format, `prefix.name`
type Config interface {

	// AddDefaultSource adds a source that provides default configuration values, used as a fallback if no other source provides a value.
	AddDefaultSource(src Source)

	// AddSource adds a source to build the root cs object. Sources are invoked in the order they are added.
	// Sources added later take precedent over sources added earlier
	AddSource(src Source)

	// AddLockedSource adds a source whose values are locked: once applied, no
	// other source — and no late-binding (env) source — may override any key
	// the source provides, nor any descendant of it ("from that point of the
	// graph down"). Locked values bypass the zero-value skip, so a lock to
	// false / 0 / "" is honored, and render with a [locked] marker in Dump.
	AddLockedSource(src Source)

	// AddLateBindingSource adds a source which is consulted at read time, meaning each property present on the
	// underlying results are looked up again with provided keys
	AddLateBindingSource(src LateBindingSource)

	// Read reads value from the key and assigns it to the provided object, which must be a pointer to a supported value
	// supported values are all primitives and a map
	Read(key string, into any) error

	// MustRead reads and panics on error
	MustRead(key string, into any)

	// SetValidatingHook sets a custom hook for validation. By default the struct is checked to implement Validating and
	// if so the method is invoked
	SetValidatingHook(func(in any) error)

	// Dump dumps out the config and metadata
	// Default dumps to stdout with descriptions and values
	Dump(opts ...DumpOption) error
}

// Validating marks a struct with the ability to validate itself after being unmarshalled
type Validating interface {
	Validate() error
}

const (
	// DescriptionKey is a reserved key for a map source to provide a metadata description
	DescriptionKey = "_description"
	// DescriptionTagName name of the description tag
	DescriptionTagName = "description"
	// KeyTagName name of the key tag
	KeyTagName = "key"
)

// Entry allows for a description to be applied via a tag to this struct
type Entry struct {
}
