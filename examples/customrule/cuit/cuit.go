// Package cuit is the predicate of the worked extensibility example: the
// Argentine CUIT, a taxpayer identifier no catalogue of general-purpose rules
// would ship.
//
// It is a pure function that generated code calls statically, so it lives in
// a package of its own that imports nothing of vogue: the generated domain
// package links this one, and must not pull the generator in with it. The
// [vogue.Rule] value pointing at it lives next door, in cuitrule.
package cuit

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
