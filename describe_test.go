package apivalidation

import (
	"testing"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// helper to create a fresh schema + ref for each test
func newTestSchemaRef() (*openapi3.Schema, *openapi3.SchemaRef) {
	schema := openapi3.NewSchema()
	ref := &openapi3.SchemaRef{
		Value: openapi3.NewSchema(),
	}
	return schema, ref
}

// helper for string-typed ref
func newTestStringSchemaRef() (*openapi3.Schema, *openapi3.SchemaRef) {
	schema := openapi3.NewSchema()
	ref := &openapi3.SchemaRef{
		Value: &openapi3.Schema{
			Type: &openapi3.Types{"string"},
		},
	}
	return schema, ref
}

func TestDescribe_Required(t *testing.T) {
	schema, ref := newTestSchemaRef()

	err := Required.Describe("name", schema, ref)
	require.NoError(t, err)

	assert.Contains(t, schema.Required, "name")
}

func TestDescribe_Required_MultipleFields(t *testing.T) {
	schema, ref := newTestSchemaRef()

	err := Required.Describe("name", schema, ref)
	require.NoError(t, err)
	err = Required.Describe("email", schema, ref)
	require.NoError(t, err)

	assert.Contains(t, schema.Required, "name")
	assert.Contains(t, schema.Required, "email")
}

func TestDescribe_Min(t *testing.T) {
	schema, ref := newTestSchemaRef()

	err := Min(5).Describe("age", schema, ref)
	require.NoError(t, err)

	require.NotNil(t, ref.Value.Min)
	assert.Equal(t, float64(5), *ref.Value.Min)
}

func TestDescribe_Max(t *testing.T) {
	schema, ref := newTestSchemaRef()

	err := Max(100).Describe("age", schema, ref)
	require.NoError(t, err)

	require.NotNil(t, ref.Value.Max)
	assert.Equal(t, float64(100), *ref.Value.Max)
}

func TestDescribe_MinMax_StringType(t *testing.T) {
	// When the ref is a string type, DescNew sets the Format field
	schema, ref := newTestStringSchemaRef()

	err := Min(0.0).Describe("amount", schema, ref)
	require.NoError(t, err)

	assert.NotEmpty(t, ref.Value.Format)
	require.NotNil(t, ref.Value.Min)
	assert.Equal(t, float64(0), *ref.Value.Min)
}

func TestDescribe_Length(t *testing.T) {
	schema, ref := newTestSchemaRef()

	err := Length(3, 255).Describe("title", schema, ref)
	require.NoError(t, err)

	require.NotNil(t, ref.Value.Min)
	require.NotNil(t, ref.Value.Max)
	assert.Equal(t, float64(3), *ref.Value.Min)
	assert.Equal(t, float64(255), *ref.Value.Max)
}

func TestDescribe_In(t *testing.T) {
	schema, ref := newTestSchemaRef()

	err := In("a", "b", "c").Describe("status", schema, ref)
	require.NoError(t, err)

	assert.Equal(t, []any{"a", "b", "c"}, ref.Value.Enum)
}

func TestDescribe_Each_SingleRule(t *testing.T) {
	schema, ref := newTestSchemaRef()

	// Each with a single rule should apply that rule
	err := Each(In("x", "y")).Describe("tags", schema, ref)
	require.NoError(t, err)

	assert.Equal(t, []any{"x", "y"}, ref.Value.Enum)
}

func TestDescribe_Each_MultipleRules(t *testing.T) {
	schema, ref := newTestSchemaRef()

	err := Each(Required, In("x", "y")).Describe("tags", schema, ref)
	require.NoError(t, err)

	// All rules should be applied
	assert.Contains(t, schema.Required, "tags")
	assert.Equal(t, []any{"x", "y"}, ref.Value.Enum)
}

func TestDescribe_Unique(t *testing.T) {
	schema, ref := newTestSchemaRef()

	err := Unique(func(i int) any { return i }, "unique items").Describe("items", schema, ref)
	require.NoError(t, err)

	assert.True(t, ref.Value.UniqueItems)
}

func TestDescribe_Date_Basic(t *testing.T) {
	schema, ref := newTestSchemaRef()

	err := Date("2006-01-02").Describe("dob", schema, ref)
	require.NoError(t, err)

	assert.Equal(t, "2006-01-02", ref.Value.Format)
}

func TestDescribe_Date_WithMinMax(t *testing.T) {
	schema, ref := newTestSchemaRef()

	minTime := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	maxTime := time.Date(2030, 12, 31, 0, 0, 0, 0, time.UTC)

	err := Date("2006-01-02").Min(minTime).Max(maxTime).Describe("eventDate", schema, ref)
	require.NoError(t, err)

	assert.Equal(t, "2006-01-02", ref.Value.Format)
	assert.Contains(t, ref.Value.Description, "> "+minTime.String())
	assert.Contains(t, ref.Value.Description, "< "+maxTime.String())
}

func TestDescribe_When_WithRules(t *testing.T) {
	schema, ref := newTestSchemaRef()

	err := When(true, "is admin", Required).Describe("role", schema, ref)
	require.NoError(t, err)

	assert.Equal(t, "when is admin: required", ref.Value.Description)
}

func TestDescribe_When_WithElse(t *testing.T) {
	schema, ref := newTestSchemaRef()

	w := When(true, "is admin", Required).Else(In("guest", "viewer"))
	err := w.Describe("role", schema, ref)
	require.NoError(t, err)

	assert.Equal(t, "when is admin: required else: one of [guest, viewer]", ref.Value.Description)
}

func TestDescribe_When_WithNil(t *testing.T) {
	schema, ref := newTestSchemaRef()

	err := When(true, "line type not 1", Nil).Describe("shipFrom", schema, ref)
	require.NoError(t, err)

	assert.Equal(t, "when line type not 1: null", ref.Value.Description)
}

func TestDescribe_When_WithMin(t *testing.T) {
	schema, ref := newTestSchemaRef()

	err := When(true, "positive", Min(0.0)).Describe("amount", schema, ref)
	require.NoError(t, err)

	assert.Equal(t, "when positive: min 0", ref.Value.Description)
}

func TestDescribe_When_EmptyDesc(t *testing.T) {
	schema, ref := newTestSchemaRef()

	err := When(true, "", Max(0.0)).Describe("amount", schema, ref)
	require.NoError(t, err)

	assert.Equal(t, "max 0", ref.Value.Description)
}

func TestDescribe_When_MultipleInnerRules(t *testing.T) {
	schema, ref := newTestSchemaRef()

	err := When(true, "active", Required, Min(1.0)).Describe("count", schema, ref)
	require.NoError(t, err)

	assert.Equal(t, "when active: required, min 1", ref.Value.Description)
}

func TestDescribe_Custom(t *testing.T) {
	schema, ref := newTestSchemaRef()

	c := Custom(func(_ any) error { return nil }, "must be special")
	err := c.Describe("field", schema, ref)
	require.NoError(t, err)

	assert.Equal(t, "must be special", ref.Value.Description)
}

func TestDescribe_Custom_AppendsDescription(t *testing.T) {
	schema, ref := newTestSchemaRef()
	ref.Value.Description = "existing"

	c := Custom(func(_ any) error { return nil }, "must be special")
	err := c.Describe("field", schema, ref)
	require.NoError(t, err)

	assert.Equal(t, "existing must be special", ref.Value.Description)
}

func TestDescribe_By(t *testing.T) {
	schema, ref := newTestSchemaRef()

	b := By(func(_ any) error { return nil }, "custom inline rule")
	err := b.Describe("field", schema, ref)
	require.NoError(t, err)

	assert.Equal(t, "custom inline rule", ref.Value.Description)
}

func TestDescribe_Describe(t *testing.T) {
	schema, ref := newTestSchemaRef()

	d := Describe("a helpful description")
	err := d.Describe("field", schema, ref)
	require.NoError(t, err)

	assert.Equal(t, "a helpful description", ref.Value.Description)
}

func TestDescribe_Describe_Appends(t *testing.T) {
	schema, ref := newTestSchemaRef()
	ref.Value.Description = "prefix"

	d := Describe("suffix")
	err := d.Describe("field", schema, ref)
	require.NoError(t, err)

	assert.Equal(t, "prefix suffix", ref.Value.Description)
}

func TestDescribe_Default(t *testing.T) {
	schema, ref := newTestSchemaRef()

	err := Default("hello").Describe("greeting", schema, ref)
	require.NoError(t, err)

	assert.Equal(t, "hello", ref.Value.Default)
}

func TestDescribe_Default_Number(t *testing.T) {
	schema, ref := newTestSchemaRef()

	err := Default(42).Describe("count", schema, ref)
	require.NoError(t, err)

	assert.Equal(t, 42, ref.Value.Default)
}

func TestDescribe_Example(t *testing.T) {
	schema, ref := newTestSchemaRef()

	err := Example("sample@email.com").Describe("email", schema, ref)
	require.NoError(t, err)

	assert.Equal(t, "sample@email.com", ref.Value.Example)
}

func TestDescribe_Deprecate(t *testing.T) {
	schema, ref := newTestSchemaRef()

	err := Deprecate().Describe("oldField", schema, ref)
	require.NoError(t, err)

	assert.True(t, ref.Value.Deprecated)
}

func TestDescribe_NotNil(t *testing.T) {
	schema, ref := newTestSchemaRef()

	err := NotNil.Describe("ptr", schema, ref)
	require.NoError(t, err)

	assert.False(t, ref.Value.Nullable)
}

func TestDescribe_Nil(t *testing.T) {
	schema, ref := newTestSchemaRef()

	err := Nil.Describe("field", schema, ref)
	require.NoError(t, err)

	assert.Equal(t, "null", ref.Value.Description)
}

func TestDescribe_Empty(t *testing.T) {
	schema, ref := newTestSchemaRef()

	err := Empty.Describe("field", schema, ref)
	require.NoError(t, err)

	assert.Equal(t, "empty", ref.Value.Description)
}

func TestDescribe_Nil_AppendsDescription(t *testing.T) {
	schema, ref := newTestSchemaRef()
	ref.Value.Description = "existing"

	err := Nil.Describe("field", schema, ref)
	require.NoError(t, err)

	assert.Equal(t, "existing null", ref.Value.Description)
}

func TestDescribe_Skip(t *testing.T) {
	schema, ref := newTestSchemaRef()

	err := Skip("not applicable").Describe("field", schema, ref)
	assert.NoError(t, err)

	assert.Contains(t, ref.Value.Description, "not applicable")
}

func TestDescribe_KeyIn(t *testing.T) {
	schema, ref := newTestSchemaRef()

	err := KeyIn("a", "b", "c").Describe("config", schema, ref)
	require.NoError(t, err)

	assert.Contains(t, ref.Value.Description, "keys must be in (a,b,c)")
}

func TestDescribe_HasAlphabetic(t *testing.T) {
	schema, ref := newTestSchemaRef()

	err := HasAlphabetic().Describe("name", schema, ref)
	require.NoError(t, err)

	assert.Contains(t, ref.Value.Description, "Must contain at least one alphabetic character.")
}

func TestDescribe_NonCreditCardNumber(t *testing.T) {
	// NonCreditCardNumber uses the same hasAlphabetic struct, so same DescNew
	schema, ref := newTestSchemaRef()

	err := NonCreditCardNumber().Describe("cardField", schema, ref)
	require.NoError(t, err)

	assert.Contains(t, ref.Value.Description, "Must contain at least one alphabetic character.")
}

func TestDescribe_StringRule(t *testing.T) {
	schema, ref := newTestSchemaRef()

	r := NewStringRuleDecimalMax(2)
	err := r.Describe("amount", schema, ref)
	require.NoError(t, err)

	assert.Equal(t, "no more than 2 decimals", ref.Value.Description)
}

func TestDescribe_StringRule_Custom(t *testing.T) {
	schema, ref := newTestSchemaRef()

	r := NewStringRule(func(s string) bool { return s != "" }, "must not be blank")
	err := r.Describe("field", schema, ref)
	require.NoError(t, err)

	assert.Equal(t, "must not be blank", ref.Value.Description)
}
