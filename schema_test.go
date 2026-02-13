package apivalidation_test

import (
	"context"
	"net/http"
	"testing"

	v "github.com/Gobd/apivalidation"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Test types for schema generation ---

type schemaBasic struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Age   int    `json:"age"`
}

func (s *schemaBasic) Rules() []*v.FieldRules {
	return []*v.FieldRules{
		v.Field(&s.Name, v.Required, v.Length(1, 100)),
		v.Field(&s.Email, v.Required),
		v.Field(&s.Age, v.Min(0), v.Max(150)),
	}
}

type schemaWithEnum struct {
	Status string `json:"status"`
}

func (s *schemaWithEnum) Rules() []*v.FieldRules {
	return []*v.FieldRules{
		v.Field(&s.Status, v.Required, v.In("active", "inactive", "pending")),
	}
}

type schemaWithDescription struct {
	Notes string `json:"notes"`
}

func (s *schemaWithDescription) Rules() []*v.FieldRules {
	return []*v.FieldRules{
		v.Field(&s.Notes, v.Describe("free-form notes field")),
	}
}

// --- Embedded struct tests ---

type schemaEmbedBase struct {
	ID string `json:"id"`
}

func (s *schemaEmbedBase) Rules() []*v.FieldRules {
	return []*v.FieldRules{
		v.Field(&s.ID, v.Required),
	}
}

type schemaWithEmbed struct {
	schemaEmbedBase
	Value string `json:"value"`
}

func (s *schemaWithEmbed) Rules() []*v.FieldRules {
	return []*v.FieldRules{
		v.Field(&s.schemaEmbedBase),
		v.Field(&s.Value, v.Required),
	}
}

// --- docs:"skip" tag test ---

type schemaWithSkipField struct {
	Public  string `json:"public"`
	Secret  string `json:"secret" docs:"skip"`
	Another string `json:"another"`
}

func (s *schemaWithSkipField) Rules() []*v.FieldRules {
	return []*v.FieldRules{
		v.Field(&s.Public, v.Required),
		v.Field(&s.Another),
	}
}

// --- Interface field test ---

type schemaWithInterface struct {
	Data any `json:"data"`
}

// --- Slice of Ruler structs ---

type schemaChild struct {
	Label string `json:"label"`
}

func (s *schemaChild) Rules() []*v.FieldRules {
	return []*v.FieldRules{
		v.Field(&s.Label, v.Required, v.Length(1, 50)),
	}
}

type schemaWithChildSlice struct {
	Items []schemaChild `json:"items"`
}

func (s *schemaWithChildSlice) Rules() []*v.FieldRules {
	return []*v.FieldRules{
		v.Field(&s.Items),
	}
}

// --- Map of Ruler structs ---

type schemaWithChildMap struct {
	Registry map[string]schemaChild `json:"registry"`
}

func (s *schemaWithChildMap) Rules() []*v.FieldRules {
	return []*v.FieldRules{
		v.Field(&s.Registry),
	}
}

// --- Nested structs: parent → []child → []grandchild ---

type schemaGrandChild struct {
	Detail string `json:"detail"`
}

func (s *schemaGrandChild) Rules() []*v.FieldRules {
	return []*v.FieldRules{
		v.Field(&s.Detail, v.Required),
	}
}

type schemaChildWithNested struct {
	Name   string             `json:"name"`
	Nested []schemaGrandChild `json:"nested"`
}

func (s *schemaChildWithNested) Rules() []*v.FieldRules {
	return []*v.FieldRules{
		v.Field(&s.Name, v.Required),
		v.Field(&s.Nested),
	}
}

type schemaParentNested struct {
	Children []schemaChildWithNested `json:"children"`
}

func (s *schemaParentNested) Rules() []*v.FieldRules {
	return []*v.FieldRules{
		v.Field(&s.Children),
	}
}

// --- ValueRuler test ---

type schemaPaymentMethod string

const (
	schemaPayACH schemaPaymentMethod = "ach"
	schemaPayCC  schemaPaymentMethod = "cc"
)

func (p schemaPaymentMethod) ValueRules() []v.Rule {
	return []v.Rule{v.In(schemaPayACH, schemaPayCC)}
}

type schemaWithValueRuler struct {
	Method schemaPaymentMethod `json:"method"`
}

func (s *schemaWithValueRuler) Rules() []*v.FieldRules {
	return []*v.FieldRules{
		v.Field(&s.Method, v.Required),
	}
}

type schemaRating int

func (r schemaRating) ValueRules() []v.Rule {
	return []v.Rule{v.Min(1), v.Max(5), v.Describe("star rating")}
}

