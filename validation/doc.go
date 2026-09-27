// Package validation is the runtime the code vogue generates depends on: the
// [FieldError] one rule produces, the [Notification] a constructor collects
// them in, and the sentinel errors a caller matches with errors.Is.
//
// It is kept deliberately small. It imports nothing but the standard library
// (errors, fmt, strconv and strings), declares no init, and is the only vogue
// package a generated value object imports, so depending on generated code
// never pulls the generator, its templates or its rule catalogue into a
// binary.
//
// # Matching failures
//
// Every validation failure matches [ErrInvalid], however deeply it was
// wrapped, so an HTTP adapter can map "the input was invalid" to a 422
// without knowing which value object produced it:
//
//	if errors.Is(err, validation.ErrInvalid) { ... }
//
// A particular failure is matched by example, with the fields that matter set
// and the rest left empty:
//
//	if errors.Is(err, validation.FieldError{Field: "title", Rule: "required"}) { ... }
//
// and the whole detail is one errors.As away:
//
//	var fe validation.FieldError
//	if errors.As(err, &fe) { log.Println(fe.Code()) }
//
// A generated Scan that is handed a source it cannot read reports
// [ErrUnsupportedSource], and one handed a binary float for an exact decimal
// reports [ErrLossySource]. Neither is an [ErrInvalid]: they describe a
// misconfigured driver, not a value that broke a rule.
package validation
