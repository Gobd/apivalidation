// Package openapi generates OpenAPI 3 specifications from struct types that
// implement [apivalidation.Ruler]. It also provides helpers for registering
// endpoints and serving Swagger UI.
//
// Use [DocBase] to create a base document, register endpoints with [Get],
// [Post], [Put], [Patch], or [Delete], and serve the Swagger UI with
// [SwaggerHandlerMust]:
//
//	doc := openapi.DocBase("my-api", "My API", "1.0")
//	openapi.Post(doc, "/orders", "createOrder", openapi.Endpoint{
//	    Request:  Order{},
//	    Response: Order{},
//	})
//	http.Handle("/swagger/", openapi.SwaggerHandlerMust("/swagger/", doc))
package openapi
