package openapi

import (
	av "github.com/Gobd/apivalidation"
	"github.com/getkin/kin-openapi/openapi3"
)

// NewSchemaRefForValue generates an OpenAPI schema for the given value,
// applying validation rules from types that implement [apivalidation.Ruler],
// [apivalidation.ContextRuler], or [apivalidation.ValueRuler].
func NewSchemaRefForValue(value any) (*openapi3.SchemaRef, error) {
	return av.NewSchemaRefForValue(value)
}
