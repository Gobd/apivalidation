package apivalidation_test

import (
	"context"
	jsonv1 "encoding/json"
	json "encoding/json/v2"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	v "github.com/Gobd/apivalidation"
	"github.com/Gobd/apivalidation/openapi"
	"github.com/Gobd/apivalidation/transform"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type regressionItem struct {
	Name string `json:"name"`
}

func (s *regressionItem) Rules() []*v.FieldRules {
	return []*v.FieldRules{v.Field(&s.Name, v.Required)}
}

type regressionParent struct {
	Items []*regressionItem `json:"items"`
}

func (s *regressionParent) Rules() []*v.FieldRules { return []*v.FieldRules{v.Field(&s.Items)} }

type regressionCode string

func (s *regressionCode) ValueRules() []v.Rule {
	return []v.Rule{v.Required, v.In(regressionCode("ok"))}
}

type regressionCodeField struct {
	Code regressionCode `json:"code"`
}

func (s *regressionCodeField) Rules() []*v.FieldRules { return []*v.FieldRules{v.Field(&s.Code)} }

func TestRegressionCollectionValidation(t *testing.T) {
	for name, value := range map[string]any{
		"pointer slice":   []*regressionItem{{}},
		"pointer map":     map[string]*regressionItem{"bad": {}},
		"interface slice": []any{&regressionItem{}},
		"interface map":   map[string]any{"bad": regressionItem{}},
		"array":           [1]*regressionItem{{}},
		"nested":          map[string][]*regressionItem{"bad": {{}}},
		"parent":          &regressionParent{Items: []*regressionItem{{}}},
		"scalar slice":    []regressionCode{"bad"},
		"scalar map":      map[string]regressionCode{"bad": "bad"},
		"scalar field":    &regressionCodeField{Code: "bad"},
	} {
		t.Run(name, func(t *testing.T) { require.Error(t, v.Validate(value)) })
	}
	for _, value := range []any{nil, (*regressionItem)(nil), []*regressionItem{nil, {Name: "ok"}}, []regressionCode{"ok"}} {
		require.NoError(t, v.Validate(value))
	}
	code := regressionCode("bad")
	require.Error(t, v.Validate(code))
	require.Error(t, v.Validate(&code))
}

type regressionNode struct {
	Name string          `json:"name"`
	Next *regressionNode `json:"next"`
}

func (s *regressionNode) Rules() []*v.FieldRules {
	return []*v.FieldRules{v.Field(&s.Name, v.Required), v.Field(&s.Next)}
}

func TestRegressionValidationCyclesAndSharedPaths(t *testing.T) {
	node := &regressionNode{Name: "ok"}
	node.Next = node
	require.NoError(t, v.Validate(node))
	node.Name = ""
	require.Error(t, v.Validate(node))
	m := map[string]any{}
	m["self"] = m
	require.NoError(t, v.Validate(m))
	s := make([]any, 1)
	s[0] = s
	require.NoError(t, v.Validate(s))
	child := &regressionItem{}
	err := v.Validate([]*regressionItem{child, child})
	var errs validation.Errors
	require.True(t, errors.As(err, &errs))
	assert.Contains(t, errs, "0")
	assert.Contains(t, errs, "1")
}

func TestRegressionWhenElseAndSkip(t *testing.T) {
	require.NoError(t, v.Nil.When(false).Validate("not nil"))
	require.Error(t, v.Nil.When(true).Validate("not nil"))
	require.NoError(t, v.Empty.When(false).Validate("not empty"))
	require.Error(t, v.When(false, "condition", v.Required).Else(v.Required).Validate(""))
	require.NoError(t, v.When(true, "condition", v.In("a")).Else(v.In("b")).Validate("a"))
	require.Error(t, v.When(false, "condition", v.In("a")).Else(v.In("b")).Validate("a"))
	item := &regressionItem{}
	require.NoError(t, v.ValidateStruct(item, []*v.FieldRules{v.Field(&item.Name, v.Skip("ignored"), v.Required)}))
	require.Error(t, v.ValidateStruct(item, []*v.FieldRules{nil}))
	parent := struct{ regressionItem }{}
	require.NotPanics(t, func() {
		require.Error(t, v.ValidateStruct(parent, []*v.FieldRules{v.Field(&parent.regressionItem)}))
	})
}

type regressionNormal struct {
	Name string `json:"name"`
}

func (s *regressionNormal) Normalize() { s.Name = strings.TrimSpace(s.Name) }

type regressionPrivate struct {
	hidden regressionNormal
	Name   string `json:"name"`
}

func TestRegressionNormalization(t *testing.T) {
	var items []regressionNormal
	require.NoError(t, v.UnmarshalAndValidate([]byte(`[{"name":" x "}]`), &items))
	assert.Equal(t, "x", items[0].Name)
	var m map[string]*regressionNormal
	require.NoError(t, v.DecodeAndValidate(strings.NewReader(`{"a":{"name":" x "}}`), &m))
	assert.Equal(t, "x", m["a"].Name)
	var private regressionPrivate
	require.NotPanics(t, func() { require.NoError(t, v.UnmarshalAndValidate([]byte(`{"name":"ok"}`), &private)) })
	value := struct {
		Array  [1]regressionNormal
		Nested map[string][]*regressionNormal
		Any    any
	}{[1]regressionNormal{{Name: " a "}}, map[string][]*regressionNormal{"x": {{Name: " b "}, nil}}, regressionNormal{Name: " c "}}
	v.Normalize(&value)
	assert.Equal(t, "a", value.Array[0].Name)
	assert.Equal(t, "b", value.Nested["x"][0].Name)
	assert.Equal(t, "c", value.Any.(regressionNormal).Name)
}

type (
	regressionContextKey   struct{}
	regressionContextChild struct {
		Name string `json:"name"`
	}
)

func (s *regressionContextChild) Normalize(ctx context.Context) {
	s.Name = ctx.Value(regressionContextKey{}).(string)
}

func (s *regressionContextChild) Rules(ctx context.Context) []*v.FieldRules {
	return []*v.FieldRules{v.Field(&s.Name, v.In(ctx.Value(regressionContextKey{}).(string)))}
}

type regressionContextParent struct {
	Children []*regressionContextChild `json:"children"`
}

func (s *regressionContextParent) Rules() []*v.FieldRules {
	return []*v.FieldRules{v.Field(&s.Children)}
}

func TestRegressionContextPropagation(t *testing.T) {
	ctx := context.WithValue(context.Background(), regressionContextKey{}, "normalized")
	for _, decode := range []func(*regressionContextParent) error{
		func(dst *regressionContextParent) error {
			return v.UnmarshalAndValidateCtx(ctx, []byte(`{"children":[{"name":"original"}]}`), dst)
		},
		func(dst *regressionContextParent) error {
			return v.DecodeAndValidateContext(ctx, strings.NewReader(`{"children":[{"name":"original"}]}`), dst)
		},
	} {
		var dst regressionContextParent
		require.NoError(t, decode(&dst))
		assert.Equal(t, "normalized", dst.Children[0].Name)
	}
}

func TestRegressionJSONV2DefaultsAndOptions(t *testing.T) {
	for _, decode := range []func(string, *regressionItem, ...json.Options) error{
		func(body string, dst *regressionItem, opts ...json.Options) error {
			return v.UnmarshalAndValidate([]byte(body), dst, opts...)
		},
		func(body string, dst *regressionItem, opts ...json.Options) error {
			return v.DecodeAndValidate(strings.NewReader(body), dst, opts...)
		},
	} {
		var dst regressionItem
		require.NoError(t, decode(`{"name":"ok","unknown":123}`, &dst))
		require.Error(t, decode(`{"name":"ok","unknown":123}`, &dst, json.RejectUnknownMembers(true)))
		for _, body := range []string{`{"name":"ok"} garbage`, `{"name":"ok"}{}`, `{"name":"a","name":"b"}`, "{\"name\":\"\xff\"}"} {
			require.Error(t, decode(body, &dst), body)
		}
		dst = regressionItem{}
		require.Error(t, decode(`{"NAME":"ok"}`, &dst)) // ignored, then Required fails
		require.NoError(t, decode(`{"NAME":"ok"}`, &dst, jsonv1.DefaultOptionsV1()))
		require.NoError(t, decode(`{"name":"a","name":"b"}`, &dst, jsonv1.DefaultOptionsV1()))
		assert.Equal(t, "b", dst.Name)
		require.Error(t, decode(`{"NAME":"ok"}`, &regressionItem{}, jsonv1.DefaultOptionsV1(), json.MatchCaseInsensitiveNames(false)))
		require.NoError(t, decode("{\"name\":\"ok\"}\n\t ", &dst))
		require.Error(t, decode(``, &dst))
		require.Error(t, decode(`null`, &dst)) // zero value fails Required
	}
}

type regressionSchema struct {
	Name   string
	Secret string            `docs:"skip"`
	Tags   []string          `json:"tags"`
	Codes  map[string]string `json:"codes"`
	Counts []int             `json:"counts"`
}

func (s *regressionSchema) Rules() []*v.FieldRules {
	return []*v.FieldRules{
		v.Field(&s.Name, v.Required, v.Length(2, 5)),
		v.Field(&s.Tags, v.Length(1, 3), v.Each(v.Required, v.In("a", "b"))),
		v.Field(&s.Codes, v.Length(1, 2), v.Each(v.In("a", "b"))),
		v.Field(&s.Counts, v.Each(v.Min(1))),
	}
}

type regressionPresence struct {
	Count   *int  `json:"count"`
	Enabled *bool `json:"enabled"`
	Scalar  int   `json:"scalar"`
}

func (s *regressionPresence) Rules() []*v.FieldRules {
	return []*v.FieldRules{v.Field(&s.Count, v.NotNil), v.Field(&s.Enabled, v.NotNil), v.Field(&s.Scalar, v.NotNil)}
}

func TestNotNilSchemaPresenceWithoutNonzeroRequirement(t *testing.T) {
	var dst regressionPresence
	require.NoError(t, v.UnmarshalAndValidate([]byte(`{"count":0,"enabled":false}`), &dst))
	ref, err := openapi.NewSchemaRefForValue(regressionPresence{})
	require.NoError(t, err)
	assert.Contains(t, ref.Value.Required, "count")
	assert.Contains(t, ref.Value.Required, "enabled")
	assert.NotContains(t, ref.Value.Required, "scalar")
	require.NoError(t, ref.Value.VisitJSON(map[string]any{"count": float64(0), "enabled": false}))
	require.Error(t, ref.Value.VisitJSON(map[string]any{"enabled": false}))
	require.Error(t, ref.Value.VisitJSON(map[string]any{"count": nil, "enabled": false}))
}

func TestRegressionSchemaSemantics(t *testing.T) {
	ref, err := openapi.NewSchemaRefForValue(regressionSchema{})
	require.NoError(t, err)
	s := ref.Value
	require.NoError(t, s.Validate(context.Background()))
	assert.NotContains(t, s.Properties, "Secret")
	assert.Contains(t, s.Required, "Name")
	assert.NotContains(t, s.Required, "tags")
	assert.Nil(t, s.Properties["Name"].Value.Min)
	assert.Nil(t, s.Properties["tags"].Value.Enum)
	assert.Empty(t, s.Properties["tags"].Value.Items.Value.Required)
	valid := map[string]any{"Name": "test", "tags": []any{"a", "b"}, "codes": map[string]any{"a": "b"}, "counts": []any{float64(1)}}
	require.NoError(t, s.VisitJSON(valid))
	for field, invalid := range map[string]any{"Name": "x", "tags": []any{"bad"}, "codes": map[string]any{"a": "bad"}, "counts": []any{float64(-1)}} {
		cp := make(map[string]any)
		for key, value := range valid {
			cp[key] = value
		}
		cp[field] = invalid
		require.Error(t, s.VisitJSON(cp), field)
	}
	unbounded := &struct{ Name string }{}
	r, err := openapi.NewSchemaRefForValue(*unbounded)
	require.NoError(t, err)
	assert.Contains(t, r.Value.Properties, "Name")
}

type regressionInterface struct {
	Data   any    `json:"custom_data"`
	Hidden string `json:"-"`
}

func TestRegressionInterfaceSchemaAndIgnoredFields(t *testing.T) {
	ref, err := openapi.NewSchemaRefForValue(regressionInterface{Data: regressionItem{}})
	require.NoError(t, err)
	assert.NotContains(t, ref.Value.Properties, "Hidden")
	assert.Contains(t, ref.Value.Properties["custom_data"].Value.Required, "name")
	assert.Nil(t, ref.Value.Properties["custom_data"].Value.Properties["name"].Value.Min)
	_, err = openapi.NewSchemaRefForValue(nil)
	require.Error(t, err)
}

func TestRegressionDateBoundsAndNamedString(t *testing.T) {
	r := v.Date(time.DateOnly).Min(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)).Max(time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC))
	for _, date := range []string{"2020-01-01", "2025-01-01", "2030-01-01"} {
		require.NoError(t, r.Validate(date))
	}
	for _, date := range []string{"2019-12-31", "2030-01-02"} {
		require.Error(t, r.Validate(date))
	}
	type named string
	require.NoError(t, v.Length(1, 1).Validate(named("界")))
	require.NoError(t, v.Min(1).Validate(named("2")))
	require.Error(t, v.Min(1).Validate(named("bad")))
}

