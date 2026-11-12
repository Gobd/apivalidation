package apivalidation

import (
	"fmt"

	"github.com/getkin/kin-openapi/openapi3"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

// Each applies rules to each element of a slice, array, or map value.
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

func (r *eachRule) Describe(_ string, _ *openapi3.Schema, ref *openapi3.SchemaRef) error {
	var item *openapi3.SchemaRef
	switch {
	case ref.Value.Type.Is(openapi3.TypeArray):
		item = ref.Value.Items
	case ref.Value.Type.Is(openapi3.TypeObject):
		item = ref.Value.AdditionalProperties.Schema
	default:
		return fmt.Errorf("Each requires an array or map schema")
	}
	if item == nil || item.Value == nil {
		return fmt.Errorf("Each requires an element schema")
	}
	// Element schemas can be shared with other fields by the generator.
	cp := *item.Value
	cp.Required = append([]string(nil), cp.Required...)
	item = &openapi3.SchemaRef{Value: &cp}
	for _, rule := range r.rules {
		if err := rule.Describe("", &cp, item); err != nil {
			return err
		}
	}
	if ref.Value.Type.Is(openapi3.TypeArray) {
		ref.Value.Items = item
	} else {
		ref.Value.AdditionalProperties.Schema = item
	}
	return nil
}
