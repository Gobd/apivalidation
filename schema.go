package apivalidation

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3gen"
)

func indirect(v any) reflect.Value {
	rv := reflect.ValueOf(v)
	for rv.Kind() == reflect.Ptr {
		rv = reflect.Indirect(rv)
	}
	return rv
}

// titleFirst uppercases the first byte of s.
// Used to convert JSON field names (e.g. "name") to Go field names ("Name").
func titleFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// getRulesForType returns validation rules for t if it implements Ruler or ContextRuler.
func getRulesForType(t reflect.Type) (any, []*FieldRules) {
	inst := reflect.New(t)
	if r, ok := inst.Interface().(Ruler); ok {
		return inst.Interface(), r.Rules()
	}
	if r, ok := inst.Interface().(ContextRuler); ok {
		return inst.Interface(), r.Rules(context.Background())
	}
	return nil, nil
}

// removeSkippedFields deletes schema properties for fields tagged with docs:"skip".
// Recurses into embedded (anonymous) struct fields.
func removeSkippedFields(structVal reflect.Value, schema *openapi3.Schema) {
	for i := range structVal.NumField() {
		sf := structVal.Type().Field(i)
		if sf.Anonymous {
			fi := structVal.Field(i)
			if sf.Type.Kind() == reflect.Ptr {
				fi = fi.Elem()
			}
			if fi.Kind() == reflect.Struct {
				removeSkippedFields(fi, schema)
			}
			continue
		}
		if strings.Split(sf.Tag.Get("docs"), ",")[0] != "skip" {
			continue
		}
		jsonTag := strings.Split(sf.Tag.Get("json"), ",")[0]
		delete(schema.Properties, jsonTag)
	}
}

// mapFieldsToTags resolves each FieldRules' fieldPtr to its JSON tag name
// using struct field address comparison.
func mapFieldsToTags(fields []*FieldRules, structVal reflect.Value) error {
	for i, fr := range fields {
		fv := reflect.ValueOf(fr.fieldPtr)
		if fv.Kind() != reflect.Ptr {
			return fmt.Errorf("rule target for field index %d must be a pointer, got %s", i, fv.Kind())
		}
		sf := findStructField(structVal, fv)
		if sf == nil {
			return fmt.Errorf("rule target for field index %d not found in struct %s", i, structVal.Type())
		}
		if sf.Anonymous {
			fields[i].tag = ""
			continue
		}
		fields[i].tag = strings.Split(sf.Tag.Get("json"), ",")[0]
	}
	return nil
}

// applyRulesToSchema calls Describe on each rule for matching schema properties.
func applyRulesToSchema(fields []*FieldRules, schema *openapi3.Schema) error {
	for k, propRef := range schema.Properties {
		for _, f := range fields {
			if f.tag != k {
				continue
			}
			for _, rule := range f.rules {
				if err := rule.Describe(k, schema, propRef); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// schemaDoc returns a SchemaCustomizer that applies validation rules to OpenAPI schemas.
// The value parameter is only used for resolving interface-typed fields to concrete types.
func schemaDoc(value any) openapi3gen.SchemaCustomizerFn {
	return func(name string, t reflect.Type, _ reflect.StructTag, schema *openapi3.Schema) error {
		// Resolve interface-typed fields to their concrete types.
		if value != nil && indirect(value).Kind() == reflect.Struct {
			fn := indirect(value).FieldByName(titleFirst(name))
			if fn.IsValid() && fn.Kind() == reflect.Interface && fn.Elem().IsValid() && fn.Elem().Kind() != reflect.Interface {
				g := openapi3gen.NewGenerator(openapi3gen.SchemaCustomizer(schemaDoc(nil)))
				ref, err := g.NewSchemaRefForValue(fn.Elem().Interface(), nil)
				if err != nil {
					return err
				}
				*schema = *ref.Value
				return nil
			}
		}

		vi, fields := getRulesForType(t)
		if vi == nil {
			return applyValueRulerSchema(t, name, schema)
		}
		structVal := indirect(vi)

		// Expand embedded Ruler fields into the parent's rule set.
		fields = expandFields(context.Background(), vi, fields)

		removeSkippedFields(structVal, schema)

		if err := mapFieldsToTags(fields, structVal); err != nil {
			return err
		}

		return applyRulesToSchema(fields, schema)
	}
}

// applyValueRulerSchema checks if a type implements ValueRuler and applies
// its rules' Describe methods to the schema. Used for non-struct types
// (e.g. type PaymentMethod string) that carry their own validation rules.
func applyValueRulerSchema(t reflect.Type, name string, schema *openapi3.Schema) error {
	inst := reflect.New(t)
	vr, ok := inst.Interface().(ValueRuler)
	if !ok {
		return nil
	}
	ref := &openapi3.SchemaRef{Value: schema}
	for _, rule := range vr.ValueRules() {
		if err := rule.Describe(name, schema, ref); err != nil {
			return err
		}
	}
	return nil
}

// NewSchemaRefForValue generates an OpenAPI schema for the given value,
// applying validation rules from types that implement [Ruler],
// [ContextRuler], or [ValueRuler].
func NewSchemaRefForValue(value any) (*openapi3.SchemaRef, error) {
	g := openapi3gen.NewGenerator(openapi3gen.SchemaCustomizer(schemaDoc(value)))
	return g.NewSchemaRefForValue(value, nil)
}
