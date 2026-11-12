package apivalidation_test

import (
	"strconv"
	"testing"

	v "github.com/Gobd/apivalidation"
)

func BenchmarkUniqueness(b *testing.B) {
	items := make([]benchmarkOrder, 100)
	for i := range items {
		items[i].Name = strconv.Itoa(i)
	}
	b.Run("index-any", func(b *testing.B) {
		r := v.Unique(func(i int) any { return items[i].Name }, "unique names")
		b.ReportAllocs()
		for b.Loop() {
			if err := r.Validate(items); err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("typed", func(b *testing.B) {
		r := v.UniqueBy(func(item benchmarkOrder) string { return item.Name }, "unique names")
		b.ReportAllocs()
		for b.Loop() {
			if err := r.Validate(items); err != nil {
				b.Fatal(err)
			}
		}
	})
}
