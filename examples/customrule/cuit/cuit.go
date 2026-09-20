// Package cuit is the custom rule of the worked extensibility example: the
// Argentine CUIT, a taxpayer identifier no catalogue of general-purpose rules
// would ship.
//
// It is written the way the built-in rules are written: a pure predicate that
// generated code calls statically, and a [vogue.Rule] value pointing at it
// through [vogue.FuncRef]. Nothing registers itself; the rule becomes real by
// being handed to the generator, which is what lets the generator reject a
// directive that names a rule nobody passed.
package cuit

import "github.com/MathiasHilgert/vogue"

// weights are the multipliers of the CUIT check digit, applied to the first
// ten digits in order. They are the published ones and are not a parameter.
var weights = [10]int{5, 4, 3, 2, 7, 6, 5, 4, 3, 2}

// Valid reports whether v is a well-formed CUIT: eleven decimal digits whose
// last one is the check digit the first ten produce.
//
// The value is taken exactly as written, with no separators: "20-12345678-6"
// is rejected, because what is stored must be what was validated. The taxpayer
// type carried by the first two digits is deliberately not checked — the set
// of legal prefixes is administrative and changes without the arithmetic
// changing — so this rule proves the identifier is internally consistent, not
// that it was ever issued.
func Valid(v string) bool {
	if len(v) != 11 {
		return false
	}
	sum := 0
	for i, weight := range weights {
		digit := int(v[i] - '0')
		if digit < 0 || digit > 9 {
			return false
		}
		sum += digit * weight
	}
	last := int(v[10] - '0')
	if last < 0 || last > 9 {
		return false
	}
	return last == checkDigit(sum)
}

// checkDigit maps the weighted sum of the first ten digits to the eleventh.
// The two carry cases are the published ones: a remainder of zero yields a
// check digit of zero, and a remainder of one yields nine.
func checkDigit(sum int) int {
	switch digit := 11 - sum%11; digit {
	case 11:
		return 0
	case 10:
		return 9
	default:
		return digit
	}
}

// Rule is the directive tag `cuit`, ready to be handed to the generator.
var Rule = vogue.Rule{
	Name:  "cuit",
	Kinds: vogue.Kinds(vogue.String),
	Doc: "Requires an Argentine CUIT: eleven decimal digits whose last one is the check digit " +
		"the first ten produce. Separators are rejected, so the value that is validated is the " +
		"value that is stored; pair it with `trim` if the input comes from a form. The taxpayer " +
		"type in the first two digits is not checked, because that set is administrative rather " +
		"than arithmetic.",
	Message: "{{.Field}} must be a valid CUIT",
	Call:    &vogue.FuncRef{Path: "github.com/MathiasHilgert/vogue/examples/customrule/cuit", Name: "Valid"},
	Examples: vogue.Examples{
		Valid: []vogue.Example{
			{In: "20123456786", Note: "an individual CUIT with a correct check digit"},
			{In: "27123456780", Note: "a check digit that carries to zero"},
		},
		Invalid: []vogue.Example{
			{In: "20123456789", Note: "the right shape with the wrong check digit"},
			{In: "2012345678", Note: "ten digits, one short"},
			{In: "20-12345678-6", Note: "a punctuated CUIT, which is not what is stored"},
			{In: "2012345678X", Note: "a non-digit in the check position"},
			{In: "", Note: "the empty string"},
		},
	},
}
