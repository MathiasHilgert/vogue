// Package testrules provides the minimal rule catalogue the generator tests
// run against. It exists so those tests stay independent of the real built-in
// catalogue, which is written later: a change to the shipped rules must not
// silently rewrite the generator golden files.
package testrules

import (
	"strings"

	"github.com/MathiasHilgert/vogue"
)

// Set returns the catalogue used by the generator tests. The `nodigits` rule is
// delegated to a static function; callPath is the import path the generated
// code should call it through, which lets a test point the rule at a package of
// its own.
func Set(callPath string) *vogue.RuleSet {
	set := &vogue.RuleSet{}
	if err := set.Add(required(), trim(), lower(), minRule(), maxRule(), noDigits(callPath)); err != nil {
		panic(err)
	}
	return set
}

// required rejects the empty string.
func required() vogue.Rule {
	return vogue.Rule{
		Name:    "required",
		Kinds:   vogue.Kinds(vogue.String),
		Doc:     "Rejects the empty value.",
		Message: "{{.Field}} is required",
		Emit:    func(c vogue.EmitContext) string { return c.Var + ` != ""` },
		Examples: vogue.Examples{
			Valid:   []vogue.Example{{In: "a"}},
			Invalid: []vogue.Example{{In: ""}},
		},
	}
}

// trim removes surrounding whitespace before the rules written after it run.
func trim() vogue.Rule {
	return vogue.Rule{
		Name:      "trim",
		Kinds:     vogue.Kinds(vogue.String),
		Doc:       "Removes leading and trailing whitespace.",
		Message:   "{{.Field}} is trimmed",
		Normalize: true,
		Imports:   []string{"strings"},
		Emit:      func(c vogue.EmitContext) string { return c.Var + " = strings.TrimSpace(" + c.Var + ")" },
	}
}

// lower folds the value to lower case before the rules written after it run.
func lower() vogue.Rule {
	return vogue.Rule{
		Name:      "lower",
		Kinds:     vogue.Kinds(vogue.String),
		Doc:       "Folds the value to lower case.",
		Message:   "{{.Field}} is lower-cased",
		Normalize: true,
		Imports:   []string{"strings"},
		Emit:      func(c vogue.EmitContext) string { return c.Var + " = strings.ToLower(" + c.Var + ")" },
	}
}

// minRule bounds a string by rune count and an int by magnitude.
func minRule() vogue.Rule {
	return vogue.Rule{
		Name:    "min",
		Kinds:   vogue.Kinds(vogue.String, vogue.Int),
		Doc:     "Rejects values below the bound.",
		Param:   vogue.ParamSpec{Presence: vogue.ParamRequired, Type: vogue.ParamInt},
		Message: "{{.Field}} must be at least {{.Param}}",
		Imports: []string{"unicode/utf8"},
		Emit:    func(c vogue.EmitContext) string { return bound(c, ">=") },
		Examples: vogue.Examples{
			Valid:   []vogue.Example{{Param: "1", In: "a"}},
			Invalid: []vogue.Example{{Param: "1", In: ""}},
		},
	}
}

// maxRule is the upper-bound counterpart of [minRule].
func maxRule() vogue.Rule {
	return vogue.Rule{
		Name:    "max",
		Kinds:   vogue.Kinds(vogue.String, vogue.Int),
		Doc:     "Rejects values above the bound.",
		Param:   vogue.ParamSpec{Presence: vogue.ParamRequired, Type: vogue.ParamInt},
		Message: "{{.Field}} must be at most {{.Param}}",
		Imports: []string{"unicode/utf8"},
		Emit:    func(c vogue.EmitContext) string { return bound(c, "<=") },
		Examples: vogue.Examples{
			Valid:   []vogue.Example{{Param: "2", In: "ab"}},
			Invalid: []vogue.Example{{Param: "2", In: "abc"}},
		},
	}
}

// bound emits the comparison a bound rule needs for the kind being generated:
// a rune count for strings, the value itself for integers.
func bound(c vogue.EmitContext, op string) string {
	if c.Kind == vogue.Int {
		return c.Var + " " + op + " " + c.Param
	}
	return "utf8.RuneCountInString(" + c.Var + ") " + op + " " + c.Param
}

// noDigits is the call-backed rule of the catalogue, exercising import
// collection and static dispatch in the generator.
func noDigits(callPath string) vogue.Rule {
	return vogue.Rule{
		Name:    "nodigits",
		Kinds:   vogue.Kinds(vogue.String),
		Doc:     "Rejects values containing a decimal digit.",
		Message: "{{.Field}} must not contain digits",
		Call:    &vogue.FuncRef{Path: callPath, Name: "NoDigits"},
		Examples: vogue.Examples{
			Valid:   []vogue.Example{{In: "abc"}},
			Invalid: []vogue.Example{{In: "a1"}},
		},
	}
}

// NoDigits reports whether v contains no decimal digit. It is the function the
// `nodigits` rule dispatches to when a test points the rule at this package.
func NoDigits(v string) bool {
	return !strings.ContainsAny(v, "0123456789")
}
