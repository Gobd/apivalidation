package apivalidation_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	v "github.com/Gobd/apivalidation"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============ Test types ============

// --- Simple struct: just Rules(), no Validate() needed ---

type valItem struct {
	Name string
}

func (i *valItem) Rules() []*v.FieldRules {
	return []*v.FieldRules{
		v.Field(&i.Name, v.Required, v.Length(1, 50)),
	}
}

// --- Map of structs ---

type valRegistry map[string]valItem

// --- Nested parent → []child (bridge auto-validates children) ---

type valChild struct {
	Name string
}

func (c *valChild) Rules() []*v.FieldRules {
	return []*v.FieldRules{
		v.Field(&c.Name, v.Required),
	}
}

type valParent struct {
	Title    string
	Children []valChild
}

func (p *valParent) Rules() []*v.FieldRules {
	return []*v.FieldRules{
		v.Field(&p.Title, v.Required),
		v.Field(&p.Children),
	}
}

// --- Nested: parent → []child → []grandchild (all via Rules(), no Validate() needed) ---

type valGrandChild struct {
	Detail string
}

func (g *valGrandChild) Rules() []*v.FieldRules {
	return []*v.FieldRules{
		v.Field(&g.Detail, v.Required),
	}
}

type valChildWithGC struct {
	Name          string
	GrandChildren []valGrandChild
}

func (c *valChildWithGC) Rules() []*v.FieldRules {
	return []*v.FieldRules{
		v.Field(&c.Name, v.Required),
		v.Field(&c.GrandChildren),
	}
}

type valParentDeep struct {
	Children []valChildWithGC
}

func (p *valParentDeep) Rules() []*v.FieldRules {
	return []*v.FieldRules{
		v.Field(&p.Children),
	}
}

// --- Embedded struct with Rules() ---

type valBase struct {
	ID string
}

func (b *valBase) Rules() []*v.FieldRules {
	return []*v.FieldRules{
		v.Field(&b.ID, v.Required),
	}
}

type valWithEmbed struct {
	valBase
	Value string
}

func (w *valWithEmbed) Rules() []*v.FieldRules {
	return []*v.FieldRules{
		v.Field(&w.valBase),
		v.Field(&w.Value, v.Required),
	}
}

// --- ProcessingFees pattern: parent struct with unique slice + element Rules ---

type processingFee struct {
	PaymentType string
	Amount      float64
}

func (f *processingFee) Rules() []*v.FieldRules {
	return []*v.FieldRules{
		v.Field(&f.PaymentType, v.Required, v.In("ach", "cc", "wire")),
		v.Field(&f.Amount, v.Required, v.Min(0.0)),
	}
}

type orderWithFees struct {
	Fees []processingFee
}

func (o *orderWithFees) Rules() []*v.FieldRules {
	return []*v.FieldRules{
		v.Field(&o.Fees, v.Unique(func(i int) any { return o.Fees[i].PaymentType }, "no duplicates")),
	}
}

// --- Items with uniqueness ---

type itemsContainer struct {
	Items []valItem
}

func (c *itemsContainer) Rules() []*v.FieldRules {
	return []*v.FieldRules{
		v.Field(&c.Items, v.Unique(func(i int) any { return c.Items[i].Name }, "unique names")),
	}
}

// --- Payment method as field on parent ---

type orderWithPayment struct {
	PaymentMethod string
}

func (o *orderWithPayment) Rules() []*v.FieldRules {
	return []*v.FieldRules{
		v.Field(&o.PaymentMethod, v.Required, v.In("ach", "cc", "wire")),
	}
}

// --- ValueRuler: non-struct type with its own rules ---

type paymentMethod string

const (
	paymentACH  paymentMethod = "ach"
	paymentCC   paymentMethod = "cc"
	paymentWire paymentMethod = "wire"
)

func (p paymentMethod) ValueRules() []v.Rule {
	return []v.Rule{v.In(paymentACH, paymentCC, paymentWire)}
}

type orderWithTypedPayment struct {
	Method paymentMethod
}

func (o *orderWithTypedPayment) Rules() []*v.FieldRules {
	return []*v.FieldRules{
		v.Field(&o.Method, v.Required),
	}
}

// --- ValueRuler with multiple rules ---

