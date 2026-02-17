// Package apivalidation provides struct validation with automatic OpenAPI 3
// schema generation.
//
// Define validation rules by implementing [Ruler] on your structs:
//
//	func (o *Order) Rules() []*FieldRules {
//	    return []*FieldRules{
//	        Field(&o.ID, Required),
//	        Field(&o.Amount, Min(0.01)),
//	    }
//	}
//
// Then validate with a single call:
//
//	err := Validate(&order)
//
// For HTTP handlers, [UnmarshalAndValidate] and [DecodeAndValidate] combine
// JSON decoding with validation in one step.
//
// Sub-packages:
//   - openapi – OpenAPI schema generation, Swagger UI serving, and endpoint helpers
//   - transform – struct string transformation utilities
//   - is – common string format validation rules
package apivalidation
