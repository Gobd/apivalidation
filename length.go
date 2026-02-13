package apivalidation

import (
	"github.com/getkin/kin-openapi/openapi3"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type lengthRule struct {
	validation.LengthRule
	min, max int
}

// Length returns a validation rule that checks if a string's rune length is within the specified range.
func Length(lo, hi int) Rule {
	return &lengthRule{
		validation.RuneLength(lo, hi),
		lo,
		hi,
	}
}

func (r *lengthRule) Describe(_ string, _ *openapi3.Schema, ref *openapi3.SchemaRef) error {
	fmin := float64(r.min)
	fmax := float64(r.max)
	ref.Value.Max = &fmax
	ref.Value.Min = &fmin
	return nil
}
