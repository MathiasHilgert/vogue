package vogue

import "github.com/MathiasHilgert/vogue/validation"

// The runtime a generated value object depends on lives in package
// [validation], which imports nothing but the standard library. These aliases
// keep code written against the names this package used to declare compiling
// unchanged.
type (
	// FieldError is an alias of [validation.FieldError].
	FieldError = validation.FieldError
	// Notification is an alias of [validation.Notification].
	Notification = validation.Notification
)

// ErrInvalid is [validation.ErrInvalid], matched by every validation failure.
var ErrInvalid = validation.ErrInvalid
