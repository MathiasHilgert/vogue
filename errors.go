package vogue

import "strings"

// FieldError is the single failure produced by one rule applied to one field.
// Generated constructors never return a bare FieldError to the caller when more
// than one rule can fail; they accumulate into a [Notification] instead.
//
// FieldError is a value type on purpose: it is comparable, cheap to copy, and
// usable as an errors.Is target without allocating.
type FieldError struct {
	// Field is the value-object field name as written in the directive, in the
	// spelling the API exposes (for example "title").
	Field string
	// Rule is the tag name of the rule that rejected the value (for example
	// "min").
	Rule string
	// Param is the rule parameter as written in the directive, empty when the
	// rule takes none.
	Param string
	// Value is the offending input rendered as text. It is carried for logging
	// and debugging and is deliberately not part of [FieldError.Error], because
	// the message is user-facing and the value may be sensitive.
	Value string
	// Message is the rendered, human-readable explanation of the failure.
	Message string
}

// Error renders the failure as
//
//	<field>: <message> (rule "<rule>", param "<param>")
//
// omitting the param clause when [FieldError.Param] is empty.
func (e FieldError) Error() string {
	var b strings.Builder
	b.Grow(len(e.Field) + len(e.Message) + len(e.Rule) + len(e.Param) + 24)
	b.WriteString(e.Field)
	b.WriteString(": ")
	b.WriteString(e.Message)
	b.WriteString(` (rule "`)
	b.WriteString(e.Rule)
	if e.Param != "" {
		b.WriteString(`", param "`)
		b.WriteString(e.Param)
	}
	b.WriteString(`")`)
	return b.String()
}

// Code returns the stable machine-readable identifier of the failure,
// `<field>.<rule>`, suitable for API error payloads and translation keys.
func (e FieldError) Code() string { return e.Field + "." + e.Rule }

// Is implements errors.Is matching by example: the target matches when every
// non-empty field of the target equals the corresponding field of the receiver.
// This lets callers write errors.Is(err, vogue.FieldError{Rule: "min"}) to ask
// "did the min rule fail?" without restating the whole failure.
func (e FieldError) Is(target error) bool {
	t, ok := target.(FieldError)
	if !ok {
		return false
	}
	return matches(t.Field, e.Field) &&
		matches(t.Rule, e.Rule) &&
		matches(t.Param, e.Param) &&
		matches(t.Value, e.Value) &&
		matches(t.Message, e.Message)
}

// matches reports whether an errors.Is target component constrains the
// candidate; an empty component is a wildcard.
func matches(target, candidate string) bool { return target == "" || target == candidate }
