package cs

import (
	"fmt"
	"reflect"
	"strings"
)

// Error represents a structure for detailed error handling with config key metadata.
type Error struct {
	// Key specifies the config key requested which produced the error
	Key string
	// TypeName indicates the type associated with the error.
	TypeName string
	// Detail contains the underlying error providing further information.
	Detail error
}

func (e Error) Error() string {
	sb := strings.Builder{}

	sb.WriteString(fmt.Sprintf("type: %s ", e.TypeName))

	if e.Key != "" {
		sb.WriteString(fmt.Sprintf("key: %s ", e.Key))
	}
	if e.Detail != nil {
		sb.WriteString(fmt.Sprintf("detail: %s", e.Detail.Error()))
	}

	return sb.String()
}

func wrapError[T any](in T, key string, err error) error {

	return Error{
		Key:      key,
		TypeName: reflect.TypeOf(in).Elem().Name(),
		Detail:   err,
	}
}