type rating int

func (r rating) ValueRules() []v.Rule {
	return []v.Rule{v.Min(1), v.Max(5)}
}

type review struct {
	Rating rating
}

func (r *review) Rules() []*v.FieldRules {
	return []*v.FieldRules{
		v.Field(&r.Rating, v.Required),
	}
}

// --- Normalizer test type ---

type valNormalizable struct {
	Name  string
	Email string
}

func (n *valNormalizable) Rules() []*v.FieldRules {
	return []*v.FieldRules{
		v.Field(&n.Name, v.Required),
		v.Field(&n.Email, v.Required),
	}
}

func (n *valNormalizable) Normalize() {
	n.Email = strings.ToLower(n.Email)
}

// --- Recursive normalization test types ---

type normAddress struct {
	Street string
	City   string
}

func (a *normAddress) Normalize() {
	v.StructTrimSpace(a)
	a.City = strings.ToUpper(a.City)
}

type normOrder struct {
	Name      string
	Addresses []normAddress
}

func (o *normOrder) Normalize() {
	v.StructTrimSpace(o)
}

// ============ Tests ============

// --- Validate auto-detects Ruler ---

func TestValidate_Ruler_Valid(t *testing.T) {
	item := valItem{Name: "test"}
	err := v.Validate(&item)
	assert.NoError(t, err)
}

func TestValidate_Ruler_Invalid(t *testing.T) {
	item := valItem{Name: ""}
	err := v.Validate(&item)
	assert.Error(t, err)
}

func TestValidate_Ruler_FieldErrors(t *testing.T) {
	item := valItem{Name: ""}
	err := v.Validate(&item)
	require.Error(t, err)

	var errs validation.Errors
	require.True(t, errors.As(err, &errs))
	assert.Contains(t, errs, "Name")
}

// --- Validate: non-Ruler value does nothing ---

func TestValidate_NonRuler(t *testing.T) {
	err := v.Validate("anything")
	assert.NoError(t, err)
}

// --- Validate: []Struct where Struct has Rules() (no Validate needed) ---

func TestValidate_SliceOfRulerStructs_AllValid(t *testing.T) {
	items := []valItem{{Name: "alpha"}, {Name: "beta"}}
	err := v.Validate(&items)
	assert.NoError(t, err)
}

func TestValidate_SliceOfRulerStructs_InvalidElement(t *testing.T) {
	items := []valItem{{Name: "alpha"}, {Name: ""}}
	err := v.Validate(&items)
	require.Error(t, err)

	var errs validation.Errors
	require.True(t, errors.As(err, &errs))
	assert.Contains(t, errs, "1")
}

// --- Validate: parent with unique items ---

func TestValidate_ItemsContainer_AllValid(t *testing.T) {
	c := itemsContainer{Items: []valItem{{Name: "alpha"}, {Name: "beta"}}}
	err := v.Validate(&c)
	assert.NoError(t, err)
}

func TestValidate_ItemsContainer_Duplicate(t *testing.T) {
	c := itemsContainer{Items: []valItem{{Name: "same"}, {Name: "same"}}}
	err := v.Validate(&c)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not unique")
}

func TestValidate_ItemsContainer_InvalidElement(t *testing.T) {
	c := itemsContainer{Items: []valItem{{Name: "alpha"}, {Name: ""}}}
	err := v.Validate(&c)
	require.Error(t, err)

	var errs validation.Errors
	require.True(t, errors.As(err, &errs))
	assert.Contains(t, errs, "Items")
}

// --- OrderWithFees pattern (unique + element validation via parent struct) ---

func TestValidate_OrderWithFees_Valid(t *testing.T) {
	o := orderWithFees{
		Fees: []processingFee{
			{PaymentType: "ach", Amount: 1.50},
			{PaymentType: "cc", Amount: 2.99},
		},
	}
	err := v.Validate(&o)
	assert.NoError(t, err)
}

func TestValidate_OrderWithFees_DuplicatePaymentType(t *testing.T) {
	o := orderWithFees{
		Fees: []processingFee{
			{PaymentType: "ach", Amount: 1.50},
			{PaymentType: "ach", Amount: 2.99},
		},
	}
	err := v.Validate(&o)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not unique")
}

