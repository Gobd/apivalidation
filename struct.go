package apivalidation

import (
	"context"
	"reflect"
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
	if !structVal.IsValid() || structVal.Kind() != reflect.Struct {
		return fields
	}

	result := make([]*FieldRules, 0, len(fields))
	for _, fr := range fields {
		fv := reflect.ValueOf(fr.fieldPtr)
		if fv.Kind() == reflect.Ptr {
			if sf := findStructField(structVal, fv); sf != nil && sf.Anonymous {
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

func findStructField(structValue reflect.Value, fieldValue reflect.Value) *reflect.StructField {
	ptr := fieldValue.Pointer()
	for i := structValue.NumField() - 1; i >= 0; i-- {
		sf := structValue.Type().Field(i)
		if ptr == structValue.Field(i).UnsafeAddr() {
			// do additional type comparison because it's possible that the address of
			// an embedded struct is the same as the first field of the embedded struct
			if sf.Type == fieldValue.Elem().Type() {
				return &sf
			}
		}
		if sf.Anonymous {
			// delve into anonymous struct to look for the field
			fi := structValue.Field(i)
			if sf.Type.Kind() == reflect.Ptr {
				fi = fi.Elem()
			}
			if fi.Kind() == reflect.Struct {
				if f := findStructField(fi, fieldValue); f != nil {
					return f
				}
			}
		}
	}
	return nil
}
