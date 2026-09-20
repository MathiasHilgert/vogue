package rules

import (
	"hash/fnv"
	"strconv"
	"strings"

	"github.com/MathiasHilgert/vogue"
)

// Import paths the emitted expressions need, named once so an expression and
// the import it depends on cannot drift apart.
const (
	importDecimal = "github.com/govalues/decimal"
	importFn      = "github.com/MathiasHilgert/vogue/rules/fn"
	importRegexp  = "regexp"
	importStrings = "strings"
	importUnicode = "unicode"
	importUTF8    = "unicode/utf8"
)

// runeCount returns the expression counting the runes of the working value.
func runeCount(c vogue.EmitContext) string {
	return "utf8.RuneCountInString(" + c.Var + ")"
}

// noRuneWhere returns an expression that is true when no rune of the working
// value satisfies pred, which is the shape every character-class check has.
// The empty string satisfies it, since it holds no offending rune.
func noRuneWhere(c vogue.EmitContext, pred string) string {
	return "strings.IndexFunc(" + c.Var + ", func(r rune) bool { return " + pred + " }) < 0"
}

// compare returns the comparison a bound rule emits for the kind being
// generated: a rune count for a string, the value itself for an integer, and a
// three-way Cmp against a once-parsed bound for a decimal.
func compare(c vogue.EmitContext, op string) string {
	switch c.Kind {
	case vogue.Int:
		return c.Var + " " + op + " " + c.Param
	case vogue.Decimal:
		return c.Var + ".Cmp(" + decimalVar(c.Param) + ") " + op + " 0"
	default:
		return runeCount(c) + " " + op + " " + c.Param
	}
}

// declareBound returns the package-level declaration a decimal bound reads,
// and nothing on the kinds whose comparison needs no parsing.
//
// Parsing the bound once at process start rather than on every constructor
// call is the point of it: the parameter was already validated at generate
// time, so MustParse cannot panic on generated code, and the constructor is
// left with a comparison and no parsing at all.
func declareBound(c vogue.EmitContext) string {
	if c.Kind != vogue.Decimal {
		return ""
	}
	name := decimalVar(c.Param)
	return "// " + name + " is the bound " + strconv.Quote(c.Param) + ", parsed once.\n" +
		"var " + name + " = decimal.MustParse(" + strconv.Quote(c.Param) + ")"
}

// decimalVar returns the deterministic name of the package-level variable
// holding a parsed bound. As with [regexpVar] it is derived from the parameter
// itself, so two directives bounded by the same number share one declaration
// and a generated file is byte-stable across runs.
func decimalVar(param string) string {
	h := fnv.New32a()
	// hash.Hash never reports an error from Write.
	_, _ = h.Write([]byte(param))
	return "_vogueDecimal" + strconv.FormatUint(uint64(h.Sum32()), 16)
}

// signCompare returns the sign test a sign rule emits for the kind being
// generated.
func signCompare(c vogue.EmitContext, op string) string {
	if c.Kind == vogue.Decimal {
		return c.Var + ".Sign() " + op + " 0"
	}
	return c.Var + " " + op + " 0"
}

// anyOf returns the disjunction a membership rule emits: every item of the
// comma-separated parameter compared against the working value. String items
// are quoted, integer items are emitted as written, because the working value
// is an int64 there.
func anyOf(c vogue.EmitContext) string {
	items := strings.Split(c.Param, ",")
	terms := make([]string, len(items))
	for i, item := range items {
		if c.Kind == vogue.Int {
			terms[i] = c.Var + " == " + item
			continue
		}
		terms[i] = c.Var + " == " + strconv.Quote(item)
	}
	return strings.Join(terms, " || ")
}

// regexpVar returns the deterministic name of the package-level variable
// holding a compiled pattern. It is derived from the pattern itself, so two
// directives written against the same pattern share one compiled regexp and a
// generated file is byte-stable across runs.
func regexpVar(pattern string) string {
	h := fnv.New32a()
	// hash.Hash never reports an error from Write.
	_, _ = h.Write([]byte(pattern))
	return "_vogueRegexp" + strconv.FormatUint(uint64(h.Sum32()), 16)
}
