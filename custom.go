package apivalidation

import (
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

type custom struct {
	f    func(any) error
	desc string
}

// Custom returns a validation rule that uses f for validation and desc for documentation.
func Custom(f func(any) error, desc string) Rule {
	return custom{
		f:    f,
		desc: desc,
	}
}

func (r custom) Describe(_ string, _ *openapi3.Schema, ref *openapi3.SchemaRef) error {
	if ref.Value.Description != "" && !strings.HasSuffix(ref.Value.Description, " ") {
		ref.Value.Description += " "
	}
	ref.Value.Description += r.desc
	return nil
}

func (r custom) Validate(value any) error {
	return r.f(value)
}
