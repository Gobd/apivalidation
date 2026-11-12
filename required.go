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

func (r requiredRule) Describe(name string, schema *openapi3.Schema, ref *openapi3.SchemaRef) error {
	if name != "" && schema != ref.Value {
		ref.Value.Nullable = false
		for _, existing := range schema.Required {
			if existing == name {
				return nil
			}
		}
		schema.Required = append(schema.Required, name)
	} else {
		// ValueRuler and Each rules describe a value, not an object property.
		if name == "" {
			ref.Value.Nullable = false
		}
		switch {
		case ref.Value.Type.Is(openapi3.TypeString):
			if ref.Value.MinLength == 0 {
				ref.Value.MinLength = 1
			}
		case ref.Value.Type.Is(openapi3.TypeArray):
			if ref.Value.MinItems == 0 {
				ref.Value.MinItems = 1
			}
		case ref.Value.Type.Is(openapi3.TypeObject):
			if ref.Value.MinProps == 0 {
				ref.Value.MinProps = 1
			}
		}
	}
	return nil
}
