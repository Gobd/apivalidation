package apivalidation

import (
	"errors"
	"reflect"

	"github.com/getkin/kin-openapi/openapi3"
)

type uniqueRule struct {
	f    func(a int) any
	desc string
}

func (r uniqueRule) Describe(_ string, _ *openapi3.Schema, ref *openapi3.SchemaRef) error {
	ref.Value.UniqueItems = true
	return nil
}

// Unique returns a validation rule that checks if all elements in a slice are unique according to f.
func Unique(f func(a int) any, desc string) Rule {
	return uniqueRule{
		desc: desc,
		f:    f,
	}
}

// Validate checks if the given value is valid or not.
func (r uniqueRule) Validate(value any) error {
	m := make(map[any]struct{})
	rv := reflect.ValueOf(value)
	if (rv.Kind() == reflect.Ptr || rv.Kind() == reflect.Interface) && rv.IsNil() {
		return nil
	}

	rv = reflect.Indirect(rv)

	switch rv.Kind() {
	case reflect.Slice, reflect.Array:
		l := rv.Len()
		for i := 0; i < l; i++ {
			m[r.f(i)] = struct{}{}
		}
		if len(m) != l {
			return errors.New("not unique")
		}
	default:
		return errors.New("must be slice")
	}
	return nil
}
