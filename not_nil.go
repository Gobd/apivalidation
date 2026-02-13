package apivalidation

import (
	"github.com/getkin/kin-openapi/openapi3"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

// NotNil is a validation rule that checks if a value is not nil.
var NotNil = notNilRule{Rule: validation.NotNil}

type notNilRule struct {
	validation.Rule
}

func (r notNilRule) Describe(_ string, _ *openapi3.Schema, ref *openapi3.SchemaRef) error {
	ref.Value.Nullable = false
	return nil
}
