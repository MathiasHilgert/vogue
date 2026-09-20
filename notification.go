package vogue

import (
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
	errs []FieldError
}

// Add appends a failure to the notification.
func (n *Notification) Add(err FieldError) {
	n.errs = append(n.errs, err)
}

// Addf appends a failure whose message is built with fmt.Sprintf. It is the
// form generated code uses when the message template has already been rendered
// into a format string at generate time.
func (n *Notification) Addf(field, rule, param, value string, format string, args ...any) {
	n.errs = append(n.errs, FieldError{
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
func (n *Notification) Merge(other *Notification) {
	if other == nil || len(other.errs) == 0 {
		return
	}
	n.errs = append(n.errs, other.errs...)
}

// Len returns the number of collected failures.
func (n *Notification) Len() int { return len(n.errs) }

// HasErrors reports whether any failure was collected.
func (n *Notification) HasErrors() bool { return len(n.errs) > 0 }

// Errors returns a copy of the collected failures in the order they were added.
// The copy keeps callers from mutating the notification through the slice they
// were handed.
func (n *Notification) Errors() []FieldError {
	if len(n.errs) == 0 {
		return nil
	}
	out := make([]FieldError, len(n.errs))
	copy(out, n.errs)
	return out
}

// Field returns a copy of the failures recorded for the named field, in order.
// It returns nil when the field has no failure.
func (n *Notification) Field(name string) []FieldError {
	var out []FieldError
	for _, e := range n.errs {
		if e.Field == name {
			out = append(out, e)
		}
	}
	return out
}

// Reset drops every collected failure while keeping the allocated capacity, so
// a notification can be reused across iterations without allocating again.
func (n *Notification) Reset() { n.errs = n.errs[:0] }

// Error renders every failure, one per line, behind a count header:
//
//	2 validation errors:
//	  - title: too short (rule "min", param "1")
//	  - email: must be a valid email address (rule "email")
//
// An empty notification renders "no validation errors"; callers should use
// [Notification.ErrOrNil] rather than returning an empty notification.
func (n *Notification) Error() string {
	switch len(n.errs) {
	case 0:
		return "no validation errors"
	case 1:
		return "1 validation error:\n  - " + n.errs[0].Error()
	}

	var b strings.Builder
	b.WriteString(strconv.Itoa(len(n.errs)))
	b.WriteString(" validation errors:")
	for i := range n.errs {
		b.WriteString("\n  - ")
		b.WriteString(n.errs[i].Error())
	}
	return b.String()
}

// Unwrap exposes the collected failures to errors.Is and errors.As, so callers
// can ask errors.Is(err, vogue.FieldError{Rule: "email"}) or extract a
// [FieldError] from a notification without type-asserting it first.
func (n *Notification) Unwrap() []error {
	if len(n.errs) == 0 {
		return nil
	}
	out := make([]error, len(n.errs))
	for i := range n.errs {
		out[i] = n.errs[i]
	}
	return out
}

// ErrOrNil returns nil when nothing was collected and the notification itself
// otherwise. It returns an untyped nil interface, never a nil *Notification
// wrapped in a non-nil error, which is the classic pitfall of returning a
// concrete pointer type as an error.
func (n *Notification) ErrOrNil() error {
	if n == nil || len(n.errs) == 0 {
		return nil
	}
	return n
}
