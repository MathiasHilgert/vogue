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
// generated: a rune count for a string, the value itself for an integer.
func compare(c vogue.EmitContext, op string) string {
	if c.Kind == vogue.Int {
		return c.Var + " " + op + " " + c.Param
	}
	return runeCount(c) + " " + op + " " + c.Param
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