type schemaWithRatingField struct {
	Score schemaRating `json:"score"`
}

func (s *schemaWithRatingField) Rules() []*v.FieldRules {
	return []*v.FieldRules{
		v.Field(&s.Score),
	}
}

// --- ContextRuler test ---

type schemaContextRuler struct {
	Title string `json:"title"`
}

func (s *schemaContextRuler) Rules(_ context.Context) []*v.FieldRules {
	return []*v.FieldRules{
		v.Field(&s.Title, v.Required),
	}
}

// schemaFor is a test helper that extracts the schema for a type via NewRequest.
func schemaFor(t *testing.T, value any) *openapi3.Schema {
	t.Helper()
	req, err := v.NewRequest(value)
	require.NoError(t, err)
	content := req.Value.Content["application/json"]
	require.NotNil(t, content)
	require.NotNil(t, content.Schema)
	require.NotNil(t, content.Schema.Value)
	return content.Schema.Value
}

// ============ Tests ============

func TestSchema_BasicStruct(t *testing.T) {
	schema := schemaFor(t, schemaBasic{})

	// Required fields
	assert.Contains(t, schema.Required, "name")
	assert.Contains(t, schema.Required, "email")

	// Properties exist
	assert.Contains(t, schema.Properties, "name")
	assert.Contains(t, schema.Properties, "email")
	assert.Contains(t, schema.Properties, "age")

	// Min/Max on age
	ageProp := schema.Properties["age"]
	require.NotNil(t, ageProp.Value)
	assert.NotNil(t, ageProp.Value.Min)
	assert.NotNil(t, ageProp.Value.Max)
	assert.Equal(t, float64(0), *ageProp.Value.Min)
	assert.Equal(t, float64(150), *ageProp.Value.Max)

	// Length on name sets min/max
	nameProp := schema.Properties["name"]
	require.NotNil(t, nameProp.Value)
	assert.NotNil(t, nameProp.Value.Min)
	assert.NotNil(t, nameProp.Value.Max)
	assert.Equal(t, float64(1), *nameProp.Value.Min)
	assert.Equal(t, float64(100), *nameProp.Value.Max)
}

func TestSchema_Enum(t *testing.T) {
	schema := schemaFor(t, schemaWithEnum{})

	statusProp := schema.Properties["status"]
	require.NotNil(t, statusProp.Value)
	assert.Equal(t, []any{"active", "inactive", "pending"}, statusProp.Value.Enum)
	assert.Contains(t, schema.Required, "status")
}

func TestSchema_Description(t *testing.T) {
	schema := schemaFor(t, schemaWithDescription{})

	notesProp := schema.Properties["notes"]
	require.NotNil(t, notesProp.Value)
	assert.Equal(t, "free-form notes field", notesProp.Value.Description)
}

func TestSchema_EmbeddedStruct(t *testing.T) {
	schema := schemaFor(t, schemaWithEmbed{})

	// Embedded field "id" should appear
	assert.Contains(t, schema.Properties, "id")
	assert.Contains(t, schema.Properties, "value")

	// value should be required
	assert.Contains(t, schema.Required, "value")

	// id should be required (from embedded Rules)
	assert.Contains(t, schema.Required, "id")
}

func TestSchema_DocsSkip(t *testing.T) {
	schema := schemaFor(t, schemaWithSkipField{})

	// "secret" should be omitted
	assert.NotContains(t, schema.Properties, "secret")

	// "public" and "another" should exist
	assert.Contains(t, schema.Properties, "public")
	assert.Contains(t, schema.Properties, "another")
}

func TestSchema_InterfaceField(t *testing.T) {
	// When the struct has a concrete value in an interface field,
	// schemaDoc resolves it. But with a zero-value struct the interface is nil,
	// so it falls through to the default openapi3gen behavior.
	schema := schemaFor(t, schemaWithInterface{})
	assert.NotNil(t, schema)
}

func TestSchema_SliceOfRulerStructs(t *testing.T) {
	schema := schemaFor(t, schemaWithChildSlice{})
	assert.Contains(t, schema.Properties, "items")

	itemsProp := schema.Properties["items"]
	require.NotNil(t, itemsProp.Value)
	assert.Equal(t, &openapi3.Types{"array"}, itemsProp.Value.Type)

	// The items schema should have the child's rules applied
	items := itemsProp.Value.Items
	require.NotNil(t, items)
	require.NotNil(t, items.Value)
	assert.Contains(t, items.Value.Properties, "label")
	assert.Contains(t, items.Value.Required, "label")
}

