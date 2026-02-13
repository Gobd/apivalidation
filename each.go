package apivalidation

import (
	"github.com/getkin/kin-openapi/openapi3"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

// Each returns a validation rule that applies the given rules to each element of a slice or array.
func Each(rules ...Rule) Rule {
	return &eachRule{
		validation.Each(convertRules(rules...)...),
		rules,
	}
}

type eachRule struct {
	validation.EachRule
	rules []Rule
}

func (r *eachRule) Describe(name string, schema *openapi3.Schema, ref *openapi3.SchemaRef) error {
	for i := range r.rules {
		if err := r.rules[i].Describe(name, schema, ref); err != nil {
			return err
		}
	}
	return nil
}
