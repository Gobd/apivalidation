package openapi

import (
	"errors"
	"net/http"

	"github.com/getkin/kin-openapi/openapi3"
)

// Response describes an HTTP response with a description and body types for schema generation.
type Response struct {
	Desc   string
	Bodies []any
}

// Endpoint describes a single API operation for the convenience helpers
// [Get], [Post], [Put], [Patch], and [Delete].
type Endpoint struct {
	Summary     string
	Description string
	Request     any                 // single request body type (convenience)
	Requests    []any               // multiple request body types (oneOf)
	Response    any                 // single 200 response type (convenience)
	Responses   map[string]Response // full response map (overrides Response if both set)
}

// NewRequestMust is like [NewRequest] but panics on error.
func NewRequestMust(vs ...any) *openapi3.RequestBodyRef {
	o, err := NewRequest(vs...)
	if err != nil {
		panic(err)
	}
	return o
}

// NewRequest generates an OpenAPI request body schema from the given value types.
func NewRequest(vs ...any) (*openapi3.RequestBodyRef, error) {
	if len(vs) == 0 {
		return nil, errors.New("no values given")
	}

	base := &openapi3.RequestBodyRef{
		Value: &openapi3.RequestBody{
			Content: openapi3.Content{
				"application/json": &openapi3.MediaType{
					Schema: &openapi3.SchemaRef{
						Value: &openapi3.Schema{
							OneOf: openapi3.SchemaRefs{},
						},
					},
				},
			},
		},
	}

	wrapper := base.Value.Content["application/json"].Schema
	for i := range vs {
		schema, err := NewSchemaRefForValue(vs[i])
		if err != nil {
			return nil, err
		}
		wrapper.Value.OneOf = append(wrapper.Value.OneOf, schema)
	}

	if len(wrapper.Value.OneOf) == 1 {
		base.Value.Content["application/json"].Schema = wrapper.Value.OneOf[0]
	}

	return base, nil
}

// NewResponseMust is like [NewResponse] but panics on error.
// Map key is status code (e.g. "200", "4xx").
func NewResponseMust(vs map[string]Response) *openapi3.Responses {
	o, err := NewResponse(vs)
	if err != nil {
		panic(err)
	}
	return o
}

// NewResponse creates an OpenAPI responses object.
// Map key is status code (e.g. "200", "4xx").
func NewResponse(vs map[string]Response) (*openapi3.Responses, error) {
	if len(vs) == 0 {
		return nil, errors.New("no values given")
	}

	opts := make([]openapi3.NewResponsesOption, 0, len(vs))

	for statusCode := range vs {
		desc := vs[statusCode].Desc

		var refs openapi3.SchemaRefs

		for k := range vs[statusCode].Bodies {
			schema, err := NewSchemaRefForValue(vs[statusCode].Bodies[k])
			if err != nil {
				return nil, err
			}
			refs = append(refs, schema)
		}

		content := openapi3.Content{
			"application/json": &openapi3.MediaType{
				Schema: &openapi3.SchemaRef{
					Value: &openapi3.Schema{
						OneOf: refs,
					},
				},
			},
		}

		if len(refs) == 1 {
			content["application/json"].Schema = refs[0]
		}

		opt := openapi3.WithName(statusCode, &openapi3.Response{
			Description: &desc,
			Content:     content,
		})
		opts = append(opts, opt)
	}

	return openapi3.NewResponses(opts...), nil
}

// DocBase returns a basic OpenAPI 3.0.3 document structure.
func DocBase(serviceName, description, version string) *openapi3.T {
	return &openapi3.T{
		OpenAPI: "3.0.3",
		Info: &openapi3.Info{
			Title:       serviceName,
			Description: description,
			Version:     version,
		},
		Paths: &openapi3.Paths{},
	}
}

// AddPath adds an operation to the OpenAPI spec at the given path and method.
func AddPath(path, method string, s *openapi3.T, op *openapi3.Operation) {
	p := s.Paths.Value(path)
	if p == nil {
		p = &openapi3.PathItem{}
	}

	switch method {
	case http.MethodGet:
		p.Get = op
	case http.MethodPost:
		p.Post = op
	case http.MethodPut:
		p.Put = op
	case http.MethodPatch:
		p.Patch = op
	case http.MethodDelete:
		p.Delete = op
	}

	s.Paths.Set(path, p)
}

// addEndpoint builds an [openapi3.Operation] from ep and registers it at path+method.
func addEndpoint(doc *openapi3.T, path, method, operationID string, ep Endpoint) {
	op := &openapi3.Operation{
		OperationID: operationID,
		Summary:     ep.Summary,
		Description: ep.Description,
	}

	// Request body
	switch {
	case len(ep.Requests) > 0:
		op.RequestBody = NewRequestMust(ep.Requests...)
	case ep.Request != nil:
		op.RequestBody = NewRequestMust(ep.Request)
	}

	// Responses
	responses := ep.Responses
	if responses == nil && ep.Response != nil {
		responses = map[string]Response{
			"200": {Desc: "OK", Bodies: []any{ep.Response}},
		}
	}
	if responses != nil {
		op.Responses = NewResponseMust(responses)
	} else {
		op.Responses = openapi3.NewResponses()
	}

	AddPath(path, method, doc, op)
}

// Get registers a GET endpoint on doc.
func Get(doc *openapi3.T, path, operationID string, ep Endpoint) {
	addEndpoint(doc, path, http.MethodGet, operationID, ep)
}

// Post registers a POST endpoint on doc.
func Post(doc *openapi3.T, path, operationID string, ep Endpoint) {
	addEndpoint(doc, path, http.MethodPost, operationID, ep)
}

// Put registers a PUT endpoint on doc.
func Put(doc *openapi3.T, path, operationID string, ep Endpoint) {
	addEndpoint(doc, path, http.MethodPut, operationID, ep)
}

// Patch registers a PATCH endpoint on doc.
func Patch(doc *openapi3.T, path, operationID string, ep Endpoint) {
	addEndpoint(doc, path, http.MethodPatch, operationID, ep)
}

// Delete registers a DELETE endpoint on doc.
func Delete(doc *openapi3.T, path, operationID string, ep Endpoint) {
	addEndpoint(doc, path, http.MethodDelete, operationID, ep)
}
