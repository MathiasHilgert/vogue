// Package rules is the catalogue of validation rules vogue ships with.
//
// Every rule is an ordinary [vogue.Rule] value exported as a package variable,
// so a project can take the ones it wants, drop the ones it does not and mix
// in its own. [All] returns the whole catalogue in catalogue order, and [Set]
// wraps it in the [vogue.RuleSet] the generator reads directives against.
//
// # Normalizers and checks
//
// A normalizer rewrites the value; a check accepts or rejects it. Normalizers
// are written first in a directive, because every rule after one sees the
// rewritten value:
//
//	//vogue:string Title trim squish required min=1 max=120
//
// There the bounds are measured on the trimmed, whitespace-collapsed value,
// which is almost always what a human means by "at most 120 characters".
//
// # The empty string
//
// Only [Required] rejects the empty string. The character-class checks —
// [Alpha], [Alphanum], [Numeric], [ASCII], [Printable], [NoSpace] — are
// satisfied vacuously by it, because there is no offending character in it,
// and so are [Prefix] and friends only when their parameter is empty too.
// Pair them with `required` when the field is mandatory; leaving them
// independent is what lets an optional field be either empty or well formed
// without a rule that means two things at once.
//
// # Counting
//
// Every length in this catalogue is a count of runes, not of bytes:
// `min=3` accepts "añó". That is what a person filling in a form counts, and
// it is what makes a bound behave the same for an English and a Spanish name.
//
// # Kinds
//
// [Min] and [Max] apply to the string, int and decimal kinds and emit the
// comparison each one needs: a rune count for a string, the value itself for
// an integer, a Cmp against a once-parsed bound for a decimal. [OneOf] spans
// string and int only, because exact equality against a written list is not a
// question a value carrying a scale answers well — 0.5 and 0.50 are the same
// number and only one of them is in the list. [Positive] and [NonNeg] read the
// sign of an int or a decimal. The rest are scoped to the one kind they make
// sense for.
//
// # Decimals
//
// The decimal kind is for exact non-integer values — rates, percentages,
// quantities in fractional units — held as a github.com/govalues/decimal
// value, never as a binary float. Besides the bounds and the sign rules it has
// [Scale], which bounds how many decimal places a value carries, and
// [NonZero]. A bound is written as a decimal literal and parsed at generate
// time, so `min=0.5` is a bound on a rate and a rejected directive on an
// integer.
package rules

import (
	"fmt"

	"github.com/MathiasHilgert/vogue"
)

// All returns the shipped rules, in the order they are catalogued: the
// normalizers first, then the checks that apply to strings, then the checks
// that apply to integers. The returned slice is fresh on every call, so a
// caller may sort or filter it without disturbing anybody else.
func All() []vogue.Rule {
	return []vogue.Rule{
		// Normalizers.
		Trim,
		Squish,
		Lower,
		Upper,

		// String checks.
		Required,
		Min,
		Max,
		Len,
		Email,
		URL,
		UUID,
		TimeZone,
		Regex,
		OneOf,
		Alpha,
		Alphanum,
		Numeric,
		ASCII,
		Printable,
		NoSpace,
		Prefix,
		Suffix,
		Contains,
		Excludes,

		// Number checks.
		Positive,
		NonNeg,
		MultipleOf,

		// Decimal checks.
		Scale,
		NonZero,
	}
}

// Set returns a [vogue.RuleSet] holding every shipped rule. It validates each
// one on the way in, so a catalogue that has been edited into an inconsistent
// state is reported here rather than as broken generated code.
func Set() (*vogue.RuleSet, error) {
	set := &vogue.RuleSet{}
	if err := set.Add(All()...); err != nil {
		return nil, fmt.Errorf("rules: building the built-in catalogue: %w", err)
	}
	return set, nil
}

// MustSet is [Set] for the package-level initialisation of a generator binary,
// where a broken built-in catalogue is a programming error rather than
// something to recover from.
func MustSet() *vogue.RuleSet {
	set, err := Set()
	if err != nil {
		panic(err)
	}
	return set
}
