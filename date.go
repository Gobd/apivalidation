package apivalidation

import (
	"strings"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

// DateRule validates that a string value matches the given date layout format.
// Use [Date] to create one, then chain [DateRule.Min] and [DateRule.Max] to
// constrain the allowed date range (inclusive).
type DateRule struct {
	validation.DateRule
	layout   string
	min, max time.Time
}

// Date creates a date validation rule with the given layout format.
// Use .Min() and .Max() to constrain the allowed date range (inclusive).
func Date(layout string) *DateRule {
	return &DateRule{
		DateRule: validation.Date(layout),
		layout:   layout,
	}
}

// Min sets the inclusive minimum date for validation and documentation.
func (r *DateRule) Min(t time.Time) *DateRule {
	r.min = t
	r.DateRule = r.DateRule.Min(t)
	return r
}

// Max sets the inclusive maximum date for validation and documentation.
func (r *DateRule) Max(t time.Time) *DateRule {
	r.max = t
	r.DateRule = r.DateRule.Max(t)
	return r
}

// Describe implements [Rule] by setting the format and date range on the schema.
func (r *DateRule) Describe(_ string, _ *openapi3.Schema, ref *openapi3.SchemaRef) error {
	switch r.layout {
	case time.DateOnly:
		ref.Value.Format = "date"
	case time.RFC3339, time.RFC3339Nano:
		ref.Value.Format = "date-time"
	default:
		if ref.Value.Description != "" && !strings.HasSuffix(ref.Value.Description, " ") {
			ref.Value.Description += " "
		}
		ref.Value.Description += "date layout: " + r.layout
	}
	if !r.min.IsZero() {
		if ref.Value.Description != "" && !strings.HasSuffix(ref.Value.Description, " ") {
			ref.Value.Description += " "
		}
		ref.Value.Description += ">= " + r.min.Format(r.layout)
	}
	if !r.max.IsZero() {
		if ref.Value.Description != "" && !strings.HasSuffix(ref.Value.Description, " ") {
			ref.Value.Description += " "
		}
		ref.Value.Description += "<= " + r.max.Format(r.layout)
	}
	return nil
}
