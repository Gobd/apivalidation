package transform

import (
	"reflect"
	"strings"

	"github.com/Gobd/apivalidation/internal/walk"
)

// StructTrimSpace runs strings.TrimSpace on strings recursively, including
// named strings, pointers, slices, arrays, interfaces, and map values.
// Unexported fields and map keys are unchanged. Cycles are safe to traverse.
func StructTrimSpace(v any) {
	StructStringFunc(v, strings.TrimSpace)
}

// StructToLower runs strings.ToLower on strings recursively.
func StructToLower(v any) {
	StructStringFunc(v, strings.ToLower)
}

// StructStringFunc applies f to every mutable string recursively.
// Pass a pointer when transforming a struct, array, or scalar value.
func StructStringFunc(v any, f func(string) string) {
	walk.Values(v, func(field reflect.Value) {
		if field.Kind() == reflect.String && field.CanSet() {
			field.SetString(f(field.String()))
		}
	})
}

// StructMulti runs all given functions on v sequentially.
func StructMulti(v any, fns ...func(any)) {
	for _, f := range fns {
		f(v)
	}
}
