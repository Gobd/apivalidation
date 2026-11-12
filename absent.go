package apivalidation

import (
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

// Nil is a validation rule that checks if a value is nil.
var Nil = absentRule{Rule: validation.Nil}

// Empty checks if a not nil value is empty.
var Empty = absentRule{Rule: validation.Empty, skipNil: true}

type absentRule struct {
	validation.Rule
	skipNil  bool
	disabled bool
}

// When enables this rule only when condition is true. Use the top-level When
// helper to also document a human-readable condition and alternative rules.
func (r absentRule) When(condition bool) absentRule {
	r.disabled = !condition
	return r
}

func (r absentRule) Validate(value any) error {
	if r.disabled {
		return nil
	}
	return r.Rule.Validate(value)
}

func (r absentRule) Describe(_ string, _ *openapi3.Schema, ref *openapi3.SchemaRef) error {
	if r.disabled {
		return nil
	}
	if ref.Value.Description != "" && !strings.HasSuffix(ref.Value.Description, " ") {
		ref.Value.Description += " "
	}
	if r.skipNil {
		ref.Value.Description += "empty"
	} else {
		ref.Value.Description += "null"
	}
	return nil
}
