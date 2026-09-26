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
)

// ValueObject is the method set every generated value object has.
type ValueObject[V any] interface {
	String() string
	IsZero() bool
	Equal(other V) bool
	MarshalText() ([]byte, error)
}

// Pointer is the pointer to a value object, which is what decodes into it.
type Pointer[V any] interface {
	*V
	encoding.TextUnmarshaler
}

// valuer returns the SQL codec of v, when it has one.
func valuer[V any](v V) (driver.Valuer, bool) {
	valuer, ok := any(v).(driver.Valuer)

	return valuer, ok
}

// scanner returns p as an sql.Scanner, when it is one.
func scanner[V any, P Pointer[V]](p P) (sql.Scanner, bool) {
	scanner, ok := any(p).(sql.Scanner)

	return scanner, ok
}

// hasScan reports whether the value object was generated with the SQL codec.
func hasScan[V any, P Pointer[V]]() bool {
	var probe V

	_, ok := scanner[V, P](&probe)

	return ok
}
