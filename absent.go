package apivalidation

import (
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

// Nil is a validation rule that checks if a value is nil.
var Nil = absentRule{validation.Nil, false}

// Empty checks if a not nil value is empty.
var Empty = absentRule{validation.Empty, true}

type absentRule struct {
	validation.Rule
	skipNil bool
}

func (r absentRule) When(_ bool) absentRule {
	return r
}

func (r absentRule) Describe(_ string, _ *openapi3.Schema, ref *openapi3.SchemaRef) error {
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
