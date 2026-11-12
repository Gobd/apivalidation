package openapi_test

import (
	"context"
	"encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	v "github.com/Gobd/apivalidation"
	"github.com/Gobd/apivalidation/openapi"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type queryRequest struct {
	Term string `json:"term"`
}

func (s *queryRequest) Rules() []*v.FieldRules { return []*v.FieldRules{v.Field(&s.Term, v.Required)} }

func TestQueryOpenAPI32(t *testing.T) {
	doc := openapi.DocBase("search", "Search API", "1.0")
	doc.OpenAPI = "3.2.0"
	openapi.Query(doc, "/search", "search", openapi.Endpoint{Request: queryRequest{}, Response: []string{}})
	p := doc.Paths.Value("/search")
	require.NotNil(t, p.Query)
	require.NotNil(t, p.Query.RequestBody)
	require.NoError(t, doc.Validate(context.Background()))
	b, err := doc.MarshalJSON()
	require.NoError(t, err)
	assert.Contains(t, string(b), `"query"`)
	_, err = openapi.SwaggerHandler("/swagger/", doc)
	require.NoError(t, err)
	doc.OpenAPI = "3.0.3"
	require.Error(t, doc.Validate(context.Background()))
	doc.OpenAPI = "3.1.0"
	require.Error(t, doc.Validate(context.Background()))
}

func TestGoQueryRequestAndRouting(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("QUERY /search", func(w http.ResponseWriter, r *http.Request) {
		var request queryRequest
		if err := v.DecodeAndValidateContext(r.Context(), r.Body, &request); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.MarshalWrite(w, []string{request.Term})
	})
	s := httptest.NewServer(mux)
	defer s.Close()
	req, err := http.NewRequest(openapi3.MethodQuery, s.URL+"/search", strings.NewReader(`{"term":"auction"}`))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.Client().Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var result []string
	require.NoError(t, json.UnmarshalRead(resp.Body, &result))
	assert.Equal(t, []string{"auction"}, result)
}

func TestBodylessResponsesAndMethods(t *testing.T) {
	doc := openapi.DocBase("test", "test", "1.0")
	openapi.Delete(doc, "/items", "delete", openapi.Endpoint{Responses: map[string]openapi.Response{"204": {Desc: "Deleted"}}})
	assert.Empty(t, doc.Paths.Value("/items").Delete.Responses.Value("204").Value.Content)
	for _, method := range []string{http.MethodHead, http.MethodOptions, http.MethodTrace} {
		op := &openapi3.Operation{OperationID: method, Responses: openapi.NewResponseMust(map[string]openapi.Response{"200": {Desc: "OK"}})}
		openapi.AddPath("/items", method, doc, op)
		assert.Same(t, op, doc.Paths.Value("/items").GetOperation(method))
	}
	require.NoError(t, doc.Validate(context.Background()))
}
