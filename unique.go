package apivalidation

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

type uniqueRule struct {
	f    func(a int) any
	desc string
}

func (r uniqueRule) Describe(_ string, _ *openapi3.Schema, ref *openapi3.SchemaRef) error {
	// OpenAPI uniqueItems compares entire JSON elements, not caller-selected
	// keys. Describe the key constraint instead of publishing the wrong rule.
	if ref.Value.Description != "" && !strings.HasSuffix(ref.Value.Description, " ") {
		ref.Value.Description += " "
	}
	desc := r.desc
	if desc == "" {
		desc = "elements must have unique keys"
	}
	ref.Value.Description += desc
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
	if value == nil {
		return nil
	}
	m := make(map[any]struct{})
	rv := reflect.ValueOf(value)
	if (rv.Kind() == reflect.Ptr || rv.Kind() == reflect.Interface) && rv.IsNil() {
		return nil
	}

	rv = reflect.Indirect(rv)

	switch rv.Kind() {
	case reflect.Slice, reflect.Array:
		l := rv.Len()
		if l > 0 && r.f == nil {
			return errors.New("uniqueness key function is nil")
		}
		for i := 0; i < l; i++ {
			key := r.f(i)
			if key != nil && !reflect.TypeOf(key).Comparable() {
				return fmt.Errorf("uniqueness key at index %d must be comparable", i)
			}
			m[key] = struct{}{}
		}
		if len(m) != l {
			return errors.New("not unique")
		}
	default:
		return errors.New("must be slice")
	}
	return nil
}
