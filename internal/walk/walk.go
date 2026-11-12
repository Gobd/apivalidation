// Package walk provides cycle-safe traversal of mutable Go values.
package walk

import "reflect"

type reference struct {
	typ reflect.Type
	ptr uintptr
	len int
}

// Values visits non-pointer, non-interface values in pre-order. Map and
// interface values are copied into addressable storage and written back after
// visiting. Unexported fields are skipped, map keys are never changed, and
// shared references are visited once to avoid cycles and duplicate mutations.
func Values(value any, visit func(reflect.Value)) {
	seen := make(map[reference]bool)
	var walk func(reflect.Value)
	walk = func(v reflect.Value) {
		if !v.IsValid() || !v.CanInterface() {
			return
		}
		switch v.Kind() {
		case reflect.Ptr, reflect.Map, reflect.Slice:
			if v.IsNil() {
				if v.Kind() == reflect.Ptr {
					return
				}
				break // nil maps/slices may have normalizers that populate them
			}
			key := referenceOf(v)
			if seen[key] {
				return
			}
			seen[key] = true
		}

		switch v.Kind() {
		case reflect.Ptr:
			walk(v.Elem())
			return
		case reflect.Interface:
			if v.IsNil() {
				return
			}
			cp := reflect.New(v.Elem().Type()).Elem()
			cp.Set(v.Elem())
			walk(cp)
			if v.CanSet() {
				v.Set(cp)
			}
			return
		}

		visit(v)
		// A visitor may replace or populate a collection. Claim its new identity
		// before descending, so self-references still cannot revisit it.
		if (v.Kind() == reflect.Map || v.Kind() == reflect.Slice) && !v.IsNil() {
			seen[referenceOf(v)] = true
		}
		switch v.Kind() {
		case reflect.Struct:
			for i := range v.NumField() {
				walk(v.Field(i))
			}
		case reflect.Slice, reflect.Array:
			for i := range v.Len() {
				walk(v.Index(i))
			}
		case reflect.Map:
			for _, key := range v.MapKeys() {
				cp := reflect.New(v.Type().Elem()).Elem()
				cp.Set(v.MapIndex(key))
				walk(cp)
				v.SetMapIndex(key, cp)
			}
		}
	}
	walk(reflect.ValueOf(value))
}

func referenceOf(v reflect.Value) reference {
	key := reference{typ: v.Type(), ptr: uintptr(v.UnsafePointer())}
	if v.Kind() == reflect.Slice {
		key.len = v.Len()
	}
	return key
}
