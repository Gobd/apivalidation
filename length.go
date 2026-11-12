package apivalidation

import (
	"fmt"
	"reflect"

	"github.com/getkin/kin-openapi/openapi3"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type lengthRule struct {
	validation.LengthRule
	min, max int
}

// Length checks string rune length or collection size. A zero bound means
// unbounded, except Length(0, 0), which requires an empty value.
// Empty values pass unless combined with Required.
func Length(lo, hi int) Rule {
	return &lengthRule{
		validation.RuneLength(lo, hi),
		lo,
		hi,
	}
}

// Validate preserves rune counting for named string types as well as string.
func (r *lengthRule) Validate(value any) error {
	value, isNil := validation.Indirect(value)
	if isNil {
		return nil
	}
	rv := reflect.ValueOf(value)
	if rv.IsValid() && rv.Kind() == reflect.String {
		value = rv.String()
	}
	return r.LengthRule.Validate(value)
}

func (r *lengthRule) Describe(_ string, _ *openapi3.Schema, ref *openapi3.SchemaRef) error {
	if r.min < 0 || r.max < 0 || (r.max > 0 && r.min > r.max) {
		return fmt.Errorf("invalid length bounds: %d, %d", r.min, r.max)
	}
	min, max := uint64(r.min), uint64(r.max)
	var upper *uint64
	if max > 0 || min == 0 {
		upper = &max
	}
	switch {
	case ref.Value.Type.Is(openapi3.TypeArray):
		ref.Value.MinItems, ref.Value.MaxItems = min, upper
	case ref.Value.Type.Is(openapi3.TypeObject):
		ref.Value.MinProps, ref.Value.MaxProps = min, upper
	default:
		ref.Value.MinLength, ref.Value.MaxLength = min, upper
	}
	return nil
}
