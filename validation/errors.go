package validation

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
func (fieldError FieldError) Error() string {
	var builder strings.Builder
	builder.Grow(len(fieldError.Field) + len(fieldError.Message) + len(fieldError.Rule) + len(fieldError.Param) + 24)
	builder.WriteString(fieldError.Field)
	builder.WriteString(": ")
	builder.WriteString(fieldError.Message)
	builder.WriteString(` (rule "`)
	builder.WriteString(fieldError.Rule)
	if fieldError.Param != "" {
		builder.WriteString(`", param "`)
		builder.WriteString(fieldError.Param)
	}
	builder.WriteString(`")`)
	return builder.String()
}

// Code returns the stable machine-readable identifier of the failure,
// `<field>.<rule>`, suitable for API error payloads and translation keys.
func (fieldError FieldError) Code() string { return fieldError.Field + "." + fieldError.Rule }

// Is implements errors.Is. Every FieldError matches [ErrInvalid]. Another
// FieldError matches by example: the target matches when every non-empty field
// of the target equals the corresponding field of the receiver. This lets
// callers write errors.Is(err, validation.FieldError{Rule: "min"}) to ask "did
// the min rule fail?" without restating the whole failure.
func (fieldError FieldError) Is(target error) bool {
	if target == ErrInvalid {
		return true
	}
	wanted, ok := target.(FieldError)
	if !ok {
		return false
	}
	return matches(wanted.Field, fieldError.Field) &&
		matches(wanted.Rule, fieldError.Rule) &&
		matches(wanted.Param, fieldError.Param) &&
		matches(wanted.Value, fieldError.Value) &&
		matches(wanted.Message, fieldError.Message)
}

// matches reports whether an errors.Is target component constrains the
// candidate; an empty component is a wildcard.
func matches(target, candidate string) bool { return target == "" || target == candidate }