func TestSchema_MapOfRulerStructs(t *testing.T) {
	schema := schemaFor(t, schemaWithChildMap{})
	assert.Contains(t, schema.Properties, "registry")

	regProp := schema.Properties["registry"]
	require.NotNil(t, regProp.Value)

	// Map generates additionalProperties or similar in openapi3gen
	// The additional properties schema should reflect the child struct
	if regProp.Value.AdditionalProperties.Schema != nil {
		childSchema := regProp.Value.AdditionalProperties.Schema.Value
		assert.Contains(t, childSchema.Properties, "label")
		assert.Contains(t, childSchema.Required, "label")
	}
}

func TestSchema_NestedStructs(t *testing.T) {
	schema := schemaFor(t, schemaParentNested{})
	assert.Contains(t, schema.Properties, "children")

	childrenProp := schema.Properties["children"]
	require.NotNil(t, childrenProp.Value)
	require.NotNil(t, childrenProp.Value.Items)

	childSchema := childrenProp.Value.Items.Value
	require.NotNil(t, childSchema)
	assert.Contains(t, childSchema.Properties, "name")
	assert.Contains(t, childSchema.Required, "name")
	assert.Contains(t, childSchema.Properties, "nested")

	nestedProp := childSchema.Properties["nested"]
	require.NotNil(t, nestedProp.Value)
	require.NotNil(t, nestedProp.Value.Items)

	grandChildSchema := nestedProp.Value.Items.Value
	require.NotNil(t, grandChildSchema)
	assert.Contains(t, grandChildSchema.Properties, "detail")
	assert.Contains(t, grandChildSchema.Required, "detail")
}

func TestSchema_ContextRuler(t *testing.T) {
	schema := schemaFor(t, schemaContextRuler{})
	assert.Contains(t, schema.Required, "title")
	assert.Contains(t, schema.Properties, "title")
}

// --- ValueRuler schema tests ---

func TestSchema_ValueRuler_Enum(t *testing.T) {
	schema := schemaFor(t, schemaWithValueRuler{})

	assert.Contains(t, schema.Properties, "method")
	assert.Contains(t, schema.Required, "method")

	methodProp := schema.Properties["method"]
	require.NotNil(t, methodProp.Value)
	assert.Equal(t, []any{schemaPayACH, schemaPayCC}, methodProp.Value.Enum)
}

func TestSchema_ValueRuler_MinMaxDescription(t *testing.T) {
	schema := schemaFor(t, schemaWithRatingField{})

	assert.Contains(t, schema.Properties, "score")

	scoreProp := schema.Properties["score"]
	require.NotNil(t, scoreProp.Value)
	assert.NotNil(t, scoreProp.Value.Min)
	assert.NotNil(t, scoreProp.Value.Max)
	assert.Equal(t, float64(1), *scoreProp.Value.Min)
	assert.Equal(t, float64(5), *scoreProp.Value.Max)
	assert.Equal(t, "star rating", scoreProp.Value.Description)
}

// --- NewRequest tests ---

func TestNewRequest_SingleType(t *testing.T) {
	req, err := v.NewRequest(schemaBasic{})
	require.NoError(t, err)
	require.NotNil(t, req)
	require.NotNil(t, req.Value)

	content := req.Value.Content["application/json"]
	require.NotNil(t, content)
	require.NotNil(t, content.Schema)

	// Single type: schema should be directly the type (no OneOf wrapper)
	assert.NotNil(t, content.Schema.Value)
	assert.Contains(t, content.Schema.Value.Properties, "name")
}

func TestNewRequest_MultipleTypes(t *testing.T) {
	req, err := v.NewRequest(schemaBasic{}, schemaWithEnum{})
	require.NoError(t, err)
	require.NotNil(t, req)

	content := req.Value.Content["application/json"]
	require.NotNil(t, content)

	// Multiple types: should use OneOf with both schemas
	require.Len(t, content.Schema.Value.OneOf, 2)
	assert.Contains(t, content.Schema.Value.OneOf[0].Value.Properties, "name")
	assert.Contains(t, content.Schema.Value.OneOf[1].Value.Properties, "status")
}

func TestNewRequest_NoValues(t *testing.T) {
	_, err := v.NewRequest()
	assert.Error(t, err)
}

func TestNewRequestMust_Panics(t *testing.T) {
	assert.Panics(t, func() {
		v.NewRequestMust() // no values → error → panic
	})
}

func TestNewRequestMust_NoPanic(t *testing.T) {
	assert.NotPanics(t, func() {
		ref := v.NewRequestMust(schemaBasic{})
		assert.NotNil(t, ref)
	})
}

// --- NewResponse tests ---

