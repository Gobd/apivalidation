package apivalidation

import (
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
