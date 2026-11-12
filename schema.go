package apivalidation

import (
	"context"
	"fmt"
	"reflect"
	"slices"
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

// Rules are constructed from fresh values, not the caller's instance: schemas
// must not depend on whether a particular request happens to be valid.
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

func removeSkippedFields(t reflect.Type, schema *openapi3.Schema) {
	for _, sf := range reflect.VisibleFields(t) {
		if strings.Split(sf.Tag.Get("docs"), ",")[0] == "skip" {
			delete(schema.Properties, fieldKey(sf))
		}
	}
}

// Rules are mapped locally without mutating FieldRules, so cached rule sets
// can also be used safely by concurrent schema generation.
type schemaFieldRules struct {
	kind  reflect.Kind
	rules []Rule
}

func mapFieldsToTags(fields []*FieldRules, structVal reflect.Value) (map[string]schemaFieldRules, error) {
	mapped := make(map[string]schemaFieldRules)
	for i, fr := range fields {
		if fr == nil {
			return nil, fmt.Errorf("nil field rule at index %d", i)
		}
		fv := reflect.ValueOf(fr.fieldPtr)
		if fv.Kind() != reflect.Ptr || fv.IsNil() {
			return nil, fmt.Errorf("rule target for field index %d must be a non-nil pointer", i)
		}
		sf := findStructField(structVal, fv)
		if sf == nil {
			return nil, fmt.Errorf("rule target for field index %d not found in struct %s", i, structVal.Type())
		}
		if sf.Anonymous && strings.Split(sf.Tag.Get("json"), ",")[0] == "" {
			continue
		}
		key := fieldKey(*sf)
		entry := mapped[key]
		entry.kind = sf.Type.Kind()
		entry.rules = append(entry.rules, fr.rules...)
		mapped[key] = entry
	}
	return mapped, nil
}

func applyRulesToSchema(fields map[string]schemaFieldRules, schema *openapi3.Schema) error {
	names := make([]string, 0, len(schema.Properties))
	for name := range schema.Properties {
		names = append(names, name)
	}
	slices.Sort(names) // stable required-field order and repeatable generated docs
	for _, name := range names {
		propRef := schema.Properties[name]
		field := fields[name]
		for _, rule := range field.rules {
			if _, ok := rule.(notNilRule); ok {
				switch field.kind {
				case reflect.Ptr, reflect.Interface, reflect.Slice, reflect.Map:
					// Missing nil-capable fields decode to nil and fail NotNil.
					// Unlike Required, their zero scalar contents may still pass.
					if err := Required.Describe(name, schema, propRef); err != nil {
						return err
					}
				}
			}
			if err := rule.Describe(name, schema, propRef); err != nil {
				return fmt.Errorf("describe field %s: %w", name, err)
			}
		}
	}
	return nil
}

// Resolve interface fields of the supplied sample by their actual JSON names,
// including custom names and embedded fields. Concrete child types get their
// normal rules, but nil interface samples remain unconstrained.
func resolveInterfaces(value any, t reflect.Type, schema *openapi3.Schema) error {
	rv := indirect(value)
	if rv.Kind() != reflect.Struct || rv.Type() != t {
		return nil
	}
	for _, sf := range reflect.VisibleFields(t) {
		name := fieldKey(sf)
		if !sf.IsExported() || schema.Properties[name] == nil {
			continue
		}
		field, err := rv.FieldByIndexErr(sf.Index)
		if err != nil || field.Kind() != reflect.Interface || field.IsNil() {
			continue
		}
		g := openapi3gen.NewGenerator(openapi3gen.UseAllExportedFields(), openapi3gen.SchemaCustomizer(schemaDoc(nil)))
		ref, err := g.NewSchemaRefForValue(field.Elem().Interface(), nil)
		if err != nil {
			return err
		}
		schema.Properties[name] = ref
	}
	return nil
}

func schemaDoc(value any) openapi3gen.SchemaCustomizerFn {
	return func(name string, t reflect.Type, _ reflect.StructTag, schema *openapi3.Schema) error {
		if t.Kind() == reflect.Struct {
			if err := resolveInterfaces(value, t, schema); err != nil {
				return err
			}
			removeSkippedFields(t, schema)
		}

		vi, fields := getRulesForType(t)
		if vi == nil {
			return applyValueRulerSchema(t, name, schema)
		}
		fields = expandFields(context.Background(), vi, fields)
		mapped, err := mapFieldsToTags(fields, indirect(vi))
		if err != nil {
			return err
		}
		return applyRulesToSchema(mapped, schema)
	}
}

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

// NewSchemaRefForValue generates an OpenAPI schema for exported JSON fields,
// applying Ruler, ContextRuler, and ValueRuler rules. Untagged fields use their
// Go names, just as JSON encoding does. ContextRuler receives a background context.
func NewSchemaRefForValue(value any) (*openapi3.SchemaRef, error) {
	if !reflect.ValueOf(value).IsValid() {
		return nil, fmt.Errorf("cannot generate a schema for nil")
	}
	g := openapi3gen.NewGenerator(openapi3gen.UseAllExportedFields(), openapi3gen.SchemaCustomizer(schemaDoc(value)))
	return g.NewSchemaRefForValue(value, nil)
}
