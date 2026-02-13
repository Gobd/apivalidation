package apivalidation

import (
	"bytes"
	"context"
	"embed"
	"io/fs"
	"net/http"
	"text/template"

	"github.com/getkin/kin-openapi/openapi3"
)

//go:embed swagger/*
var swagFS embed.FS

// SwaggerHandler returns an http.Handler that serves the Swagger UI for the
// given OpenAPI spec. The prefix is stripped automatically, so just mount it:
//
//	http.Handle("/swagger/", apivalidation.SwaggerHandlerMust("/swagger/", spec))
func SwaggerHandler(prefix string, s *openapi3.T) (http.Handler, error) {
	if err := s.Validate(context.Background()); err != nil {
		return nil, err
	}

	specJSON, err := s.MarshalJSON()
	if err != nil {
		return nil, err
	}

	tmpl, err := template.ParseFS(swagFS, "swagger/index.html")
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, map[string]any{"Docs": string(specJSON)}); err != nil {
		return nil, err
	}
	index := buf.Bytes()

	static, err := fs.Sub(swagFS, "swagger")
	if err != nil {
		return nil, err
	}
	files := http.FileServer(http.FS(static))

	return http.StripPrefix(prefix, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "", "/":
			_, _ = w.Write(index)
		case "/docs.json":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(specJSON)
		default:
			files.ServeHTTP(w, r)
		}
	})), nil
}

// SwaggerHandlerMust is like SwaggerHandler but panics on error.
func SwaggerHandlerMust(prefix string, s *openapi3.T) http.Handler {
	h, err := SwaggerHandler(prefix, s)
	if err != nil {
		panic(err)
	}
	return h
}
