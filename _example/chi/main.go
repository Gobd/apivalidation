// Command chi demonstrates apivalidation with a chi router.
//
// Run:
//
//	cd _example/chi && go run .
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
	"github.com/go-chi/chi/v5"
)

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

type ErrorResponse struct {
	Error string `json:"error"`
}

func main() {
	doc := openapi.DocBase("Example API (chi)", "Demonstrates apivalidation with chi", "0.1.0")

	openapi.Post(doc, "/orders", "createOrder", openapi.Endpoint{
		Summary:  "Create an order",
		Request:  Order{},
		Response: Order{},
	})

	r := chi.NewRouter()

	r.Handle("/swagger/*", openapi.SwaggerHandlerMust("/swagger/", doc))

	r.Post("/orders", func(w http.ResponseWriter, r *http.Request) {
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
	log.Fatal(http.ListenAndServe(":8080", r))
}
