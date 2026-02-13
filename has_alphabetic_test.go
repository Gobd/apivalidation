package apivalidation_test

import (
	"testing"

	"github.com/Gobd/apivalidation"
	"github.com/stretchr/testify/assert"
)

type AlphabeticTester struct {
	Value any `json:"value"`
}

func (f AlphabeticTester) Validate() error {
	return apivalidation.ValidateStruct(f.Rules())
}

func (f AlphabeticTester) Rules() (interface{}, []*apivalidation.FieldRules) {
	return &f, []*apivalidation.FieldRules{
		apivalidation.Field(&f.Value, apivalidation.HasAlphabetic()),
	}
}

func TestHasAlphabetic(t *testing.T) {
	tests := []struct {
		name   string
		in     any
		errStr string
	}{
		{
			name:   "alpha",
			in:     "1234-1234 \nabc",
			errStr: "",
		},
		{
			name:   "empty", // Allow when not required
			in:     "",
			errStr: "",
		},
		{
			name:   "non alpha",
			in:     "1234-1234 \n",
			errStr: "value: must contain at least one alphabetic character.",
		},
		{
			name:   "wrong type",
			in:     1234,
			errStr: "value: expected string, got int.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := AlphabeticTester{Value: tt.in}.Validate()
			var errStr string
			if err != nil {
				errStr = err.Error()
			}
			assert.Equal(t, tt.errStr, errStr)
		})
	}
}

type CreditCardTester struct {
	Value any `json:"value"`
}

func (f CreditCardTester) Validate() error {
	return apivalidation.ValidateStruct(f.Rules())
}

func (f CreditCardTester) Rules() (interface{}, []*apivalidation.FieldRules) {
	return &f, []*apivalidation.FieldRules{
		apivalidation.Field(&f.Value, apivalidation.NonCreditCardNumber()),
	}
}

func TestNonCreditCardNumber(t *testing.T) {
	tests := []struct {
		name   string
		in     any
		errStr string
	}{
		{
			name:   "credit card numbers just numbers",
			in:     "1234567890122345",
			errStr: "value: must not be a credit card number.",
		},
		{
			name:   "credit card numbers with spaces",
			in:     "1234 5678 9012 2345",
			errStr: "value: must not be a credit card number.",
		},
		{
			name:   "credit card numbers with dashes",
			in:     "1234-5678-9012-2345",
			errStr: "value: must not be a credit card number.",
		},
		{
			name:   "almost credit card number length",
			in:     "1234-5678-9012-234",
			errStr: "",
		},
		{
			name:   "almost credit card number alpha",
			in:     "1234-5678-9012-234a",
			errStr: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CreditCardTester{Value: tt.in}.Validate()
			var errStr string
			if err != nil {
				errStr = err.Error()
			}
			assert.Equal(t, tt.errStr, errStr)
		})
	}
}
