# apivalidation

Go validation, normalization, and OpenAPI schemas from one set of rules. Define
rules as Go code instead of maintaining separate validation tags and API docs.

**Requires Go 1.27.4 or newer.** JSON decoding uses the standard library's
`encoding/json/v2`, with caller-supplied options and an explicit v1 compatibility
option. See [JSON behaviour](#json-behaviour-and-options) and [UPGRADE.md](UPGRADE.md).

```sh
go get github.com/Gobd/apivalidation
```

## Quick start

```go
import (
    v "github.com/Gobd/apivalidation"
    "github.com/Gobd/apivalidation/openapi"
)

type Order struct {
    Name   string  `json:"name"`
    Amount float64 `json:"amount"`
    Status string  `json:"status"`
}

func (o *Order) Rules() []*v.FieldRules {
    return []*v.FieldRules{
        v.Field(&o.Name, v.Required, v.Length(1, 100)),
        v.Field(&o.Amount, v.Required, v.Min(0.01)),
        v.Field(&o.Status, v.Required, v.In("pending", "paid", "cancelled")),
    }
}
```

Use a **pointer receiver** for `Rules()` so its field pointers belong to the
instance being validated.

```go
err := v.Validate(&order)                         // already decoded
err = v.UnmarshalAndValidate(bodyBytes, &order)    // []byte
err = v.DecodeAndValidate(r.Body, &order)           // io.Reader / HTTP body
req, err := openapi.NewRequest(Order{})            // OpenAPI request schema
```

`Validate` does not normalize. The JSON helpers decode, normalize recursively,
then validate. `ValidateCtx`, `UnmarshalAndValidateCtx`, and
`DecodeAndValidateContext` propagate a context to `ContextRuler`,
`ContextNormalizer`, and context-aware rules.

For a complete working HTTP server:

```sh
go run ./_example
# Browse http://localhost:8080/swagger/
curl -X POST http://localhost:8080/orders \
  -H 'Content-Type: application/json' \
  -d '{"customer_name":"Alice","item_count":2,"total":10}'
```

Router-specific examples: `cd _example/chi && go run .` or
`cd _example/gorilla && go run .`.

## JSON behaviour and options

Both JSON helpers use **JSON v2 defaults**:

- Unknown object members are **allowed**, not rejected.
- Struct field names are matched case-sensitively.
- Duplicate object names and invalid UTF-8 are rejected.
- Input must contain exactly one JSON value, followed only by whitespace.

For example, `{"name":"Alice"} garbage` and `{"name":"Alice"}{}` are not
single-value JSON documents. The reader helper now rejects them instead of
silently accepting the first object. It reads until EOF; it is not a decoder for
an open-ended stream of JSON messages.

Pass any `encoding/json/v2` or `encoding/json/jsontext` options directly:

```go
import (
    jsonv1 "encoding/json"
    jsonv2 "encoding/json/v2"
)

// Full v1 decoding semantics, including case-insensitive names and duplicates:
err := v.UnmarshalAndValidate(body, &order, jsonv1.DefaultOptionsV1())
err = v.DecodeAndValidate(r.Body, &order, jsonv1.DefaultOptionsV1())

// Options are optional; later options override earlier ones:
err = v.UnmarshalAndValidate(body, &order,
    jsonv1.DefaultOptionsV1(),
    jsonv2.MatchCaseInsensitiveNames(false),
)

// Reject unknown fields only if your application deliberately requests it:
err = v.DecodeAndValidate(r.Body, &order, jsonv2.RejectUnknownMembers(true))
```

The v1 option changes JSON semantics, **not** the single-document requirement.
For JSON streams or custom decoding, decode separately, then call `v.Normalize`
and `v.Validate`. For HTTP requests, bound the body before decoding:

```go
r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MiB
err := v.DecodeAndValidateContext(r.Context(), r.Body, &order)
```

Decoding, normalization, and validation can partially modify the destination even
when an error is returned. Decode into a fresh local value before persisting it.

## Normalization

```go
import "github.com/Gobd/apivalidation/transform"

func (o *Order) Normalize() {
    transform.StructTrimSpace(o)
    o.Status = strings.ToLower(o.Status)
}
```

The JSON helpers call normalizers parent-first through structs, pointers,
slices, arrays, map values, and interfaces—even if only a child implements the
interface. Unexported fields are skipped. Cycles are safe; shared references are
visited once. Normalizers should be idempotent, especially when a parent also
uses recursive transforms.

For non-JSON values or separate decoding:

```go
v.Normalize(&order)
v.NormalizeCtx(ctx, &order)
```

String utilities handle the same collections, including named string types:

```go
transform.StructTrimSpace(&s)
transform.StructToLower(&s)
transform.StructStringFunc(&s, myFunc)
transform.StructMulti(&s, transform.StructTrimSpace, transform.StructToLower)
```

Map keys and unexported fields are not modified.

## Nested values and collections

Child structs implementing `Ruler` or `ContextRuler` are validated when their
field is declared in the parent's rules. Elements in slices, arrays, maps,
pointer collections, interfaces, and nested collections are also validated.
Errors retain their field names, map keys, and element indices.

```go
func (c *Cart) Rules() []*v.FieldRules {
    return []*v.FieldRules{
        v.Field(&c.Items, v.UniqueBy(
            func(item LineItem) string { return item.SKU }, "unique SKUs")),
    }
}
```

`UniqueBy[T, K]` receives an element and uses a comparable key; it avoids
capturing a request-specific slice. The older index-based `Unique` remains
available. Key-based uniqueness is described in OpenAPI, not encoded as
`uniqueItems`, which would compare entire objects instead of their keys.

Embedded `Ruler` structs declared with `v.Field(&s.Base)` have flat error keys
and flat schema properties when JSON also embeds their fields.

For named scalar types, implement `ValueRuler`:

```go
type PaymentMethod string

func (p PaymentMethod) ValueRules() []v.Rule {
    return []v.Rule{v.In(PaymentMethod("ach"), PaymentMethod("cc"))}
}
```

These rules apply to declared struct fields and collection elements, and are
used in schema generation. Parent fields may add rules such as `Required`.

## Typed custom rules

`Field[T]` is already generic. Typed callbacks avoid assertions inside custom
rules, while `FieldCheck` also checks that the field and callback types agree:

```go
v.FieldCheck(&o.Name, func(name string) error {
    if strings.Contains(name, "\n") {
        return errors.New("must be a single line")
    }
    return nil
}, "single-line name", v.Required)

// Or combine a typed check with existing field rules:
v.Field(&o.Name, v.Required, v.ByTyped(func(name string) error {
    return checkName(name)
}, "acceptable name"))
```

The existing `By`, `Custom`, and `Rule` APIs remain supported. A typed rule
applied to the wrong field type returns an error instead of panicking.

## Important validation semantics

- `Required` means **non-empty**, not merely present in JSON. Numeric zero and
  `false` fail it. Use pointer fields plus `NotNil` when zero/false are valid but
  the field must be supplied.
- Most ozzo rules skip empty values. Add `Required` when emptiness is invalid.
- `Length` counts Unicode runes for strings and size for collections. A zero
  maximum is unbounded, except `Length(0, 0)`, which requires emptiness.
- `Default`, `Example`, and `Describe` are documentation-only; they do not set
  runtime values. `Date(...).Min(...).Max(...)` enforces inclusive date bounds.
- `When(...).Else(...)` validates the selected branch. OpenAPI describes these
  conditions in prose rather than applying a branch unconditionally.
- Nil pointers pass unless a containing field uses `Required` or `NotNil`.
- Values without `Ruler`/`ValueRuler` have no rules and pass validation.
- `is.Email` performs a DNS/MX lookup; account for network latency and failures.
  Use `is.EmailFormat` for syntax-only validation without network I/O.

`ValidateStruct` is an advanced explicit-rule helper for types you cannot add
`Rules()` to. Normal application code should use `Validate`.

## Tags and rule coverage

| Tag | Effect |
|-----|--------|
| `json:"name"` | Name used in JSON, field errors, and schemas |
| No JSON name | Exported field uses its Go name |
| `json:"-"` | Excluded from JSON/schema; explicit validation rules still run |
| `docs:"skip"` | Excluded from OpenAPI schema and `MissingRules` checks |
| `validate:"-"` | Intentionally excluded from `MissingRules` checks, not validation |

Use `MissingRules` in tests to catch forgotten fields:

```go
assert.Empty(t, v.MissingRules(&Order{}))
assert.Empty(t, v.MissingRules(&Cart{}))
```

A struct without `Ruler` reports its eligible fields as missing. This checks rule
coverage, not the correctness of the rules themselves.

## OpenAPI documents and Swagger UI

```go
doc := openapi.DocBase("my-service", "My API", "1.0.0")

openapi.Post(doc, "/orders", "createOrder", openapi.Endpoint{
    Request: Order{},
    Responses: map[string]openapi.Response{
        "200": {Desc: "Created", Bodies: []any{Order{}}},
        "400": {Desc: "Bad request", Bodies: []any{ErrorResponse{}}},
    },
})

http.Handle("/swagger/", openapi.SwaggerHandlerMust("/swagger/", doc))
```

The handler validates the spec and serves `/swagger/docs.json` plus embedded UI
assets. Use `SwaggerHandler` instead of the `Must` form to handle startup errors
without panicking. The handlers serve a snapshot; rebuild one after changing a
spec. Responses without `Bodies`, such as `204`, have no content schema.

For lower-level assembly, use `NewRequest`, `NewResponse`, `AddPath`, and
`NewSchemaRefForValue`. Multiple body types generate `oneOf`. Schema generation
uses fresh values and a background context; do not depend on live request data
inside `Rules()` to generate documentation. Some runtime semantics (including
empty-value skipping and arbitrary callbacks) cannot be expressed exactly in
OpenAPI; test generated schemas against representative payloads when exact
client-side parity matters.

### Register routes and documentation together (optional)

Keep an ordinary router and supply one adapter. The library does not import Chi
or Gorilla, install middleware, or take ownership of decoding/responses:

```go
mux := http.NewServeMux()
routes := openapi.NewRoutes(doc, func(method, path string, h http.Handler) {
    mux.Handle(method+" "+path, h)
})

routes.HandleFunc(http.MethodPost, "/orders", "createOrder", openapi.Endpoint{
    Request: Order{}, Response: Order{},
}, createOrderHandler)
```

For Chi, the adapter body is `r.Method(method, path, h)`. For Gorilla, it is
`r.Handle(path, h).Methods(method)`; check the returned route's `GetError()` if
registration errors should fail startup. See the runnable examples for all three.

Composition remains standard Go HTTP composition:

```go
routes.Handle(http.MethodPost, "/orders", "createOrder", endpoint,
    authenticate(logRequests(http.HandlerFunc(createOrderHandler))))

// Or apply application-wide middleware inside the registration callback:
routes = openapi.NewRoutes(doc, func(method, path string, h http.Handler) {
    mux.Handle(method+" "+path, authenticate(logRequests(h)))
})

// Full OpenAPI control, without Endpoint's JSON-body convenience defaults:
routes.HandleOperation(http.MethodPost, "/upload", customOperation, uploadHandler)
```

`Endpoint.Parameters` accepts explicit OpenAPI path/query/header/cookie
parameters. Request/response models and parameters cannot be inferred from an
arbitrary `http.Handler`; declaring them does **not** decode or validate requests.
Use OpenAPI `{name}` paths and adapt router-specific syntax in the callback.
Build the Swagger handler **after** all route registration. Registration helpers
panic on invalid setup/schema definitions, like the existing endpoint helpers;
applications needing error-returning assembly can use `NewRequest`, `NewResponse`,
and `AddPath` directly. These are optional approaches, not a required framework.

### HTTP QUERY

`DocBase` defaults to OpenAPI 3.0.3 for compatibility. HTTP `QUERY` and custom
methods need OpenAPI 3.2:

```go
doc.OpenAPI = "3.2.0"
openapi.Query(doc, "/search", "search", openapi.Endpoint{
    Request: SearchRequest{}, Response: SearchResponse{},
})

mux.HandleFunc("QUERY /search", searchHandler)
```

Go supports string method names without a `net/http.MethodQuery` constant.
`openapi3.MethodQuery` is available from kin-openapi. Validate the complete spec
before serving; proxies, API gateways, and clients also need to support QUERY.

## Packages and performance

- `apivalidation`: rule API, validation, normalization, and schema generation.
- `openapi`: documents, endpoint helpers, and Swagger UI.
- `transform`: recursive string transformations.
- `is`: string-format rules.
- `internal/walk`: shared traversal; not a public API.

This library retains ozzo-validation instead of maintaining a fork. Typed
helpers are additive; existing rule definitions do not need a new DSL. Prefer
`UniqueBy` over index-based `Unique` for typed keys. Immutable rule values may be
constructed once and reused, but **do not cache field pointers across request
instances**, or mutate shared rule objects during validation. For example:

```go
var nameRules = []v.Rule{v.Required, v.Length(1, 100)} // build immutable rules once

func (o *Order) Rules() []*v.FieldRules {
    return []*v.FieldRules{v.Field(&o.Name, nameRules...)} // bind this instance
}
```

Run checks and allocation benchmarks:

```sh
go run mvdan.cc/gofumpt@v0.12.0 -w .
go test -race -cover ./...
go vet ./...
go test -run '^$' -bench . -benchmem .
(cd _example/chi && go test ./...)
(cd _example/gorilla && go test ./...)
```
