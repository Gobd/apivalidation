package apivalidation

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

// Validate is the single entry point for all validation.
// If value implements Ruler, validates struct fields via Rules().
// If value implements ValueRuler, applies its rules to the value directly.
// Collection elements implementing Ruler are auto-validated.
func Validate(value any) error {
	return validateCore(context.Background(), value)
}

// ValidateCtx is like Validate but passes a context to ContextRuler.Rules().
func ValidateCtx(ctx context.Context, value any) error {
	return validateCore(ctx, value)
}

// ValidateStruct validates a struct with explicit field rules.
// Prefer Validate for types implementing Ruler.
func ValidateStruct(structPtr any, fields []*FieldRules) error {
	return validation.ValidateStruct(structPtr, convertFieldRules(context.Background(), structPtr, fields...)...)
}

// UnmarshalAndValidate decodes JSON from r into dst, then validates.
// If dst implements Normalizer, recursively normalizes (top level first, then
// nested structs, slices, maps) before validation.
func UnmarshalAndValidate(b []byte, dst any) error {
	return UnmarshalAndValidateCtx(context.Background(), b, dst)
}

// UnmarshalAndValidateCtx is like UnmarshalAndValidate but passes a context to
// ContextNormalizer.Normalize and ContextRuler.Rules.
func UnmarshalAndValidateCtx(ctx context.Context, b []byte, dst any) error {
	if err := json.Unmarshal(b, dst); err != nil {
		return err
	}
	normalizeRecursive(ctx, dst)
	return ValidateCtx(ctx, dst)
}

// DecodeAndValidate reads JSON from r into dst using a streaming decoder,
// then normalizes and validates. Use this instead of [UnmarshalAndValidate]
// when reading directly from an [io.Reader] such as an HTTP request body.
func DecodeAndValidate(r io.Reader, dst any) error {
	return DecodeAndValidateContext(context.Background(), r, dst)
}

// DecodeAndValidateContext is like DecodeAndValidate but passes a context to
// ContextNormalizer.Normalize and ContextRuler.Rules.
func DecodeAndValidateContext(ctx context.Context, r io.Reader, dst any) error {
	decoder := json.NewDecoder(r)
	if err := decoder.Decode(dst); err != nil {
		return err
	}
	normalizeRecursive(ctx, dst)
	return ValidateCtx(ctx, dst)
}

func validateCore(ctx context.Context, value any) error {
	rv := reflect.ValueOf(value)
	if rv.Kind() == reflect.Ptr && rv.IsNil() {
		return nil
	}

	// Ruler/ContextRuler: validate struct fields.
	if r, ok := value.(Ruler); ok {
		return validation.ValidateStruct(value, convertFieldRules(ctx, value, r.Rules()...)...)
	}
	if r, ok := value.(ContextRuler); ok {
		return validation.ValidateStruct(value, convertFieldRules(ctx, value, r.Rules(ctx)...)...)
	}
	// Non-pointer struct value: check if *T implements Ruler/ContextRuler.
	// This happens when ozzo passes a struct field value to the bridge rule.
	if rv.Kind() == reflect.Struct {
		ptr := reflect.New(rv.Type())
		ptr.Elem().Set(rv)
		pi := ptr.Interface()
		if r, ok := pi.(Ruler); ok {
			return validation.ValidateStruct(pi, convertFieldRules(ctx, pi, r.Rules()...)...)
		}
		if r, ok := pi.(ContextRuler); ok {
			return validation.ValidateStruct(pi, convertFieldRules(ctx, pi, r.Rules(ctx)...)...)
		}
	}

	// ValueRuler: non-struct types with their own validation rules.
	if vr, ok := value.(ValueRuler); ok {
		return validateValueRules(value, vr.ValueRules())
	}

	// Auto-validate collection elements that implement Ruler.
	if (rv.Kind() == reflect.Ptr || rv.Kind() == reflect.Interface) && rv.IsNil() {
		return nil
	}

	rv = reflect.Indirect(rv)

	switch rv.Kind() {
	case reflect.Map:
		if shouldAutoValidate(rv.Type().Elem()) {
			return validateMap(ctx, rv)
		}
	case reflect.Slice, reflect.Array:
		if shouldAutoValidate(rv.Type().Elem()) {
			return validateSlice(ctx, rv)
		}
	case reflect.Ptr, reflect.Interface:
		return validateCore(ctx, rv.Elem().Interface())
	}

	return nil
}

