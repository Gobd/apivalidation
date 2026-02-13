package apivalidation

import (
	"github.com/getkin/kin-openapi/openapi3"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type requiredRule struct {
	validation.RequiredRule
	desc string
}

// Required is a validation rule that checks if a value is not empty.
var Required = requiredRule{
	validation.Required,
	"required",
}

func (r requiredRule) Describe(name string, schema *openapi3.Schema, _ *openapi3.SchemaRef) error {
	schema.Required = append(schema.Required, name)
	return nil
}