func TestValidate_OrderWithFees_InvalidElement(t *testing.T) {
	o := orderWithFees{
		Fees: []processingFee{
			{PaymentType: "ach", Amount: 1.50},
			{PaymentType: "", Amount: 2.99},
		},
	}
	err := v.Validate(&o)
	require.Error(t, err)

	var errs validation.Errors
	require.True(t, errors.As(err, &errs))
	assert.Contains(t, errs, "Fees")
}

func TestValidate_OrderWithFees_InvalidPaymentType(t *testing.T) {
	o := orderWithFees{
		Fees: []processingFee{
			{PaymentType: "bitcoin", Amount: 1.50},
		},
	}
	err := v.Validate(&o)
	require.Error(t, err)
}

func TestValidate_OrderWithFees_Empty(t *testing.T) {
	o := orderWithFees{}
	err := v.Validate(&o)
	assert.NoError(t, err)
}

// --- PaymentMethod as parent field ---

func TestValidate_PaymentField_Valid(t *testing.T) {
	o := orderWithPayment{PaymentMethod: "ach"}
	err := v.Validate(&o)
	assert.NoError(t, err)
}

func TestValidate_PaymentField_Invalid(t *testing.T) {
	o := orderWithPayment{PaymentMethod: "bitcoin"}
	err := v.Validate(&o)
	assert.Error(t, err)
}

func TestValidate_PaymentField_Missing(t *testing.T) {
	o := orderWithPayment{}
	err := v.Validate(&o)
	assert.Error(t, err)
}

// --- Validate: ValueRuler types auto-validate via rulerBridge ---

func TestValidate_ValueRuler_Valid(t *testing.T) {
	o := orderWithTypedPayment{Method: paymentACH}
	err := v.Validate(&o)
	assert.NoError(t, err)
}

func TestValidate_ValueRuler_Invalid(t *testing.T) {
	o := orderWithTypedPayment{Method: "bitcoin"}
	err := v.Validate(&o)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must be one of")
}

func TestValidate_ValueRuler_Missing(t *testing.T) {
	o := orderWithTypedPayment{}
	err := v.Validate(&o)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be blank")
}

func TestValidate_ValueRuler_MultipleRules_Valid(t *testing.T) {
	r := review{Rating: 3}
	err := v.Validate(&r)
	assert.NoError(t, err)
}

func TestValidate_ValueRuler_MultipleRules_TooLow(t *testing.T) {
	r := review{Rating: 0}
	err := v.Validate(&r)
	assert.Error(t, err)
}

func TestValidate_ValueRuler_MultipleRules_TooHigh(t *testing.T) {
	r := review{Rating: 10}
	err := v.Validate(&r)
	assert.Error(t, err)
}

// --- Validate: map[string]Struct where Struct has Rules() ---

func TestValidate_MapOfRulerStructs_AllValid(t *testing.T) {
	reg := valRegistry{
		"first":  {Name: "alpha"},
		"second": {Name: "beta"},
	}
	err := v.Validate(&reg)
	assert.NoError(t, err)
}

func TestValidate_MapOfRulerStructs_InvalidValue(t *testing.T) {
	reg := valRegistry{
		"ok":  {Name: "alpha"},
		"bad": {Name: ""},
	}
	err := v.Validate(&reg)
	require.Error(t, err)

	var errs validation.Errors
	require.True(t, errors.As(err, &errs))
	assert.Contains(t, errs, "bad")
}

// --- Validate: nil and empty edge cases ---

func TestValidate_NilSlice(t *testing.T) {
	var items []valItem
	err := v.Validate(&items)
	assert.NoError(t, err)
}

func TestValidate_EmptySlice(t *testing.T) {
	items := []valItem{}
	err := v.Validate(&items)
	assert.NoError(t, err)
}

func TestValidate_NilMap(t *testing.T) {
	var reg valRegistry
	err := v.Validate(&reg)
	assert.NoError(t, err)
}

func TestValidate_EmptyMap(t *testing.T) {
	reg := valRegistry{}
	err := v.Validate(&reg)
	assert.NoError(t, err)
}

func TestValidate_NilPtr(t *testing.T) {
	var p *valItem
	err := v.Validate(p)
	assert.NoError(t, err)
}

