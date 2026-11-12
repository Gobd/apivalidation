package apivalidation

import (
	"context"
	json "encoding/json/v2"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

// Validate validates Ruler structs, ValueRuler values, and their collection
// elements recursively. Values without either interface have no rules and pass.
// Nil pointers pass unless a containing field has Required or NotNil.
func Validate(value any) error {
	return ValidateCtx(context.Background(), value)
}

// ValidateCtx is like Validate but passes ctx to ContextRuler and context-aware rules.
func ValidateCtx(ctx context.Context, value any) error {
	return validateCore(newValidationContext(ctx), value)
}

// ValidateStruct validates a struct with explicit field rules.
// Prefer Validate for types implementing Ruler.
func ValidateStruct(structPtr any, fields []*FieldRules) error {
	ctx := newValidationContext(context.Background())
	return validation.ValidateStructWithContext(ctx, structPtr, convertFieldRules(ctx, structPtr, fields...)...)
}

// UnmarshalAndValidate decodes exactly one JSON value into dst, recursively
// normalizes it (parent first), then validates. It uses encoding/json/v2 defaults:
// unknown members are allowed, names are case-sensitive, and duplicate names
// and invalid UTF-8 are rejected. Pass encoding/json.DefaultOptionsV1() for
// legacy JSON semantics. Later options override earlier ones.
func UnmarshalAndValidate(b []byte, dst any, opts ...json.Options) error {
	return UnmarshalAndValidateCtx(context.Background(), b, dst, opts...)
}

// UnmarshalAndValidateCtx is like UnmarshalAndValidate but passes ctx to
// ContextNormalizer, ContextRuler, and context-aware rules.
func UnmarshalAndValidateCtx(ctx context.Context, b []byte, dst any, opts ...json.Options) error {
	if err := json.Unmarshal(b, dst, opts...); err != nil {
		return err
	}
	normalizeRecursive(ctx, dst)
	return ValidateCtx(ctx, dst)
}

// DecodeAndValidate reads exactly one JSON value from r until EOF, then
// normalizes and validates. It uses the same JSON v2 defaults and options as
// UnmarshalAndValidate. Extra JSON values or non-whitespace trailing data are
// rejected. For HTTP bodies, callers should use http.MaxBytesReader to limit size.
func DecodeAndValidate(r io.Reader, dst any, opts ...json.Options) error {
	return DecodeAndValidateContext(context.Background(), r, dst, opts...)
}

// DecodeAndValidateContext is like DecodeAndValidate but passes ctx to
// ContextNormalizer, ContextRuler, and context-aware rules.
func DecodeAndValidateContext(ctx context.Context, r io.Reader, dst any, opts ...json.Options) error {
	if err := json.UnmarshalRead(r, dst, opts...); err != nil {
		return err
	}
	normalizeRecursive(ctx, dst)
	return ValidateCtx(ctx, dst)
}

type validationContextKey struct{}

type validationReference struct {
	typ reflect.Type
	ptr uintptr
	len int
}

func newValidationContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, validationContextKey{}, make(map[validationReference]bool))
}

func validateCore(ctx context.Context, value any) error {
	rv := reflect.ValueOf(value)
	if !rv.IsValid() {
		return nil
	}
	// Skip only active ancestors: shared children must report errors at every
	// path, while self-referential values must not recurse forever.
	switch rv.Kind() {
	case reflect.Ptr, reflect.Map, reflect.Slice:
		if rv.IsNil() {
			return nil
		}
		key := validationReference{typ: rv.Type(), ptr: uintptr(rv.UnsafePointer())}
		if rv.Kind() == reflect.Slice {
			key.len = rv.Len()
		}
		active := ctx.Value(validationContextKey{}).(map[validationReference]bool)
		if active[key] {
			return nil
		}
		active[key] = true
		defer delete(active, key)
	}

	if r, ok := value.(Ruler); ok {
		return validation.ValidateStructWithContext(ctx, value, convertFieldRules(ctx, value, r.Rules()...)...)
	}
	if r, ok := value.(ContextRuler); ok {
		return validation.ValidateStructWithContext(ctx, value, convertFieldRules(ctx, value, r.Rules(ctx)...)...)
	}
	if vr, ok := value.(ValueRuler); ok {
		return validateValueRules(ctx, value, vr.ValueRules())
	}

	// Ozzo passes field values rather than pointers. An addressable copy allows
	// pointer-receiver rules to work for struct and named scalar field values.
	if rv.Kind() != reflect.Ptr && rv.Kind() != reflect.Interface && hasPointerRules(rv.Type()) {
		ptr := reflect.New(rv.Type())
		ptr.Elem().Set(rv)
		pi := ptr.Interface()
		if r, ok := pi.(Ruler); ok {
			return validation.ValidateStructWithContext(ctx, pi, convertFieldRules(ctx, pi, r.Rules()...)...)
		}
		if r, ok := pi.(ContextRuler); ok {
			return validation.ValidateStructWithContext(ctx, pi, convertFieldRules(ctx, pi, r.Rules(ctx)...)...)
		}
		if vr, ok := pi.(ValueRuler); ok {
			return validateValueRules(ctx, value, vr.ValueRules())
		}
	}

	switch rv.Kind() {
	case reflect.Map:
		return validateMap(ctx, rv)
	case reflect.Slice, reflect.Array:
		return validateSlice(ctx, rv)
	case reflect.Ptr, reflect.Interface:
		if !rv.IsNil() {
			return validateCore(ctx, rv.Elem().Interface())
		}
	}
	return nil
}

