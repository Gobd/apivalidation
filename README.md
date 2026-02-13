# apivalidation

Validation + OpenAPI 3 schema generation from a single source of truth. Define rules once as Go code, get runtime validation and generated docs for free.

## Quick Start

Implement `Ruler` on your struct:

```go
type Order struct {
    Name   string  `json:"name"`
    Amount float64 `json:"amount"`
    Status string  `json:"status"`
}

func (o *Order) Rules() []*v.FieldRules {
    return []*v.FieldRules{
        v.Field(&o.Name, v.Required, v.Length(1, 100)),
        v.Field(&o.Amount, v.Required, v.Min(0.0)),
        v.Field(&o.Status, v.Required, v.In("pending", "paid", "cancelled")),
    }
}
```

Validate:

```go
err := v.Validate(&order)
```

Unmarshal + normalize + validate in one call:

```go
err := v.UnmarshalAndValidate(r.Body, &order)
```

Generate OpenAPI schema:

```go
req, err := v.NewRequest(Order{})
```

That's it. `Rules()` drives all three.

## Struct Tags

| Tag | Effect |
|-----|--------|
| `json:"name"` | Field name in errors and schema |
| `json:"-"` | Field excluded from schema and validation |
| `docs:"skip"` | Field excluded from OpenAPI schema |
| `validate:"-"` | Field intentionally has no rules (for `MissingRules` check) |

## Nested Structs, Slices, Maps

Child structs that implement `Ruler` are validated automatically. Just declare the field in the parent's `Rules()`:

```go
type LineItem struct {
    SKU string
    Qty int
}

func (l *LineItem) Rules() []*v.FieldRules {
    return []*v.FieldRules{
        v.Field(&l.SKU, v.Required),
        v.Field(&l.Qty, v.Required, v.Min(1)),
    }
}

type Cart struct {
    Items []LineItem
}

func (c *Cart) Rules() []*v.FieldRules {
    return []*v.FieldRules{
        v.Field(&c.Items, v.Unique(func(i int) any { return c.Items[i].SKU }, "unique SKUs")),
    }
}
```

`v.Validate(&cart)` validates the cart, checks uniqueness, and validates every `LineItem` — all automatically via `Rules()`.

Works the same for `map[string]Ruler`, `[]*Ruler`, and nested collections like `map[string][]Ruler`.

## Embedded Structs

Embedded `Ruler` structs get flat error keys (not nested under the embedded type name):

```go
type Base struct {
    ID string
}

func (b *Base) Rules() []*v.FieldRules {
    return []*v.FieldRules{v.Field(&b.ID, v.Required)}
}

type Product struct {
    Base
    Name string
}

func (p *Product) Rules() []*v.FieldRules {
    return []*v.FieldRules{
        v.Field(&p.Base),
        v.Field(&p.Name, v.Required),
    }
}
// Error keys: {"ID": "...", "Name": "..."} — not {"Base": {"ID": "..."}}
```

## Value Types (Non-Struct)

For named types like `type PaymentMethod string`, implement `ValueRuler` to define validation rules once. They apply automatically wherever the type appears as a struct field — both validation and OpenAPI schema generation.

```go
type PaymentMethod string

const (
    PaymentACH PaymentMethod = "ach"
    PaymentCC  PaymentMethod = "cc"
)

func (p PaymentMethod) ValueRules() []v.Rule {
    return []v.Rule{v.In(PaymentACH, PaymentCC)}
}
```

Use it in any struct — no extra rules needed for the field's own validation:

```go
type Checkout struct {
    Method PaymentMethod `json:"method"`
    Total  float64       `json:"total"`
}

func (c *Checkout) Rules() []*v.FieldRules {
    return []*v.FieldRules{
        v.Field(&c.Method, v.Required), // In() comes from ValueRules automatically
        v.Field(&c.Total, v.Required, v.Min(0.01)),
    }
}
```

Any rule works in `ValueRules`: `In`, `Min`, `Max`, `Length`, `Describe`, custom rules — all of it.

## Normalization

Implement `Normalizer` to run custom logic after JSON decoding and before validation:

```go
func (o *Order) Normalize() {
    v.StructTrimSpace(o)
    o.Status = strings.ToLower(o.Status)
}
```

`UnmarshalAndValidate` calls `Normalize()` automatically and recurses into nested structs, slices, and maps that also implement `Normalizer`. Top level runs first, then children.

Use `ContextNormalizer` with `UnmarshalAndValidateCtx` when you need a context.

## Transform Utilities

```go
v.StructTrimSpace(&s)                          // strings.TrimSpace on all string fields
v.StructToLower(&s)                             // strings.ToLower on all string fields
v.StructStringFunc(&s, myFunc)                  // any func(string) string
v.StructMulti(&s, v.StructTrimSpace, v.StructToLower) // chain multiple
```

These walk struct fields, pointers, slices, and map values recursively.

## Catching Forgotten Fields

`MissingRules` returns field names that have no rule. Use in tests to ensure full coverage:

```go
func TestRulesCoverage(t *testing.T) {
    assert.Empty(t, v.MissingRules(&Order{}))
    assert.Empty(t, v.MissingRules(&Cart{}))
}
```

Tag fields that intentionally have no rules with `validate:"-"` so they don't show up as missing.

## OpenAPI Schema Generation

```go
doc := v.DocBase("my-service", "My API", "1.0.0")

req := v.NewRequestMust(Order{})
resp, _ := v.NewResponse(map[string]v.Response{
    "200": {Desc: "success", V: []any{Order{}}},
    "400": {Desc: "bad request", V: []any{ErrorResponse{}}},
})

v.AddPath("/orders", http.MethodPost, doc, &openapi3.Operation{
    OperationID: "createOrder",
    RequestBody: req,
    Responses:   resp,
})
```

Serve a Swagger UI with `SwaggerHandler` or `SwaggerHandlerMust` (standard `http.Handler`):

```go
http.Handle("/swagger/", v.SwaggerHandlerMust("/swagger/", doc))
```
