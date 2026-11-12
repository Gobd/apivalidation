package openapi_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Gobd/apivalidation/openapi"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRoutesRegisterHandlerAndDocs(t *testing.T) {
	doc := openapi.DocBase("test", "test", "1.0")
	mux := http.NewServeMux()
	var registeredMethod, registeredPath string
	routes := openapi.NewRoutes(doc, func(method, path string, h http.Handler) {
		registeredMethod, registeredPath = method, path
		mux.Handle(method+" "+path, h)
	})
	routes.HandleFunc(http.MethodGet, "/items/{id}", "getItem", openapi.Endpoint{
		Response:   "",
		Parameters: openapi3.Parameters{{Value: openapi3.NewPathParameter("id").WithSchema(openapi3.NewStringSchema())}},
	}, func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(r.PathValue("id"))) })
	assert.Equal(t, http.MethodGet, registeredMethod)
	assert.Equal(t, "/items/{id}", registeredPath)
	assert.Equal(t, "getItem", doc.Paths.Value("/items/{id}").Get.OperationID)
	require.NoError(t, doc.Validate(context.Background()))
	result := httptest.NewRecorder()
	mux.ServeHTTP(result, httptest.NewRequest(http.MethodGet, "/items/42", nil))
	assert.Equal(t, http.StatusOK, result.Code)
	assert.Equal(t, "42", result.Body.String())
	result = httptest.NewRecorder()
	mux.ServeHTTP(result, httptest.NewRequest(http.MethodPost, "/items/42", nil))
	assert.Equal(t, http.StatusMethodNotAllowed, result.Code)
}

func TestRoutesFailureDoesNotRegisterPartialOperation(t *testing.T) {
	doc := openapi.DocBase("test", "test", "1.0")
	registered := false
	routes := openapi.NewRoutes(doc, func(string, string, http.Handler) { registered = true })
	require.Panics(t, func() {
		routes.HandleFunc(http.MethodPost, "/bad", "bad", openapi.Endpoint{Requests: []any{nil}}, func(http.ResponseWriter, *http.Request) {})
	})
	assert.False(t, registered)
	assert.Nil(t, doc.Paths.Value("/bad"))
	routes = openapi.NewRoutes(doc, func(string, string, http.Handler) { panic("router rejected route") })
	require.Panics(t, func() {
		routes.HandleFunc(http.MethodGet, "/bad", "bad", openapi.Endpoint{}, func(http.ResponseWriter, *http.Request) {})
	})
	assert.Nil(t, doc.Paths.Value("/bad"))
}

func TestRoutesPreserveMiddlewareAndRawOperation(t *testing.T) {
	doc := openapi.DocBase("test", "test", "1.0")
	mux := http.NewServeMux()
	middleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Middleware", "present")
			next.ServeHTTP(w, r)
		})
	}
	var registered http.Handler
	routes := openapi.NewRoutes(doc, func(method, path string, h http.Handler) {
		registered = h
		mux.Handle(method+" "+path, middleware(h))
	})
	op := &openapi3.Operation{
		OperationID: "custom",
		Tags:        []string{"custom"},
		Responses:   openapi.NewResponseMust(map[string]openapi.Response{"202": {Desc: "Accepted"}}),
	}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusAccepted) })
	wrapped := &struct{ http.Handler }{middleware(handler)}
	routes.HandleOperation(http.MethodPost, "/custom", op, wrapped)
	assert.Same(t, wrapped, registered)
	assert.Same(t, op, doc.Paths.Value("/custom").Post)
	require.NoError(t, doc.Validate(context.Background()))
	result := httptest.NewRecorder()
	mux.ServeHTTP(result, httptest.NewRequest(http.MethodPost, "/custom", nil))
	assert.Equal(t, http.StatusAccepted, result.Code)
	assert.Equal(t, "present", result.Header().Get("X-Middleware"))
}

func TestRoutesRequireInputs(t *testing.T) {
	doc := openapi.DocBase("test", "test", "1.0")
	require.Panics(t, func() { openapi.NewRoutes(nil, func(string, string, http.Handler) {}) })
	require.Panics(t, func() { openapi.NewRoutes(doc, nil) })
	routes := openapi.NewRoutes(doc, func(string, string, http.Handler) {})
	require.Panics(t, func() {
		routes.HandleOperation(http.MethodGet, "/bad", nil, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	})
	require.Panics(t, func() { routes.HandleOperation(http.MethodGet, "/bad", &openapi3.Operation{}, nil) })
	require.Panics(t, func() { routes.Handle(http.MethodGet, "/bad", "bad", openapi.Endpoint{}, nil) })
	require.Panics(t, func() { routes.HandleFunc(http.MethodGet, "/bad", "bad", openapi.Endpoint{}, nil) })
	assert.Nil(t, doc.Paths.Value("/bad"))
}
