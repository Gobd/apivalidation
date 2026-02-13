package apivalidation_test

import (
	"fmt"
	"testing"

	"github.com/Gobd/apivalidation"
)

type Foo struct {
	Bar any
}

func (f Foo) Validate() error {
	return apivalidation.ValidateStruct(f.Rules())
}

func (f Foo) Rules() (interface{}, []*apivalidation.FieldRules) {
	return &f, []*apivalidation.FieldRules{
		apivalidation.Field(&f.Bar,
			apivalidation.Required,
			apivalidation.Custom(
				func(_ any) error {
					return fmt.Errorf("custom error")
				},
				"custom description",
			),
		),
	}
}

func TestCustom(t *testing.T) {
	foo := Foo{Bar: struct{}{}}
	err := foo.Validate()
	if err == nil {
		t.Fatal("should have returned error")
	}
	if err.Error() != "Bar: custom error." {
		t.Error("wrong error:", err.Error())
	}
}
