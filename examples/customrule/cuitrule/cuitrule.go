// Package cuitrule is the [vogue.Rule] of the worked extensibility example:
// the directive tag `cuit`, pointing at the predicate in package cuit through
// [vogue.FuncRef].
//
// It is kept apart from the predicate on purpose. The generator binary imports
// this package, and so pulls vogue in; the generated domain package imports
// only cuit, so it never does. Nothing registers itself: the rule becomes real
// by being handed to the generator, which is what lets the generator reject a
// directive that names a rule nobody passed.
package cuitrule

import "github.com/MathiasHilgert/vogue"

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
