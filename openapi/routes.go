package openapi

import (
	"net/http"

	"github.com/getkin/kin-openapi/openapi3"
)

// RegisterFunc adapts an application's router without importing that router.
// Registration usually happens once at startup, before serving requests.
//
// For a standard http.ServeMux:
//
//	func(method, path string, h http.Handler) { mux.Handle(method+" "+path, h) }
//
// For chi:
//
//	func(method, path string, h http.Handler) { r.Method(method, path, h) }
//
// For gorilla/mux:
//
//	func(method, path string, h http.Handler) { r.Handle(path, h).Methods(method) }
//
// Router-specific path syntax is kept in the application. OpenAPI paths must
// use {name} for path parameters. The callback can translate paths if needed.
type RegisterFunc func(method, path string, handler http.Handler)

// Routes couples route registration with OpenAPI documentation. It does not
// serve requests or wrap handlers: authentication, middleware, decoding,
// validation, and response writing remain under the application's control.
// Like most routers, Routes is intended for single-threaded startup registration.
type Routes struct {
	doc      *openapi3.T
	register RegisterFunc
}

// NewRoutes creates a router-neutral documented route registrar.
// It panics for a nil document or registration callback. Build and validate the
// complete document before serving SwaggerHandler.
func NewRoutes(doc *openapi3.T, register RegisterFunc) *Routes {
	if doc == nil || register == nil {
		panic("openapi.NewRoutes requires a document and registration callback")
	}
	return &Routes{doc: doc, register: register}
}

// Handle registers handler and its operation together. Like the endpoint
// convenience helpers, invalid schema definitions panic at startup. Request and
// response models must be declared in ep; they cannot be inferred from Handler.
func (r *Routes) Handle(method, path, operationID string, ep Endpoint, handler http.Handler) {
	if handler == nil {
		panic("openapi.Routes.Handle requires a handler")
	}
	r.HandleOperation(method, path, operationForEndpoint(operationID, ep), handler)
}

// HandleOperation registers an existing handler and a fully customized OpenAPI
// operation. Use it instead of Handle when Endpoint's convenience fields are
// insufficient (for example security, non-JSON bodies, or custom media types).
// The handler is passed unchanged to the router adapter, so middleware can be
// applied before registration or inside the adapter.
func (r *Routes) HandleOperation(method, path string, op *openapi3.Operation, handler http.Handler) {
	if op == nil || handler == nil {
		panic("openapi.Routes.HandleOperation requires an operation and handler")
	}
	r.register(method, path, handler)
	AddPath(path, method, r.doc, op)
}

// HandleFunc is Handle for an ordinary HTTP handler function.
func (r *Routes) HandleFunc(method, path, operationID string, ep Endpoint, handler http.HandlerFunc) {
	if handler == nil {
		panic("openapi.Routes.HandleFunc requires a handler")
	}
	r.Handle(method, path, operationID, ep, handler)
}
