package generator

// ValidationContract describes the type generated constructors record their
// failures on, with a reference implementation short enough to copy. It is
// what the command prints when -validation is missing, and what the README
// repeats.
const ValidationContract = `Generated constructors record their failures on a type your project owns, so the
generated code imports nothing of vogue. Name it as <import path>.<Type>:

	//go:generate go run github.com/MathiasHilgert/vogue/cmd/vogue -validation=example.com/app/fault.Validation

The type needs a zero value that is ready to use and two methods:

	Add(field, rule, message string)   records one failure
	Err() error                        nil until something was added; otherwise an error that
	                                   also has Has(field, rule string) bool, which the
	                                   generated tests use

A reference implementation to copy into your own package:

` + ValidationReference

// ValidationRequired is the message of a run that names no failure type.
const ValidationRequired = "vogue: the failure type is required: pass -validation=<import path>.<Type> " +
	"(generator.WithValidation from Go).\n\n" + ValidationContract

// ValidationReference is a complete failure type, and the same code as
// examples/validation, condensed. Err builds the error only when there is a
// failure, so a valid value object allocates nothing.
const ValidationReference = `	package fault

	import "strings"

	type failure struct{ field, rule, message string }

	type Validation struct{ failures []failure }

	func (validation *Validation) Add(field, rule, message string) {
		validation.failures = append(validation.failures, failure{field, rule, message})
	}

	func (validation *Validation) Err() error {
		if len(validation.failures) == 0 {
			return nil
		}
		return &Error{failures: validation.failures}
	}

	type Error struct{ failures []failure }

	func (invalid *Error) Error() string {
		messages := make([]string, len(invalid.failures))
		for index, failed := range invalid.failures {
			messages[index] = "invalid " + failed.field + ": " + failed.message + " (" + failed.rule + ")"
		}
		return strings.Join(messages, "; ")
	}

	func (invalid *Error) Has(field, rule string) bool {
		for _, failed := range invalid.failures {
			if failed.field == field && failed.rule == rule {
				return true
			}
		}
		return false
	}
`
