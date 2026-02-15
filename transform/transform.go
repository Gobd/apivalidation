package transform

import (
	"reflect"
	"strings"
)

// StructTrimSpace runs [strings.TrimSpace] on all string fields in the struct recursively,
// including nested structs, pointer fields, slices, and map values.
func StructTrimSpace(v any) {
	stringFunc(v, strings.TrimSpace)
}

// StructToLower runs [strings.ToLower] on all string fields in the struct recursively.
func StructToLower(v any) {
	stringFunc(v, strings.ToLower)
}

// StructStringFunc applies f to every string field in the struct recursively.
func StructStringFunc(v any, f func(string) string) {
	stringFunc(v, f)
}

// StructMulti runs all given functions on the struct pointer sequentially.
func StructMulti(v any, fns ...func(any)) {
	for _, f := range fns {
		f(v)
	}
}

func stringFunc(a any, f func(string) string) { //nolint:revive // reflection walker is inherently complex
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
			stringFunc(field.Addr().Interface(), f)
		case reflect.Ptr:
			if field.IsNil() {
				continue
			}
			switch field.Elem().Kind() {
			case reflect.String:
				field.Elem().SetString(f(field.Elem().String()))
			case reflect.Struct:
				stringFunc(field.Interface(), f)
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
					stringFunc(elem.Addr().Interface(), f)
				case reflect.Ptr:
					if !elem.IsNil() {
						switch elem.Elem().Kind() {
						case reflect.String:
							elem.Elem().SetString(f(elem.Elem().String()))
						case reflect.Struct:
							stringFunc(elem.Interface(), f)
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
					stringFunc(cp.Addr().Interface(), f)
					field.SetMapIndex(key, cp)
				}
			}
		}
	}
}
