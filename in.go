package apivalidation

import (
	"fmt"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

// In returns a validation rule that checks if a value is one of the allowed values.
func In(values ...any) Rule {
	want := make([]string, len(values))
	for i := range values {
		want[i] = fmt.Sprintf("'%v'", values[i])
	}
	return &inRule{
		validation.In(values...).Error(fmt.Sprintf("must be one of %s", strings.Join(want, ", "))),
		values,
	}
}

// inRule is a validation rule that validates if a value can be found in the given list of values.
type inRule struct {
	validation.InRule
	values []any
}

func (r *inRule) Validate(value any) error {
	err := r.InRule.Validate(value)
	if err != nil {
		return fmt.Errorf("%s got '%v'", err, value)
	}
	return nil
}

func (r *inRule) Describe(_ string, _ *openapi3.Schema, ref *openapi3.SchemaRef) error {
	ref.Value.Enum = r.values
	return nil
}