// validateValueRules applies a set of rules to a single value.
// Used for ValueRuler types (non-struct types with their own rules).
func validateValueRules(value any, rules []Rule) error {
	for _, rule := range rules {
		if err := rule.Validate(value); err != nil {
			return err
		}
	}
	return nil
}

// shouldAutoValidate checks if elements of the given type can be auto-validated.
// Recurses into nested collections (e.g. map[string][]Ruler).
func shouldAutoValidate(elemType reflect.Type) bool {
	if elemType.Kind() == reflect.Struct {
		if _, ok := reflect.New(elemType).Interface().(Ruler); ok {
			return true
		}
		if _, ok := reflect.New(elemType).Interface().(ContextRuler); ok {
			return true
		}
	}
	if elemType.Kind() == reflect.Slice || elemType.Kind() == reflect.Array {
		return shouldAutoValidate(elemType.Elem())
	}
	if elemType.Kind() == reflect.Map {
		return shouldAutoValidate(elemType.Elem())
	}
	return false
}

// validateElement validates a single collection element.
// Ruler structs are validated via validateCore. Nested collections are recursed.
func validateElement(ctx context.Context, v reflect.Value) error {
	if (v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface) && v.IsNil() {
		return nil
	}

	// Get a pointer for pointer-receiver interfaces.
	var ptr reflect.Value
	if v.CanAddr() {
		ptr = v.Addr()
	} else if v.Type().Kind() == reflect.Struct {
		ptr = reflect.New(v.Type())
		ptr.Elem().Set(v)
	}

	// Ruler: delegate to validateCore which handles everything.
	if ptr.IsValid() {
		pi := ptr.Interface()
		if _, ok := pi.(Ruler); ok {
			return validateCore(ctx, pi)
		}
		if _, ok := pi.(ContextRuler); ok {
			return validateCore(ctx, pi)
		}
	}

	// Nested collections (e.g. map[string][]Ruler): delegate to validateCore.
	switch v.Kind() {
	case reflect.Slice, reflect.Array, reflect.Map:
		return validateCore(ctx, v.Interface())
	}

	return nil
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

// rulerBridge is an ozzo validation.Rule that bridges Ruler fields back into
// our validateCore. When ozzo validates a struct field, this rule fires and
// recursively validates Ruler structs, []Ruler slices, and map[K]Ruler maps.
type rulerBridge struct {
	ctx context.Context
}

func (b *rulerBridge) Validate(value any) error {
	if value == nil {
		return nil
	}
	return validateCore(b.ctx, value)
}

// convertFieldRules translates our FieldRules into ozzo's FieldRules.
// Embedded Ruler fields are expanded via expandFields for flat error keys.
// A rulerBridge is appended to each field so ozzo recurses into Ruler children.
func convertFieldRules(ctx context.Context, structPtr any, fields ...*FieldRules) []*validation.FieldRules {
	flat := ExpandFields(ctx, structPtr, fields)

	vFields := make([]*validation.FieldRules, len(flat))
	for i, fr := range flat {
		rules := make([]validation.Rule, len(fr.rules), len(fr.rules)+1)
		for j, r := range fr.rules {
			rules[j] = validation.Rule(r)
		}
		rules = append(rules, &rulerBridge{ctx: ctx})
		vFields[i] = validation.Field(fr.fieldPtr, rules...)
	}
	return vFields
}

func convertRules(rules ...Rule) []validation.Rule {
	vRules := make([]validation.Rule, len(rules))
	for i := range rules {
		vRules[i] = validation.Rule(rules[i])
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
