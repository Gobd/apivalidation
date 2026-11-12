package walk_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/Gobd/apivalidation/internal/walk"
	"github.com/stretchr/testify/assert"
)

func trim(v reflect.Value) {
	if v.Kind() == reflect.String && v.CanSet() {
		v.SetString(strings.TrimSpace(v.String()))
	}
}

func TestValues(t *testing.T) {
	type node struct {
		Name   string
		Next   *node
		hidden string
	}
	n := &node{Name: " x ", hidden: " keep "}
	n.Next = n
	root := struct {
		Child     *node
		Array     [1]string
		Map       map[string]any
		Interface any
		Nil       *node
	}{n, [1]string{" a "}, map[string]any{" keep key ": " m "}, " i ", nil}
	walk.Values(&root, trim)
	assert.Equal(t, "x", n.Name)
	assert.Equal(t, " keep ", n.hidden)
	assert.Equal(t, "a", root.Array[0])
	assert.Equal(t, "m", root.Map[" keep key "])
	assert.Equal(t, "i", root.Interface)
	walk.Values(nil, trim)
	walk.Values((*node)(nil), trim)
	walk.Values(struct{ Any any }{}, trim)
}

func TestCollectionsAndSharedReferences(t *testing.T) {
	type word string
	m := map[string][]word{"x": {" x "}}
	walk.Values(m, trim)
	assert.Equal(t, word("x"), m["x"][0])
	cyclic := make([]any, 1)
	cyclic[0] = cyclic
	walk.Values(cyclic, trim)
	var nilSlice []string
	walk.Values(&nilSlice, func(v reflect.Value) {
		if v.Kind() == reflect.Slice && v.IsNil() {
			v.Set(reflect.ValueOf([]string{"new"}))
		}
	})
	assert.Equal(t, []string{"new"}, nilSlice)
	var nilMap map[string]string
	walk.Values(&nilMap, func(v reflect.Value) {
		if v.Kind() == reflect.Map && v.IsNil() {
			v.Set(reflect.ValueOf(map[string]string{"x": "new"}))
		}
	})
	assert.Equal(t, "new", nilMap["x"])
	shared := "x"
	walk.Values([]*string{&shared, &shared}, func(v reflect.Value) {
		if v.Kind() == reflect.String {
			v.SetString(v.String() + "!")
		}
	})
	assert.Equal(t, "x!", shared)
}
