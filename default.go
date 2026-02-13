package apivalidation

import (
	"github.com/getkin/kin-openapi/openapi3"
)

type defaulter struct {
	a any
}

// Default returns a documentation-only rule that sets the schema default value.
func Default(a any) Rule {
	return defaulter{
		a: a,
	}
}

func (r defaulter) Describe(_ string, _ *openapi3.Schema, ref *openapi3.SchemaRef) error {
	ref.Value.Default = r.a
	return nil
}

func (r defaulter) Validate(_ any) error {
	return nil
}
