package apivalidation_test

import (
	"testing"

	"github.com/Gobd/apivalidation"
	"github.com/stretchr/testify/assert"
)

type KeyInTest struct {
	Map     any
	Allowed []string
}

func (f *KeyInTest) Rules() []*apivalidation.FieldRules {
	return []*apivalidation.FieldRules{
		apivalidation.Field(&f.Map,
			apivalidation.Required,
			apivalidation.KeyIn(f.Allowed...),
		),
	}
}

type keyInType string

func TestKeyInt(t *testing.T) {
	tests := []struct {
		name        string
		in          any
		allowed     []string
		expectError bool
	}{
		{
			name:        "basic",
			in:          map[string]string{"a": "b"},
			allowed:     []string{"a"},
			expectError: false,
		},
		{
			name:        "basic failure",
			in:          map[string]string{"a": "b"},
			allowed:     []string{"c"},
			expectError: true,
		},
		{
			name:        "complex",
			in:          map[string]any{"a": struct{}{}},
			allowed:     []string{"a"},
			expectError: false,
		},
		{
			name:        "number",
			in:          map[string]int{"a": 1},
			allowed:     []string{"a"},
			expectError: false,
		},
		{
			name:        "alias type as key",
			in:          map[keyInType]any{"a": "b"},
			allowed:     []string{"a"},
			expectError: false,
		},
		{
			name:        "non map",
			in:          "a",
			allowed:     []string{"a"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := KeyInTest{Map: tt.in, Allowed: tt.allowed}
			err := apivalidation.Validate(&v)
			if err != nil {
				t.Log(err)
			}
			assert.Equal(t, tt.expectError, err != nil)
		})
	}
}
