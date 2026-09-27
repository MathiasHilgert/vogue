// Package schema is the neutral description of how a generated value object
// crosses a JSON boundary: its JSON type, format, pattern, enum members and
// limits.
//
// A value object generated with `-schema` has a method
//
//	func (T) JSONSchema() schema.Schema
//
// and therefore implements [Provider]. The package imports nothing but the
// standard library and knows no OpenAPI framework: a domain package can carry
// the description without depending on the transport, and the HTTP adapter
// translates it into its framework's schema type. With Huma, that is a
// registry wrapper that asks every type it registers whether it is a
// [Provider]; see the README.
//
// Every generated value object marshals through encoding.TextMarshaler, so
// its JSON type is always a string — an int value object is "42", not 42.
// Minimum, Maximum and ExclusiveMinimum therefore describe the value the
// string holds rather than a JSON number; an adapter may keep them as
// documentation or drop them.
package schema

import (
	"regexp"
	"slices"
	"strconv"
	"unicode/utf8"
)

// Type is a JSON Schema type.
type Type string

// String is the JSON type of every generated value object.
const String Type = "string"

// The formats generated value objects declare. They are JSON Schema formats
// where one exists (email, uri, uuid) and descriptive names where none does.
const (
	FormatEmail   = "email"
	FormatURI     = "uri"
	FormatUUID    = "uuid"
	FormatInt64   = "int64"
	FormatDecimal = "decimal"
)

// IntegerPattern is the pattern of the text an int value object reads.
const IntegerPattern = `^[+-]?[0-9]+$`

// Schema describes a value object on the wire. The zero value of a field
// means "not constrained".
type Schema struct {
	// Type is the JSON type, String for every generated value object.
	Type Type
	// Format is the JSON Schema format, empty when none applies.
	Format string
	// Pattern is an RE2 pattern the text matches, empty when none applies.
	Pattern string
	// Enum lists the only values accepted, nil when any value may be.
	Enum []string
	// MinLength and MaxLength bound the length of the text in runes.
	MinLength, MaxLength Length
	// Minimum and Maximum bound the value the text holds, inclusively, and
	// ExclusiveMinimum from below, exclusively.
	Minimum, Maximum, ExclusiveMinimum Number
}

// Length is an optional length limit.
type Length struct {
	// Set says whether the limit applies.
	Set bool
	// Value is the limit, in runes.
	Value int
}

// Number is an optional numeric limit.
type Number struct {
	// Set says whether the limit applies.
	Set bool
	// Value is the limit.
	Value float64
}

// Provider is implemented by every value object generated with `-schema`.
type Provider interface {
	JSONSchema() Schema
}

// Accepts reports whether text satisfies every constraint of the schema. It
// is what the generated tests check the schema against: a value the
// constructor accepted must be one the schema describes as valid.
func (description Schema) Accepts(text string) bool {
	if description.Enum != nil && !slices.Contains(description.Enum, text) {
		return false
	}
	if !description.acceptsLength(utf8.RuneCountInString(text)) {
		return false
	}
	if description.Pattern != "" {
		compiled, err := regexp.Compile(description.Pattern)
		if err != nil || !compiled.MatchString(text) {
			return false
		}
	}
	return description.acceptsNumber(text)
}

// acceptsLength checks the length limits.
func (description Schema) acceptsLength(length int) bool {
	return (!description.MinLength.Set || length >= description.MinLength.Value) &&
		(!description.MaxLength.Set || length <= description.MaxLength.Value)
}

// acceptsNumber checks the numeric limits, which only a number satisfies.
func (description Schema) acceptsNumber(text string) bool {
	if !description.Minimum.Set && !description.Maximum.Set && !description.ExclusiveMinimum.Set {
		return true
	}
	value, err := strconv.ParseFloat(text, 64)
	if err != nil {
		return false
	}
	return (!description.Minimum.Set || value >= description.Minimum.Value) &&
		(!description.Maximum.Set || value <= description.Maximum.Value) &&
		(!description.ExclusiveMinimum.Set || value > description.ExclusiveMinimum.Value)
}
