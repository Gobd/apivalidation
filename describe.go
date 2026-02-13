package apivalidation

import (
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

type describe struct {
	desc string
}

// Describe returns a documentation-only rule that appends desc to the schema description.
func Describe(desc string) Rule {
	return &describe{desc: desc}
}

func (r *describe) Describe(_ string, _ *openapi3.Schema, ref *openapi3.SchemaRef) error {
	if ref.Value.Description != "" && !strings.HasSuffix(ref.Value.Description, " ") {
		ref.Value.Description += " "
	}
	ref.Value.Description += r.desc
	return nil
}

func (r *describe) Validate(_ any) error {
	return nil
}
