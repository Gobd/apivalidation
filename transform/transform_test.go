package transform_test

import (
	"strings"
	"testing"

	"github.com/Gobd/apivalidation/transform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type name string

func TestRecursiveStrings(t *testing.T) {
	pointer := name(" E ")
	value := struct {
		Name   name
		Array  [2]name
		Maps   map[string]name
		Nested map[string][]*name
		Any    any
		hidden string
	}{
		Name: " A ", Array: [2]name{" B ", " C "}, Maps: map[string]name{" keep key ": " D "},
		Nested: map[string][]*name{"x": {&pointer, nil}}, Any: map[string]any{"x": " F "}, hidden: " unchanged ",
	}
	require.NotPanics(t, func() { transform.StructMulti(&value, transform.StructTrimSpace, transform.StructToLower) })
	assert.Equal(t, name("a"), value.Name)
	assert.Equal(t, [2]name{"b", "c"}, value.Array)
	assert.Equal(t, name("d"), value.Maps[" keep key "])
	assert.Equal(t, name("e"), pointer)
	assert.Equal(t, "f", value.Any.(map[string]any)["x"])
	assert.Equal(t, " unchanged ", value.hidden)
}

func TestCyclesAndSharedReferences(t *testing.T) {
	type node struct {
		Name string
		Next *node
	}
	n := &node{Name: " n "}
	n.Next = n
	transform.StructTrimSpace(n)
	assert.Equal(t, "n", n.Name)
	m := map[string]any{"name": " m "}
	m["self"] = m
	transform.StructTrimSpace(m)
	assert.Equal(t, "m", m["name"])
	s := make([]any, 2)
	s[0], s[1] = " s ", s
	transform.StructTrimSpace(s)
	assert.Equal(t, "s", s[0])
	shared := "a"
	transform.StructStringFunc([]*string{&shared, &shared}, func(s string) string { return s + "!" })
	assert.Equal(t, "a!", shared)
}

func TestScalarAndNil(t *testing.T) {
	n := name(" X ")
	transform.StructTrimSpace(&n)
	assert.Equal(t, name("X"), n)
	require.NotPanics(t, func() {
		transform.StructTrimSpace(nil)
		transform.StructTrimSpace((*string)(nil))
	})
	m := map[string]*string{"nil": nil}
	transform.StructStringFunc(m, strings.ToLower)
	assert.Nil(t, m["nil"])
}
