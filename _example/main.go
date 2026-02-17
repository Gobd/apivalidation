// Command example demonstrates apivalidation with an HTTP server serving
// a Swagger UI and a validated JSON endpoint.
//
// Run:
//
//	go run ./_example
//
// Then open http://localhost:8080/swagger/ in your browser.
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	v "github.com/Gobd/apivalidation"
	"github.com/Gobd/apivalidation/openapi"
)

// Order is a sample request/response type.
type Order struct {
	CustomerName string  `json:"customer_name"`
	ItemCount    int     `json:"item_count"`
	Total        float64 `json:"total"`
}

func (o *Order) Rules() []*v.FieldRules {
	return []*v.FieldRules{
		v.Field(&o.CustomerName, v.Required, v.Length(1, 200)),
		v.Field(&o.ItemCount, v.Required, v.Min(1)),
		v.Field(&o.Total, v.Required, v.Min(0.01)),
	}
}

// ErrorResponse is a standard error envelope.
type ErrorResponse struct {
	Error string `json:"error"`
}

func main() {
	// Build the OpenAPI spec.
	doc := openapi.DocBase("Example API", "Demonstrates apivalidation", "0.1.0")

	openapi.Post(doc, "/orders", "createOrder", openapi.Endpoint{
		Summary:  "Create an order",
		Request:  Order{},
		Response: Order{},
		Responses: map[string]openapi.Response{
			"200": {Desc: "Created order", Bodies: []any{Order{}}},
			"400": {Desc: "Validation error", Bodies: []any{ErrorResponse{}}},
		},
	})

	// Swagger UI
	http.Handle("/swagger/", openapi.SwaggerHandlerMust("/swagger/", doc))

	// API endpoint
	http.HandleFunc("/orders", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var order Order
		if err := v.DecodeAndValidate(r.Body, &order); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(ErrorResponse{Error: err.Error()})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(order)
	})

	fmt.Println("Listening on http://localhost:8080")
	fmt.Println("Swagger UI: http://localhost:8080/swagger/")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
