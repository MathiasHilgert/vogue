package validation

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Notification implements Martin Fowler's Notification pattern: instead of
// aborting on the first invalid rule, a constructor collects every failure and
// hands the caller the complete picture in one error.
//
// The zero value is ready to use and allocates nothing until the first
// [Notification.Add]. Because validation succeeds on the happy path, a value
// object constructed from valid input never allocates at all.
//
// A Notification is not safe for concurrent use; generated constructors keep
// one on the stack for the duration of a single call.
type Notification struct {
	failures []FieldError
}

// Add appends a failure to the notification.
func (notification *Notification) Add(err FieldError) {
	notification.failures = append(notification.failures, err)
}

// Reject records that rule rejected value for field. It is [Notification.Add]
// with the five fields of a [FieldError] spelled as arguments, which is the
// form generated constructors use: every argument is a literal rendered at
// generate time, so the call reads as one line per failure and no struct
// literal is left for a linter to find incomplete.
func (notification *Notification) Reject(field, rule, param, value, message string) {
	notification.failures = append(notification.failures, FieldError{
		Field:   field,
		Rule:    rule,
		Param:   param,
		Value:   value,
		Message: message,
	})
}

// Addf appends a failure whose message is built with fmt.Sprintf. It is the
// form generated code uses when the message template has already been rendered
// into a format string at generate time.
func (notification *Notification) Addf(field, rule, param, value, format string, args ...any) {
	notification.failures = append(notification.failures, FieldError{
		Field:   field,
		Rule:    rule,
		Param:   param,
		Value:   value,
		Message: fmt.Sprintf(format, args...),
	})
}

// Merge appends every failure of other, preserving order. A nil other is a
// no-op, so callers can merge the result of an optional sub-validation without
// a guard. The other notification is left untouched.
func (notification *Notification) Merge(other *Notification) {
	if other == nil || len(other.failures) == 0 {
		return
	}
	notification.failures = append(notification.failures, other.failures...)
}

// Collect folds the failures an error carries into the notification. It is
// how a composite value object validates its parts and reports every failure
// of all of them at once:
//
//	var n validation.Notification
//	latitude, err := NewLatitude(lat)
//	if err := n.Collect(err); err != nil {
//		return Coordinates{}, err
//	}
//
// A nil error is ignored. A [Notification] or a [FieldError], however deeply
// wrapped, is merged and Collect returns nil. Any other error is not a
// validation failure — a driver or a minting error — and is returned
// unchanged for the caller to handle.
func (notification2 *Notification) Collect(err error) error {
	if err == nil {
		return nil
	}
	var notification *Notification
	if errors.As(err, &notification) {
		notification2.Merge(notification)
		return nil
	}
	var field FieldError
	if errors.As(err, &field) {
		notification2.Add(field)
		return nil
	}
	return err
}

// Len returns the number of collected failures.
func (notification *Notification) Len() int { return len(notification.failures) }

// HasErrors reports whether any failure was collected.
func (notification *Notification) HasErrors() bool { return len(notification.failures) > 0 }

// Errors returns a copy of the collected failures in the order they were added.
// The copy keeps callers from mutating the notification through the slice they
// were handed.
func (notification *Notification) Errors() []FieldError {
	if len(notification.failures) == 0 {
		return nil
	}
	out := make([]FieldError, len(notification.failures))
	copy(out, notification.failures)
	return out
}

// Field returns a copy of the failures recorded for the named field, in order.
// It returns nil when the field has no failure.
func (notification *Notification) Field(name string) []FieldError {
	var out []FieldError
	for _, fieldError := range notification.failures {
		if fieldError.Field == name {
			out = append(out, fieldError)
		}
	}
	return out
}

// Reset drops every collected failure while keeping the allocated capacity, so
// a notification can be reused across iterations without allocating again.
func (notification *Notification) Reset() { notification.failures = notification.failures[:0] }

// Error renders every failure, one per line, behind a count header:
//
//	2 validation errors:
//	  - title: too short (rule "min", param "1")
//	  - email: must be a valid email address (rule "email")
//
// An empty notification renders "no validation errors"; callers should use
// [Notification.ErrOrNil] rather than returning an empty notification.
func (notification *Notification) Error() string {
	switch len(notification.failures) {
	case 0:
		return "no validation errors"
	case 1:
		return "1 validation error:\n  - " + notification.failures[0].Error()
	}

	var builder strings.Builder
	builder.WriteString(strconv.Itoa(len(notification.failures)))
	builder.WriteString(" validation errors:")
	for index := range notification.failures {
		builder.WriteString("\n  - ")
		builder.WriteString(notification.failures[index].Error())
	}
	return builder.String()
}

// Unwrap exposes the collected failures to errors.Is and errors.As, so callers
// can ask errors.Is(err, validation.FieldError{Rule: "email"}) or extract a
// [FieldError] from a notification without type-asserting it first.
func (notification *Notification) Unwrap() []error {
	if len(notification.failures) == 0 {
		return nil
	}
	out := make([]error, len(notification.failures))
	for index := range notification.failures {
		out[index] = notification.failures[index]
	}
	return out
}

// ErrOrNil returns nil when nothing was collected and the notification itself
// otherwise. It returns an untyped nil interface, never a nil *Notification
// wrapped in a non-nil error, which is the classic pitfall of returning a
// concrete pointer type as an error.
func (notification *Notification) ErrOrNil() error {
	if notification == nil || len(notification.failures) == 0 {
		return nil
	}
	return notification
}
