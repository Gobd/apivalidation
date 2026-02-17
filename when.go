package apivalidation

import (
	"fmt"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

// WhenRule validates conditionally: it applies one set of rules when the
// condition is true, and an optional alternative set (via [WhenRule.Else])
// when false. Use [When] to create one.
type WhenRule struct {
	validation.WhenRule
	desc      string
	whenRules []Rule
	elseRules []Rule
}

// When returns a conditional validation rule that applies rules only when condition is true.
func When(condition bool, desc string, rules ...Rule) *WhenRule {
	return &WhenRule{
		WhenRule:  validation.When(condition, convertRules(rules...)...),
		desc:      desc,
		whenRules: rules,
		elseRules: nil,
	}
}

// Else specifies alternative rules to apply when the [When] condition is false.
func (r *WhenRule) Else(rules ...Rule) *WhenRule {
	r.elseRules = rules
	return r
}

// describeRules calls Describe on each rule using a temporary schema/ref,
// then extracts a human-readable summary of the schema mutations.
func describeRules(name string, rules []Rule) (string, error) {
	if len(rules) == 0 {
		return "", nil
	}

	schema := openapi3.NewSchema()
	ref := &openapi3.SchemaRef{Value: openapi3.NewSchema()}

	for _, r := range rules {
		if err := r.Describe(name, schema, ref); err != nil {
			return "", err
		}
	}

	var parts []string

	if ref.Value.Description != "" {
		parts = append(parts, ref.Value.Description)
	}
	if len(schema.Required) > 0 {
		parts = append(parts, "required")
	}
	if ref.Value.Min != nil {
		parts = append(parts, fmt.Sprintf("min %g", *ref.Value.Min))
	}
	if ref.Value.Max != nil {
		parts = append(parts, fmt.Sprintf("max %g", *ref.Value.Max))
	}
	if len(ref.Value.Enum) > 0 {
		vals := make([]string, len(ref.Value.Enum))
		for i, v := range ref.Value.Enum {
			vals[i] = fmt.Sprint(v)
		}
		parts = append(parts, "one of ["+strings.Join(vals, ", ")+"]")
	}
	if ref.Value.UniqueItems {
		parts = append(parts, "unique")
	}

	return strings.Join(parts, ", "), nil
}

// Describe implements [Rule] by appending a human-readable summary of the
// conditional rules to the schema description.
func (r *WhenRule) Describe(name string, _ *openapi3.Schema, ref *openapi3.SchemaRef) error {
	if len(r.whenRules) > 0 {
		desc, err := describeRules(name, r.whenRules)
		if err != nil {
			return err
		}
		if desc != "" {
			if ref.Value.Description != "" && !strings.HasSuffix(ref.Value.Description, " ") {
				ref.Value.Description += " "
			}
			if r.desc != "" {
				ref.Value.Description += fmt.Sprintf("when %s: %s", r.desc, desc)
			} else {
				ref.Value.Description += desc
			}
		}
	}

	if len(r.elseRules) > 0 {
		desc, err := describeRules(name, r.elseRules)
		if err != nil {
			return err
		}
		if desc != "" {
			if ref.Value.Description != "" && !strings.HasSuffix(ref.Value.Description, " ") {
				ref.Value.Description += " "
			}
			ref.Value.Description += "else: " + desc
		}
	}
	return nil
}
