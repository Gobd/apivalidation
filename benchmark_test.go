package apivalidation_test

import (
	"testing"

	v "github.com/Gobd/apivalidation"
)

type benchmarkOrder struct {
	Name   string  `json:"name"`
	Status string  `json:"status"`
	Email  string  `json:"email"`
	Note   string  `json:"note"`
	Count  int     `json:"count"`
	Amount float64 `json:"amount"`
}

func (s *benchmarkOrder) Rules() []*v.FieldRules {
	return []*v.FieldRules{
		v.Field(&s.Name, v.Required, v.Length(1, 100)),
		v.Field(&s.Status, v.Required, v.In("pending", "paid")),
		v.Field(&s.Email, v.Required),
		v.Field(&s.Note, v.Length(0, 200)),
		v.Field(&s.Count, v.Required, v.Min(1)),
		v.Field(&s.Amount, v.Required, v.Min(0.01)),
	}
}

func BenchmarkValidateOrder(b *testing.B) {
	order := benchmarkOrder{Name: "Alice", Status: "paid", Email: "a@example.com", Note: "note", Count: 2, Amount: 10}
	b.ReportAllocs()
	for b.Loop() {
		if err := v.Validate(&order); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkValidateOrderInvalid(b *testing.B) {
	order := benchmarkOrder{}
	b.ReportAllocs()
	for b.Loop() {
		if err := v.Validate(&order); err == nil {
			b.Fatal("expected error")
		}
	}
}

// Immutable rules can be reused while field pointers are rebound per instance.
var benchmarkCachedRules = [][]v.Rule{
	{v.Required, v.Length(1, 100)},
	{v.Required, v.In("pending", "paid")},
	{v.Required},
	{v.Length(0, 200)},
	{v.Required, v.Min(1)},
	{v.Required, v.Min(0.01)},
}

type benchmarkCachedOrder benchmarkOrder

func (s *benchmarkCachedOrder) Rules() []*v.FieldRules {
	return []*v.FieldRules{
		v.Field(&s.Name, benchmarkCachedRules[0]...),
		v.Field(&s.Status, benchmarkCachedRules[1]...),
		v.Field(&s.Email, benchmarkCachedRules[2]...),
		v.Field(&s.Note, benchmarkCachedRules[3]...),
		v.Field(&s.Count, benchmarkCachedRules[4]...),
		v.Field(&s.Amount, benchmarkCachedRules[5]...),
	}
}

func BenchmarkValidateOrderCachedRules(b *testing.B) {
	order := benchmarkCachedOrder{Name: "Alice", Status: "paid", Email: "a@example.com", Note: "note", Count: 2, Amount: 10}
	b.ReportAllocs()
	for b.Loop() {
		if err := v.Validate(&order); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkJSONPipeline(b *testing.B) {
	body := []byte(`{"name":"Alice","status":"paid","email":"a@example.com","note":"note","count":2,"amount":10}`)
	b.ReportAllocs()
	for b.Loop() {
		var order benchmarkOrder
		if err := v.UnmarshalAndValidate(body, &order); err != nil {
			b.Fatal(err)
		}
	}
}
