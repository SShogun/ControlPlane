// Package validator provides reusable form validation helpers.
// It is intentionally decoupled from HTTP so it can be used in
// handler tests without importing net/http.
package validator

import (
	"strings"
	"unicode/utf8"
)

// Validator tracks field-level validation errors.
// Embed it in a form struct to get validation behaviour for free.
type Validator struct {
	FieldErrors map[string]string
}

// Valid returns true when no field errors have been recorded.
func (v *Validator) Valid() bool {
	return len(v.FieldErrors) == 0
}

// AddFieldError records an error for a specific form field.
// If an error already exists for that field, the call is a no-op
// so only the first error per field is shown.
func (v *Validator) AddFieldError(field, message string) {
	if v.FieldErrors == nil {
		v.FieldErrors = map[string]string{}
	}
	if _, exists := v.FieldErrors[field]; !exists {
		v.FieldErrors[field] = message
	}
}

// Check adds a field error when ok is false.
// It is the primary way to register validation rules.
//
//	v.Check(title != "", "title", "Title is required")
func (v *Validator) Check(ok bool, field, message string) {
	if !ok {
		v.AddFieldError(field, message)
	}
}

// --- Standalone rule functions ------------------------------------------------
// These are pure functions so they can be tested independently and composed
// inside Check calls.

// NotBlank returns true if the value is not empty after trimming whitespace.
func NotBlank(value string) bool {
	return strings.TrimSpace(value) != ""
}

// MaxChars returns true if the value contains at most n Unicode characters.
func MaxChars(value string, n int) bool {
	return utf8.RuneCountInString(value) <= n
}

// MinChars returns true if the value contains at least n Unicode characters.
func MinChars(value string, n int) bool {
	return utf8.RuneCountInString(value) >= n
}

// PositiveInt returns true if the value is greater than zero.
func PositiveInt(value int) bool {
	return value > 0
}
