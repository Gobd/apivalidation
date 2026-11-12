package apivalidation

import (
	"context"
	"reflect"
	"strings"
)

// Field creates a FieldRules binding a struct field pointer to its validation rules.
func Field[T any](fieldPtr *T, rules ...Rule) *FieldRules {
	return &FieldRules{
		fieldPtr: fieldPtr,
		rules:    rules,
	}
}

// expandFields flattens embedded Ruler/ContextRuler field rules into the parent's rule set.
// Non-embedded fields are returned as-is. Embedded Ruler fields have their Rules() inlined
// recursively, so error keys and schema properties are flat (not nested under the embedded name).
func expandFields(ctx context.Context, structPtr any, fields []*FieldRules) []*FieldRules {
	structVal := reflect.Indirect(reflect.ValueOf(structPtr))
	if !structVal.IsValid() || structVal.Kind() != reflect.Struct || !structVal.CanAddr() {
		return fields
	}

	result := make([]*FieldRules, 0, len(fields))
	for _, fr := range fields {
		if fr == nil {
			result = append(result, fr)
			continue
		}
		fv := reflect.ValueOf(fr.fieldPtr)
		// Scalar fields cannot be embedded Rulers, so avoid scanning the parent
		// for them. This is the common path for most API request rules.
		if fv.Kind() == reflect.Ptr && !fv.IsNil() && fv.Elem().Kind() == reflect.Struct {
			if sf := findStructField(structVal, fv); sf != nil && sf.Anonymous && strings.Split(sf.Tag.Get("json"), ",")[0] == "" {
				embeddedPtr := fv.Interface()
				if r, ok := embeddedPtr.(Ruler); ok {
					result = append(result, expandFields(ctx, embeddedPtr, r.Rules())...)
					continue
				}
				if r, ok := embeddedPtr.(ContextRuler); ok {
					result = append(result, expandFields(ctx, embeddedPtr, r.Rules(ctx))...)
					continue
				}
			}
		}
		result = append(result, fr)
	}
	return result
}
