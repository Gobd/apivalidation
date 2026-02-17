package apivalidation

import (
	"strings"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

// DateRule validates that a string value matches the given date layout format.
// Use [Date] to create one, then chain [DateRule.Min] and [DateRule.Max] to
// constrain the date range for documentation.
type DateRule struct {
	validation.DateRule
	layout   string
	min, max time.Time
}

// Date creates a date validation rule with the given layout format.
// Use .Min() and .Max() to constrain the date range for documentation.
func Date(layout string) *DateRule {
	return &DateRule{
		DateRule: validation.Date(layout),
		layout:   layout,
	}
}

// Min sets the minimum allowed date for documentation.
func (r *DateRule) Min(t time.Time) *DateRule {
	r.min = t
	return r
}

// Max sets the maximum allowed date for documentation.
func (r *DateRule) Max(t time.Time) *DateRule {
	r.max = t
	return r
}

// Describe implements [Rule] by setting the format and date range on the schema.
func (r *DateRule) Describe(_ string, _ *openapi3.Schema, ref *openapi3.SchemaRef) error {
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