func TestNewResponse_SingleStatusCode(t *testing.T) {
	resp, err := v.NewResponse(map[string]v.Response{
		"200": {Desc: "success", V: []any{schemaBasic{}}},
	})
	require.NoError(t, err)
	require.NotNil(t, resp)

	r := resp.Value("200")
	require.NotNil(t, r)
	assert.Equal(t, "success", *r.Value.Description)
}

func TestNewResponse_MultipleStatusCodes(t *testing.T) {
	resp, err := v.NewResponse(map[string]v.Response{
		"200": {Desc: "success", V: []any{schemaBasic{}}},
		"400": {Desc: "bad request", V: []any{schemaWithEnum{}}},
	})
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.NotNil(t, resp.Value("200"))
	assert.NotNil(t, resp.Value("400"))
}

func TestNewResponse_MultipleTypesOneOf(t *testing.T) {
	resp, err := v.NewResponse(map[string]v.Response{
		"200": {Desc: "success", V: []any{schemaBasic{}, schemaWithEnum{}}},
	})
	require.NoError(t, err)

	r := resp.Value("200")
	require.NotNil(t, r)
	content := r.Value.Content["application/json"]
	require.NotNil(t, content)
	assert.Len(t, content.Schema.Value.OneOf, 2)
}

func TestNewResponse_NoValues(t *testing.T) {
	_, err := v.NewResponse(map[string]v.Response{})
	assert.Error(t, err)
}

func TestNewResponseMust_Panics(t *testing.T) {
	assert.Panics(t, func() {
		v.NewResponseMust(map[string]v.Response{})
	})
}

// --- DocBase tests ---

func TestDocBase(t *testing.T) {
	doc := v.DocBase("test-service", "A test service", "1.0.0")

	assert.Equal(t, "3.0.3", doc.OpenAPI)
	assert.Equal(t, "test-service", doc.Info.Title)
	assert.Equal(t, "A test service", doc.Info.Description)
	assert.Equal(t, "1.0.0", doc.Info.Version)
	assert.Empty(t, doc.Servers)
	assert.Nil(t, doc.Components)
	assert.Empty(t, doc.Security)
	assert.NotNil(t, doc.Paths)

	// Validate the structure
	err := doc.Validate(context.Background())
	require.NoError(t, err)
}

// --- AddPath tests ---

func TestAddPath_Methods(t *testing.T) {
	doc := v.DocBase("test", "test", "1.0")

	methods := []string{
		http.MethodGet,
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
	}

	for _, method := range methods {
		op := &openapi3.Operation{
			OperationID: method + "-test",
			Responses:   openapi3.NewResponses(),
		}
		v.AddPath("/test-"+method, method, doc, op)
	}

	assert.NotNil(t, doc.Paths.Value("/test-GET").Get)
	assert.NotNil(t, doc.Paths.Value("/test-POST").Post)
	assert.NotNil(t, doc.Paths.Value("/test-PUT").Put)
	assert.NotNil(t, doc.Paths.Value("/test-PATCH").Patch)
	assert.NotNil(t, doc.Paths.Value("/test-DELETE").Delete)
}

func TestAddPath_SamePath(t *testing.T) {
	doc := v.DocBase("test", "test", "1.0")

	getOp := &openapi3.Operation{
		OperationID: "getItems",
		Responses:   openapi3.NewResponses(),
	}
	postOp := &openapi3.Operation{
		OperationID: "createItem",
		Responses:   openapi3.NewResponses(),
	}

	v.AddPath("/items", http.MethodGet, doc, getOp)
	v.AddPath("/items", http.MethodPost, doc, postOp)

	path := doc.Paths.Value("/items")
	require.NotNil(t, path)
	assert.NotNil(t, path.Get)
	assert.NotNil(t, path.Post)
	assert.Equal(t, "getItems", path.Get.OperationID)
	assert.Equal(t, "createItem", path.Post.OperationID)
}

// --- Full round-trip: build a complete doc and validate it ---

func TestFullDocRoundTrip(t *testing.T) {
	doc := v.DocBase("api", "API", "1.0")

	req := v.NewRequestMust(schemaBasic{})
	resp, err := v.NewResponse(map[string]v.Response{
		"200": {Desc: "ok", V: []any{schemaBasic{}}},
	})
	require.NoError(t, err)

	op := &openapi3.Operation{
		OperationID: "createBasic",
		RequestBody: req,
		Responses:   resp,
	}
	v.AddPath("/basics", http.MethodPost, doc, op)

	err = doc.Validate(context.Background())
	require.NoError(t, err)

	// Ensure it marshals
	b, err := doc.MarshalJSON()
	require.NoError(t, err)
	assert.NotEmpty(t, b)
}
