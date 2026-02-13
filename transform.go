package apivalidation

import (
	"context"
	"reflect"
	"strings"
)

// Normalizer is implemented by types that need custom normalization after unmarshaling.
// Called by UnmarshalAndValidate before validation. When the top-level type implements
// Normalizer, normalization recurses into struct fields, slices, maps, and embedded
// structs, calling Normalize on any nested type that also implements it.
// Top level is always called first, then children depth-first.
type Normalizer interface {
	Normalize()
}

// ContextNormalizer is like Normalizer but receives a context.
// Called by UnmarshalAndValidateCtx before validation.
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

// StructTrimSpace runs strings.TrimSpace on all string fields in the struct recursively,
// including nested structs, pointer fields, slices, and map values.
func StructTrimSpace(v any) {
	structStringFunc(v, strings.TrimSpace)
}

// StructToLower runs strings.ToLower on all string fields in the struct recursively.
func StructToLower(v any) {
	structStringFunc(v, strings.ToLower)
}

// StructStringFunc applies f to every string field in the struct recursively.
func StructStringFunc(v any, f func(string) string) {
	structStringFunc(v, f)
}

// StructMulti runs all given functions on the struct pointer sequentially.
func StructMulti(v any, fns ...func(any)) {
	for _, f := range fns {
		f(v)
	}
}

func structStringFunc(a any, f func(string) string) { //nolint:revive // reflection walker is inherently complex
	v := reflect.ValueOf(a)
	if v.Kind() == reflect.Pointer {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return
	}
	for i := range v.NumField() {
		field := v.Field(i)
		if !field.CanSet() {
			continue
		}
		switch field.Kind() {
		case reflect.String:
			field.SetString(f(field.String()))
		case reflect.Struct:
			structStringFunc(field.Addr().Interface(), f)
		case reflect.Ptr:
			if field.IsNil() {
				continue
			}
			switch field.Elem().Kind() {
			case reflect.String:
				field.Elem().SetString(f(field.Elem().String()))
			case reflect.Struct:
				structStringFunc(field.Interface(), f)
			}
		case reflect.Interface:
			// Skip interface fields — concrete type is unknown at compile time
			// and modifying them via reflect can cause subtle bugs.
		case reflect.Slice:
			for j := range field.Len() {
				elem := field.Index(j)
				switch elem.Kind() {
				case reflect.String:
					elem.SetString(f(elem.String()))
				case reflect.Struct:
					structStringFunc(elem.Addr().Interface(), f)
				case reflect.Ptr:
					if !elem.IsNil() {
						switch elem.Elem().Kind() {
						case reflect.String:
							elem.Elem().SetString(f(elem.Elem().String()))
						case reflect.Struct:
							structStringFunc(elem.Interface(), f)
						}
					}
				}
			}
		case reflect.Map:
			for _, key := range field.MapKeys() {
				val := field.MapIndex(key)
				switch val.Kind() {
				case reflect.String:
					field.SetMapIndex(key, reflect.ValueOf(f(val.String())))
				case reflect.Struct:
					cp := reflect.New(val.Type()).Elem()
					cp.Set(val)
					structStringFunc(cp.Addr().Interface(), f)
					field.SetMapIndex(key, cp)
				}
			}
		}
	}
}
