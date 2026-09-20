// Package testrules provides the minimal rule catalogue the generator tests
// run against. It exists so those tests stay independent of the real built-in
// catalogue, which is written later: a change to the shipped rules must not
// silently rewrite the generator golden files.
package testrules

import (
	"hash/fnv"
	"strconv"
	"strings"

	"github.com/MathiasHilgert/vogue"
)

// Set returns the catalogue used by the generator tests. The `nodigits` rule is
// delegated to a static function; callPath is the import path the generated
// code should call it through, which lets a test point the rule at a package of
// its own.
func Set(callPath string) *vogue.RuleSet {
	set := &vogue.RuleSet{}
	if err := set.Add(required(), trim(), lower(), minRule(), maxRule(), scale(), noDigits(callPath)); err != nil {
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
			Valid:   []vogue.Example{{In: "a", Note: "a single rune"}},
			Invalid: []vogue.Example{{In: "", Note: "the empty string"}},
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
		Examples: vogue.Examples{
			Normalized: []vogue.Normalization{
				{In: "  a  ", Out: "a", Note: "trim removes the blanks around a value"},
			},
		},
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
		Examples: vogue.Examples{
			Normalized: []vogue.Normalization{
				{In: "A", Out: "a", Note: "lower folds an upper-case value"},
			},
		},
	}
}

// minRule bounds a string by rune count and an int by magnitude.
func minRule() vogue.Rule {
	return vogue.Rule{
		Name:    "min",
		Kinds:   vogue.Kinds(vogue.String, vogue.Int, vogue.Decimal),
		Doc:     "Rejects values below the bound.",
		Param:   vogue.ParamSpec{Presence: vogue.ParamRequired, Type: vogue.ParamNumber},
		Message: "{{.Field}} must be at least {{.Param}}",
		Imports: []string{"unicode/utf8", "github.com/govalues/decimal"},
		Emit:    func(c vogue.EmitContext) string { return bound(c, ">=") },
		Declare: boundDecl,
		Examples: vogue.Examples{
			Valid: []vogue.Example{
				{Kinds: vogue.Kinds(vogue.String), Param: "1", In: "a"},
				{Kinds: vogue.Kinds(vogue.Int), Param: "1", In: "1"},
				{Kinds: vogue.Kinds(vogue.Decimal), Param: "0", In: "1.5", Note: "a weight above the floor"},
			},
			Invalid: []vogue.Example{
				{Kinds: vogue.Kinds(vogue.String), Param: "1", In: "", Note: "the empty string"},
				{Kinds: vogue.Kinds(vogue.Int), Param: "1", In: "0", Note: "a table with nobody at it"},
				{Kinds: vogue.Kinds(vogue.Decimal), Param: "0", In: "-0.25", Note: "a weight below zero"},
			},
		},
	}
}

// maxRule is the upper-bound counterpart of [minRule].
func maxRule() vogue.Rule {
	return vogue.Rule{
		Name:    "max",
		Kinds:   vogue.Kinds(vogue.String, vogue.Int, vogue.Decimal),
		Doc:     "Rejects values above the bound.",
		Param:   vogue.ParamSpec{Presence: vogue.ParamRequired, Type: vogue.ParamNumber},
		Message: "{{.Field}} must be at most {{.Param}}",
		Imports: []string{"unicode/utf8", "github.com/govalues/decimal"},
		Emit:    func(c vogue.EmitContext) string { return bound(c, "<=") },
		Declare: boundDecl,
		Examples: vogue.Examples{
			Valid: []vogue.Example{
				{Kinds: vogue.Kinds(vogue.String), Param: "2", In: "ab"},
				{Kinds: vogue.Kinds(vogue.String), Param: "120", In: "a tab name"},
				{Kinds: vogue.Kinds(vogue.Int), Param: "200", In: "200"},
			},
			Invalid: []vogue.Example{
				{Kinds: vogue.Kinds(vogue.String), Param: "2", In: "abc"},
				{Kinds: vogue.Kinds(vogue.Int), Param: "200", In: "201", Note: "one guest more than the house holds"},
			},
		},
	}
}

// scale bounds the number of decimal places of a decimal value.
func scale() vogue.Rule {
	return vogue.Rule{
		Name:    "scale",
		Kinds:   vogue.Kinds(vogue.Decimal),
		Doc:     "Rejects values carrying more decimal places than the bound.",
		Param:   vogue.ParamSpec{Presence: vogue.ParamRequired, Type: vogue.ParamInt},
		Message: "{{.Field}} must have at most {{.Param}} decimal places",
		Emit:    func(c vogue.EmitContext) string { return c.Var + ".Scale() <= " + c.Param },
		Examples: vogue.Examples{
			Valid:   []vogue.Example{{Param: "3", In: "1.25", Note: "two decimal places fit in three"}},
			Invalid: []vogue.Example{{Param: "3", In: "0.1234", Note: "four decimal places do not"}},
		},
	}
}

// bound emits the comparison a bound rule needs for the kind being generated:
// a rune count for strings, the value itself for integers, and a Cmp against a
// once-parsed package-level decimal for the decimal kind.
func bound(c vogue.EmitContext, op string) string {
	switch c.Kind {
	case vogue.Int:
		return c.Var + " " + op + " " + c.Param
	case vogue.Decimal:
		return c.Var + ".Cmp(" + boundVar(c.Param) + ") " + op + " 0"
	default:
		return "utf8.RuneCountInString(" + c.Var + ") " + op + " " + c.Param
	}
}

// boundDecl declares the parsed bound a decimal comparison reads.
func boundDecl(c vogue.EmitContext) string {
	if c.Kind != vogue.Decimal {
		return ""
	}
	return "var " + boundVar(c.Param) + " = decimal.MustParse(" + strconv.Quote(c.Param) + ")"
}

// boundVar names the declaration deterministically from the bound itself.
func boundVar(param string) string {
	h := fnv.New32a()
	_, _ = h.Write([]byte(param))
	return "_vogueDecimal" + strconv.FormatUint(uint64(h.Sum32()), 16)
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
			Invalid: []vogue.Example{{In: "a1", Note: "a value carrying a digit"}},
		},
	}
}

// NoDigits reports whether v contains no decimal digit. It is the function the
// `nodigits` rule dispatches to when a test points the rule at this package.
func NoDigits(v string) bool {
	return !strings.ContainsAny(v, "0123456789")
}
