package apivalidation

import (
	"fmt"
	"reflect"

	"github.com/getkin/kin-openapi/openapi3"
)

// ByTyped wraps a typed custom validator as a Rule. It avoids type assertions
// inside the callback and accepts T or pointers to T. A mismatched field type
// returns an error rather than panicking. Nil pointers are treated as absent;
// combine with Required or NotNil when they must be present.
func ByTyped[T any](f func(T) error, desc string) Rule {
	return typedRule[T]{f: f, desc: desc}
}

// FieldCheck binds a field to a typed custom validator, with compile-time
// agreement between the field and callback types. Other rules may be supplied
// before the custom check (for example Required).
func FieldCheck[T any](fieldPtr *T, f func(T) error, desc string, rules ...Rule) *FieldRules {
	return Field(fieldPtr, append(rules, ByTyped(f, desc))...)
}

type typedRule[T any] struct {
	f    func(T) error
	desc string
}

func (r typedRule[T]) Validate(value any) error {
	v, present, err := typedValue[T](value)
	if err != nil || !present {
		return err
	}
	if r.f == nil {
		return fmt.Errorf("typed validator for %s is nil", reflect.TypeFor[T]())
	}
	return r.f(v)
}

func (r typedRule[T]) Describe(name string, schema *openapi3.Schema, ref *openapi3.SchemaRef) error {
	return Describe(r.desc).Describe(name, schema, ref)
}

// UniqueBy requires slice or array elements to have distinct keys. K must be
// comparable, preventing the unhashable-key panic possible with Unique.
// Unlike Unique, the callback receives an element, not an index, so it does not
// need to capture a particular request instance. Nil collections pass.
// Key-based uniqueness is documented as a description, not uniqueItems.
func UniqueBy[T any, K comparable](key func(T) K, desc string) Rule {
	return uniqueByRule[T, K]{key: key, desc: desc}
}

type uniqueByRule[T any, K comparable] struct {
	key  func(T) K
	desc string
}

func (r uniqueByRule[T, K]) Validate(value any) error {
	rv := reflect.ValueOf(value)
	seen := make(map[uintptr]bool)
	for rv.IsValid() && (rv.Kind() == reflect.Ptr || rv.Kind() == reflect.Interface) {
		if rv.IsNil() {
			return nil
		}
		if rv.Kind() == reflect.Ptr {
			if seen[rv.Pointer()] {
				return fmt.Errorf("cyclic pointer passed to UniqueBy")
			}
			seen[rv.Pointer()] = true
		}
		rv = rv.Elem()
	}
	if !rv.IsValid() {
		return nil
	}
	if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
		return fmt.Errorf("UniqueBy requires a slice or array, got %s", rv.Type())
	}
	if !rv.Type().Elem().AssignableTo(reflect.TypeFor[T]()) {
		return fmt.Errorf("UniqueBy expects elements of type %s, got %s", reflect.TypeFor[T](), rv.Type().Elem())
	}
	if r.key == nil {
		return fmt.Errorf("uniqueness key function is nil")
	}
	// A single slice conversion also handles named slice types. The hot loop
	// stays typed, avoiding reflect.Interface and boxing for each element.
	if rv.Kind() == reflect.Slice && rv.Type().ConvertibleTo(reflect.TypeFor[[]T]()) {
		items := rv.Convert(reflect.TypeFor[[]T]()).Interface().([]T)
		return r.validateKeys(len(items), func(i int) K { return r.key(items[i]) })
	}
	return r.validateKeys(rv.Len(), func(i int) K { return r.key(rv.Index(i).Interface().(T)) })
}

func (r uniqueByRule[T, K]) validateKeys(length int, keyAt func(int) K) error {
	keys := make(map[K]struct{}, length)
	checkDynamicKey := typeContainsInterface(reflect.TypeFor[K]())
	for i := range length {
		key := keyAt(i)
		// Interface keys may contain an unhashable dynamic value despite
		// satisfying comparable. Ordinary typed keys need no reflection here.
		if checkDynamicKey {
			if k := reflect.ValueOf(key); k.IsValid() && !k.Comparable() {
				return fmt.Errorf("uniqueness key at index %d must be comparable", i)
			}
		}
		if _, exists := keys[key]; exists {
			return fmt.Errorf("not unique")
		}
		keys[key] = struct{}{}
	}
	return nil
}

func (r uniqueByRule[T, K]) Describe(name string, schema *openapi3.Schema, ref *openapi3.SchemaRef) error {
	return uniqueRule{desc: r.desc}.Describe(name, schema, ref)
}

func typeContainsInterface(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Interface:
		return true
	case reflect.Array:
		return typeContainsInterface(t.Elem())
	case reflect.Struct:
		for i := range t.NumField() {
			if typeContainsInterface(t.Field(i).Type) {
				return true
			}
		}
	}
	return false
}

func typedValue[T any](value any) (T, bool, error) {
	var zero T
	rv := reflect.ValueOf(value)
	seen := make(map[uintptr]bool)
	for rv.IsValid() {
		if (rv.Kind() == reflect.Ptr || rv.Kind() == reflect.Interface) && rv.IsNil() {
			return zero, false, nil
		}
		if v, ok := rv.Interface().(T); ok {
			return v, true, nil
		}
		if rv.Kind() != reflect.Ptr && rv.Kind() != reflect.Interface {
			break
		}
		if rv.Kind() == reflect.Ptr {
			if seen[rv.Pointer()] {
				return zero, false, fmt.Errorf("cyclic pointer passed to typed validator")
			}
			seen[rv.Pointer()] = true
		}
		rv = rv.Elem()
	}
	if !rv.IsValid() {
		return zero, false, nil
	}
	return zero, false, fmt.Errorf("typed validator expects %s, got %s", reflect.TypeFor[T](), rv.Type())
}
