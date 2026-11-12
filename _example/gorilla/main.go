// Command gorilla demonstrates documented routes without coupling apivalidation to gorilla/mux.
// Run: cd _example/gorilla && go run .
// Browse: http://localhost:8080/swagger/
package main

import (
	"encoding/json/v2"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	v "github.com/Gobd/apivalidation"
	"github.com/Gobd/apivalidation/openapi"
	"github.com/gorilla/mux"
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

func (o *Order) Normalize() { o.CustomerName = strings.TrimSpace(o.CustomerName) }

type ErrorResponse struct {
	Error string `json:"error"`
}

func main() {
	doc := openapi.DocBase("Example API (gorilla)", "Demonstrates documented gorilla routes", "0.1.0")
	r := mux.NewRouter()
	routes := openapi.NewRoutes(doc, func(method, path string, h http.Handler) {
		if err := r.Handle(path, h).Methods(method).GetError(); err != nil {
			panic(err)
		}
	})
	routes.HandleFunc(http.MethodPost, "/orders", "createOrder", openapi.Endpoint{
		Summary: "Create an order",
		Request: Order{},
		Responses: map[string]openapi.Response{
			"200": {Desc: "Created order", Bodies: []any{Order{}}},
			"400": {Desc: "Invalid request", Bodies: []any{ErrorResponse{}}},
		},
	}, func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		var order Order
		if err := v.DecodeAndValidateContext(r.Context(), r.Body, &order); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.MarshalWrite(w, ErrorResponse{Error: err.Error()})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.MarshalWrite(w, order)
	})
	r.PathPrefix("/swagger/").Handler(openapi.SwaggerHandlerMust("/swagger/", doc))
	server := &http.Server{Addr: ":8080", Handler: r, ReadHeaderTimeout: 5 * time.Second}
	fmt.Println("Listening on http://localhost:8080; Swagger UI: /swagger/")
	log.Fatal(server.ListenAndServe())
}
