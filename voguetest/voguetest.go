// Package voguetest holds the assertions the tests vogue generates run.
//
// A generated test used to spell every assertion out for every value object,
// which made two value objects of the same kind produce two copies of the
// same fifty lines — exactly what a duplicate-code linter reports. The
// generated test now states only what is particular to its value object, the
// constructor and the examples its rules declare, and hands them to one of the
// suites below:
//
//	func TestCountryCode(t *testing.T) {
//		t.Parallel()
//
//		voguetest.Scalar[CountryCode, *CountryCode, string]{
//			Field:      "countryCode",
//			New:        NewCountryCode,
//			...
//		}.Run(t)
//	}
//
// Every suite runs its checks as parallel subtests. A value object generated
// without the SQL codec is not asked to implement it: the suites look for
// driver.Valuer and sql.Scanner and skip the database checks when they are
// absent.
package voguetest

import (
	"database/sql"
	"database/sql/driver"
	"encoding"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/MathiasHilgert/vogue/schema"
)

// ValueObject is the method set every generated value object has.
type ValueObject[Object any] interface {
	String() string
	IsZero() bool
	Equal(other Object) bool
	MarshalText() ([]byte, error)
}

// Pointer is the pointer to a value object, which is what decodes into it.
type Pointer[Object any] interface {
	*Object
	encoding.TextUnmarshaler
}

// valuer returns the SQL codec of v, when it has one.
func valuer[Object any](value Object) (driver.Valuer, bool) {
	valuer, ok := any(value).(driver.Valuer)

	return valuer, ok
}

// scanner returns p as an sql.Scanner, when it is one.
func scanner[Object any, Reference Pointer[Object]](reference Reference) (sql.Scanner, bool) {
	scanner, ok := any(reference).(sql.Scanner)

	return scanner, ok
}

// hasScan reports whether the value object was generated with the SQL codec.
func hasScan[Object any, Reference Pointer[Object]]() bool {
	var probe Object

	_, ok := scanner[Object, Reference](&probe)

	return ok
}

// describes proves that a value object generated with a JSONSchema method
// describes its own valid values: the text of every accepted value must
// satisfy the schema. A value object without the method is not asked.
func describes[Object ValueObject[Object]](t *testing.T, accepted Object) {
	t.Helper()

	provider, ok := any(accepted).(schema.Provider)
	if !ok {
		return
	}

	assert.True(t, provider.JSONSchema().Accepts(accepted.String()),
		"%s was accepted by its constructor but not by its JSONSchema", accepted.String())
}
