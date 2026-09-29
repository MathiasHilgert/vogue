// Package validation is the reference implementation of the type vogue's
// generated constructors record their failures on.
//
// Generated code imports nothing of vogue, so this type is not imported by
// anything vogue generates: copy it into your own module, adapt it, and name
// it to the generator with -validation=<import path>.Validation. The
// generator asks for three things, all of which are here:
//
//   - a zero value that is ready to use, so a constructor declares
//     `var failures validation.Validation` and nothing else;
//   - Add(field, rule, message string), which records one failure;
//   - Err() error, which is nil until something was added, and otherwise an
//     error that also has Has(field, rule string) bool, which the generated
//     tests use to check that the expected rule rejected an input.
//
// Err builds the error only when there is a failure, so constructing a valid
// value object allocates nothing.
package validation

import "strings"

// failure is one rule that rejected one field.
type failure struct{ field, rule, message string }

// Validation collects the failures of one or more value objects. Its zero
// value is ready to use.
type Validation struct{ failures []failure }

// Add records that the rule rejected the field. The message reads after the
// field name: "is required", "must be at most 120 characters long".
func (validation *Validation) Add(field, rule, message string) {
	validation.failures = append(validation.failures, failure{field: field, rule: rule, message: message})
}

// Err returns nil when nothing was added, and otherwise an *Error listing
// every failure.
func (validation *Validation) Err() error {
	if len(validation.failures) == 0 {
		return nil
	}

	return &Error{failures: validation.failures}
}

// Error is the error Err returns. It never carries the rejected value, so it
// is safe to log.
type Error struct{ failures []failure }

// Error lists the failures as "invalid <field>: <message> (<rule>)", joined
// by "; ".
func (invalid *Error) Error() string {
	messages := make([]string, len(invalid.failures))
	for index, failed := range invalid.failures {
		messages[index] = "invalid " + failed.field + ": " + failed.message + " (" + failed.rule + ")"
	}

	return strings.Join(messages, "; ")
}

// Has reports whether the rule rejected the field.
func (invalid *Error) Has(field, rule string) bool {
	for _, failed := range invalid.failures {
		if failed.field == field && failed.rule == rule {
			return true
		}
	}

	return false
}
