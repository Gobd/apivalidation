package apivalidation

import validation "github.com/go-ozzo/ozzo-validation/v4"

// ValidationErrors is a map of field names to their validation errors.
// It is an alias for [validation.Errors] from ozzo-validation and implements
// the error interface with a JSON-friendly string representation.
type ValidationErrors = validation.Errors
