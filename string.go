package apivalidation

import (
	"fmt"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type stringRule struct {
	validation.StringRule
	desc string
}

// NewStringRuleWithError returns a string validation rule with a custom error and schema description.
func NewStringRuleWithError(validator func(string) bool, err validation.Error, desc string) Rule {
	return stringRule{
		validation.NewStringRuleWithError(validator, err),
		desc,
	}
}

// NewStringRule returns a string validation rule using desc as both the error message and schema description.
func NewStringRule(validator func(string) bool, desc string) Rule {
	return stringRule{
		validation.NewStringRule(validator, desc),
		desc,
	}
}

// NewStringRuleDecimalMax returns a validation rule that limits the number of decimal places in a numeric string.
func NewStringRuleDecimalMax(i uint) Rule {
	desc := fmt.Sprintf("no more than %d decimals", i)
	return stringRule{
		validation.NewStringRule(func(s string) bool {
			spl := strings.Split(s, ".")
			if len(spl) < 2 {
				return true
			}
			return len(spl[1]) <= int(i)
		}, desc),
		desc,
	}
}

func (r stringRule) Describe(_ string, _ *openapi3.Schema, ref *openapi3.SchemaRef) error {
	if ref.Value.Description != "" && !strings.HasSuffix(ref.Value.Description, " ") {
		ref.Value.Description += " "
	}
	ref.Value.Description += r.desc
	return nil
}