func TestRegressionTypedRules(t *testing.T) {
	var cycle any
	cycle = &cycle
	require.Error(t, v.ByTyped(func(int) error { return nil }, "int").Validate(cycle))
	require.Error(t, v.UniqueBy(func(n int) int { return n }, "keys").Validate(cycle))
	called := false
	require.NoError(t, v.ByTyped(func(*int) error { called = true; return nil }, "pointer").Validate((*int)(nil)))
	assert.False(t, called)
	positive := v.ByTyped(func(n int) error {
		if n < 1 {
			return fmt.Errorf("must be positive")
		}
		return nil
	}, "positive")
	require.NoError(t, positive.Validate(2))
	n := 2
	require.NoError(t, positive.Validate(&n))
	require.Error(t, positive.Validate(-1))
	require.Error(t, positive.Validate("wrong type"))
	require.NoError(t, positive.Validate((*int)(nil)))
	item := &regressionItem{Name: "ok"}
	require.NoError(t, v.ValidateStruct(item, []*v.FieldRules{v.FieldCheck(&item.Name, func(s string) error { return nil }, "custom", v.Required)}))
	key := func(i regressionItem) string { return i.Name }
	unique := v.UniqueBy(key, "unique names")
	require.NoError(t, unique.Validate([]regressionItem{{Name: "a"}, {Name: "b"}}))
	require.Error(t, unique.Validate([]regressionItem{{Name: "a"}, {Name: "a"}}))
	require.NoError(t, unique.Validate([2]regressionItem{{Name: "a"}, {Name: "b"}}))
	require.Error(t, unique.Validate([]string{}))
	require.NoError(t, unique.Validate(nil))
	type items []regressionItem
	require.NoError(t, unique.Validate(items{{Name: "a"}}))
	require.Error(t, v.Unique(func(int) any { return []int{1} }, "keys").Validate([]int{1}))
	require.Error(t, v.UniqueBy(func(int) any { return []int{1} }, "keys").Validate([]int{1}))
	type compositeKey struct{ Value any }
	require.Error(t, v.UniqueBy(func(int) compositeKey { return compositeKey{[]int{1}} }, "keys").Validate([]int{1}))
}

func TestMissingRulesWithoutRuler(t *testing.T) {
	value := struct {
		Name    string
		Ignored string `validate:"-"`
	}{}
	assert.Equal(t, []string{"Name"}, v.MissingRules(&value))
	assert.Empty(t, v.MissingRules((*regressionItem)(nil)))
	assert.Empty(t, v.MissingRules(regressionItem{}))
}

func FuzzUnmarshalAndValidate(f *testing.F) {
	for _, seed := range []string{`{"name":"ok"}`, `null`, `{`, `{"name":"a","name":"b"}`, `{"unknown":1}`} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, body string) {
		var dst regressionItem
		_ = v.UnmarshalAndValidate([]byte(body), &dst)
	})
}

// Keep a transform use in this package to verify the public subpackage API.
func ExampleNormalize() {
	item := regressionNormal{Name: " Alice "}
	v.Normalize(&item)
	transform.StructToLower(&item)
	fmt.Println(item.Name)
	// Output: alice
}