// --- Validate: parent with []child field — bridge auto-validates children ---

func TestValidate_Parent_Valid(t *testing.T) {
	p := valParent{
		Title:    "parent",
		Children: []valChild{{Name: "child"}},
	}
	err := v.Validate(&p)
	assert.NoError(t, err)
}

func TestValidate_Parent_MissingTitle(t *testing.T) {
	p := valParent{
		Title:    "",
		Children: []valChild{{Name: "child"}},
	}
	err := v.Validate(&p)
	require.Error(t, err)

	var errs validation.Errors
	require.True(t, errors.As(err, &errs))
	assert.Contains(t, errs, "Title")
}

func TestValidate_Parent_InvalidChild(t *testing.T) {
	p := valParent{
		Title:    "parent",
		Children: []valChild{{Name: "ok"}, {Name: ""}},
	}
	err := v.Validate(&p)
	require.Error(t, err)

	var errs validation.Errors
	require.True(t, errors.As(err, &errs))
	assert.Contains(t, errs, "Children")
}

// --- Validate: nested child with grandchild (all via Rules, no manual Validate) ---

func TestValidate_NestedSlices_AllValid(t *testing.T) {
	parent := valParentDeep{
		Children: []valChildWithGC{
			{Name: "child1", GrandChildren: []valGrandChild{{Detail: "gc1"}}},
			{Name: "child2", GrandChildren: []valGrandChild{{Detail: "gc2"}}},
		},
	}
	err := v.Validate(&parent)
	assert.NoError(t, err)
}

func TestValidate_NestedSlices_InvalidGrandChild(t *testing.T) {
	parent := valParentDeep{
		Children: []valChildWithGC{
			{
				Name: "child1",
				GrandChildren: []valGrandChild{
					{Detail: "ok"},
					{Detail: ""},
				},
			},
		},
	}
	err := v.Validate(&parent)
	require.Error(t, err)
}

func TestValidate_NestedSlices_InvalidChild(t *testing.T) {
	parent := valParentDeep{
		Children: []valChildWithGC{
			{Name: "", GrandChildren: []valGrandChild{{Detail: "ok"}}},
		},
	}
	err := v.Validate(&parent)
	require.Error(t, err)
}

// --- Validate: embedded struct with Rules() ---

func TestValidate_Embedded_Valid(t *testing.T) {
	w := valWithEmbed{
		valBase: valBase{ID: "abc"},
		Value:   "hello",
	}
	err := v.Validate(&w)
	assert.NoError(t, err)
}

func TestValidate_Embedded_MissingEmbeddedField(t *testing.T) {
	w := valWithEmbed{
		valBase: valBase{ID: ""},
		Value:   "hello",
	}
	err := v.Validate(&w)
	require.Error(t, err)

	// Embedded field errors should be flat (not nested under "valBase").
	var errs validation.Errors
	require.True(t, errors.As(err, &errs))
	assert.Contains(t, errs, "ID")
	assert.NotContains(t, errs, "valBase")
}

func TestValidate_Embedded_MissingOwnField(t *testing.T) {
	w := valWithEmbed{
		valBase: valBase{ID: "abc"},
		Value:   "",
	}
	err := v.Validate(&w)
	require.Error(t, err)

	var errs validation.Errors
	require.True(t, errors.As(err, &errs))
	assert.Contains(t, errs, "Value")
}

func TestValidate_Embedded_BothInvalid(t *testing.T) {
	w := valWithEmbed{
		valBase: valBase{ID: ""},
		Value:   "",
	}
	err := v.Validate(&w)
	require.Error(t, err)

	var errs validation.Errors
	require.True(t, errors.As(err, &errs))
	assert.Contains(t, errs, "ID")
	assert.Contains(t, errs, "Value")
}

// --- Validate: map of slices ---

func TestValidate_MapOfSlices_AllValid(t *testing.T) {
	m := map[string][]valItem{
		"group1": {{Name: "a"}, {Name: "b"}},
		"group2": {{Name: "c"}},
	}
	err := v.Validate(&m)
	assert.NoError(t, err)
}

