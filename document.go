package apivalidation

import (
	"context"
	"reflect"

	"github.com/getkin/kin-openapi/openapi3"
)

type (
	// RuleFunc is a function type that validates a value and returns an error if invalid.
	RuleFunc func(value any) error

	// Rule is the interface that all validation rules must implement.
	Rule interface {
		Validate(value any) error
		Describe(name string, schema *openapi3.Schema, ref *openapi3.SchemaRef) error
	}

	// FieldRules binds a struct field pointer to its validation rules.
	FieldRules struct {
		fieldPtr any
		tag      string
		rules    []Rule
	}

	// ValueRuler is implemented by non-struct types (e.g. type PaymentMethod string)
	// that carry their own validation rules. The returned rules are automatically
	// applied during both validation and OpenAPI schema generation wherever the
	// type appears as a struct field.
	//
	//	type PaymentMethod string
	//
	//	const (
	//	    PaymentACH  PaymentMethod = "ach"
	//	    PaymentCC   PaymentMethod = "cc"
	//	)
	//
	//	func (p PaymentMethod) ValueRules() []Rule {
	//	    return []Rule{In(PaymentACH, PaymentCC)}
	//	}
	ValueRuler interface {
		ValueRules() []Rule
	}
)

// Ruler is implemented by types that define validation rules for their fields.
// Use a pointer receiver so field pointers are stable:
//
//	func (s *MyStruct) Rules() []*FieldRules {
//	    return []*FieldRules{Field(&s.Name, Required)}
//	}
type Ruler interface {
	Rules() []*FieldRules
}

// ContextRuler is like [Ruler] but receives a context (for conditional rules).
type ContextRuler interface {
	Rules(context.Context) []*FieldRules
}

// findStructField returns the reflect.StructField whose address matches fieldValue
// within structValue. It recurses into anonymous (embedded) struct fields.
// Returns nil if no match is found.
func findStructField(structValue reflect.Value, fieldValue reflect.Value) *reflect.StructField {
	ptr := fieldValue.Pointer()
	for i := structValue.NumField() - 1; i >= 0; i-- {
		sf := structValue.Type().Field(i)
		if ptr == structValue.Field(i).UnsafeAddr() {
			// do additional type comparison because it's possible that the address of
			// an embedded struct is the same as the first field of the embedded struct
			if sf.Type == fieldValue.Elem().Type() {
				return &sf
			}
		}
		if sf.Anonymous {
			// delve into anonymous struct to look for the field
			fi := structValue.Field(i)
			if sf.Type.Kind() == reflect.Ptr {
				fi = fi.Elem()
			}
			if fi.Kind() == reflect.Struct {
				if f := findStructField(fi, fieldValue); f != nil {
					return f
				}
			}
		}
	}
	return nil
}
