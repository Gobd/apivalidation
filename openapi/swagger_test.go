package openapi_test

import (
	"encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Gobd/apivalidation/openapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSwaggerRoutes(t *testing.T) {
	doc := openapi.DocBase("test", "test document", "1.0")
	openapi.Get(doc, "/health", "health", openapi.Endpoint{})
	for _, prefix := range []string{"/swagger/", "/swagger", "/", ""} {
		t.Run(prefix, func(t *testing.T) {
			h, err := openapi.SwaggerHandler(prefix, doc)
			require.NoError(t, err)
			root := strings.TrimSuffix(prefix, "/")
			for _, path := range []string{"/", "/docs.json", "/swagger-ui.css", "/swagger-ui-bundle.js"} {
				r := httptest.NewRecorder()
				h.ServeHTTP(r, httptest.NewRequest(http.MethodGet, root+path, nil))
				require.Equal(t, http.StatusOK, r.Code, path)
				switch path {
				case "/":
					assert.Contains(t, r.Header().Get("Content-Type"), "text/html")
					assert.Contains(t, r.Body.String(), "JSON.parse(")
				case "/docs.json":
					assert.Equal(t, "application/json", r.Header().Get("Content-Type"))
					var spec map[string]any
					require.NoError(t, json.Unmarshal(r.Body.Bytes(), &spec))
					assert.Equal(t, "3.0.3", spec["openapi"])
				}
			}
			r := httptest.NewRecorder()
			h.ServeHTTP(r, httptest.NewRequest(http.MethodGet, root+"/missing", nil))
			assert.Equal(t, http.StatusNotFound, r.Code)
		})
	}
}

func TestSwaggerEscapesEmbeddedJSON(t *testing.T) {
	doc := openapi.DocBase("test", `</script><script>alert("xss")</script>`, "1.0")
	openapi.Get(doc, "/health", "health", openapi.Endpoint{})
	h, err := openapi.SwaggerHandler("/swagger/", doc)
	require.NoError(t, err)
	r := httptest.NewRecorder()
	h.ServeHTTP(r, httptest.NewRequest(http.MethodGet, "/swagger/", nil))
	assert.NotContains(t, r.Body.String(), `</script><script>alert`)
	// Decode the JavaScript string literal and then its JSON content to verify
	// that safe escaping preserves the exact spec description.
	body := r.Body.String()
	start := strings.Index(body, "JSON.parse(") + len("JSON.parse(")
	end := start + strings.Index(body[start:], "),")
	var specJSON string
	require.NoError(t, json.Unmarshal([]byte(body[start:end]), &specJSON))
	var spec map[string]any
	require.NoError(t, json.Unmarshal([]byte(specJSON), &spec))
	assert.Equal(t, doc.Info.Description, spec["info"].(map[string]any)["description"])
}

func TestSwaggerErrors(t *testing.T) {
	_, err := openapi.SwaggerHandler("/swagger/", nil)
	require.Error(t, err)
	_, err = openapi.SwaggerHandler("not-a-path", openapi.DocBase("test", "", "1.0"))
	require.Error(t, err)
	require.Panics(t, func() { openapi.SwaggerHandlerMust("/swagger/", nil) })
}
