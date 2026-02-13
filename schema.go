package apivalidation

import (
	"context"
	"errors"
	"fmt"
	"net/http"
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
// The struct pointer is derived from reflect.New(t), so Rules() no longer needs to return it.
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
			// Embedded fields are handled by recursion, not direct rule application.
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

// Response describes an HTTP response with a description and body types for schema generation.
type Response struct {
	Desc string
	V    []any
}

// NewRequestMust is like NewRequest but panics on error.
func NewRequestMust(vs ...any) *openapi3.RequestBodyRef {
	o, err := NewRequest(vs...)
	if err != nil {
		panic(err)
	}
	return o
}

// NewRequest generates an OpenAPI request body schema from the given value types.
func NewRequest(vs ...any) (*openapi3.RequestBodyRef, error) {
	if len(vs) == 0 {
		return nil, errors.New("no values given")
	}

	base := &openapi3.RequestBodyRef{
		Value: &openapi3.RequestBody{
			Content: openapi3.Content{
				"application/json": &openapi3.MediaType{
					Schema: &openapi3.SchemaRef{
						Value: &openapi3.Schema{
							OneOf: openapi3.SchemaRefs{},
						},
					},
				},
			},
		},
	}

	wrapper := base.Value.Content["application/json"].Schema
	for i := range vs {
		schema, err := newSchemaRefForValue(vs[i])
		if err != nil {
			return nil, err
		}
		wrapper.Value.OneOf = append(wrapper.Value.OneOf, schema)
	}

	if len(wrapper.Value.OneOf) == 1 {
		base.Value.Content["application/json"].Schema = wrapper.Value.OneOf[0]
	}

	return base, nil
}

// NewResponseMust is like NewResponse but panics on error.
// Map key is status code (e.g. "200", "4xx").
func NewResponseMust(vs map[string]Response) *openapi3.Responses {
	o, err := NewResponse(vs)
	if err != nil {
		panic(err)
	}
	return o
}

// NewResponse creates an OpenAPI responses object.
// Map key is status code (e.g. "200", "4xx").
func NewResponse(vs map[string]Response) (*openapi3.Responses, error) {
	if len(vs) == 0 {
		return nil, errors.New("no values given")
	}

	opts := make([]openapi3.NewResponsesOption, 0, len(vs))

	for statusCode := range vs {
		desc := vs[statusCode].Desc

		var refs openapi3.SchemaRefs

		for k := range vs[statusCode].V {
			schema, err := newSchemaRefForValue(vs[statusCode].V[k])
			if err != nil {
				return nil, err
			}
			refs = append(refs, schema)
		}

		content := openapi3.Content{
			"application/json": &openapi3.MediaType{
				Schema: &openapi3.SchemaRef{
					Value: &openapi3.Schema{
						OneOf: refs,
					},
				},
			},
		}

		if len(refs) == 1 {
			content["application/json"].Schema = refs[0]
		}

		opt := openapi3.WithName(statusCode, &openapi3.Response{
			Description: &desc,
			Content:     content,
		})
		opts = append(opts, opt)
	}

	return openapi3.NewResponses(opts...), nil
}

// newSchemaRefForValue generates an OpenAPI schema for the given value,
// applying validation rules from types that implement Ruler, ContextRuler, or ValueRuler.
func newSchemaRefForValue(value any) (*openapi3.SchemaRef, error) {
	g := openapi3gen.NewGenerator(openapi3gen.SchemaCustomizer(schemaDoc(value)))
	return g.NewSchemaRefForValue(value, nil)
}

// Ruler is implemented by types that define validation rules for their fields.
// Use a pointer receiver so field pointers are stable:
//
//	func (s *MyStruct) Rules() []*FieldRules {
//	    return []*FieldRules{Field(&s.Name, Required)}
//	}
type Ruler interface {
	Rules() []*FieldRules
}

// ContextRuler is like Ruler but receives a context (for conditional rules).
type ContextRuler interface {
	Rules(context.Context) []*FieldRules
}

// DocBase returns a basic OpenAPI 3.0.3 document structure.
func DocBase(serviceName, description, version string) *openapi3.T {
	return &openapi3.T{
		OpenAPI: "3.0.3",
		Info: &openapi3.Info{
			Title:       serviceName,
			Description: description,
			Version:     version,
		},
		Paths: &openapi3.Paths{},
	}
}

// AddPath adds an operation to the OpenAPI spec at the given path and method.
func AddPath(path, method string, s *openapi3.T, op *openapi3.Operation) {
	p := s.Paths.Value(path)
	if p == nil {
		p = &openapi3.PathItem{}
	}

	switch method {
	case http.MethodGet:
		p.Get = op
	case http.MethodPost:
		p.Post = op
	case http.MethodPut:
		p.Put = op
	case http.MethodPatch:
		p.Patch = op
	case http.MethodDelete:
		p.Delete = op
	}

	s.Paths.Set(path, p)
}
