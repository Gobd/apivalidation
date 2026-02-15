package apivalidation

import (
	"context"
	"reflect"
	"strings"
)

// MissingRules returns the names of exported struct fields that have no
// corresponding rule in the Ruler's Rules(). Embedded Ruler fields are
// expanded and their inner fields checked recursively.
//
// Automatically excluded:
//   - json:"-"
//   - docs:"skip"
//   - validate:"-"  (field intentionally has no rules)
//
// Use in tests to catch forgotten fields:
//
//	assert.Empty(t, v.MissingRules(&MyStruct{}))
//	assert.Empty(t, v.MissingRules(&MyStruct{}, "OptionalField"))
func MissingRules(structPtr any, exclude ...string) []string {
	var fields []*FieldRules
	switch r := structPtr.(type) {
	case Ruler:
		fields = r.Rules()
	case ContextRuler:
		fields = r.Rules(context.Background())
	default:
		return nil
	}

	// Expand embedded Ruler fields into flat list.
	fields = ExpandFields(context.Background(), structPtr, fields)

	// Build set of covered field keys.
	structVal := reflect.Indirect(reflect.ValueOf(structPtr))
	covered := map[string]bool{}
	for _, fr := range fields {
		fv := reflect.ValueOf(fr.fieldPtr)
		if fv.Kind() != reflect.Ptr {
			continue
		}
		sf := FindStructField(structVal, fv)
		if sf == nil {
			continue
		}
		covered[fieldKey(*sf)] = true
	}

	// Build exclude set (accepts both Go field name and json tag name).
	excl := map[string]bool{}
	for _, e := range exclude {
		excl[e] = true
	}

	// Collect uncovered fields.
	var missing []string
	collectUncovered(structVal.Type(), excl, covered, &missing)
	return missing
}

// fieldKey returns the json tag name if present, otherwise the Go field name.
func fieldKey(sf reflect.StructField) string {
	tag := strings.Split(sf.Tag.Get("json"), ",")[0]
	if tag != "" && tag != "-" {
		return tag
	}
	return sf.Name
}

// collectUncovered walks the struct type recursively (into embedded structs)
// and appends any uncovered exported field names to missing.
func collectUncovered(t reflect.Type, excl, covered map[string]bool, missing *[]string) { //nolint:revive // many early-return branches inflate complexity
	for i := range t.NumField() {
		sf := t.Field(i)
		if sf.Anonymous {
			inner := sf.Type
			if inner.Kind() == reflect.Ptr {
				inner = inner.Elem()
			}
			if inner.Kind() == reflect.Struct {
				collectUncovered(inner, excl, covered, missing)
			}
			continue
		}
		if !sf.IsExported() {
			continue
		}
		jsonTag := strings.Split(sf.Tag.Get("json"), ",")[0]
		if jsonTag == "-" {
			continue
		}
		if strings.Split(sf.Tag.Get("docs"), ",")[0] == "skip" {
			continue
		}
		if sf.Tag.Get("validate") == "-" {
			continue
		}
		key := fieldKey(sf)
		if excl[key] || excl[sf.Name] {
			continue
		}
		if !covered[key] {
			*missing = append(*missing, key)
		}
	}
}
