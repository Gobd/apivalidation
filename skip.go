package apivalidation

import (
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

type skipRule struct {
	skip bool
	desc string
}

// Skip returns a rule that skips all subsequent validation and adds desc to the schema description.
func Skip(desc string) Rule {
	return &skipRule{skip: true, desc: desc}
}

func (r *skipRule) Describe(_ string, _ *openapi3.Schema, ref *openapi3.SchemaRef) error {
	if ref.Value.Description != "" && !strings.HasSuffix(ref.Value.Description, " ") {
		ref.Value.Description += " "
	}
	ref.Value.Description += r.desc
	return nil
}

func (r *skipRule) Validate(any) error {
	return nil
}

// When determines if all rules following it should be skipped.
func (r *skipRule) When(condition bool) Rule {
	r.skip = condition
	return r
}
