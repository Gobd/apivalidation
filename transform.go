package apivalidation

import (
	"context"
	"reflect"
)

// Normalizer is implemented by types that need custom normalization after unmarshaling.
// Called by [UnmarshalAndValidate] before validation. When the top-level type implements
// Normalizer, normalization recurses into struct fields, slices, maps, and embedded
// structs, calling Normalize on any nested type that also implements it.
// Top level is always called first, then children depth-first.
type Normalizer interface {
	Normalize()
}

// ContextNormalizer is like [Normalizer] but receives a context.
// Called by [UnmarshalAndValidateCtx] before validation.
type ContextNormalizer interface {
	Normalize(context.Context)
}

// normalizeRecursive calls Normalize on v (top level first), then recursively
// walks struct fields, slices, maps, and pointers calling Normalize on any
// nested value that implements Normalizer (or ContextNormalizer when ctx != nil).
func normalizeRecursive(ctx context.Context, a any) {
	if a == nil {
		return
	}
	callNormalize(ctx, a)
	rv := reflect.ValueOf(a)
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return
		}
		rv = rv.Elem()
	}
	if rv.Kind() == reflect.Struct {
		walkNormalize(ctx, rv)
	}
}

func callNormalize(ctx context.Context, v any) {
	if n, ok := v.(ContextNormalizer); ok {
		n.Normalize(ctx)
		return
	}
	if n, ok := v.(Normalizer); ok {
		n.Normalize()
	}
}

func walkNormalize(ctx context.Context, rv reflect.Value) { //nolint:revive // reflection walker is inherently complex
	for i := range rv.NumField() {
		field := rv.Field(i)
		switch field.Kind() {
		case reflect.Struct:
			if field.CanAddr() {
				callNormalize(ctx, field.Addr().Interface())
			}
			walkNormalize(ctx, field)
		case reflect.Ptr:
			if !field.IsNil() {
				callNormalize(ctx, field.Interface())
				if field.Elem().Kind() == reflect.Struct {
					walkNormalize(ctx, field.Elem())
				}
			}
		case reflect.Slice:
			for j := range field.Len() {
				elem := field.Index(j)
				switch elem.Kind() {
				case reflect.Struct:
					if elem.CanAddr() {
						callNormalize(ctx, elem.Addr().Interface())
					}
					walkNormalize(ctx, elem)
				case reflect.Ptr:
					if !elem.IsNil() {
						callNormalize(ctx, elem.Interface())
						if elem.Elem().Kind() == reflect.Struct {
							walkNormalize(ctx, elem.Elem())
						}
					}
				}
			}
		case reflect.Map:
			for _, key := range field.MapKeys() {
				val := field.MapIndex(key)
				// Map values aren't addressable; copy, normalize, put back.
				if val.Kind() == reflect.Struct {
					cp := reflect.New(val.Type())
					cp.Elem().Set(val)
					callNormalize(ctx, cp.Interface())
					walkNormalize(ctx, cp.Elem())
					field.SetMapIndex(key, cp.Elem())
				}
			}
		}
	}
}
