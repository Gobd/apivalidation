package apivalidation

import (
	"github.com/getkin/kin-openapi/openapi3"
)

type deprecate struct{}

// Deprecate returns a documentation-only rule that marks the field as deprecated in the schema.
func Deprecate() Rule {
	return &deprecate{}
}

func (r *deprecate) Describe(_ string, _ *openapi3.Schema, ref *openapi3.SchemaRef) error {
	ref.Value.Deprecated = true
	return nil
}

func (r *deprecate) Validate(_ any) error {
	return nil
}
