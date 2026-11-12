package apivalidation

import (
	"context"
	"reflect"

	"github.com/Gobd/apivalidation/internal/walk"
)

// Normalizer is implemented by types needing custom normalization after JSON
// decoding. The decode helpers visit parents before children, including structs,
// pointers, slices, arrays, maps, and interfaces. Unexported fields are skipped
// and shared references are visited once. Normalizers should be idempotent.
type Normalizer interface {
	Normalize()
}

// ContextNormalizer is like Normalizer but receives the decoding context.
type ContextNormalizer interface {
	Normalize(context.Context)
}

// Normalize recursively calls Normalizer methods, parent first. Use it when
// decoding separately or normalizing non-JSON values. It does not validate.
func Normalize(value any) {
	NormalizeCtx(context.Background(), value)
}

// NormalizeCtx is like Normalize but passes ctx to ContextNormalizer methods.
func NormalizeCtx(ctx context.Context, value any) {
	normalizeRecursive(ctx, value)
}

func normalizeRecursive(ctx context.Context, value any) {
	walk.Values(value, func(v reflect.Value) {
		if v.CanAddr() {
			callNormalize(ctx, v.Addr().Interface())
		} else {
			callNormalize(ctx, v.Interface())
		}
	})
}

func callNormalize(ctx context.Context, v any) {
	if n, ok := v.(ContextNormalizer); ok {
		n.Normalize(ctx)
	} else if n, ok := v.(Normalizer); ok {
		n.Normalize()
	}
}