func TestValidate_MapOfSlices_InvalidInner(t *testing.T) {
	m := map[string][]valItem{
		"group1": {{Name: "a"}},
		"group2": {{Name: ""}},
	}
	err := v.Validate(&m)
	require.Error(t, err)

	var errs validation.Errors
	require.True(t, errors.As(err, &errs))
	assert.Contains(t, errs, "group2")
}

// --- Validate: Skip rule ---

// --- Unique rule standalone tests ---

func TestUnique_Valid(t *testing.T) {
	vals := []string{"a", "b", "c"}
	r := v.Unique(func(i int) any { return vals[i] }, "unique")
	err := r.Validate(&vals)
	assert.NoError(t, err)
}

func TestUnique_Duplicate(t *testing.T) {
	vals := []string{"a", "b", "a"}
	r := v.Unique(func(i int) any { return vals[i] }, "unique")
	err := r.Validate(&vals)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not unique")
}

func TestUnique_NilSlice(t *testing.T) {
	var vals *[]string
	r := v.Unique(func(_ int) any { return "" }, "unique")
	err := r.Validate(vals)
	assert.NoError(t, err)
}

func TestUnique_NonSlice(t *testing.T) {
	r := v.Unique(func(i int) any { return i }, "unique")
	err := r.Validate("not a slice")
	assert.Error(t, err)
}

// --- KeyIn standalone tests ---

func TestKeyIn_Valid(t *testing.T) {
	m := map[string]any{"a": 1, "b": 2}
	r := v.KeyIn("a", "b", "c")
	err := r.Validate(m)
	assert.NoError(t, err)
}

func TestKeyIn_Invalid(t *testing.T) {
	m := map[string]any{"a": 1, "bad": 2}
	r := v.KeyIn("a", "b", "c")
	err := r.Validate(m)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "bad")
}

// --- Custom rule standalone tests ---

func TestCustom_Valid(t *testing.T) {
	c := v.Custom(func(_ any) error { return nil }, "ok")
	err := c.Validate("anything")
	assert.NoError(t, err)
}

func TestCustom_Invalid(t *testing.T) {
	c := v.Custom(func(_ any) error { return fmt.Errorf("nope") }, "nope")
	err := c.Validate("anything")
	assert.Error(t, err)
}

// --- Date rule standalone tests ---

func TestDate_ValidFormat(t *testing.T) {
	d := v.Date("2006-01-02")
	err := d.Validate("2024-03-15")
	assert.NoError(t, err)
}

func TestDate_InvalidFormat(t *testing.T) {
	d := v.Date("2006-01-02")
	err := d.Validate("not-a-date")
	assert.Error(t, err)
}

func TestDate_Empty(t *testing.T) {
	d := v.Date("2006-01-02")
	err := d.Validate("")
	assert.NoError(t, err)
}

// --- In rule error messages ---

func TestIn_Valid(t *testing.T) {
	err := v.In("a", "b", "c").Validate("b")
	assert.NoError(t, err)
}

func TestIn_Invalid_ErrorMessage(t *testing.T) {
	err := v.In("a", "b", "c").Validate("x")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must be one of")
	assert.Contains(t, err.Error(), "'a'")
	assert.Contains(t, err.Error(), "got 'x'")
}

// --- Absent rules ---

func TestNil_Valid(t *testing.T) {
	err := v.Nil.Validate(nil)
	assert.NoError(t, err)
}

func TestNil_Invalid(t *testing.T) {
	err := v.Nil.Validate("not nil")
	assert.Error(t, err)
}

func TestEmpty_Valid(t *testing.T) {
	err := v.Empty.Validate("")
	assert.NoError(t, err)
}

func TestEmpty_Invalid(t *testing.T) {
	err := v.Empty.Validate("not empty")
	assert.Error(t, err)
}

// --- NotNil rule ---

func TestNotNil_Valid(t *testing.T) {
	s := "hello"
	err := v.NotNil.Validate(&s)
	assert.NoError(t, err)
}

func TestNotNil_Invalid(t *testing.T) {
	err := v.NotNil.Validate(nil)
	assert.Error(t, err)
}

// --- Describe rule (validation is no-op) ---

func TestDescribe_AlwaysPasses(t *testing.T) {
	d := v.Describe("some desc")
	err := d.Validate("anything")
	assert.NoError(t, err)

	err = d.Validate(nil)
	assert.NoError(t, err)
}