func validateValueRules(ctx context.Context, value any, rules []Rule) error {
	// Use ozzo's rule runner so Skip and context-aware conditional rules retain
	// the same semantics for scalar values as they do for struct fields.
	return validation.ValidateWithContext(ctx, value, convertRules(rules...)...)
}

func hasPointerRules(t reflect.Type) bool {
	pt := reflect.PointerTo(t)
	return pt.Implements(reflect.TypeFor[Ruler]()) ||
		pt.Implements(reflect.TypeFor[ContextRuler]()) ||
		pt.Implements(reflect.TypeFor[ValueRuler]())
}

func validateElement(ctx context.Context, v reflect.Value) error {
	if v.CanAddr() && v.Kind() != reflect.Ptr && v.Kind() != reflect.Interface && hasPointerRules(v.Type()) {
		return validateCore(ctx, v.Addr().Interface())
	}
	return validateCore(ctx, v.Interface())
}

func validateSlice(ctx context.Context, rv reflect.Value) error {
	errs := validation.Errors{}
	for i := range rv.Len() {
		if err := validateElement(ctx, rv.Index(i)); err != nil {
			errs[strconv.Itoa(i)] = err
		}
	}
	if len(errs) > 0 {
		return errs
	}
	return nil
}

func validateMap(ctx context.Context, rv reflect.Value) error {
	errs := validation.Errors{}
	for _, key := range rv.MapKeys() {
		if err := validateElement(ctx, rv.MapIndex(key)); err != nil {
			errs[fmt.Sprintf("%v", key.Interface())] = err
		}
	}
	if len(errs) > 0 {
		return errs
	}
	return nil
}

// rulerBridge connects ozzo field validation to our recursive rule runner.
type rulerBridge struct {
	ctx context.Context
}

func (b *rulerBridge) Validate(value any) error {
	return validateCore(b.ctx, value)
}

func convertFieldRules(ctx context.Context, structPtr any, fields ...*FieldRules) []*validation.FieldRules {
	flat := expandFields(ctx, structPtr, fields)
	vFields := make([]*validation.FieldRules, len(flat))
	for i, fr := range flat {
		if fr == nil {
			vFields[i] = validation.Field(nil)
			continue
		}
		rules := append(convertRules(fr.rules...), &rulerBridge{ctx: ctx})
		vFields[i] = validation.Field(fr.fieldPtr, rules...)
	}
	return vFields
}

func convertRules(rules ...Rule) []validation.Rule {
	vRules := make([]validation.Rule, len(rules))
	for i, rule := range rules {
		if r, ok := rule.(*skipRule); ok {
			vRules[i] = validation.Skip.When(r.skip)
		} else {
			vRules[i] = rule
		}
	}
	return vRules
}

// By wraps a RuleFunc into a Rule.
func By(f RuleFunc, desc string) Rule {
	return &inlineRule{validation.By(validation.RuleFunc(f)), f, desc}
}

type inlineRule struct {
	validation.Rule
	f    RuleFunc
	desc string
}

func (r *inlineRule) Describe(_ string, _ *openapi3.Schema, ref *openapi3.SchemaRef) error {
	if ref.Value.Description != "" && !strings.HasSuffix(ref.Value.Description, " ") {
		ref.Value.Description += " "
	}
	ref.Value.Description += r.desc
	return nil
}
