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
// [Min], [Max] and [OneOf] apply to both the string and the int kind and emit
// the comparison each one needs: a rune count for a string, the value itself
// for an integer. The rest are scoped to the one kind they make sense for.
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

		// Integer checks.
		Positive,
		NonNeg,
		MultipleOf,
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
