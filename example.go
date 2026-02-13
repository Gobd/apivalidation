package apivalidation

import (
	"github.com/getkin/kin-openapi/openapi3"
)

type example struct {
	ex any
}

// Example returns a documentation-only rule that sets the schema example value.
func Example(ex any) Rule {
	return &example{ex: ex}
}

func (r *example) Describe(_ string, _ *openapi3.Schema, ref *openapi3.SchemaRef) error {
	ref.Value.Example = r.ex
	return nil
}

func (r *example) Validate(_ any) error {
	return nil
}