// --- Default rule (validation is no-op) ---

func TestDefault_AlwaysPasses(t *testing.T) {
	d := v.Default("fallback")
	err := d.Validate("anything")
	assert.NoError(t, err)
}

// --- Deprecate rule (validation is no-op) ---

func TestDeprecate_AlwaysPasses(t *testing.T) {
	d := v.Deprecate()
	err := d.Validate("anything")
	assert.NoError(t, err)
}

// --- Example rule (validation is no-op) ---

func TestExample_AlwaysPasses(t *testing.T) {
	e := v.Example("ex")
	err := e.Validate("anything")
	assert.NoError(t, err)
}

// --- Skip rule ---

func TestSkip_AlwaysPasses(t *testing.T) {
	err := v.Skip("skipped").Validate("anything")
	assert.NoError(t, err)
}

// --- Each rule ---

func TestEach_Valid(t *testing.T) {
	r := v.Each(v.In("a", "b", "c"))
	err := r.Validate([]string{"a", "b"})
	assert.NoError(t, err)
}

func TestEach_Invalid(t *testing.T) {
	r := v.Each(v.In("a", "b", "c"))
	err := r.Validate([]string{"a", "x"})
	assert.Error(t, err)
}

// --- StringRule ---

func TestStringRuleDecimalMax_Valid(t *testing.T) {
	r := v.NewStringRuleDecimalMax(2)
	err := r.Validate("1.23")
	assert.NoError(t, err)
}

func TestStringRuleDecimalMax_Invalid(t *testing.T) {
	r := v.NewStringRuleDecimalMax(2)
	err := r.Validate("1.234")
	assert.Error(t, err)
}

func TestStringRuleDecimalMax_NoDecimal(t *testing.T) {
	r := v.NewStringRuleDecimalMax(2)
	err := r.Validate("123")
	assert.NoError(t, err)
}

// --- UnmarshalAndValidate ---

func TestUnmarshalAndValidate_Valid(t *testing.T) {
	body := `{"Name":"test"}`
	var item valItem
	err := v.UnmarshalAndValidate([]byte(body), &item)
	assert.NoError(t, err)
	assert.Equal(t, "test", item.Name)
}

func TestUnmarshalAndValidate_InvalidJSON(t *testing.T) {
	body := `{bad json`
	var item valItem
	err := v.UnmarshalAndValidate([]byte(body), &item)
	assert.Error(t, err)
}

func TestUnmarshalAndValidate_ValidationFails(t *testing.T) {
	body := `{"Name":""}`
	var item valItem
	err := v.UnmarshalAndValidate([]byte(body), &item)
	assert.Error(t, err)

	var errs validation.Errors
	require.True(t, errors.As(err, &errs))
	assert.Contains(t, errs, "Name")
}

func TestUnmarshalAndValidate_DoesNotTrim(t *testing.T) {
	body := `{"Name":"  test  "}`
	var item valItem
	err := v.UnmarshalAndValidate([]byte(body), &item)
	assert.NoError(t, err)
	assert.Equal(t, "  test  ", item.Name)
}

func TestUnmarshalAndValidate_Normalizes(t *testing.T) {
	body := `{"Name":"Test","Email":"UPPER@EMAIL.COM"}`
	var n valNormalizable
	err := v.UnmarshalAndValidate([]byte(body), &n)
	assert.NoError(t, err)
	assert.Equal(t, "Test", n.Name)
	assert.Equal(t, "upper@email.com", n.Email)
}

func TestUnmarshalAndValidate_NormalizesRecursive(t *testing.T) {
	body := `{"Name":"  order  ","Addresses":[{"Street":"  123 Main  ","City":"  seattle  "}]}`
	var o normOrder
	err := v.UnmarshalAndValidate([]byte(body), &o)
	assert.NoError(t, err)

	// Top-level Normalize trims all strings via StructTrimSpace.
	assert.Equal(t, "order", o.Name)
	// Nested normAddress.Normalize trims then uppercases City.
	assert.Equal(t, "123 Main", o.Addresses[0].Street)
	assert.Equal(t, "SEATTLE", o.Addresses[0].City)
}

// --- StructTrimSpace ---

