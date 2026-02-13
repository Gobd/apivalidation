package apivalidation

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMinMax(t *testing.T) {
	minTests := []struct {
		min         float64
		value       any
		expectError bool
	}{
		{min: 0, value: 1.0, expectError: false},
		{min: 0, value: 1, expectError: true}, // 1 is an int not a float
		{min: 0, value: "1", expectError: false},
		{min: 0, value: "-1", expectError: true},
		{min: 0, value: "abc", expectError: true},
		{min: 0, value: nil, expectError: false}, // Skips empty
		{min: 0, value: []int{1}, expectError: true},
		{min: 0, value: json.Number("1"), expectError: false},
	}
	for _, tt := range minTests {
		t.Run(fmt.Sprintf("min:%v,v:%d", tt.min, tt.value), func(t *testing.T) {
			r := Min(tt.min)
			err := r.Validate(tt.value)
			if tt.expectError {
				require.NotNil(t, err)
			} else {
				require.Nil(t, err)
			}
		})
	}

	maxTests := []struct {
		max         float64
		value       any
		expectError bool
	}{
		{max: 2, value: "2", expectError: false},
		{max: 2, value: "3", expectError: true},
		{max: 2, value: "1", expectError: false},
		{max: 5.5, value: "5.6", expectError: true},
		{max: 5.5, value: "5.4", expectError: false},
		{max: 5.5, value: "5.5", expectError: false},
	}
	for _, tt := range maxTests {
		t.Run(fmt.Sprintf("max:%v,v:%d", tt.max, tt.value), func(t *testing.T) {
			r := Max(tt.max)
			err := r.Validate(tt.value)
			if tt.expectError {
				require.NotNil(t, err)
			} else {
				require.Nil(t, err)
			}
		})
	}
}
