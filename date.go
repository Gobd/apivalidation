package apivalidation

import (
	"strings"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type dateRule struct {
	validation.DateRule
	layout   string
	min, max time.Time
}

// Date creates a date validation rule with the given layout format.
// Use .Min() and .Max() to constrain the date range for documentation.
func Date(layout string) *dateRule { //nolint:revive // unexported return enables chaining
	return &dateRule{
		DateRule: validation.Date(layout),
		layout:   layout,
	}
}

// Min sets the minimum allowed date for documentation.
func (r *dateRule) Min(t time.Time) *dateRule {
	r.min = t
	return r
}

// Max sets the maximum allowed date for documentation.
func (r *dateRule) Max(t time.Time) *dateRule {
	r.max = t
	return r
}

func (r *dateRule) Describe(_ string, _ *openapi3.Schema, ref *openapi3.SchemaRef) error {
	ref.Value.Format = r.layout
	if !r.min.IsZero() {
		if ref.Value.Description != "" && !strings.HasSuffix(ref.Value.Description, " ") {
			ref.Value.Description += " "
		}
		ref.Value.Description += "> " + r.min.String()
	}
	if !r.max.IsZero() {
		if ref.Value.Description != "" && !strings.HasSuffix(ref.Value.Description, " ") {
			ref.Value.Description += " "
		}
		ref.Value.Description += "< " + r.max.String()
	}
	return nil
}