func TestStructTrimSpace(t *testing.T) {
	type inner struct {
		Val string
	}
	type outer struct {
		Name  string
		Inner inner
		Items []string
	}
	o := outer{
		Name:  "  hello  ",
		Inner: inner{Val: " world "},
		Items: []string{" a ", " b "},
	}
	v.StructTrimSpace(&o)
	assert.Equal(t, "hello", o.Name)
	assert.Equal(t, "world", o.Inner.Val)
	assert.Equal(t, []string{"a", "b"}, o.Items)
}

func TestStructTrimSpace_Nested(t *testing.T) {
	type child struct {
		Name string
	}
	type parent struct {
		Children []child
	}
	p := parent{Children: []child{{Name: "  a  "}, {Name: " b "}}}
	v.StructTrimSpace(&p)
	assert.Equal(t, "a", p.Children[0].Name)
	assert.Equal(t, "b", p.Children[1].Name)
}

func TestStructTrimSpace_MapValues(t *testing.T) {
	type s struct {
		Data map[string]string
	}
	x := s{Data: map[string]string{"k": "  val  "}}
	v.StructTrimSpace(&x)
	assert.Equal(t, "val", x.Data["k"])
}

func TestStructTrimSpace_PointerField(t *testing.T) {
	type inner struct {
		Val string
	}
	type outer struct {
		Inner *inner
	}
	o := outer{Inner: &inner{Val: "  trimme  "}}
	v.StructTrimSpace(&o)
	assert.Equal(t, "trimme", o.Inner.Val)
}

func TestStructTrimSpace_NilPointer(_ *testing.T) {
	type inner struct {
		Val string
	}
	type outer struct {
		Inner *inner
	}
	o := outer{Inner: nil}
	v.StructTrimSpace(&o) // should not panic
}

func TestStructStringFunc(t *testing.T) {
	type s struct {
		Name string
	}
	x := s{Name: "Hello World"}
	v.StructStringFunc(&x, strings.ToUpper)
	assert.Equal(t, "HELLO WORLD", x.Name)
}

// --- MissingRules ---

type checkComplete struct {
	Name  string
	Email string
}

func (c *checkComplete) Rules() []*v.FieldRules {
	return []*v.FieldRules{
		v.Field(&c.Name, v.Required),
		v.Field(&c.Email, v.Required),
	}
}

type checkMissing struct {
	Name  string
	Email string
	Age   int
}

func (c *checkMissing) Rules() []*v.FieldRules {
	return []*v.FieldRules{
		v.Field(&c.Name, v.Required),
	}
}

type checkTagSkip struct {
	Name    string
	TraceID string `validate:"-"` //nolint:revive // used by MissingRules
	Counter int    `validate:"-"` //nolint:revive // used by MissingRules
}

func (c *checkTagSkip) Rules() []*v.FieldRules {
	return []*v.FieldRules{
		v.Field(&c.Name, v.Required),
	}
}

type checkJSONSkip struct {
	Name     string
	Internal string `json:"-"`
}

func (c *checkJSONSkip) Rules() []*v.FieldRules {
	return []*v.FieldRules{
		v.Field(&c.Name, v.Required),
	}
}

func TestMissingRules_AllCovered(t *testing.T) {
	missing := v.MissingRules(&checkComplete{})
	assert.Empty(t, missing)
}

func TestMissingRules_FieldsMissing(t *testing.T) {
	missing := v.MissingRules(&checkMissing{})
	assert.Contains(t, missing, "Email")
	assert.Contains(t, missing, "Age")
	assert.NotContains(t, missing, "Name")
}

func TestMissingRules_ExcludeParam(t *testing.T) {
	missing := v.MissingRules(&checkMissing{}, "Email", "Age")
	assert.Empty(t, missing)
}

func TestMissingRules_ValidateTagSkip(t *testing.T) {
	missing := v.MissingRules(&checkTagSkip{})
	assert.Empty(t, missing)
}

func TestMissingRules_JSONDashSkip(t *testing.T) {
	missing := v.MissingRules(&checkJSONSkip{})
	assert.Empty(t, missing)
}

func TestMissingRules_Embedded(t *testing.T) {
	missing := v.MissingRules(&valWithEmbed{})
	assert.Empty(t, missing)
}
