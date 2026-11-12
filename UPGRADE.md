# Upgrade notes

This update keeps the `Rules()` / `Field()` API, but deliberately changes JSON
and validation behaviour. Review these items before updating a service.

## Go and dependencies

All three modules now require **Go 1.27.4**, the latest stable release checked
while preparing this update. This includes standard-library security fixes.
No `GOEXPERIMENT` flag is needed for `encoding/json/v2` on this toolchain.

Updated direct dependencies:

| Dependency | Previous | Updated |
|---|---|---|
| kin-openapi | v0.133.0 | v0.153.0 |
| ozzo-validation/v4 | v4.3.0 | v4.5.0 |
| testify | v1.9.0 | v1.12.1 |
| govalidator | unversioned 2023 pseudo-version | **govalidator/v12 v12.0.0** |
| Chi (example only) | v5.2.5 | v5.5.0 |
| Gorilla/mux (example only) | v1.8.1 | v1.8.1 (already latest) |
| Embedded Swagger UI | v5.4.2 | v5.34.1 |

The govalidator major upgrade includes an import-path change in `is`. Its old
module remains indirect solely because ozzo's dependency tests import the legacy
path; it is not used by this module's build. No router dependency was added to the
main module. Used indirect dependencies are also updated, including jsonpointer
v1.0.2 and YAML v4. GitHub Actions are upgraded to their latest major releases.
Swagger assets were verified against the npm tarball's SHA-512 integrity value;
their upstream LICENSE and NOTICE are included.

## JSON v2 defaults

The following functions accept additional `...jsonv2.Options` parameters:

- `UnmarshalAndValidate`
- `UnmarshalAndValidateCtx`
- `DecodeAndValidate`
- `DecodeAndValidateContext`

Ordinary calls remain source-compatible. Code that stores one of these functions
in an explicitly non-variadic function type must update that type or wrap the
function in a closure.

Unknown fields remain **allowed**. Defaults now match `encoding/json/v2`:
case-sensitive field matching, rejection of duplicate names and invalid UTF-8,
and v2's documented merge/null/type semantics. Normalization runs only after
successful decoding, and validation runs only after normalization.

For legacy JSON semantics:

```go
import jsonv1 "encoding/json"

err := v.UnmarshalAndValidate(body, &dst, jsonv1.DefaultOptionsV1())
err = v.DecodeAndValidate(r.Body, &dst, jsonv1.DefaultOptionsV1())
```

Later options override earlier ones. Pass `jsonv2.RejectUnknownMembers(true)`
only if your application intentionally wants that policy.

Both byte and reader helpers require **exactly one JSON value**. The reader
helper reads to EOF and rejects extra objects or trailing garbage. Whitespace is
allowed. `DefaultOptionsV1()` does not restore the old decoder's acceptance of
trailing data. For a JSON message stream, decode separately and call `Normalize`
and `Validate` yourself. Bound HTTP request bodies with `http.MaxBytesReader`.

`KeyIn` also uses JSON v2 for its JSON-object key inspection.

## Previously accepted invalid values now fail

- Pointer/interface collections and named `ValueRuler` collection elements are
  recursively validated. Requests that slipped through these paths may now fail.
- `When(...).Else(...)` now actually enforces its alternative rules.
- `Skip` now really skips subsequent field rules and automatic child validation.
- `Nil.When(false)` and `Empty.When(false)` now disable those rules.
- `Date(...).Min(...).Max(...)` now enforces **inclusive** date bounds instead of
  documenting bounds without validating them.
- Invalid uniqueness key types return errors instead of reflection/map panics.

Nil pointers still pass unless a containing field requires presence. Shared
children report errors at every path; recursive cycles do not overflow the stack.
`MissingRules` now reports eligible fields of non-Ruler structs instead of
silently returning an empty result. Nil/non-struct inputs return no missing fields.

## Normalization and transforms

One shared cycle-safe walker now handles pointers, interfaces, arrays, nested
collections, map values, and named strings. It skips unexported fields and leaves
map keys unchanged. Shared references are transformed/normalized once per pass.
Normalizers run parent-first and should be idempotent. `Normalize` and
`NormalizeCtx` are available for separately decoded or non-JSON inputs.

Newly supported collection paths now run normalizers that were previously skipped;
check custom normalizers for unexpected side effects or non-idempotent mutations.

## Schema changes

- Untagged exported fields are included using their Go names.
- `docs:"skip"` is respected even on structs without `Rules()`.
- String lengths use `minLength`/`maxLength`, collections use item/property counts.
- `Each` modifies the element/value schema, not the collection schema.
- Required pointer fields are non-nullable. `NotNil` on nil-capable fields also
  marks them required without rejecting zero/false contents.
- Standard date layouts produce `date`/`date-time` formats. Custom Go layouts
  and numeric-string bounds are descriptions, not ineffective schema keywords.
- Key-based uniqueness is described rather than mapped to `uniqueItems`, which
  compares entire JSON elements and is not equivalent to key uniqueness.
- Responses without body types do not publish a fictitious JSON body.
- Empty endpoint metadata gets an explicit bodyless `200` response.
- Standard HTTP methods are delegated to kin-openapi; `QUERY` and custom methods
  require setting `doc.OpenAPI = "3.2.0"`. `DocBase` still defaults to 3.0.3.

Regenerate and review published schemas/client code. Not every runtime behaviour
is representable in OpenAPI, especially empty-value skipping, conditional Go
logic, or arbitrary callbacks.

`json:"-"` excludes a field from JSON/schema, **not** explicitly declared
validation. The previous README claim that it disabled validation was incorrect.
`validate:"-"` affects `MissingRules` checks only.

## Additive APIs and composition

- `ByTyped[T]`: typed custom validation callbacks.
- `FieldCheck[T]`: field and callback types checked together at compile time.
- `UniqueBy[T, K comparable]`: typed key uniqueness without request-slice capture.
- `is.EmailFormat`: syntax-only email validation without DNS/MX lookups.
- `openapi.Query`: HTTP QUERY documentation.
- `openapi.NewRoutes`: optional, router-neutral registration and documentation.
- `Routes.HandleOperation`: standard handlers plus fully customized operations.
- `Endpoint.Parameters`: explicit path/query/header/cookie metadata.

Existing `ValidateStruct`, `By`, `Custom`, `Unique`, `AddPath`, and endpoint helpers
remain available. Router registration does not wrap handlers or impose middleware,
decoding, validation, serialization, or authentication. Root file names were made
clearer; package import paths are unchanged. No ozzo fork or new rule DSL is required.

## Checks

CI enforces gofumpt v0.12.0 formatting, module tidiness, vet, race tests,
coverage, builds, and known
vulnerabilities for the main module and both router examples. Unit/integration
coverage includes actual HTTP QUERY requests, route/middleware composition,
Swagger asset/spec routes, and generated schema validation against JSON values.
Allocation benchmarks and a JSON fuzz target are included.
