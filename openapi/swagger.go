package openapi

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

//go:embed swagger/*
var swagFS embed.FS

// SwaggerHandler returns an [http.Handler] that serves the Swagger UI for the
// given OpenAPI spec. The prefix is stripped automatically, so just mount it:
//
//	http.Handle("/swagger/", openapi.SwaggerHandlerMust("/swagger/", spec))
func SwaggerHandler(prefix string, s *openapi3.T) (http.Handler, error) {
	if s == nil {
		return nil, fmt.Errorf("OpenAPI document must not be nil")
	}
	if prefix != "" && !strings.HasPrefix(prefix, "/") {
		return nil, fmt.Errorf("Swagger prefix must start with /: %q", prefix)
	}
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

	return http.StripPrefix(strings.TrimSuffix(prefix, "/"), http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "", "/":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write(index)
		case "/docs.json":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(specJSON)
		default:
			files.ServeHTTP(w, r)
		}
	})), nil
}

// SwaggerHandlerMust is like [SwaggerHandler] but panics on error.
func SwaggerHandlerMust(prefix string, s *openapi3.T) http.Handler {
	h, err := SwaggerHandler(prefix, s)
	if err != nil {
		panic(err)
	}
	return h
}
